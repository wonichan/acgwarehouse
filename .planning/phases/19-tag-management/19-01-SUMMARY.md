---
phase: 19-tag-management
plan: 01
subsystem: api
tags: [go, gin, sqlite, tag-governance]

requires: []
provides:
  - governance list contract with enriched tag rows and delete safety signals
  - explicit source-to-target tag merge endpoint with transactional reassignment
  - safe delete preview and selected cleanup batch contract
affects: [desktop-tag-management, tag-provider-governance]

tech-stack:
  added: []
  patterns: [admin service orchestration, preview-first destructive actions, explicit merge target only]

key-files:
  created: [internal/service/tag_admin_service.go]
  modified: [internal/service/tag_admin_service_test.go, internal/handler/tag_handler.go, internal/handler/tag_handler_test.go, internal/handler/routes.go]

key-decisions:
  - "Delete path is preview-first and blocks used tags with explicit blocking_reason"
  - "Merge remains explicit target selection; no fuzzy candidate auto-picking"
  - "Batch cleanup accepts selected tag_ids only and returns deleted/blocked/failed arrays"

patterns-established:
  - "Governance APIs return affected_image_count for destructive flows"
  - "Tag cleanup uses TagAdminService orchestration instead of handler-owned loops"

requirements-completed: [DSK-04]

duration: 11 min
completed: 2026-04-05
---

# Phase 19 Plan 01: Tag Governance Backend Contracts Summary

**Tag governance backend now ships explicit merge, delete preview, and selected cleanup contracts with safe-delete enforcement and affected-image transparency.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-05T11:41:16+08:00
- **Completed:** 2026-04-05T11:52:03+08:00
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added `TagAdminService` governance orchestration for list rows, explicit merge, delete preview, and selected cleanup.
- Exposed `GET /api/v1/tags/governance`, `POST /api/v1/tags/:id/merge`, `GET /api/v1/tags/:id/delete-preview`, and `POST /api/v1/tags/batch/cleanup` in handler/routes.
- Enforced safe delete semantics: in-use tags return `409` with `affected_image_count` and `merge_or_reclassify_required`; successful delete returns `affected_image_count: 0`.

## Task Commits

1. **Task 1 RED:** `9101666` — `test(19-01): add failing governance list and merge endpoint coverage`
2. **Task 1 GREEN:** `4158971` — `feat(19-01): add tag governance list and explicit merge contracts`
3. **Task 2 RED:** `b4f4417` — `test(19-01): add failing safe delete and cleanup coverage`
4. **Task 2 GREEN:** `fe2c225` — `feat(19-01): enforce safe delete preview and selected cleanup contracts`
5. **Task 2 REFACTOR:** `728000c` — `refactor(19-01): route tag cleanup through admin governance service`
6. **Rule 1 follow-up:** `4405228` — `fix(19-01): default blocked delete reason constant`

## Files Created/Modified
- `internal/service/tag_admin_service.go` - Governance orchestration and transactional merge/delete-preview/cleanup logic.
- `internal/service/tag_admin_service_test.go` - Service TDD coverage for governance list, merge, delete preview, and selected cleanup.
- `internal/handler/tag_handler.go` - Governance endpoints, safe delete contract enforcement, and selected cleanup response shaping.
- `internal/handler/tag_handler_test.go` - Handler coverage for governance list/merge/delete-preview/delete/cleanup contracts.
- `internal/handler/routes.go` - API route wiring for governance, merge, delete-preview, and selected cleanup.

## Decisions Made
- Used a dedicated admin service for governance contracts to keep handlers thin and deterministic.
- Kept merge target selection explicit-only, aligned with exact-match-first governance decisions.
- Dropped legacy cleanup route from active Phase 19 path in favor of selected cleanup endpoint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added explicit default blocking reason literal in handler**
- **Found during:** Task 2 acceptance check
- **Issue:** `internal/handler/tag_handler.go` did not contain explicit `merge_or_reclassify_required` literal required by acceptance criteria.
- **Fix:** Added `mergeOrReclassifyRequired` constant and fallback mapping in blocked-delete response.
- **Files modified:** `internal/handler/tag_handler.go`
- **Verification:** Targeted delete/cleanup suite and acceptance string checks passed.
- **Committed in:** `4405228`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** No scope creep; fix only aligned behavior and acceptance contract wording.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Backend contracts required by Phase 19 desktop governance work are available and test-covered.
- Ready for `19-02-PLAN.md` frontend service/provider governance orchestration.

## Verification Results
- `go test ./internal/service ./internal/handler -run "TestTag(AdminService|GetGovernance|Merge)" -count=1` ✅ pass
- `go test ./internal/service ./internal/handler -run "TestTag(Delete|Cleanup|DeletePreview)" -count=1` ✅ pass

## Self-Check: PASSED
