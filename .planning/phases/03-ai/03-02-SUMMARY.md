---
phase: 03-ai
plan: 02
subsystem: database
tags: [sqlite, repository, tags, governance, ai]
requires:
  - phase: 03-01
    provides: AI provider abstraction, rate-limited AI client, async tag generation handler
provides:
  - ImageTag domain model for image_tags associations
  - SQLite repositories for tags, observations, aliases, and image-tag links
  - Tag governance merge service for exact-match reuse and pending-tag creation
affects: [03-03, 03-04, tag-api, flutter-tags]
tech-stack:
  added: []
  patterns: [context-aware repositories, sqlite-backed repository tests, exact-match tag merge workflow]
key-files:
  created:
    - internal/domain/image_tag.go
    - internal/repository/tag_repository.go
    - internal/repository/tag_alias_repository.go
    - internal/repository/image_tag_repository.go
    - internal/service/tag_governance_service.go
  modified:
    - internal/repository/schema.go
    - internal/repository/tag_observation_repository.go
    - internal/worker/ai_tag_handler.go
    - internal/worker/ai_tag_handler_test.go
key-decisions:
  - "Repository tests use EnsureScanSchema so tag data layer coverage stays aligned with runtime SQLite tables."
  - "MergeTags increments existing tag usage counts and seeds new tags at pending review state with the association created immediately."
  - "Alias normalization trims whitespace and lowercases labels before persistence for stable exact alias lookup."
patterns-established:
  - "Repository pattern: pass context.Context through all tag data access methods."
  - "Governance pattern: exact label match first, create pending tag otherwise, then persist image-tag association."
requirements-completed: [AIRE-03, TAGS-01, TAGS-03]
duration: 9 min
completed: 2026-03-15
---

# Phase 03 Plan 02: Tag Data Layer Summary

**SQLite-backed tag repositories and an exact-match governance merge service now connect AI observations to governed tags and image associations.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-15T16:35:00+08:00
- **Completed:** 2026-03-15T08:44:07.715Z
- **Tasks:** 6
- **Files modified:** 15

## Accomplishments
- Added `ImageTag` as the domain model for `image_tags` composite-key associations.
- Implemented SQLite repositories for tag CRUD, AI observations, aliases, and image-tag state changes with dedicated tests.
- Added `TagGovernanceService.MergeTags` to reuse exact matches, create pending tags, increment usage counts, and persist image links.

## task Commits

Each task was committed atomically:

1. **task 1: 创建 ImageTag domain 模型** - `2232405` (feat)
2. **task 2: 实现 TagRepository** - `14692f1` (feat)
3. **task 3: 实现 TagObservationRepository** - `0a52fc6` (feat)
4. **task 4: 实现 TagAliasRepository** - `b5e80cb` (feat)
5. **task 5: 实现 ImageTagRepository** - `13efa70` (feat)
6. **task 6: 实现标签归并服务** - `23d05f8` (feat)

## Files Created/Modified
- `internal/domain/image_tag.go` - Defines the image-tag association model and composite key helper.
- `internal/domain/image_tag_test.go` - Verifies the composite key helper behavior.
- `internal/repository/schema.go` - Expands the SQLite test schema with tag, alias, and image-tag tables.
- `internal/repository/tag_repository.go` - Implements tag CRUD, exact lookup, fuzzy lookup, and usage state updates.
- `internal/repository/tag_repository_test.go` - Covers tag save, exact lookup, fuzzy lookup ordering, and state/count updates.
- `internal/repository/tag_observation_repository.go` - Adds context-aware observation persistence plus provider filtering.
- `internal/repository/tag_observation_repository_test.go` - Covers save, image lookup ordering, and provider filtering.
- `internal/repository/tag_alias_repository.go` - Implements alias persistence, normalization, search, and deletion.
- `internal/repository/tag_alias_repository_test.go` - Covers alias save, normalized lookup, per-tag lookup, and deletion.
- `internal/repository/image_tag_repository.go` - Implements image-tag persistence, lookup, review-state updates, and batch updates.
- `internal/repository/image_tag_repository_test.go` - Covers image-tag save, lookup, state update, and delete flows.
- `internal/service/tag_governance_service.go` - Implements exact-match merge behavior for AI-generated tags.
- `internal/service/tag_governance_service_test.go` - Covers existing tag reuse, new tag creation, pending state, and usage count increments.
- `internal/worker/ai_tag_handler.go` - Adapts AI observation persistence to the updated repository contract.
- `internal/worker/ai_tag_handler_test.go` - Keeps worker mocks aligned with the observation repository interface.

## Decisions Made
- Used context-aware repository methods across the tag data layer so service and worker flows can share cancellation-aware database access.
- Reused `EnsureScanSchema` for repository/service tests instead of bespoke test-only DDL to keep schema coverage close to production migrations.
- Kept merge behavior strict to exact label matches; unmatched labels become new pending tags immediately linked to the source image.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Expanded the shared SQLite test schema for tag tables**
- **Found during:** task 2 (实现 TagRepository)
- **Issue:** `EnsureScanSchema` only created images, jobs, and observations, so tag repositories could not run against the shared SQLite test harness.
- **Fix:** Added `tags`, `tag_aliases`, and `image_tags` tables plus indexes to `internal/repository/schema.go`.
- **Files modified:** `internal/repository/schema.go`
- **Verification:** `go test -v ./internal/repository/...`
- **Committed in:** `14692f1`

**2. [Rule 3 - Blocking] Updated AI worker integration for the new observation repository contract**
- **Found during:** task 3 (实现 TagObservationRepository)
- **Issue:** The worker still called `obsRepo.Save` without `context.Context`, which broke compilation after the repository interface was expanded.
- **Fix:** Updated `internal/worker/ai_tag_handler.go` and `internal/worker/ai_tag_handler_test.go` to use the context-aware interface and added the new provider query stub in tests.
- **Files modified:** `internal/worker/ai_tag_handler.go`, `internal/worker/ai_tag_handler_test.go`
- **Verification:** `go test ./internal/worker/...`
- **Committed in:** `0a52fc6`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were required to keep the planned repository work testable and compiling. No scope creep.

## Issues Encountered
- None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Tag repositories and merge service are ready for the Phase 03-03 tag API layer.
- Alias normalization and image-tag review-state operations are in place for upcoming UI and API review flows.

## Self-Check: PASSED
- Verified summary file exists at `.planning/phases/03-ai/03-02-SUMMARY.md`.
- Verified task commits `2232405`, `14692f1`, `0a52fc6`, `b5e80cb`, `13efa70`, and `23d05f8` exist in git history.
