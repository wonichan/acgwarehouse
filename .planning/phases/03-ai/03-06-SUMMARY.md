---
phase: 03-ai
plan: 06
subsystem: api
tags: [images, tags, filtering, merge, statistics, governance]

# Dependency graph
requires:
  - phase: 03-03
    provides: tag CRUD API and image-tag review endpoints
  - phase: 03-05
    provides: AI worker governance merge wiring
provides:
  - Image list API with governed tag filtering (AND semantics)
  - Image-tag merge endpoint for AI review items
  - Tag statistics API with usage/pending/AI/Manual counts
affects: [03-04, flutter-gallery, tag-filter-drawer]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - SQL GROUP BY HAVING for AND-semantics tag filtering
    - Source observation ID tracking for AI vs manual distinction

key-files:
  created:
    - internal/handler/image_handler.go
    - internal/repository/image_repository_test.go
    - internal/handler/image_handler_test.go
  modified:
    - internal/repository/image_repository.go
    - internal/repository/image_tag_repository.go
    - internal/handler/image_tag_handler.go
    - internal/handler/tag_handler.go
    - internal/handler/routes.go

key-decisions:
  - "Image filtering uses AND semantics - images must have ALL requested tags"
  - "Tag statistics distinguish AI-sourced (has source_observation_id) vs manual associations"

patterns-established:
  - "Filter pattern: GROUP BY image_id HAVING COUNT(DISTINCT tag_id) = len(requested_tag_ids)"
  - "Merge pattern: delete old association, create new with target tag preserving confidence/review_state"

requirements-completed: [AIRE-05, TAGS-03, TAGS-05]

# Metrics
duration: 15 min
completed: 2026-03-15
---

# Phase 03 Plan 06: Backend Gap Closure Summary

**Image list API with governed tag filtering, image-tag merge endpoint, and tag statistics API for Phase 3 gap closure.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-03-15T12:19:46Z
- **Completed:** 2026-03-15T12:35:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Image list API supports tag_ids query parameter with AND semantics filtering
- Image-tag merge endpoint allows reassigning AI tags to existing governed tags
- Tag statistics endpoint returns usage, pending, confirmed, AI, and manual counts

## Task Commits

Each task was committed atomically:

1. **Task 1: add real image list filtering by governed standard tags**
   - `2314cfe` test(03-06): add failing image filter API tests
   - `fe224ad` feat(03-06): add governed tag filtering to image list endpoints

2. **Task 2: expose review merge and governance statistics APIs**
   - `7079bd3` feat(03-06): add image-tag merge and tag statistics repository methods
   - `afbd058` feat(03-06): expose review merge and governance statistics endpoints

## Files Created/Modified
- `internal/handler/image_handler.go` - Image list and get endpoints with tag filtering
- `internal/repository/image_repository.go` - FindByTagIDs and CountByTagIDs methods
- `internal/repository/image_tag_repository.go` - MergeImageTag and GetTagStats methods
- `internal/handler/image_tag_handler.go` - MergeImageTag endpoint
- `internal/handler/tag_handler.go` - GetTagStats endpoint
- `internal/handler/routes.go` - Route registration for new endpoints
- `internal/repository/image_repository_test.go` - Tests for tag filtering
- `internal/handler/image_handler_test.go` - Tests for image handler

## Decisions Made
- AND semantics for tag filtering: only images with ALL requested tags are returned
- AI vs manual distinction based on source_observation_id presence in image_tags
- Merge preserves confidence and review_state from source association

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Test helper `getDBFromRepo` created a separate database connection, causing merge tests to fail. Fixed by seeding tag 3 in the existing test setup instead of creating it in a separate database.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Backend gaps from 03-VERIFICATION.md are closed for TAGS-03, TAGS-05, and merge portion of AIRE-05
- Flutter frontend can now consume the filtering, merge, and statistics endpoints
- Remaining gaps: AI result display in Flutter (03-07)

---
*Phase: 03-ai*
*Completed: 2026-03-15*

## Self-Check: PASSED
- Verified summary file exists at `.planning/phases/03-ai/03-06-SUMMARY.md`.
- Verified task commits `2314cfe`, `fe224ad`, `7079bd3`, and `afbd058` exist in git history.