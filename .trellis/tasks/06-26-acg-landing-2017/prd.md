# ACG gallery landing page on 2017 behind nginx

## Goal

Serve a static "二次元图库浏览" (anime gallery browsing) landing page at
`https://acgwarehouse.cloud`, hosted by an nginx static server on port 2017
and reverse-proxied by the public-facing nginx HTTPS server.

User value: visitors hitting the domain see a polished gallery landing page
instead of a raw MinIO S3 API response.

## Background / Confirmed Facts

- Environment is BT panel (宝塔) nginx 1.26.3.
  - Active master config: `/www/server/nginx/conf/nginx.conf`
  - Active vhost dir: `/www/server/panel/vhost/nginx/*.conf`
  - The `/etc/nginx/*` tree is NOT the active config (no `/etc/nginx/nginx.conf`).
- Current active vhost `/www/server/panel/vhost/nginx/minio-proxy.conf`:
  - `80`/`8080` → 301 redirect to HTTPS for `acgwarehouse.cloud www.acgwarehouse.cloud`.
  - `443 ssl` / `8443 ssl`, `http2 on`, same server_name.
    - cert: `/etc/ssl/cloudflare/cert.pem` + `/etc/ssl/cloudflare/privkey.pem` (working, exp 2036).
    - `location /` → `proxy_pass http://127.0.0.1:19005/` (MinIO S3 API, cached).
    - `location /console/` → `proxy_pass http://127.0.0.1:19006/` (MinIO Console).
- Ports currently listening: 80, 443, 8080 (nginx). Port 2017 is FREE.
- A stale duplicate exists in `/etc/nginx/` (separate cert `/etc/nginx/ssl/acgwarehouse.*`),
  but it is not loaded by the running nginx; do not edit it.

## Decisions (from brainstorm)

- D1: Root `/` MinIO S3 image API is taken DOWN, replaced by the landing page.
- D2: MinIO is fully removed from this domain — both `/` and `/console/` are dropped.
  The domain serves only the landing page.
- D3: Landing page is purely static (placeholder/sample art demonstrating a gallery
  browsing experience). No backend, no real image data fetch.
- D4: The 2017 static service is hosted by a new nginx server block listening on
  `127.0.0.1:2017` serving static files directly (no Go/Python process).
- D5: HTTPS keeps the existing working cert `/etc/ssl/cloudflare/*.pem` and the
  existing 80→443 redirect.

## Requirements

- R1: Create a static landing page (HTML/CSS, optional vanilla JS) for an anime
  gallery browse experience: hero section, responsive image grid with
  placeholder/sample art, feature highlights, footer. Self-contained assets
  (no external CDN dependency required to render core layout).
- R2: Static files live at docroot `/opt/acgwarehouse/landing`.
- R3: An nginx server block listens on `127.0.0.1:2017` and serves the docroot
  with `index index.html` and a sensible `try_files`.
- R4: The public `443`/`8443` server for `acgwarehouse.cloud www.acgwarehouse.cloud`
  reverse-proxies `location /` to `http://127.0.0.1:2017`, replacing the MinIO
  `location /` and removing `location /console/`.
- R5: 80→443 redirect remains intact.
- R6: A backup of the current `minio-proxy.conf` is kept before editing.
- R7: `nginx -t` passes and nginx is reloaded.

## Acceptance Criteria

- [ ] AC1: `curl -s http://127.0.0.1:2017/` returns HTTP 200 and the landing page HTML.
- [ ] AC2: `curl -sk -H 'Host: acgwarehouse.cloud' https://127.0.0.1/` (or via
      `--resolve`) returns the landing page HTML, NOT a MinIO/S3 XML response.
- [ ] AC3: `curl -s -o /dev/null -w '%{http_code} %{redirect_url}' -H 'Host: acgwarehouse.cloud' http://127.0.0.1/`
      shows a 301 to `https://acgwarehouse.cloud/`.
- [ ] AC4: `curl -sk -H 'Host: acgwarehouse.cloud' https://127.0.0.1/console/`
      no longer reaches MinIO console (404/landing behavior, not 19006).
- [ ] AC5: `nginx -t` passes; nginx reloaded without error.
- [ ] AC6: Landing page renders correctly (layout intact) when opened — verified
      via served HTML structure and asset 200s.

## Out of Scope

- Real image data / MinIO integration on this domain.
- Certificate issuance/renewal (existing cert reused).
- Editing the stale `/etc/nginx/` tree.
- Migrating MinIO to a different host/path (it is simply removed from this domain).

## Open Questions

- None blocking.
