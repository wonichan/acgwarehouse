---
phase: 13-backend-monitoring-queue-control
plan: 02
subsystem: backend+admin-ui
tags: [admin, monitoring, queue, task-platform, api, javascript]
requires:
  - phase: 13-backend-monitoring-queue-control
    provides: 13-01 batch-first admin shell and task/batch page structure
provides:
  - platform overview contract with queue runtime + batch/task status counts
  - admin API endpoint `/admin/api/task-platform/overview`
  - phase-13 admin page wiring that prefers overview API over legacy summary derivation
affects: [13-03, 13-04, wave-3-control-actions]
tech-stack:
  added: []
  patterns: [TDD red-green commits, admin handler endpoint extension, phase-13 overview-first page loading]
key-files:
  created:
    - .planning/phases/13-backend-monitoring-queue-control/13-02-SUMMARY.md
  modified:
    - internal/service/admin_service.go
    - internal/service/admin_service_test.go
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go
    - internal/handler/routes.go
    - web/admin/app.js
    - .planning/STATE.md
    - .planning/ROADMAP.md
key-decisions:
  - "Overview contract aggregates from task-platform batch/task read models, not async_jobs-only guessing."
  - "Legacy `/admin/api/summary` remains intact for compatibility; Phase 13 page loads `/admin/api/task-platform/overview` first."
  - "Queue runtime exposes paused/running/queue size/worker count in one payload to support upcoming control actions."
patterns-established:
  - "Service-level overview paging aggregates all batch/task counts without truncating to a single page."
  - "Admin page top cards consume `queue`, `batches`, and `tasks` from overview payload while preserving existing batch-first UI sections."
requirements-completed: [PIPE-02, OPS-01]
duration: 22 min
completed: 2026-03-27
---

# Phase 13 Plan 02: Backend Platform Overview Contract + Admin Wiring Summary

Phase 13 now has a dedicated platform overview contract and endpoint, and the batch-first admin page top overview is wired to this new source instead of deriving platform state from old summary/job assumptions.

## Accomplishments

- Added `AdminService.GetTaskPlatformOverview` in `internal/service/admin_service.go` with queue runtime (`is_running`, `is_paused`, `queue_size`, `worker_count`) and platform counters (`batches`, `tasks`) aggregated from task-platform read models.
- Added TDD coverage in `internal/service/admin_service_test.go` for queue runtime + status aggregation, nil/empty read-model behavior, and legacy health/config/library preservation.
- Added `GET /admin/api/task-platform/overview` in `internal/handler/admin_handler.go` and registered it in `internal/handler/routes.go`.
- Added handler tests in `internal/handler/admin_handler_test.go` for 200 success payload and 500 error behavior.
- Updated `web/admin/app.js` so top overview loading prefers `/admin/api/task-platform/overview` and renders queue runtime + platform counters directly from the new contract.

## Task Commits

Each task was committed atomically (red then green):

1. **Task 1: 平台概览 service 聚合 DTO 与测试**
   - `6a2205e` — `test(13-02): add failing platform overview service coverage`
   - `6d4b5cc` — `feat(13-02): add task platform overview aggregation`
2. **Task 2: handler/routes/frontend 接线**
   - `b7f2833` — `test(13-02): add failing task platform overview handler coverage`
   - `69981e1` — `feat(13-02): expose task platform overview API to admin page`

## Verification Evidence

Executed exact plan verification commands and recorded output:

1. `go test ./internal/service/... -run "Admin|TaskRead" -count=1`
   - `ok   github.com/wonichan/acgwarehouse-backend/internal/service  1.563s`
2. `go test ./internal/handler/... -run "AdminHandler|Routes" -count=1 && go test ./internal/service/... -run "Admin|TaskRead" -count=1`
   - `ok   github.com/wonichan/acgwarehouse-backend/internal/handler  0.236s`
   - `ok   github.com/wonichan/acgwarehouse-backend/internal/service  1.321s`
3. `go test ./internal/service/... ./internal/handler/... -run "Admin|TaskRead|Routes" -count=1`
   - `ok   github.com/wonichan/acgwarehouse-backend/internal/service  1.621s`
   - `ok   github.com/wonichan/acgwarehouse-backend/internal/handler  0.230s`

Additional checks:

- `lsp_diagnostics` reports no errors for modified Go files (`internal/service/admin_service.go`, `internal/service/admin_service_test.go`, `internal/handler/admin_handler.go`, `internal/handler/admin_handler_test.go`, `internal/handler/routes.go`).
- JavaScript LSP diagnostics are unavailable in this environment because `typescript-language-server` is not installed.

## Deviations from Plan

- Auto-fixed test harness stability issue: switched service test DB helper to temporary-file SQLite database in `internal/service/admin_service_test.go` to avoid `:memory:` multi-connection table visibility issues during aggregation queries.
- No scope expansion beyond 13-02 objectives.

## Progress Updates

- Updated `.planning/ROADMAP.md` Phase 13 plan progress from `1/4` to `2/4`.
- Updated `.planning/STATE.md` to `Completed 13-02-PLAN.md`, with Phase 13 current position moved to plan `3 of 4`.

## Wave 3 Readiness

Wave 3 can now build queue control actions directly on the new overview contract:

- queue runtime flags and counts are available in one endpoint,
- batch/task status distributions are platform-derived,
- Phase 13 page top cards already consume the new payload.
