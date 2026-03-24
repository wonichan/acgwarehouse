---
phase: 11-task-platform-batch-model
plan: 04
subsystem: api
tags: [task-batch, platform-task, admin-api, read-model, sqlite]

requires:
  - phase: 11-task-platform-batch-model
    provides: 11-03 unified task platform entrypoints and lifecycle-backed execution records
provides:
  - admin-readable task batch list API with source summaries, skip summaries, failure summaries, and status counters
  - admin task detail API filtered by batch_id and backed by platform task read models instead of raw async jobs
  - dedicated read repository/service layer for task_batches, task_batch_sources, platform_tasks, and async_jobs aggregation
affects: [phase-11, phase-13, admin-dashboard, task-platform, monitoring]

tech-stack:
  added: []
  patterns: [admin service delegates read aggregation to TaskReadService, admin routes expose batch and task reads under /admin/api, read SQL lives in task_batch_read_repository instead of job_repository]

key-files:
  created:
    - internal/repository/task_batch_read_repository.go
    - internal/service/task_read_service.go
    - internal/service/task_read_service_test.go
  modified:
    - internal/service/admin_service.go
    - internal/service/admin_service_test.go
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go
    - internal/handler/routes.go
    - internal/app/app.go

key-decisions:
  - "后台批次读模型直接聚合 task_batches、task_batch_sources、platform_tasks 与 images，而不是回退到裸 async_jobs 视角。"
  - "AdminService 通过 TaskReadService 暴露批次/任务查询，沿用既有 admin service/handler/router 分层，不新增平行路由风格。"
  - "为保持验证命令稳定，admin_service_test 的测试数据库改为复用 EnsureScanSchema，而不是维护过时的手写 async_jobs schema。"

patterns-established:
  - "Admin read APIs return batch-centric JSON collections: task_batches and tasks."
  - "Read-model SQL is isolated in repository layer, while service layer shapes admin-facing DTOs."

requirements-completed: [PIPE-01, PIPE-03]
duration: 10 min
completed: 2026-03-24
---

# Phase 11 Plan 04: 提供后台查询所需的批次 / 任务读模型 Summary

**后台现在可以通过批次与平台任务读模型直接查询导入后任务平台状态，而不是只能查看裸 `async_jobs` 列表。**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-24T15:59:39Z
- **Completed:** 2026-03-24T16:09:16Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- 新增 `task_batch_read_repository` 与 `TaskReadService`，为后台聚合批次来源摘要、跳过统计、失败摘要、状态/任务类型计数。
- 扩展 `AdminService`、`AdminHandler` 与路由，提供 `/admin/api/task-batches` 与 `/admin/api/tasks` 两类读接口。
- 把应用启动阶段的 admin service 装配接入读模型服务，使 Phase 13 监控页可直接消费真实平台数据。

## task Commits

Each task was committed atomically:

1. **task 1: 为批次 / 任务读模型聚合写失败测试并实现查询服务** - `3cf7f85` (test), `061c6a1` (feat)
2. **task 2: 扩展 admin service/handler/routes 暴露批次与任务查询接口** - `153f8bc` (test), `e58c9ea` (feat)

**Plan metadata:** `pending at summary creation`

## Files Created/Modified
- `internal/repository/task_batch_read_repository.go` - 提供批次汇总查询与按批次任务明细查询 SQL。
- `internal/service/task_read_service.go` - 将 repository 读记录整理为后台 API 可直接返回的 DTO。
- `internal/service/task_read_service_test.go` - 锁定来源摘要、跳过摘要、失败摘要、排序与过滤行为。
- `internal/service/admin_service.go` - 通过 `TaskReadService` 暴露批次/任务读取能力。
- `internal/service/admin_service_test.go` - 让 admin service 验证使用最新平台 schema，避免旧 schema 阻塞当前计划验证。
- `internal/handler/admin_handler.go` - 新增批次列表与任务列表接口，并保持现有 admin JSON 风格。
- `internal/handler/admin_handler_test.go` - 验证 `/admin/api/task-batches` 与 `/admin/api/tasks?batch_id=` 响应。
- `internal/handler/routes.go` - 注册新的 admin 读接口路由。
- `internal/app/app.go` - 为 `AdminService` 注入 `TaskReadService`。

## Decisions Made
- 后台读模型直接返回批次中心视图，批次摘要里包含来源、跳过、失败与任务状态统计，避免让 handler 自己拼装跨表数据。
- 任务明细接口选择沿用 `/admin/api/tasks?batch_id=` 的查询式风格，和现有 admin `/jobs` 读接口保持一致。
- admin service 继续作为后台聚合入口，但新增的聚合 SQL 被收敛到独立 read repository，避免把 Phase 11 读模型塞回 `job_repository.go`。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 修复 admin service 测试仍依赖旧版 async_jobs schema 的验证阻塞**
- **Found during:** task 1（读模型服务实现后运行 `go test ./internal/service/... -run "TaskRead|Admin" -count=1`）
- **Issue:** `internal/service/admin_service_test.go` 手写的测试数据库 schema 没有 `platform_task_id` 列，导致当前 Phase 11 的 admin 验证命令在无关断言前就被旧 schema 卡住。
- **Fix:** 改为让测试数据库直接复用 `repository.EnsureScanSchema()`，与当前任务平台数据库结构保持一致。
- **Files modified:** `internal/service/admin_service_test.go`
- **Verification:** `go test ./internal/service/... -run "TaskRead|Admin" -count=1`
- **Committed in:** `061c6a1`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 仅修复了当前验证命令的旧 schema 阻塞，确保 11-04 能在现有 Phase 11 数据模型上完成验证，没有扩张到无关范围。

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 11 已具备“批次可追踪 + 平台任务可观测 + 后台可查询”的闭环，可结束本阶段并为 Phase 12 自动入队规则提供后台验证入口。
- Phase 13 后台监控页可以直接复用当前 `/admin/api/task-batches` 与 `/admin/api/tasks` 数据结构，而无需回头补做跨表聚合。

## Self-Check: PASSED
- FOUND: `.planning/phases/11-task-platform-batch-model/11-04-SUMMARY.md`
- FOUND: `3cf7f85`
- FOUND: `061c6a1`
- FOUND: `153f8bc`
- FOUND: `e58c9ea`

---
*Phase: 11-task-platform-batch-model*
*Completed: 2026-03-24*
