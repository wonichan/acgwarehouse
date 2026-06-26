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

---

## Session 2: ACG gallery backend stages 00-01

**Date**: 2026-06-26
**Task**: ACG图库后端服务 (`.trellis/tasks/06-26-acg-gallery-backend`)
**Branch**: `master`

### Summary

Started the backend Trellis task and implemented stages 00/01 only. Stage 00 added the Go module, Hertz web skeleton, env config, zap logger wrapper, unified error codes/response helpers, CORS middleware, SQLite WAL read/write pool setup, `/api/v1/ping`, and graceful shutdown hooks. Stage 01 added user po/do/dto, user repository/service, bcrypt registration/login, local HS256 JWT sign/parse with strict `typ=JWT` + `alg=HS256` header validation, Auth/RequireAdmin middleware, register/login/me routes, admin bootstrap, and unit tests for repository/service/JWT behavior.

### Main Changes

- Created Go backend module `github.com/yachiyo/acgwarehouse` with `cmd/`, `internal/`, and `pkg/` layout.
- Implemented foundation files under `internal/conf`, `internal/handler`, `internal/infra/db`, `pkg/errors`, and `pkg/logger`.
- Implemented authentication files under `internal/model/{do,dto,po}`, `internal/repository`, `internal/service`, `internal/handler`, `internal/handler/middleware`, and `pkg/jwt`.
- Addressed review blocker: `pkg/jwt.Manager.Parse` now decodes and validates JOSE header `typ=JWT` and `alg=HS256`; tests cover invalid alg/type/malformed header.
- Updated stage 00/01 implementation checklist and ran `codegraph sync` after stages.

### Verification

- [OK] `go mod tidy`
- [OK] `go test ./pkg/jwt`
- [OK] `go test ./...`
- [OK] `go build ./...`
- [OK] `go vet ./...`
- [OK] `gofmt -s -l .` produced no output
- [OK] Runtime smoke test on temp DB/port: `/api/v1/users/me` without token returned 401; register returned 200; login returned JWT; `/me` with Bearer token returned 200; admin bootstrap login returned 200; SIGINT graceful shutdown exited 0.
- [INFO] `gopls_go_diagnostics` returned `no views` in this environment.

### Status

[OK] **Stages 00 and 01 completed; stages 02+ not started.**

### Next Steps

- Continue with stage 02 image sync when requested.
- Before commit/finish, run full Trellis check and decide whether to update backend spec for the local JWT implementation / deferred dependency strategy.

