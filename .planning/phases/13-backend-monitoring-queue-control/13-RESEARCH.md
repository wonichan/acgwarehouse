# Phase 13: 后台监控与队列控制 - Research

**Researched:** 2026-03-27
**Status:** Ready for planning

## Goal

回答“为了把 Phase 13 计划写准，我需要知道什么？”聚焦后台管理页的批次优先监控、全局队列控制、批次/任务取消、失败重试与验证策略。

## Existing Reusable Assets

- `internal/service/task_read_service.go` 已提供批次中心读模型：`TaskBatchReadModel` 含 `status_counts`、`task_type_counts`、`failure_summary`，`TaskReadModel` 含文件名、任务类型、状态、错误摘要与最新 async job 关联。
- `internal/handler/admin_handler.go` 与 `internal/handler/routes.go` 已暴露 `/admin/api/task-batches`、`/admin/api/tasks`、`/admin/api/actions/jobs/pause`、`/resume`、`/retry-failed` 等路由骨架，延续统一 `success/message/data` 风格。
- `internal/worker/job_manager.go` 已提供运行态能力：`Pause()`、`Resume()`、`IsPaused()`、`QueueSize()`、`GetStats()`，可作为顶部运行态概览的直接来源。
- `web/admin/index.html` 与 `web/admin/app.js` 已有后台页壳、toast、自动刷新、全局按钮和最近错误区，可在既有页面上扩展，不必另起 Flutter 页面或新路由。
- Phase 11 已锁定“批次是自然单位”，Phase 12 已锁定自动入队和活跃任务排除规则；Phase 13 应继续围绕 `task_batches` / `platform_tasks`，而不是回退到裸 `async_jobs`。

## Locked Decisions to Preserve

- 页面主结构必须是“批次列表在上、任务明细在下”的同页监控布局。
- 批次列表使用表格行 + 状态徽标，不做卡片堆叠。
- 顶部概览同时显示平台统计与队列运行态，不只显示裸 job 数。
- `pause` / `resume` 只做全局队列控制；`cancel` 以批次级为主入口，任务级为补充入口。
- `clear queue` 只清空全局 `pending` / `queued`，不能影响 `running`。
- `cancel` / `clear queue` 必须强确认，并明确影响范围与数量。
- 批次级失败重试是主入口，任务级重试是补充入口。
- 仅 `failed` 任务允许重试；重试结果必须形成新批次，并在反馈中给出新批次入口。

## Missing Backend Contracts

### 1. 队列概览接口仍偏旧版 job 视角

`AdminService.GetSummary()` 当前只返回健康、配置、旧 `async_jobs` 计数和图库统计，缺少：

- 队列暂停态 / 运行态
- 当前内存队列规模 / 容量
- 平台批次状态统计
- 平台任务状态统计（尤其 `pending/queued/running/completed/failed/cancelled/skipped`）
- 最近失败摘要或异常聚合

建议新增独立平台监控接口（如 `/admin/api/task-platform/overview`），避免把旧 `summary` 契约改成混合结构过大；也可保留 `/summary` 兼容旧卡片，前端 Phase 13 使用新接口。

### 2. 控制动作未覆盖 Phase 13 全量要求

当前已有：
- 全局 `pause`
- 全局 `resume`
- 旧语义 `retry-failed`（针对 failed async jobs，不是平台任务）

当前缺失：
- 批次取消
- 单任务取消
- 全局清空 `pending/queued`
- 批次级失败重试
- 单任务失败重试
- 失败重试返回“新批次数量 / 新批次 ID 列表”而非仅旧 job count

### 3. 服务层需要从“旧 worker job 控制”抬升到“平台任务控制”

Phase 13 的动作语义应围绕 `platform_tasks` / `task_batches`：

- `cancel batch`：将目标批次中可取消的 `pending/queued/running` 任务改到取消语义，并刷新批次聚合状态。
- `cancel task`：只作用于单个平台任务。
- `clear queue`：只处理全局 `pending/queued` 平台任务，不能误改已运行任务。
- `retry failed`：针对 `failed` 平台任务重新通过 `TaskPlatformService` 生成新批次，而不是简单把旧 `async_jobs.status` 改回 `ready`。

## Missing Frontend States & Interactions

- 需要批次筛选栏：仅保留 `status` 与 `source_type`，符合锁定决策。
- 需要批次选中态与任务明细联动：点击批次后加载该批次任务，并默认把异常 / 进行中排前。
- 需要顶部概览卡同时呈现：批次状态、任务状态、队列暂停态、待处理规模、最近失败提示。
- 需要 destructive action confirm：至少覆盖批次取消、任务取消、清空队列；确认文本展示受影响数量。
- 需要重试成功 toast：展示创建的重试任务数量，并提供跳到新批次的入口。
- 需要自动刷新与手动刷新并存；自动刷新间隔可延续 30 秒。

## Recommended Implementation Order

1. **先补后端监控读契约**：新增平台概览接口，补齐批次/任务排序与控制所需计数字段，保证前端不靠拼接旧 `/summary` 与 `/jobs` 做业务判断。
2. **再补控制动作服务与路由**：先全局 pause/resume/clear，再批次/任务 cancel，再失败 retry；所有动作都返回统一反馈载荷。
3. **最后扩前端页面**：先完成批次优先监控布局与数据加载，再挂接控制动作、强确认和 retry 跳转反馈。
4. **验证顺序**：后端 service/handler 测试 → 前端最小 DOM 行为测试（若已有）或以 handler/service 自动化为主 → 最后人工验证后台页交互。

## Validation Architecture

### Test Surface

- **Service 层**：`internal/service/admin_service_test.go` 适合锁定平台概览、pause/resume、clear/cancel/retry 语义。
- **Handler 层**：`internal/handler/admin_handler_test.go` 适合锁定新路由、参数校验、JSON 反馈结构。
- **Repository / read model 层**：若新增平台概览聚合 SQL，优先新增独立测试文件或扩 `task_read_service_test.go`。
- **Frontend**：当前 `web/admin` 是静态页面，无现成 JS 测试基建；Phase 13 计划不应把关键 correctness 压给手工测试，核心行为需以后端自动化为主，页面只保留人工验证。

### Fast Automated Commands

- 快速服务验证：`go test ./internal/service/... -run "Admin|TaskRead" -count=1`
- 快速 handler 验证：`go test ./internal/handler/... -run "Admin" -count=1`
- 波次回归：`go test ./internal/service/... ./internal/handler/... -count=1`

### Validation Architecture Requirements

- 每个计划任务都要有 `<automated>` 命令；若页面交互需要人工确认，仍要先以 handler/service 自动化证明后端契约正确。
- 破坏性动作必须既测成功路径，也测边界：空队列、非法 ID、非 `failed` 重试、`running` 不受 clear 影响。
- 失败重试必须验证“新批次语义”而非“原任务复活”。

## Pitfalls To Avoid

- **不要把 Phase 13 退回到裸 `async_jobs` 模式**：这会破坏 Phase 11/12 已建立的平台语义。
- **不要把重试实现成 `job.status=ready`**：这与“新触发动作 = 新批次”的锁定决策冲突。
- **不要让 clear queue 影响 running**：用户已锁定范围仅限 `pending/queued`。
- **不要把 pause/resume 做成批次级控制**：本阶段只允许全局队列暂停与恢复。
- **不要只做前端展示，不补自动化**：Nyquist 会要求每个任务都有 automated verify。
- **不要引入 Phase 14 范围**：补跑恢复、失败隔离增强、高级优先级都不在本阶段。

## Planning Implications

- 最合理的拆分仍是 ROADMAP 里的 4 个计划：
  - `13-01` 批次主视图与平台概览
  - `13-02` 队列统计与任务明细接口补强
  - `13-03` 暂停 / 继续 / 取消 / 清空控制动作
  - `13-04` 失败任务重试与操作反馈
- 为保证并行度，可把读契约与 UI 主视图分成波次 1 / 2：先后端契约，再页面接线；控制动作与失败重试再串行接在后面，避免共享 `admin_service` / `admin_handler` 文件冲突。

---

*Phase: 13-backend-monitoring-queue-control*
*Research completed: 2026-03-27*
