# Backend Operations and API Logging Implementation Plan

> **For agentic workers:** REQUIRED: Use `trellis-before-dev` before editing Go files, then use TDD. Steps use checkbox syntax for tracking.

**Goal:** Add file-based structured operations logs and `/api/v1` access logs for the Go Hertz backend.

**Architecture:** Keep the existing Zap-backed `pkg/logger` API, replace stdout/stderr output with rotatefile-backed app/access files, and register a custom access log middleware only on the `/api/v1` route group. Operations logs stay in app logs; access logs use a dedicated access logger.

**Tech Stack:** Go, Hertz, Zap, `gookit/rotatefile`, Trellis specs, stdlib `testing`.

---

## Files

- Modify: `go.mod`, `go.sum` — add `github.com/gookit/rotatefile`.
- Modify: `pkg/logger/logger.go` — configure app/access Zap loggers, rotatefile writers, and access logging API.
- Test: `pkg/logger/logger_test.go` — verify file-only output, app/access split, and logger reset behavior.
- Create: `internal/handler/middleware/access_log.go` — `/api/v1` access log middleware.
- Test: `internal/handler/middleware/access_log_test.go` — required fields, forbidden data, byte counts, status, duration.
- Modify: `internal/handler/router/router.go` — attach access log middleware only to `/api/v1` group.
- Test: `internal/handler/router/*_test.go` or new `access_log_test.go` — route-scope smoke if practical.
- Modify: `cmd/web/main.go` — initialize file logging and add lifecycle/background task logs.

## Task 1: Prepare logging dependency and logger file-output tests

- [ ] Add `github.com/gookit/rotatefile` to `go.mod` with `go get` or equivalent module update.
- [ ] Write failing tests in `pkg/logger/logger_test.go` for app/access file separation using `t.TempDir()`.
- [ ] Test that app logger output does not appear in access file and access logger output does not appear in app file.
- [ ] Test that no stdout/stderr writer is configured indirectly by writing logs and asserting only temp log files receive entries.
- [ ] Run: `go test ./pkg/logger/...` and confirm failure for missing rotatefile/file logging behavior.

## Task 2: Implement rotatefile-backed logger setup

- [ ] Extend `pkg/logger` with a configuration value or constructor inputs for log directory and rotation settings.
- [ ] Create `data/log` or configured log directory with `os.MkdirAll` during logger setup.
- [ ] Configure app writer with rotatefile ModeCreate daily files, 100MB max size, compression enabled.
- [ ] Configure access writer with the same rotation/compression policy.
- [ ] Keep `logger.Info/Warn/Error` writing only to app logger.
- [ ] Add a dedicated access logging function that writes only to access logger at Info level.
- [ ] Preserve `ReplaceGlobal`/test reset behavior so tests remain isolated.
- [ ] Run: `go test ./pkg/logger/...` and make the logger tests pass.

## Task 3: Add access log middleware with tests first

- [ ] Write failing tests in `internal/handler/middleware/access_log_test.go` using a Hertz test engine and test logger output.
- [ ] Cover: required fields `route/method/path/client_ip/user_agent/request_body_bytes/response_body_bytes/duration_ms/status_code`.
- [ ] Cover: status code and response bytes are captured after handler execution.
- [ ] Cover: negative/unknown request Content-Length logs `request_body_bytes=0`.
- [ ] Cover: query string, request body, response body, Authorization and Cookie values are absent from access log output.
- [ ] Run: `go test ./internal/handler/middleware/...` and confirm failure for missing middleware.
- [ ] Implement `middleware.AccessLog()` with `time.Now()` before `ctx.Next(c)` and post-response field collection.
- [ ] Use `ctx.FullPath()` with fallback to `path`; use `ctx.Path()` so query is not logged.
- [ ] Run: `go test ./internal/handler/middleware/...` and make tests pass.

## Task 4: Register middleware only under `/api/v1`

- [ ] Modify `internal/handler/router/router.go` so `v1 := engine.Group("/api/v1")` uses `middleware.AccessLog()` before route registration.
- [ ] Add or update router smoke tests to verify `/api/v1/ping` emits access log.
- [ ] Add or update route-scope test to verify a non-`/api/v1` route does not emit access log if practical in current router test harness.
- [ ] Run: `go test ./internal/handler/router/...`.

## Task 5: Add operations lifecycle logs

- [ ] In `cmd/web/main.go`, log non-sensitive startup/config summary after logger setup.
- [ ] Log SQLite opened and search index opened after successful initialization.
- [ ] Log RankingJob and ViewBuffer started after `Start(ctx)`.
- [ ] Log graceful shutdown started and completed in shutdown hook path.
- [ ] Keep scope limited to lifecycle/background status; do not add login/favorite/rating audit logs.
- [ ] Run: `go test ./cmd/web/...`.

## Task 6: Full validation and review

- [ ] Run: `go test ./pkg/logger/...`.
- [ ] Run: `go test ./internal/handler/middleware/...`.
- [ ] Run: `go test ./internal/handler/router/...`.
- [ ] Run: `go test ./cmd/web/...`.
- [ ] Run: `go test ./...`.
- [ ] If available, run stricter Go checks from project/tooling docs; otherwise report they are unavailable.
- [ ] Inspect changed files for no `fmt.Println`, native `log.Print`, sensitive data logging, query/body/header leakage, or stdout/stderr outputs.
- [ ] Confirm generated log filenames follow `app.YYYYMMDD.log` and `access.YYYYMMDD.log` under `data/log`.

## Risks and rollback points

- Global logger replacement can make tests order-sensitive; isolate with reset functions and temp directories.
- Access logs must use the dedicated access logger, not the app logger, to avoid file mixing.
- `ctx.Response.BodyBytes()` may be empty for streaming responses; logging `0` is acceptable.
- `ContentLength()` may be negative for chunked/unknown request bodies; normalize to `0`.
- If rotatefile setup fails, service should fail startup instead of silently writing stdout.
- Rollback by removing rotatefile dependency, logger file setup, access middleware registration, and lifecycle log additions.
