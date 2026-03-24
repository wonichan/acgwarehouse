---
phase: 11-task-platform-batch-model
plan: 02
subsystem: api
tags: [sqlite, task-batch, platform-task, dedupe, lifecycle, async-jobs]

requires:
  - phase: 11-task-platform-batch-model
    provides: 11-01 domain contracts and schema baseline for task batches, platform tasks, and async job linkage
provides:
  - SQLite-backed task batch persistence with lifecycle aggregation and partial_failed semantics
  - platform task dedupe lookup and async job platform_task_id round-trip support
  - task platform planning service that backfills missing task types without re-enqueueing unchanged work
affects: [phase-11, phase-12, phase-13, phase-14, scheduler, admin-read-model]

tech-stack:
  added: []
  patterns: [batch status derived from platform task terminal states, dedupe by image_version_key plus task type, async_jobs linked to platform_tasks as execution records]

key-files:
  created:
    - internal/service/task_platform_service.go
    - internal/service/task_platform_service_test.go
  modified:
    - internal/repository/task_batch_repository.go
    - internal/repository/platform_task_repository.go
    - internal/repository/job_repository.go
    - internal/repository/task_batch_repository_test.go
    - internal/repository/platform_task_repository_test.go
    - internal/repository/job_repository_test.go

key-decisions:
  - "Use image_version_key + task_type as the stable dedupe key so path-only moves do not re-enqueue unchanged work."
  - "Keep batch status derived from platform task aggregation, including partial_failed when at least one task fails after all tasks reach terminal states."

patterns-established:
  - "Repository aggregation owns batch lifecycle refresh instead of import-time shortcuts."
  - "Service planning creates a new batch record per trigger while only backfilling missing task types for each image version."

requirements-completed: [PIPE-03, SAFE-03]
duration: 7 min
completed: 2026-03-24
---

# Phase 11 Plan 02: 建立任务持久化与状态流转规则 Summary

**任务批次生命周期聚合、平台任务去重补缺规则与 async job 平台关联已在 SQLite repository/service 层闭环。**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-24T15:17:09Z
- **Completed:** 2026-03-24T15:24:19Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- 先以 TDD 补齐了批次聚合、平台任务 dedupe、async job 关联与 service 规划规则的失败测试。
- 把 `task_batches`、`platform_tasks` 与 `async_jobs.platform_task_id` 的 repository 契约补成了真实 SQLite 持久化实现。
- 新增 `TaskPlatformService`，支持按任务类型补缺、避免未变更重复入队，并按任务终态刷新批次状态。

## task Commits

Each task was committed atomically:

1. **task 1: 为生命周期聚合与重复保护写失败测试** - `f8dd7e3` (test)
2. **task 2: 实现 repository 持久化与 async job 关联** - `5d82071` (feat)
3. **task 3: 实现任务平台 service 的规划、去重与状态聚合规则** - `9d2beba` (feat)

**Plan metadata:** `3ec007f`

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified
- `internal/repository/task_batch_repository.go` - 实现批次创建、来源记录、列表查询、状态聚合刷新与 `partial_failed` 计算。
- `internal/repository/platform_task_repository.go` - 实现平台任务增删查改、按 dedupe key 查重、按图片+任务类型补缺查询。
- `internal/repository/job_repository.go` - 增加按 `platform_task_id` 查询 async job 关联能力。
- `internal/repository/task_batch_repository_test.go` - 锁定“未全部终态前不得 completed、失败后聚合 partial_failed”的 RED/GREEN 用例。
- `internal/repository/platform_task_repository_test.go` - 锁定 dedupe 查询与按任务类型补缺查询行为。
- `internal/repository/job_repository_test.go` - 锁定 `platform_task_id` 的保存、更新与反查行为。
- `internal/service/task_platform_service.go` - 实现统一批次规划、去重跳过统计与批次状态刷新入口。
- `internal/service/task_platform_service_test.go` - 验证重复保护、只补缺失任务类型与 lifecycle 聚合规则。

## Decisions Made
- 去重键采用 `image_version_key + task_type`，确保去重按内容/版本而非文件路径生效。
- 批次状态完全由平台任务聚合得出；存在未终态任务时保持 `running`，全部终态且含失败时聚合为 `partial_failed`。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 11-03 可直接把导入入口和调度骨架挂到 `TaskPlatformService`，不必再在导入链路里散落判断 dedupe/补缺逻辑。
- 后续后台读模型已可复用批次聚合状态、dedupe 语义与 `platform_task_id` 执行链路。

## Self-Check: PASSED
- FOUND: `.planning/phases/11-task-platform-batch-model/11-02-SUMMARY.md`
- FOUND: `f8dd7e3`
- FOUND: `5d82071`
- FOUND: `9d2beba`

---
*Phase: 11-task-platform-batch-model*
*Completed: 2026-03-24*
