---
phase: 16-duplicate-detection-migration
plan: 02
subsystem: database
tags: [sqlite, go-domain, repository, phash, duplicate-detection]

requires:
  - phase: 16-01
    provides: Python 侧重复检测计算与结果结构
  - phase: 15-03
    provides: Go 侧可观测性与 sidecar 基础运行边界
provides:
  - images 表新增 `phash_hex TEXT` 迁移列
  - duplicate_relations 表新增 `recommendation_score REAL` 与 `recommendation_rationale TEXT` 迁移列
  - Go domain 支持 `PHashHex` 与结构化 `RecommendationRationale`
  - Repository 提供 `UpdateImagePHashHex` 明确写路径并完成新字段读写映射
affects: [phase-16-plan-03, duplicate-repository, image-repository]

tech-stack:
  added: []
  patterns: [sqlite-safe-column-migration, json-rawmessage-rationale-mapping, explicit-phash-write-path]

key-files:
  created:
    - .planning/phases/16-duplicate-detection-migration/16-02-SUMMARY.md
  modified:
    - internal/repository/schema.go
    - internal/domain/image.go
    - internal/domain/duplicate_group.go
    - internal/repository/duplicate_repository.go
    - internal/repository/image_repository.go
    - internal/repository/duplicate_repository_test.go
    - internal/repository/image_repository_test.go

key-decisions:
  - "沿用 ensureColumnExists 进行增量迁移，不改 CREATE TABLE，保障已有 SQLite 库平滑升级"
  - "推荐依据在 Go 侧使用 json.RawMessage，数据库仍持久化为 TEXT，兼顾结构化输出与存储兼容"

patterns-established:
  - "256-bit pHash 存储模式：保留旧 phash INTEGER，新增 phash_hex TEXT 并走显式回写路径"
  - "重复关系扩展模式：INSERT/SELECT 同步补齐 recommendation_score 与 recommendation_rationale"

requirements-completed: [COMP-03, COMP-04]

duration: 12 min
completed: 2026-04-04
---

# Phase 16 Plan 02: Go Domain 与 Schema 扩展总结

**Go 数据层已完成 256-bit pHash 与结构化推荐依据扩展，能够稳定接收并持久化 Python 侧返回的 phash 与推荐结果。**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-04T13:47:14+08:00
- **Completed:** 2026-04-04T13:54:28+08:00
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- 通过 `EnsureScanSchema` 增量迁移新增 `phash_hex`、`recommendation_score`、`recommendation_rationale` 三个列
- 扩展 `domain.Image`、`domain.DuplicateRelation`、`domain.DuplicateImage`，支持 256-bit pHash 与结构化推荐依据
- 更新 `DuplicateRepository` 与 `ImageRepository` 的读写路径，新增 `UpdateImagePHashHex` 显式回写方法
- 新增并通过仓储测试，验证新列/新字段的插入、查询、JSON 有效性与 round-trip 一致性

## Task Commits

Each task was committed atomically:

1. **Task 1: Schema 列迁移与列级 round-trip 测试** - `fb32fff` (feat)
2. **Task 2: Domain/Repository 扩展与结构化依据持久化** - `6c3d2f3` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `internal/repository/schema.go` - 新增 `phash_hex` 与 recommendation 相关列迁移
- `internal/domain/image.go` - 新增 `PHashHex`
- `internal/domain/duplicate_group.go` - 新增推荐评分与 `json.RawMessage` 推荐依据字段
- `internal/repository/duplicate_repository.go` - 重复关系新增推荐字段写入与读取映射
- `internal/repository/image_repository.go` - 所有图片查询补齐 `phash_hex` 映射，并新增 `UpdateImagePHashHex`
- `internal/repository/duplicate_repository_test.go` - 新增推荐列与结构化依据持久化测试
- `internal/repository/image_repository_test.go` - 新增 `phash_hex` 列与更新回写测试

## Decisions Made
- 继续保留旧 `phash` 字段，仅新增 `phash_hex` 承接 256-bit 迁移，降低历史数据兼容风险。
- 推荐依据采用 `json.RawMessage` 出口语义，避免二次转义字符串影响后续 API 结构化输出。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Go 侧 schema/domain/repository 扩展已具备，满足 Plan 03 接入 Python 结果落库前置条件。
- 若 16-01 同步完成，可直接推进 16-03 的 sidecar 调用链路改造。

## Self-Check: PASSED
- Summary file exists on disk.
- Task commits `fb32fff` 与 `6c3d2f3` 可在 git history 中检索到。
