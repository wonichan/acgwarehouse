---
phase: 11-task-platform-batch-model
plan: 01
subsystem: database
tags: [sqlite, async-jobs, task-batch, platform-task, repository]

requires:
  - phase: 10-theme-unification
    provides: unified backend/repository baseline and existing async job infrastructure
provides:
  - import task batch domain model and status vocabulary
  - platform task domain model with dedupe and async job linkage
  - SQLite schema baseline for task_batches, task_batch_sources, platform_tasks, and async_jobs.platform_task_id
affects: [phase-11, phase-12, phase-13, phase-14, scheduler, admin-read-model]

tech-stack:
  added: []
  patterns: [platform task layered above async_jobs, batch/task schema with nullable execution linkage]

key-files:
  created:
    - internal/domain/task_batch.go
    - internal/domain/platform_task.go
    - internal/repository/task_batch_repository.go
    - internal/repository/platform_task_repository.go
    - internal/repository/task_batch_repository_test.go
    - internal/repository/platform_task_repository_test.go
  modified:
    - internal/domain/async_job.go
    - internal/repository/schema.go
    - internal/repository/job_repository.go
    - internal/repository/job_repository_test.go

key-decisions:
  - "Keep async_jobs as execution-layer storage and attach platform_task_id instead of replacing the table."
  - "Model import processing with separate task_batches and platform_tasks tables so later phases can aggregate by batch, image, task type, and state."

patterns-established:
  - "Batch/task platform data lives beside async_jobs, not inside ad-hoc job payload semantics."
  - "Schema evolution adds nullable linkage columns compatibly via EnsureScanSchema idempotent setup."

requirements-completed: [PIPE-01, PIPE-03]
duration: 4 min
completed: 2026-03-24
---

# Phase 11 Plan 01: 定义导入批次、任务与任务状态模型 Summary

**导入后任务批次模型、平台任务去重语义与 async job 关联 schema 基线已落地到 SQLite/repository 契约。**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-24T14:58:40Z
- **Completed:** 2026-03-24T15:03:30Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- 以 TDD 方式先补齐了 task batch / platform task / async job 关联的失败测试骨架。
- 新增 `task_batches`、`task_batch_sources`、`platform_tasks` 三张表，并把 `async_jobs` 扩展为可空 `platform_task_id` 关联。
- 建立了批次与平台任务领域模型、状态常量，以及后续 11-02 可直接实现的 repository 接口边界。

## task Commits

Each task was committed atomically:

1. **task 1: 为批次与平台任务契约写失败测试骨架** - `af048bc` (test)
2. **task 2: 定义领域模型、状态常量与 repository 契约** - `ad7aec3` (feat)

**Plan metadata:** `pending`

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified
- `internal/domain/task_batch.go` - 定义批次来源、状态常量、批次摘要与来源明细模型。
- `internal/domain/platform_task.go` - 定义平台任务状态、任务类型、去重键和最新 async job 关联。
- `internal/domain/async_job.go` - 为底层执行记录增加 `PlatformTaskID`。
- `internal/repository/schema.go` - 增加任务平台三张表、索引及 `async_jobs.platform_task_id` 兼容迁移。
- `internal/repository/task_batch_repository.go` - 定义批次 create/find/list/aggregate 契约。
- `internal/repository/platform_task_repository.go` - 定义平台任务 create/list/dedupe/update 契约。
- `internal/repository/task_batch_repository_test.go` - 断言批次表与 async job 关联字段存在。
- `internal/repository/platform_task_repository_test.go` - 断言平台任务去重字段与索引存在。
- `internal/repository/job_repository.go` - 让 async job 持久化逻辑读写 `platform_task_id`。
- `internal/repository/job_repository_test.go` - 同步测试夹具 schema，确保现有 job repository 回归通过。

## Decisions Made
- 保留 `async_jobs` 作为执行层持久化表，只增加 `platform_task_id` 关联字段，避免把产品语义重新塞回裸 job。
- 先落地批次/平台任务的领域与 repository 契约，具体持久化实现延后到 11-02，以保证 Phase 11 后续计划复用同一主模型边界。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 同步 job repository 测试夹具到新 async_jobs schema**
- **Found during:** task 2 (定义领域模型、状态常量与 repository 契约)
- **Issue:** `job_repository_test.go` 仍使用旧版 `async_jobs` 建表语句，导致新增 `platform_task_id` 后 repository 同包回归测试直接失败。
- **Fix:** 给测试夹具补上 `platform_task_id` 字段和对应索引，使现有 job repository 测试与扩展后的 schema 保持一致。
- **Files modified:** `internal/repository/job_repository_test.go`
- **Verification:** `go test ./internal/repository/... -count=1`
- **Committed in:** `ad7aec3` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 该修复仅用于消除当前任务直接引入的 repository 回归，不改变 11-01 的范围或设计。

## Issues Encountered
- `job_repository_test.go` 的本地建表夹具未随 schema 变更同步，已在 task 2 内修复并通过整包回归验证。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 11-02 可以直接在既有 `task_batches` / `platform_tasks` / `async_jobs.platform_task_id` 契约上实现持久化与状态流转。
- 批次聚合、去重查询与后台读模型都已有统一字段边界，无需再次重定主模型。

## Self-Check: PASSED
- FOUND: `.planning/phases/11-task-platform-batch-model/11-01-SUMMARY.md`
- FOUND: `af048bc`
- FOUND: `ad7aec3`

---
*Phase: 11-task-platform-batch-model*
*Completed: 2026-03-24*
