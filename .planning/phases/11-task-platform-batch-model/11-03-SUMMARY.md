---
phase: 11-task-platform-batch-model
plan: 03
subsystem: api
tags: [task-batch, platform-task, scanner, ai-tag, bootstrap, worker-lifecycle]

requires:
  - phase: 11-task-platform-batch-model
    provides: 11-02 task batch persistence, dedupe rules, and platform task lifecycle storage
provides:
  - import_scan batches created from scanner runs with thumbnail platform task planning
  - manual_single and manual_batch AI triggers routed through platform batches with queued async jobs linked by platform_task_id
  - bootstrap wrapper that syncs async job execution state back into platform tasks and batch aggregation
affects: [phase-11, phase-12, phase-13, phase-14, scanner, ai-tagging, worker-runtime]

tech-stack:
  added: []
  patterns: [scanner creates platform batches before execution, manual AI requests return batch/task/job identifiers, worker handlers update platform task lifecycle around execution]

key-files:
  created: []
  modified:
    - internal/service/scanner_service.go
    - internal/service/scanner_service_test.go
    - internal/service/task_platform_service.go
    - internal/service/task_platform_service_test.go
    - internal/handler/ai_tag_handler.go
    - internal/handler/ai_tag_handler_test.go
    - internal/handler/routes.go
    - internal/app/bootstrap.go
    - internal/worker/ai_tag_handler.go

key-decisions:
  - "Scanner runs now always create an import_scan batch, but only newly imported images are eligible to plan thumbnail tasks."
  - "Manual AI trigger responses expose batch/task/job identifiers so single-image actions still live inside the unified batch model."
  - "Platform task lifecycle sync is implemented as bootstrap-level handler wrappers around existing worker.Manager registrations instead of changing the manager API."

patterns-established:
  - "Entry points first create or reuse platform tasks, then queue async jobs as execution records."
  - "Legacy image_imported jobs remain internal dispatch triggers while thumbnail_generate and ai_tag_generation stay the product-facing platform task types."

requirements-completed: [PIPE-01, PIPE-03, SAFE-03]
duration: 9 min
completed: 2026-03-24
---

# Phase 11 Plan 03: 接入统一调度入口与分发骨架 Summary

**导入扫描与手动 AI 入口已统一写入任务批次，并通过 bootstrap 包装器把 async job 执行状态回写到平台任务生命周期。**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-24T15:40:41Z
- **Completed:** 2026-03-24T15:49:41Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments
- 扫描入口现在会为每次导入创建 `import_scan` 批次，并只为新导入图片规划缩略图平台任务。
- 单图与批量 AI 触发都改为创建 `manual_single` / `manual_batch` 批次，并返回批次、平台任务、async job 概览。
- bootstrap 现已用平台任务包装现有 worker handler，在 queued/running/completed/failed 各阶段同步平台任务与批次状态。

## task Commits

Each task was committed atomically:

1. **task 1: 让导入扫描入口创建 import_scan 批次与平台任务** - `7a12125` (feat)
2. **task 2: 让手动 AI 入口改为创建 manual_single / manual_batch 批次** - `f2ae96d` (feat)
3. **task 3: 在 bootstrap 建立统一调度与状态回写骨架** - `002c984` (feat)

**Plan metadata:** `2d6031d`

## Files Created/Modified
- `internal/service/scanner_service.go` - 为扫描结果补充批次字段，并在扫描结束后统一规划 `import_scan` 批次。
- `internal/service/scanner_service_test.go` - 锁定导入批次创建、未变更跳过与平台任务规划行为。
- `internal/service/task_platform_service.go` - 增加图片版本键/摘要辅助函数、任务入队与按 job 回写平台任务状态的方法。
- `internal/service/task_platform_service_test.go` - 验证 queued/running/completed/failed 生命周期与批次聚合回写。
- `internal/handler/ai_tag_handler.go` - 把单图/批量 AI 触发改为平台批次规划并返回批次/任务/job 标识。
- `internal/handler/ai_tag_handler_test.go` - 锁定 `manual_single`、`manual_batch` 与重复任务跳过行为。
- `internal/handler/routes.go` - 路由装配 AI handler 所需的 task platform 依赖。
- `internal/app/bootstrap.go` - 用平台任务包装器注册 thumbnail/AI handler，并把 legacy `image_imported` 收敛为内部调度起点。
- `internal/worker/ai_tag_handler.go` - 导出可复用的 AI job handler 以便 bootstrap 包装生命周期同步。

## Decisions Made
- 扫描批次会始终记录一次触发动作，但只有本次新导入图片才允许真正规划缩略图任务，保持现有导入语义不向 Phase 12 提前扩张。
- 手动 AI 入口直接返回批次级结果，不再暴露平台外裸 `job_id` 作为唯一语义。
- 不改动 `worker.Manager` 公共 API，而是在 bootstrap 注册时包裹 handler，同步平台任务状态并保留现有 worker 行为。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 修复 watcher 与路由对新调度签名的编译阻塞**
- **Found during:** task 1 / task 2
- **Issue:** `ScannerService` 构造函数与 `importFile()` 返回值变化后，watcher 与 AI 路由装配立即编译失败，导致目标测试命令无法运行。
- **Fix:** 同步更新 watcher 的 scanner 调用方式，并在路由层即时装配 task platform 依赖，保持现有入口可编译、可测试。
- **Files modified:** `internal/service/watcher_service.go`, `internal/service/watcher_service_test.go`, `internal/handler/routes.go`
- **Verification:** `go test ./internal/service/... -run "Scanner|TaskPlatform" -count=1`；`go test ./internal/handler/... -run "AITag" -count=1`
- **Committed in:** `7a12125`, `f2ae96d`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 仅修复与 11-03 直接相关的编译阻塞，未扩张范围。

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 11-04 现在可以直接复用 `task_batches` / `platform_tasks` 的真实入口与 lifecycle 数据来构建后台读模型。
- 导入与手动 AI 都已进入统一平台语义，后续后台不必再兼容平台外裸 async job 入口。

## Self-Check: PASSED
- FOUND: `.planning/phases/11-task-platform-batch-model/11-03-SUMMARY.md`
- FOUND: `7a12125`
- FOUND: `f2ae96d`
- FOUND: `002c984`

---
*Phase: 11-task-platform-batch-model*
*Completed: 2026-03-24*
