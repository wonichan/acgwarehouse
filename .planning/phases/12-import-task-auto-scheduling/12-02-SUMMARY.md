---
phase: 12-import-task-auto-scheduling
plan: 02
subsystem: infra
tags: [ai-tagging, scheduler, config, task-platform]
requires:
  - phase: 12-01
    provides: image tag source semantics and eligible-image query support
provides:
  - AI auto scheduling configuration under the existing ai config block
  - periodic import-scan scheduler for AI tag platform tasks
  - AI-tag source persistence in the governance save path
affects: [12-03, 12-04, admin-observability]
tech-stack:
  added: []
  patterns: [nested AI scheduling config, ticker-driven background scheduler, task-platform enqueue reuse]
key-files:
  created: [internal/service/ai_tag_auto_scheduler.go, internal/service/ai_tag_auto_scheduler_test.go]
  modified: [internal/config/config.go, internal/config/config_test.go, deploy/config/config.example.yaml, internal/service/tag_governance_service.go, internal/worker/ai_tag_handler_test.go]
key-decisions:
  - "Nested the auto scheduling settings under config.AI to match the existing config shape instead of introducing a new top-level block."
  - "Queued AI auto-scan work through TaskPlatformService.PlanBatch and QueueTask so dedupe and lifecycle rules stay centralized."
  - "Marked AI-generated image tags in TagGovernanceService because that is the actual persistence boundary for worker-produced tags."
patterns-established:
  - "AI auto scheduling config lives under the ai section with env overrides for each operational knob."
  - "Background schedulers should depend on narrow interfaces and an injected ticker for deterministic tests."
requirements-completed: [AIQ-01]
duration: 5 min
completed: 2026-03-26
---

# Phase 12 Plan 02: 定时扫描服务实现 Summary

**AI auto scheduling config, periodic import-scan enqueueing, and AI-source tag persistence for the task platform**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-26T16:12:21Z
- **Completed:** 2026-03-26T16:17:59Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- Added AI auto-scheduling settings to `config.AI` with defaults, env overrides, and example config coverage.
- Implemented `AITagAutoScheduler` with configurable scan limits and ticker-driven enqueue behavior.
- Fixed worker-driven tag persistence so AI-generated tags save with `source='ai'`.

## task Commits

Each task was committed atomically:

1. **task 1: 添加 auto_ai_tag_on_import 配置项** - `810b283` (`feat`)
2. **task 2: 实现 AITagAutoScheduler 服务** - `f3bec4c` (`feat`)
3. **task 3: 更新 AI 标签保存时设置 source='ai'** - `77a4a31` (`fix`)

**Plan metadata:** pending

## Files Created/Modified
- `internal/config/config.go` - Adds nested AI auto-scheduling fields, defaults, and env overrides.
- `internal/config/config_test.go` - Verifies config parsing, defaults, and overrides for auto scheduling.
- `deploy/config/config.example.yaml` - Documents the example AI auto-scheduling settings.
- `internal/service/ai_tag_auto_scheduler.go` - Scans eligible images and queues AI tag tasks through the platform service.
- `internal/service/ai_tag_auto_scheduler_test.go` - Covers scan limits, enqueue behavior, lifecycle control, and disabled mode.
- `internal/service/tag_governance_service.go` - Persists AI-generated tag associations with `source='ai'`.
- `internal/worker/ai_tag_handler_test.go` - Adds regression coverage for AI-source persistence and updates the job repo mock.

## Decisions Made
- Nested the new settings under `config.AI` because the repo already keeps AI operational knobs in the `ai:` config section.
- Reused `TaskPlatformService` instead of hand-rolling queue writes so import-scan AI tasks inherit existing dedupe and status behavior.
- Applied the `source='ai'` fix in `TagGovernanceService` because worker code delegates the actual `image_tags` save there.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Aligned scheduler config with the existing `ai:` config shape**
- **Found during:** task 1 (添加 auto_ai_tag_on_import 配置项)
- **Issue:** The plan text suggested a new top-level scheduler config block, but the repo already groups AI operational settings under `config.AI`.
- **Fix:** Added `AutoAITagOnImport`, `AutoScanIntervalMinutes`, and `AutoScanBatchSize` to `AIConfig` instead of introducing a parallel top-level structure.
- **Files modified:** `internal/config/config.go`, `internal/config/config_test.go`, `deploy/config/config.example.yaml`
- **Verification:** `go test ./internal/config/... -run "AutoAITag" -count=1`
- **Committed in:** `810b283`

**2. [Rule 1 - Bug] Fixed AI tag source persistence at the real save boundary**
- **Found during:** task 3 (更新 AI 标签保存时设置 source='ai')
- **Issue:** The plan pointed at `internal/worker/ai_tag_handler.go`, but the `image_tags` rows are persisted by `TagGovernanceService`, which was still defaulting AI-created rows to `manual`.
- **Fix:** Set `Source` to `domain.ImageTagSourceAI` when an observation-backed AI merge is saved.
- **Files modified:** `internal/service/tag_governance_service.go`, `internal/worker/ai_tag_handler_test.go`
- **Verification:** `go test ./internal/worker/... -run "AITag" -count=1`
- **Committed in:** `77a4a31`

**3. [Rule 3 - Blocking] Updated the AI worker mock to match the current job repository interface**
- **Found during:** task 3 (更新 AI 标签保存时设置 source='ai')
- **Issue:** `mockJobRepoForAI` was missing `FindByPlatformTaskID`, which blocked the worker test suite from compiling before the new regression could run.
- **Fix:** Added the missing mock method in the existing test helper.
- **Files modified:** `internal/worker/ai_tag_handler_test.go`
- **Verification:** `go test ./internal/worker/... -run "AITag" -count=1`
- **Committed in:** `77a4a31`

---

**Total deviations:** 3 auto-fixed (2 bug, 1 blocking)
**Impact on plan:** All deviations were required to fit the existing codebase shape and keep the plan verifiable without scope creep.

## Issues Encountered
- Worker tests exposed a stale mock interface before the AI-source regression could execute; updating the mock unblocked normal verification.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The scheduler service and config knobs are ready to be wired into application startup in plan `12-03`.
- End-to-end auto-enqueue validation can now build on deterministic scheduler and source-persistence behavior.

## Self-Check: PASSED

- Verified `12-02-SUMMARY.md` exists on disk.
- Verified task commits `810b283`, `f3bec4c`, and `77a4a31` exist in git history.
