# Design — ACG gallery landing page on 2017 behind nginx

## Architecture

```
Internet ──HTTPS──> nginx (public)                      nginx (local static)
                    listen 443/8443 ssl                 listen 127.0.0.1:2017
                    server_name acgwarehouse.cloud  ──>  root /opt/acgwarehouse/landing
                    location / proxy_pass :2017           index index.html
                    (replaces MinIO / and /console/)
         ──HTTP───> nginx listen 80/8080 -> 301 https
```

Single nginx instance (BT). Two server blocks in one vhost file:
1. Public TLS termination + reverse proxy to the internal static server.
2. Internal static file server on `127.0.0.1:2017`.

Rationale: D4 chose nginx-served static on 2017 over a Go/Python daemon — zero
extra process, starts with nginx, simplest to operate.

## Files / Boundaries

- Docroot: `/opt/acgwarehouse/landing/`
  - `index.html` — landing page markup (semantic sections).
  - `styles.css` — self-contained styling (responsive grid, hero, dark anime theme).
  - `app.js` — optional vanilla JS (lightbox/hover niceties); page must render without it.
  - Placeholder art: inline SVG / CSS gradients (no external CDN needed for core render).
- nginx vhost: `/www/server/panel/vhost/nginx/minio-proxy.conf`
  - Repurpose this active file. Keep a timestamped backup first.
  - Final state:
    - `server { listen 80; listen 8080; ... return 301 https://$host$request_uri; }` (kept)
    - `server { listen 443 ssl; listen 8443 ssl; http2 on; ... location / { proxy_pass http://127.0.0.1:2017; proxy_set_header Host/X-Real-IP/X-Forwarded-*; } }`
      - Remove MinIO `location /` (19005) and `location /console/` (19006).
      - Remove the now-unused `proxy_cache_path` / minio-cache directives.
    - `server { listen 127.0.0.1:2017; root /opt/acgwarehouse/landing; index index.html; location / { try_files $uri $uri/ /index.html; } }`
  - Consider renaming intent via a new file `acg-landing.conf` and removing the
    minio file — but since both are loaded by the same glob, simplest + lowest
    risk is to rewrite the existing `minio-proxy.conf` in place (keeps server_name
    ownership in one place, avoids duplicate-server_name warnings). Keep the
    `.bak` already present plus a fresh timestamped backup.

## Reverse proxy contract

- Upstream: `http://127.0.0.1:2017` (plain HTTP, loopback).
- Headers forwarded: `Host $host`, `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto $scheme`.
- No proxy cache needed for a tiny static page (drop minio-cache).

## Compatibility / Migration

- MinIO at 19005/19006 keeps running; it is only no longer exposed via this domain.
  Anything that relied on `https://acgwarehouse.cloud/<object>` or `/console/`
  will break by design (D2). Accepted.

## Rollback

- `cp minio-proxy.conf minio-proxy.conf.bak.<ts>` before edit.
- Rollback = restore the backup and `nginx -t && nginx -s reload`.
- Docroot is additive; removing `/www/wwwroot/acg-landing` is harmless.

## Risks

- Duplicate `server_name` if a second vhost also claims `acgwarehouse.cloud`.
  Mitigation: rewrite in place; `nginx -t` will warn on conflicts.
- SELinux/AppArmor on 2017: environment runs nginx as root master; loopback
  listen on 2017 is allowed (port is free). Verify with `nginx -t` + curl.
