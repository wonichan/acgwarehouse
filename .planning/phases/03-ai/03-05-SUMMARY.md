---
phase: 03-ai
plan: 05
subsystem: ai
tags: [ai, governance, worker, aliases, image-tags]
requires:
  - phase: 03-01
    provides: AI provider abstraction and async worker registration
  - phase: 03-02
    provides: tag repositories, alias normalization, and governance merge service
  - phase: 03-03
    provides: runtime server dependency wiring for tag APIs and jobs
provides:
  - AI worker observation-to-image-tag governance wiring
  - Alias-aware merge resolution for governed tags
  - Pending reviewable AI image tags for downstream review UI
affects: [03-04, 03-06, ai-review-ui, tag-governance]
tech-stack:
  added: []
  patterns:
    - exact preferred-label lookup before normalized alias lookup in AI governance merges
    - save observation first, then persist pending governed image tags from the worker
key-files:
  created:
    - cmd/server/main_test.go
  modified:
    - cmd/server/main.go
    - internal/service/tag_governance_service.go
    - internal/service/tag_governance_service_test.go
    - internal/worker/ai_tag_handler.go
    - internal/worker/ai_tag_handler_test.go
key-decisions:
  - "AI governance keeps exact preferred-label matches first, then normalized alias lookup, and only then creates a new pending governed tag."
  - "AI-created image tag associations stay at pending review even when they resolve to an existing confirmed governed tag."
  - "Server bootstrap delegates AI worker registration through a helper so governance injection is testable without changing the non-fatal provider-not-configured behavior."
patterns-established:
  - "Worker pattern: persist tag_observations first and pass the saved observation ID into governance merges."
  - "Governance pattern: alias lookup reuses the governed tag while leaving review_state pending on image_tags for human review."
requirements-completed: [AIRE-03, AIRE-05]
duration: 7 min
completed: 2026-03-15
---

# Phase 03 Plan 05: AI Worker Governance Gap Closure Summary

**AI background jobs now turn saved observations into alias-aware governed image tags that stay pending for review in the existing tag workflow.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-15T20:03:25+08:00
- **Completed:** 2026-03-15T12:10:53Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Extended `TagGovernanceService.MergeTags` to reuse exact preferred labels first, then normalized aliases, while still creating pending AI review records.
- Wired the AI worker to call governance merges after `tag_observations` are saved, passing the persisted observation ID and model confidence.
- Added bootstrap coverage so `cmd/server/main.go` injects the governance service into `RegisterAITagHandler` without changing the current optional-provider startup behavior.

## task Commits

Each task was committed atomically:

1. **task 1: add alias-aware governance merge coverage first** - `e34211d`, `122f009` (test, feat)
2. **task 2: wire the AI worker to persist governed image_tags** - `448598b`, `cfdeb5d` (test, feat)

**Plan metadata:** pending

## Files Created/Modified
- `internal/service/tag_governance_service.go` - Adds normalized alias reuse and forces AI-created image tag associations into pending review state.
- `internal/service/tag_governance_service_test.go` - Covers exact-match reuse, alias reuse, and pending `image_tags` output.
- `internal/worker/ai_tag_handler.go` - Persists observations, then invokes governance merges with saved observation IDs.
- `internal/worker/ai_tag_handler_test.go` - Covers governance invocation and end-to-end persistence of reviewable AI `image_tags`.
- `cmd/server/main.go` - Injects governance into AI worker registration through a testable bootstrap helper.
- `cmd/server/main_test.go` - Verifies server bootstrap passes governance into worker registration.

## Decisions Made
- Preferred-label exact matches remain the first resolution path to preserve the Phase 3 no-fuzzy-match governance rule.
- Alias reuse is keyed by `tag_aliases.normalized_label` so synonym handling stays deterministic and consistent with existing alias storage.
- AI-generated `image_tags` always persist as `pending` review items, even when they point at existing confirmed governed tags, so the UI still has reviewable work to present.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated AI worker job mocks for the expanded repository contract**
- **Found during:** task 2 (wire the AI worker to persist governed image_tags)
- **Issue:** `mockJobRepoForAI` no longer satisfied `repository.JobRepository` because the repository layer now requires `FindByType`, which blocked the new worker tests from compiling.
- **Fix:** Added the missing `FindByType` stub while updating worker tests for governance wiring.
- **Files modified:** `internal/worker/ai_tag_handler_test.go`
- **Verification:** `go test ./internal/worker/... -run TestAITagHandler`
- **Committed in:** `cfdeb5d` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix kept the planned worker wiring testable after earlier repository interface growth. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The AI review UI can now surface real pending `image_tags` generated by background jobs instead of stopping at `tag_observations`.
- Alias-aware governance reuse is in place for backend gap closure, leaving the remaining Phase 3 gaps focused on filter wiring, AI result display, and tag statistics.

## Self-Check: PASSED
- Verified summary file exists at `.planning/phases/03-ai/03-05-SUMMARY.md`.
- Verified task commits `e34211d`, `122f009`, `448598b`, and `cfdeb5d` exist in git history.
