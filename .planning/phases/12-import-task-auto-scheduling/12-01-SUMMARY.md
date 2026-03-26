---
phase: 12-import-task-auto-scheduling
plan: 01
subsystem: database
tags: [sqlite, migration, repository, ai-tag]

requires:
  - phase: 11-task-platform-batch-model
    provides: task platform model and batch/task semantics
provides:
  - image_tags source classification for ai/manual tags
  - repository query for thumbnail-ready images without ai tags
  - source-aware image tag persistence and hydration
affects: [12-02, ai-tag-auto-scheduler, import-scan-filtering]

tech-stack:
  added: []
  patterns: [schema compatibility via EnsureScanSchema, NOT EXISTS ai-source filtering]

key-files:
  created:
    - internal/repository/image_tag_source_schema_test.go
    - migrations/003_add_image_tag_source.up.sql
    - migrations/003_add_image_tag_source.down.sql
  modified:
    - internal/domain/image_tag.go
    - internal/repository/schema.go
    - internal/repository/image_repository.go
    - internal/repository/image_repository_test.go
    - internal/repository/image_tag_repository.go
    - internal/repository/image_tag_repository_test.go

key-decisions:
  - "Use image_tags.source enum values ai/manual with manual default"
  - "Auto-queue eligibility query checks thumbnail_small_url is non-empty and excludes source='ai'"
  - "Backfill existing ai tags using source_observation_id during migration/schema compatibility"

patterns-established:
  - "ImageTag source fallback: empty source persists/reads as manual"
  - "Eligibility scan SQL uses ORDER BY image id ASC for FIFO queueing"

requirements-completed: [AIQ-02]

duration: 10 min
completed: 2026-03-26
---

# Phase 12 Plan 01: 数据模型变更与查询基础 Summary

**Shipped `image_tags.source` data semantics plus a repository query that returns thumbnail-ready images with no AI-source tags for auto-scheduling eligibility.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-26T15:49:13Z
- **Completed:** 2026-03-26T15:59:15Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments
- Added migration `003` and schema compatibility logic for `image_tags.source` with AI backfill.
- Added `ImageRepository.FindImagesWithoutAITags(ctx, limit)` with FIFO ordering and limit support.
- Updated `ImageTagRepository.Save`/scan paths to persist and read `source`, including manual fallback compatibility.

## task Commits

Each task was committed atomically:

1. **task 1: 添加 image_tag.source 字段迁移** - `aa4f89e` (feat)
2. **task 2: 实现查询无 AI 标签图片的方法** - `7dfc272` (feat)
3. **task 3: 更新 ImageTagRepository.Save 支持 source 字段** - `6979ccd` (feat)

**Plan metadata:** (in docs commit for this summary/state update)

## Files Created/Modified
- `internal/domain/image_tag.go` - Added source constants and `ImageTag.Source` field.
- `internal/repository/schema.go` - Added `image_tags.source` schema definition + compatibility backfill.
- `internal/repository/image_repository.go` - Added `FindImagesWithoutAITags` implementation.
- `internal/repository/image_repository_test.go` - Added 4 scenario tests for AI-tag eligibility query.
- `internal/repository/image_tag_repository.go` - Persisted/scanned `source` in save/find/merge paths.
- `internal/repository/image_tag_repository_test.go` - Added source persistence/default tests.
- `internal/repository/image_tag_source_schema_test.go` - Added migration/schema source behavior tests.
- `migrations/003_add_image_tag_source.up.sql` - Added source column migration and AI backfill.
- `migrations/003_add_image_tag_source.down.sql` - Added rollback for source column.

## Decisions Made
- Kept source values explicit (`ai`/`manual`) at domain level for direct query semantics.
- Used repository-level compatibility fallback (`manual`) to keep old/empty records safe.
- Scoped auto-scheduling eligibility to thumbnail-ready images with no `source='ai'` association.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] SQLite does not support `ADD COLUMN IF NOT EXISTS` in current runtime**
- **Found during:** task 1 (migration idempotency test)
- **Issue:** Migration SQL using `IF NOT EXISTS` failed with syntax error.
- **Fix:** Switched migration to standard `ALTER TABLE ... ADD COLUMN`; ensured idempotent compatibility through `EnsureScanSchema` + `ensureColumnExists` tests.
- **Files modified:** `migrations/003_add_image_tag_source.up.sql`, `internal/repository/image_tag_source_schema_test.go`, `internal/repository/schema.go`
- **Verification:** `go test ./internal/repository/... -run "ImageTagSource|Schema" -count=1`
- **Committed in:** `aa4f89e`

**2. [Rule 1 - Bug] Empty thumbnail URLs were incorrectly treated as thumbnail-ready**
- **Found during:** task 2 (query behavior tests)
- **Issue:** Existing rows can store empty string rather than NULL for `thumbnail_small_url`.
- **Fix:** Added `thumbnail_small_url != ''` condition to eligibility query.
- **Files modified:** `internal/repository/image_repository.go`
- **Verification:** `go test ./internal/repository/... -run "FindImagesWithoutAITags" -count=1`
- **Committed in:** `7dfc272`

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Fixes were required for correctness and runtime compatibility; no scope creep beyond plan objective.

## Issues Encountered
- Migration idempotency expectation required schema-compatibility handling because runtime SQLite syntax support is limited.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 12-01 outcomes are complete and verified; scheduler implementation can rely on `FindImagesWithoutAITags` and `image_tags.source`.
- Ready for `12-02` (定时扫描服务实现).

---
*Phase: 12-import-task-auto-scheduling*
*Completed: 2026-03-26*
