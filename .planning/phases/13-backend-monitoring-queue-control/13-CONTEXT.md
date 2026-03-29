# Phase 13: 后台监控与队列控制 - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付后台任务平台页中的按批次监控、队列状态查看，以及暂停 / 继续 / 取消 / 清空 / 失败重试这些核心控制动作。

范围基于既有 v3.0 平台模型、批次 / 任务读模型和后台管理页面展开，不在这里扩张为 Phase 14 的补跑恢复闭环、失败隔离增强、更多任务类型，或高级优先级 / 调度系统。

</domain>

<decisions>
## Implementation Decisions

### 批次主视图
- 任务平台页采用“批次列表在上、任务明细在下”的同页结构，以批次作为默认入口，不做纯任务首页或独立详情页作为主路径。
- 批次列表默认按“未完成优先，其次按最新时间倒序”排序，优先把 `running` / `pending` / `queued` / `partial_failed` 批次顶到前面。
- 批次列表使用表格行 + 状态徽标的呈现方式，而不是卡片堆叠；首页筛选先只暴露批次状态与来源类型。
- 每个批次默认直接显示状态计数、任务类型计数、来源摘要和最新失败摘要；任务类型分布以紧凑方式直接露出。
- 选中批次后，任务明细默认优先展示异常与进行中的任务，而不是先把全部已完成任务铺开。
- 页面保留自动刷新 + 手动刷新，维持典型监控页的浏览节奏。

### 控制作用域
- `pause` / `resume` 仅作用于全局队列，不提供批次级或任务级暂停。
- `cancel` 以批次级为主入口，并允许在任务明细中对单个任务执行补充取消。
- `clear queue` 表示清空全局尚未执行的 `pending` / `queued` 任务，不影响已经 `running` 的任务。
- `cancel` 与 `clear queue` 都属于破坏性动作，必须强确认，并在确认中明确影响范围与任务数量。
- 管理后台要服务运营控制，而不是把管理员带回逐图干预模式。

### 失败重试入口
- 失败重试的主入口放在批次级，管理员优先从批次视角重试该批次内的失败任务。
- 任务明细中保留单任务补充重试，用于少量异常任务的精细处理。
- 仅 `failed` 状态任务允许进入重试；`cancelled`、`skipped`、`completed` 不纳入本阶段重试入口。
- 重试后给出 toast 反馈，明确创建的重试任务数量，并提供跳转到新重试批次的入口。
- 沿用 Phase 11 “一次触发动作 = 一个批次”的语义，重试视作新的触发动作，而不是在原批次内静默覆盖状态。

### 状态与信息密度
- 页面顶部同时显示平台统计与队列运行态，兼顾业务监控与运行态感知。
- 顶层概览不能只看裸 `async_jobs` 数量，还应反映批次 / 平台任务状态以及队列是否暂停、待处理规模等关键信号。
- 任务明细表优先展示文件名、任务类型、状态、错误摘要；文件路径与底层 ID 属于次级信息。
- 保留独立异常区，延续现有“最近错误”区域的思路，同时在批次和任务行内显示错误摘要。
- 整体信息密度偏“监控 + 排障”，不是纯日志流，也不是极简状态页。

### OpenCode's Discretion
- 自动刷新具体间隔（可沿用当前 30 秒基线）与手动刷新交互细节。
- 状态徽标颜色、图标、文案细节，以及批次表格列宽和响应式布局。
- 强确认弹窗的文案、风险提示形式，以及任务数量的摘要展示方式。
- “新重试批次”在界面中的跳转表现（按钮、链接或 toast 内动作）。

</decisions>

<specifics>
## Specific Ideas

- 页面应该更像“批次优先的监控台”，不是纯日志流，也不是逐任务控制台。
- 默认先把异常与进行中的内容顶出来，管理员进入页面就能先看到需要处理的东西。
- 重试不是悄悄改原状态，而是明确形成新的可追踪动作，并把人带到新批次。
- 控制动作以运营效率为主，但破坏性动作必须明显告知影响范围。

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/service/task_read_service.go`: 已经把批次 / 任务读模型封装为后台可直接消费的 JSON 模型，包含状态计数、任务类型计数、失败摘要等字段。
- `internal/repository/task_batch_read_repository.go`: 已支持按 `status`、`source_type`、`batch_id`、`task_type` 查询批次与任务，是 Phase 13 首页筛选与下钻明细的直接数据来源。
- `internal/handler/admin_handler.go`: 已有 `GetTaskBatches`、`GetTasks`、`PauseBackgroundTasks`、`ResumeBackgroundTasks`、`RetryFailedJobs` 等后台接口骨架。
- `internal/handler/routes.go`: 已注册 `/admin/api/task-batches`、`/admin/api/tasks` 和现有的队列控制路由，可在此基础上扩展取消 / 清空动作。
- `web/admin/index.html` 与 `web/admin/app.js`: 已有管理后台页面骨架、全局控制按钮、最近错误区和自动刷新机制，可直接扩展为批次优先监控页。
- `internal/worker/job_manager.go`: 已提供 `Pause`、`Resume`、`IsPaused`、`QueueSize`、`GetStats` 等运行态能力，可支撑全局队列状态展示。
- `internal/domain/task_batch.go` 与 `internal/domain/platform_task.go`: 已定义批次 / 任务状态、来源类型、取消态与任务类型，是本阶段展示与动作语义的基础。

### Established Patterns
- 后台接口统一走 `/admin/api/*` 路由与 JSON 返回风格，动作接口使用 `success` / `message` / `data` 结构。
- Phase 11 已锁定“批次是自然单位”，后台应围绕 `task_batches` / `platform_tasks` 读模型，而不是退回到裸 `async_jobs` 视角。
- Phase 11 已预留 `cancelled`、`partial_failed` 等平台语义；Phase 13 的工作是把这些状态与动作真正暴露到后台监控页。
- Phase 12 已锁定自动入队由定时扫描补偿驱动，因此本阶段重点是可见性与控制，而不是重新讨论入队策略。

### Integration Points
- 需要在现有 admin 路由 / handler / service 链路上扩展取消、清空与更细粒度重试动作。
- 需要把批次列表与任务明细接到现有 `web/admin` 页面结构中，形成“批次在上、明细在下”的同页监控布局。
- 需要把 `job_manager` 的暂停状态、队列规模等运行态与平台批次 / 任务统计一起暴露到监控页顶部概览。
- 需要保持“重试形成新触发动作”的批次语义，与 Phase 11 的批次模型保持一致。

</code_context>

<deferred>
## Deferred Ideas

无 —— 本次讨论保持在 Phase 13 范围内。

</deferred>

---

*Phase: 13-backend-monitoring-queue-control*
*Context gathered: 2026-03-27*
