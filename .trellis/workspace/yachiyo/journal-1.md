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



## Session 2: ACG gallery backend stages 04-07 complete

**Date**: 2026-06-27
**Task**: ACG gallery backend stages 04-07 complete
**Branch**: `master`

### Summary

Implemented tag, rating, collection, and ranking modules with TDD. Each stage added model/DTO/PO, repository/service/handler/tests, HTTP routes, DB migration, updated code-spec, and recorded task artifact. All Go tests/build/vet/gofmt pass.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f49f074` | (see git log) |
| `b6fffe3` | (see git log) |
| `f930458` | (see git log) |
| `cdc237f` | (see git log) |
| `3a6d5a8` | (see git log) |
| `ceaec96` | (see git log) |
| `ffc5c6e` | (see git log) |
| `f4c3eb8` | (see git log) |
| `0756506` | (see git log) |
| `a596f9d` | (see git log) |
| `446b1e3` | (see git log) |
| `d83e6cc` | (see git log) |
| `58ea428` | (see git log) |
| `2cc948e` | (see git log) |
| `ec18b38` | (see git log) |
| `52d36af` | (see git log) |
| `88ac855` | (see git log) |
| `b3cc4ac` | (see git log) |
| `a4fa436` | (see git log) |
| `bbecbd6` | (see git log) |
| `822f363` | (see git log) |
| `bd800ff` | (see git log) |
| `252805a` | (see git log) |
| `73008e2` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete
