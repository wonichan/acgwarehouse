---
phase: 20-operations-monitoring
plan: 01
subsystem: api
tags: [go, gin, websocket, monitoring, sidecar]
requires:
  - phase: 15-compute-sidecar-infrastructure
    provides: sidecar runtime lifecycle and admin overview diagnostics
  - phase: 13-task-platform
    provides: task batch and task read models used by monitoring snapshots
provides:
  - monitoring event bus for real-time overview broadcasts
  - authenticated admin WebSocket endpoint for monitoring streams
  - sidecar restart endpoint with interrupted task impact reporting
  - app lifecycle wiring for monitoring bus startup and shutdown
affects: [phase-20-02, phase-20-03, flutter-monitoring-page]
tech-stack:
  added: [github.com/gorilla/websocket]
  patterns:
    - buffered event-bus fan-out for monitoring snapshots
    - WebSocket auth reusing admin Basic Auth semantics
    - paged running-task impact counting before destructive sidecar restart
key-files:
  created:
    - internal/service/monitoring_event_bus.go
    - internal/service/monitoring_event_bus_test.go
    - internal/handler/ws_handler.go
    - internal/handler/ws_handler_test.go
  modified:
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go
    - internal/handler/routes.go
    - internal/app/app.go
    - go.mod
    - go.sum
key-decisions:
  - "Monitoring snapshots are polled from AdminService and fanned out through a buffered in-process event bus."
  - "WebSocket connections reuse admin Basic Auth when credentials are configured."
  - "Sidecar restart impact is counted from paged running-task reads before Stop/Start execution."
patterns-established:
  - "Monitoring event delivery: poll overview -> marshal payload -> broadcast non-blocking to subscribers."
  - "WebSocket cleanup: reader goroutine signals disconnect and handler unsubscribes before returning."
requirements-completed: [OPS-01, OPS-02]
duration: session
completed: 2026-04-05
---

# Phase 20 Plan 01: Real-time monitoring backend summary

**Real-time monitoring backend now ships with an overview event bus, an authenticated monitoring WebSocket, and a sidecar restart endpoint that reports interrupted running-task impact before execution.**

## Performance

- **Completed:** 2026-04-05T18:39:46.0286887+08:00
- **Tasks:** 4
- **Files modified:** 10

## Accomplishments

- Added `MonitoringEventBus` snapshot polling, subscriber fan-out, and unsubscribe cleanup.
- Added `WSHandler` with WebSocket upgrade, event pumping, disconnect cleanup, and optional Basic Auth.
- Added `HandleSidecarRestart` with exact running-task impact counting plus Stop→Start orchestration.
- Wired monitoring bus lifecycle and new admin routes into `routes.go` and `app.go`.

## Task Commits

1. **Task 1 RED** — `5de7138` `test(20-01): add failing monitoring event bus coverage`
2. **Task 1 GREEN** — `7cab13b` `feat(20-01): add monitoring event bus with subscriber fan-out`
3. **Task 2 deps** — `5c1cd68` `deps(20-01): add gorilla/websocket for monitoring WebSocket`
4. **Task 2 RED** — `58c3475` `test(20-01): add failing WebSocket handler coverage`
5. **Task 2 GREEN** — `b137118` `feat(20-01): add WebSocket handler with event pump and auth`
6. **Task 3 RED** — `17e99eb` `test(20-01): add failing sidecar restart endpoint coverage`
7. **Task 3 GREEN** — `83a11d0` `feat(20-01): add sidecar restart endpoint with impact reporting`
8. **Task 4** — `3e626b2` `feat(20-01): wire monitoring WebSocket and sidecar restart routes`

## Files Created/Modified

- `internal/service/monitoring_event_bus.go` - polling event bus with buffered subscribers and controlled start/stop
- `internal/service/monitoring_event_bus_test.go` - RED/GREEN coverage for subscription, broadcast, unsubscribe, and polling
- `internal/handler/ws_handler.go` - authenticated WebSocket upgrade and event streaming handler
- `internal/handler/ws_handler_test.go` - upgrade, disconnect cleanup, and auth rejection coverage
- `internal/handler/admin_handler.go` - sidecar restart handler and runtime injection support
- `internal/handler/admin_handler_test.go` - restart impact, sequencing, missing-runtime, and failure-path coverage
- `internal/handler/routes.go` - `/admin/api/monitoring/ws` and `/admin/api/actions/sidecar/restart` wiring
- `internal/app/app.go` - monitoring bus initialization, dependency injection, startup loop, and shutdown cleanup
- `go.mod` - `github.com/gorilla/websocket` dependency
- `go.sum` - checksum entries for `github.com/gorilla/websocket`

## Decisions Made

- Used a buffered per-subscriber channel (`16`) so monitoring fan-out never blocks on a slow WebSocket client.
- Kept the WebSocket auth path aligned with existing admin Basic Auth behavior instead of introducing a separate auth contract.
- Counted running tasks through paged `GetTasks` reads so restart impact reporting remains exact without adding a new service method.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Captured `go.sum` alongside `go.mod` for the WebSocket dependency**
- **Found during:** Task 2 dependency installation
- **Issue:** `go get github.com/gorilla/websocket` updated both module files; committing only `go.mod` would leave the dependency change incomplete.
- **Fix:** Included `go.sum` in the dependency commit.
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `go build ./...` and handler tests passed with the resolved dependency graph.
- **Committed in:** `5c1cd68`

**2. [Rule 1 - Bug] Fixed the admin-service test double to honor task filters**
- **Found during:** Task 3 GREEN verification
- **Issue:** `mockAdminService.GetTasks` ignored the `Status` filter, causing restart tests to count non-running tasks.
- **Fix:** Applied batch/task-type/status filtering plus offset/limit slicing in the mock.
- **Files modified:** `internal/handler/admin_handler_test.go`
- **Verification:** `go test ./internal/handler/ -run "TestSidecarRestart" -count=1`
- **Committed in:** `83a11d0`

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both deviations were required for correctness and reproducible verification. No product-scope creep.

## Issues Encountered

- Existing unrelated workspace changes were present before execution (`.planning/ROADMAP.md`, `.planning/STATE.md`, `.gitignore`, docs, `config.yaml`, `data/`). Task commits staged only the plan-specific files.

## User Setup Required

None - no external service configuration required.

## Verification Evidence

- `go test ./internal/service/ -run "TestMonitoringEventBus" -count=1` → `ok   github.com/wonichan/acgwarehouse-backend/internal/service 0.147s`
- `go test ./internal/handler/ -run "TestWSHandler" -count=1` → `ok   github.com/wonichan/acgwarehouse-backend/internal/handler 0.207s`
- `go test ./internal/handler/ -run "TestSidecarRestart" -count=1` → `ok   github.com/wonichan/acgwarehouse-backend/internal/handler 0.176s`
- `go build ./...` → exit 0
- `go test ./internal/service/ ./internal/handler/ -count=1` → `ok   github.com/wonichan/acgwarehouse-backend/internal/service 8.810s` and `ok   github.com/wonichan/acgwarehouse-backend/internal/handler 2.555s`
- `lsp_diagnostics` on all touched Go files → no diagnostics found

## Self-Check

PASSED

- Monitoring event bus implemented and tested.
- Monitoring WebSocket route and handler implemented and authenticated.
- Sidecar restart endpoint implemented with interrupted-task impact reporting.
- App wiring added for monitoring bus startup and shutdown.
- Tracking files (`.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/REQUIREMENTS.md`) intentionally left untouched per user instruction.

## Next Phase Readiness

- Backend contracts for Phase 20 real-time monitoring are ready for downstream Flutter consumption.
- The remaining Phase 20 plans can build on `/admin/api/monitoring/ws`, `/admin/api/task-platform/overview`, and `/admin/api/actions/sidecar/restart` without further backend foundation work.
