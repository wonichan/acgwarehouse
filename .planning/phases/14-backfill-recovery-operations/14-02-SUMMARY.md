---
phase: 14-backfill-recovery-operations
plan: 02
subsystem: testing, api
tags: [failure-isolation, failure-groups, retry-hints, batch-read-model, tdd]

# Dependency graph
requires:
  - phase: 11-task-platform-batch-model
    provides: batch/task model, TaskPlatformService, PlatformTask status lifecycle
  - phase: 13-backend-monitoring-queue-control
    provides: batch-first read model, TaskBatchReadRepository, TaskReadService
provides:
  - Isolation regression tests proving single-task failure does not block siblings
  - TaskBatchFailureGroup struct with reason_key, count, retry_recommended, retry_hint
  - LoadFailureGroups repository aggregation method
  - Deterministic retryability classifier (timeout/network=retryable, auth/config=not)
affects: [admin-ui, batch-monitoring, recovery-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Error prefix extraction: first token before ': ' in error_summary used as grouping key"
    - "Deterministic retry classification: backend-owned map of retryable vs non-retryable prefixes"

key-files:
  created: []
  modified:
    - internal/service/task_platform_service_test.go
    - internal/service/task_read_service.go
    - internal/service/task_read_service_test.go
    - internal/repository/task_batch_read_repository.go
    - internal/repository/image_repository.go

key-decisions:
  - "Failure groups aggregated by error_summary prefix (text before ': ') for deterministic grouping"
  - "Retryability classified by backend map, not client-side inference"
  - "Existing FailureSummary string preserved for backward compatibility alongside FailureGroups array"

patterns-established:
  - "Error prefix pattern: 'category: details' format enables automatic grouping and retry classification"

requirements-completed:
  - SAFE-01
  - SAFE-02

# Metrics
duration: 10min
completed: 2026-03-29
---

# Phase 14 Plan 02: Failure Isolation & Grouped Failure Summary Summary

**Single-image failure isolation regression tests + grouped failure summary contract with deterministic retry hints for batch-level admin visibility**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-29T04:10:16Z
- **Completed:** 2026-03-29T04:19:47Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added 3 isolation regression tests proving one failed task does not block siblings, does not affect unrelated tasks, and queue processing continues after failure
- Verified existing code already implements correct isolation semantics — no bugs found, tests serve as regression protection
- Added `TaskBatchFailureGroup` JSON contract with `reason_key`, `reason_label`, `count`, `retry_recommended`, `retry_hint`
- Added `LoadFailureGroups` repository method that aggregates failed tasks by error prefix with counts
- Added deterministic retryability classifier: timeout/rate_limit/network/connection = retryable; auth/config/malformed/missing = not retryable
- Preserved existing `failure_summary` string field for backward compatibility

## Task Commits

Each task was committed atomically:

1. **Task 1: Lock single-image failure isolation with regression tests** - `ce2d9ed` (test)
2. **Task 2: Add grouped failure summaries and retry hints to batch read model** - `f4a3488` (feat)

_Note: Task 1 tests passed immediately (GREEN confirmed), confirming existing isolation correctness. Task 2 combined RED+GREEN._

## Files Created/Modified
- `internal/service/task_platform_service_test.go` - Added 3 isolation regression tests (TestMarkJobFailedIsolation_OneFailsSiblingSucceeds, TestMarkJobFailedIsolation_OnlyAffectsOwnTask, TestIsolation_QueueProcessingContinuesAfterFailure)
- `internal/service/task_read_service.go` - Added TaskBatchFailureGroup struct, FailureGroups field, classifyFailureGroups function with retryability map
- `internal/service/task_read_service_test.go` - Added 3 failure group tests (TestTaskReadServiceFailureSummary_GroupedReasonsWithCounts, TestTaskReadServiceRetryHint_TransientVsNonRetryable, TestTaskReadServiceTaskReadReturnsErrorSummaryPerTask)
- `internal/repository/task_batch_read_repository.go` - Added FailureGroupRecord type, LoadFailureGroups interface method, SQL aggregation implementation
- `internal/repository/image_repository.go` - Added stub implementations for 5 backfill candidate methods (14-01 in progress)

## Decisions Made
- Error prefix extraction uses the text before the first `: ` in error_summary as the grouping key. This convention was already established by the existing error messages.
- Retryability is classified by a backend-owned map rather than requiring client-side inference, fulfilling D-16's requirement.
- The `FailureSummary` string is preserved alongside `FailureGroups` for backward compatibility, fulfilling D-17's batch-list level visibility while not breaking existing consumers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added stub implementations for backfill candidate methods**
- **Found during:** Task 1 (running tests)
- **Issue:** Parallel executor (14-01) had added interface methods to ImageRepository but not yet provided implementations, causing compilation failure
- **Fix:** Added 5 stub methods (FindBackfillCandidates, CountBackfillCandidates, CountBackfillSkippedWithAITag, CountBackfillSkippedWithActiveTask, CountBackfillHitCount) returning zero/nil
- **Files modified:** internal/repository/image_repository.go
- **Verification:** All tests compile and pass
- **Committed in:** ce2d9ed (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minimal — stubs will be replaced by 14-01's full implementation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Failure groups contract is ready for downstream admin UI rendering
- Retry hints are deterministic and backend-owned, admin UI only needs to display them
- Phase 14-03 can now build on this foundation for the backfill entry point

---
*Phase: 14-backfill-recovery-operations*
*Completed: 2026-03-29*

## Self-Check: PASSED

All files exist:
- internal/service/task_platform_service_test.go ✓
- internal/service/task_read_service.go ✓
- internal/service/task_read_service_test.go ✓
- internal/repository/task_batch_read_repository.go ✓
- 14-02-SUMMARY.md ✓

All commits found:
- ce2d9ed ✓
- f4a3488 ✓
