# Journal - yachiyo (Part 1)

> AI development session journal
> Started: 2026-06-26

---



## Session 1: ACG gallery landing page on 2017 behind nginx

**Date**: 2026-06-26
**Task**: ACG gallery landing page on 2017 behind nginx
**Branch**: `master`

### Summary

Built a self-contained static anime gallery landing page (landing/index.html, styles.css, app.js) served by nginx on 127.0.0.1:2017 and reverse-proxied at https://acgwarehouse.cloud. Replaced the MinIO S3 root proxy and removed /console/. Fixed a Cloudflare Flexible-mode ERR_TOO_MANY_REDIRECTS loop by making port 80 honor X-Forwarded-Proto instead of unconditionally 301->https; user switched Cloudflare to Full mode and public access verified 200. git init'd the repo (first commit 23923b1). Also updated local Go toolchain 1.26.0->1.26.4 earlier in the session.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `23923b1` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete
