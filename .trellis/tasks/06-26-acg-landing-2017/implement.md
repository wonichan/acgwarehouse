# Implement — ACG gallery landing page on 2017 behind nginx

## Ordered checklist

1. [ ] Create docroot `/opt/acgwarehouse/landing/`.
2. [ ] Write `index.html` — hero + responsive gallery grid (placeholder art) +
       features + footer. Anime/二次元 visual theme. Renders without JS.
3. [ ] Write `styles.css` — self-contained responsive styling.
4. [ ] Write `app.js` — optional progressive enhancement (lightbox/filter); page
       works if it fails to load.
5. [ ] Backup active vhost: `cp /www/server/panel/vhost/nginx/minio-proxy.conf
       /www/server/panel/vhost/nginx/minio-proxy.conf.bak.<ts>`.
6. [ ] Rewrite `/www/server/panel/vhost/nginx/minio-proxy.conf`:
       - keep 80/8080 → 301 https server
       - 443/8443 ssl server: `location /` → `proxy_pass http://127.0.0.1:2017`,
         remove MinIO `/` and `/console/`, drop minio-cache directives
       - add `127.0.0.1:2017` static server with `try_files`
7. [ ] `nginx -t`.
8. [ ] `nginx -s reload`.

## Validation commands

```bash
# AC5
nginx -t

# AC1 — local static server
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:2017/
curl -s http://127.0.0.1:2017/ | grep -i '<title'

# AC2 — public HTTPS path serves landing, not MinIO XML
curl -sk --resolve acgwarehouse.cloud:443:127.0.0.1 https://acgwarehouse.cloud/ | grep -i '<title'

# AC3 — 80 -> 301 https
curl -s -o /dev/null -w '%{http_code} %{redirect_url}\n' \
  --resolve acgwarehouse.cloud:80:127.0.0.1 http://acgwarehouse.cloud/

# AC4 — /console/ no longer hits MinIO console
curl -sk -o /dev/null -w '%{http_code}\n' \
  --resolve acgwarehouse.cloud:443:127.0.0.1 https://acgwarehouse.cloud/console/

# AC6 — assets load
curl -s -o /dev/null -w 'css %{http_code}\n' http://127.0.0.1:2017/styles.css
```

## Risky points / rollback

- Editing the live vhost is the only risky step. Backup taken in step 5.
- Rollback: restore `.bak.<ts>`, `nginx -t && nginx -s reload`.
- If `nginx -t` fails after rewrite, do NOT reload; fix or rollback first.

## Notes

- Static HTML/CSS/JS are not covered by the Go programming skill's toolchain;
  no go.mod involved. Keep files small and self-contained.
