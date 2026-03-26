---
phase: 12-import-task-auto-scheduling
plan: 03
subsystem: infra
tags: [app-lifecycle, scheduler, hot-reload, ai-tagging]
requires:
  - phase: 12-02
    provides: AI auto scheduler service, config knobs, and enqueue logic
provides:
  - app-owned AI auto scheduler lifecycle wiring
  - guarded startup and shutdown for the import-scan scheduler
  - hot-reload restart behavior for enablement and scan setting changes
affects: [12-04, application-lifecycle, hot-reload]
tech-stack:
  added: []
  patterns: [app-owned background service lifecycle, restart-on-config-change scheduler reload]
key-files:
  created: []
  modified: [internal/app/app.go, internal/app/bootstrap.go, internal/app/app_test.go]
key-decisions:
  - "Keep the production field as `*service.AITagAutoScheduler` while routing lifecycle actions through app-owned helpers so tests can fake Start/Stop behavior."
  - "Treat `AutoAITagOnImport`, `AutoScanIntervalMinutes`, and `AutoScanBatchSize` changes as scheduler restart triggers to avoid stale config pointers after hot reload."
patterns-established:
  - "Background schedulers are constructed in `New()` and started/stopped from `Run()`/`Shutdown()` helpers instead of ad hoc goroutines."
  - "Lifecycle shutdown paths use app-level guards before closing shared channels or background services."
requirements-completed: [AIQ-01, AIQ-02]
duration: 40 min
completed: 2026-03-26
---

# Phase 12 Plan 03: 集成调度服务到应用启动流程 Summary

**Application lifecycle wiring now constructs, starts, stops, and hot-reloads the AI import-scan scheduler without double-start/double-stop races or stale scheduler config.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-03-26T15:56:09Z
- **Completed:** 2026-03-26T16:36:09Z
- **Tasks:** 4
- **Files modified:** 3

## Accomplishments
- Added `App.autoScheduler` construction in `New()` so the scheduler is always available to lifecycle hooks.
- Started the scheduler from `Run()` only when AI auto-tagging is enabled, with guards against duplicate starts.
- Stopped and reloaded the scheduler safely from `Shutdown()` and config hot-reload paths.

## task Commits

Each task was committed atomically:

1. **task 1: 在 App 结构添加 autoScheduler 字段** - `775811f` (`feat`)
2. **task 2: 在 Run 方法中启动调度服务** - `b5c284e` (`feat`)
3. **task 3: 在 Shutdown 方法中停止调度服务** - `5ed22f3` (`fix`)
4. **task 4: 添加配置热重载支持** - `8cb3cf2` (`feat`)

**Plan metadata:** pending

## Files Created/Modified
- `internal/app/app.go` - Adds the app-owned scheduler field, startup hook, and shutdown guard.
- `internal/app/bootstrap.go` - Centralizes scheduler construction plus start/stop/reload helper methods.
- `internal/app/app_test.go` - Covers scheduler init, guarded startup, safe shutdown, and config-reload restarts.

## Decisions Made
- Kept the concrete `autoScheduler *service.AITagAutoScheduler` field for plan compatibility while using small app helpers for lifecycle control and testability.
- Rebuilt the scheduler on scan-setting changes instead of mutating the existing instance because the service keeps a config pointer that would otherwise go stale after hot reload.
- Guarded `Shutdown()` with `sync.Once` so scheduler stop, config reloader stop, and refill channel close remain safe on repeated teardown calls.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Rebuilt the scheduler on hot reload to prevent stale config usage**
- **Found during:** task 4 (添加配置热重载支持)
- **Issue:** `AITagAutoScheduler` stores a config pointer, so only toggling flags in `App` would leave interval and batch-size changes stuck on the old instance.
- **Fix:** Restarted the scheduler whenever enablement, scan interval, or batch size changes and reinitialized it with the new config snapshot.
- **Files modified:** `internal/app/app.go`, `internal/app/bootstrap.go`, `internal/app/app_test.go`
- **Verification:** `go test ./internal/app/... -run "ConfigReload|OnChange" -count=1`
- **Committed in:** `8cb3cf2`

**2. [Rule 2 - Missing Critical] Added double-stop protection around shutdown lifecycle**
- **Found during:** task 3 (在 Shutdown 方法中停止调度服务)
- **Issue:** Repeated shutdown calls would re-close `refillStopCh` and could re-run background teardown, which is unsafe once the scheduler joins the app lifecycle.
- **Fix:** Wrapped shutdown teardown in `sync.Once` and stopped the scheduler before worker/database teardown.
- **Files modified:** `internal/app/app.go`, `internal/app/bootstrap.go`, `internal/app/app_test.go`
- **Verification:** `go test ./internal/app/... -run "Shutdown" -count=1`
- **Committed in:** `5ed22f3`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both fixes were required for correct lifecycle behavior under hot reload and repeated shutdown. No scope creep beyond the scheduler integration objective.

## Issues Encountered
- `go build ./cmd/server/...` emits a root `server.exe` artifact in this workspace, so it had to be removed between verification steps to keep the tree clean.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan `12-04` can now validate real application startup and import-scan auto enqueue behavior against a safe lifecycle-managed scheduler.
- The user pre-approved auto-advancing the `12-04` human-verify checkpoint, so downstream execution can continue without waiting for manual approval.

## Self-Check: PASSED

- Verified `.planning/phases/12-import-task-auto-scheduling/12-03-SUMMARY.md` exists on disk.
- Verified task commits `775811f`, `b5c284e`, `5ed22f3`, and `8cb3cf2` exist in git history.
