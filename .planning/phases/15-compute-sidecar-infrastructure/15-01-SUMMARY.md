---
phase: 15-compute-sidecar-infrastructure
plan: 01
subsystem: infra
tags: [go, sidecar, lifecycle, health, degraded]

requires:
  - phase: 14-import-task-platform-polish
    provides: v3 task-platform orchestration baseline
provides:
  - Go-owned sidecar runtime state machine with startup timeout and shutdown reap guarantees
  - App lifecycle wiring for sidecar startup degraded/ready mode and idempotent stop
  - Layered health endpoint regression protection for Go-scoped semantics
affects: [phase-15-plan-02, phase-15-plan-03, phase-16]

tech-stack:
  added: []
  patterns: [stateful sidecar lifecycle, degraded availability signaling, layered health boundaries]

key-files:
  created:
    - internal/sidecar/runtime.go
    - internal/sidecar/runtime_test.go
    - internal/handler/health_handler_test.go
  modified:
    - internal/app/app.go
    - internal/app/bootstrap.go
    - internal/app/app_test.go
    - internal/handler/health_handler.go
    - internal/handler/routes_test.go

key-decisions:
  - "App startup records degraded mode when sidecar startup fails instead of aborting Go service startup."
  - "Sidecar shutdown enforces graceful attempt then kill/reap fallback to avoid leaked child processes."
  - "Base /health and /ready responses remain explicitly Go-scoped and do not expose sidecar diagnostics."

patterns-established:
  - "Go is lifecycle owner: sidecar start/stop runs inside app startup and shutdown flow."
  - "Degraded mode is explicit: sidecar failure does not imply Go liveness/readiness failure."

requirements-completed: [COMP-01, COMP-02]

duration: 8 h 11 m
completed: 2026-04-03
---

# Phase 15 Plan 01: Sidecar lifecycle infrastructure summary

**Go-owned sidecar runtime now enforces bounded startup, degraded transition, and shutdown reap semantics while app lifecycle and base health boundaries remain stable.**

## Performance

- **Duration:** 8 h 11 m
- **Started:** 2026-04-04T00:50:30+08:00
- **Completed:** 2026-04-03T17:02:01Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- Added `internal/sidecar/runtime.go` with explicit state transitions: `not_started`, `starting`, `ready`, `degraded`, `stopping`, `stopped`.
- Added runtime tests covering startup success, startup-timeout degraded behavior, and shutdown kill+wait/reap guarantees.
- Wired app-owned sidecar lifecycle into startup/shutdown with non-fatal degraded startup mode and idempotent sidecar stop.
- Added health regression tests and explicit Go-scoped payload markers for `/health` and `/ready`.

## Task Commits

1. **Task 1 RED** — `7ffd6e7` (`test`)
2. **Task 1 GREEN** — `c4cda77` (`feat`)
3. **Task 2 RED** — `ed502af` (`test`)
4. **Task 2 GREEN** — `03e8afd` (`feat`)
5. **Task 3 RED** — `c386347` (`test`)
6. **Task 3 GREEN** — `58ee99a` (`fix`)

## Files Created/Modified
- `internal/sidecar/runtime.go` - sidecar runtime lifecycle component
- `internal/sidecar/runtime_test.go` - lifecycle behavior coverage
- `internal/app/app.go` - startup/shutdown sidecar orchestration wiring
- `internal/app/bootstrap.go` - default sidecar runtime bootstrap assembly
- `internal/app/app_test.go` - app lifecycle degraded/ready/idempotent shutdown tests
- `internal/handler/health_handler.go` - explicit Go-scoped health payload fields
- `internal/handler/health_handler_test.go` - layered health semantic regressions
- `internal/handler/routes_test.go` - route registration and no direct Python route coupling assertion

## Decisions Made
- Keep sidecar startup failure non-fatal for app startup; mark degraded and keep Go available.
- Keep sidecar process lifecycle under `App` ownership with `shutdownOnce` to ensure exactly-once stop behavior.
- Keep base health endpoints Go-scoped and reserve sidecar diagnostics for admin-focused surfaces.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed startup-timeout cleanup hang in sidecar runtime**
- **Found during:** Task 1 GREEN verification (`go test ./internal/sidecar/...` timed out)
- **Issue:** Startup-timeout branch could wait forever if probe never succeeded but process was not force-terminated before wait.
- **Fix:** Added forced kill-and-reap path on startup-timeout transition to guarantee child-process cleanup.
- **Files modified:** `internal/sidecar/runtime.go`
- **Verification:** `go test ./internal/sidecar/... -run "Runtime|Lifecycle|Degraded|Shutdown" -count=1`
- **Committed in:** `c4cda77`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Auto-fix was required for correctness and process-lifecycle safety; no scope creep introduced.

## Authentication Gates

None.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 15-01 runtime/lifecycle/health foundation is complete and verified.
- Ready for `15-02-PLAN.md` runtime manifest and Flutter startup discovery work.

## Self-Check: PASSED

- Verified created/modified key files exist on disk.
- Verified all six task commit hashes are present in git history.
