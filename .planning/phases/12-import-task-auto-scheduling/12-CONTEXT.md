# Phase 12: 导入后任务接入与自动调度 - Context

**Gathered:** 2026-03-26
**Status:** Ready for planning

<domain>
## Phase Boundary

将 AI 标签与导入后处理链路接入统一平台，实现导入完成后的自动入队与条件过滤。本阶段聚焦"自动入队"和"条件过滤"两个核心能力，不涉及后台监控 UI（Phase 13）和失败恢复闭环（Phase 14）。

**Success Criteria:**
1. 用户导入图片后，符合条件的图片会自动加入 AI 打标签队列。
2. 默认只有没有 AI 标签的图片才会被自动加入 AI 打标签队列。
3. 导入后的处理不再依赖逐图人工触发。

</domain>

<decisions>
## Implementation Decisions

### 触发时机与方式
- **定时任务扫描补偿**：不采用导入后立即触发或缩略图链式触发，而是通过定时任务扫描需要 AI 标签的图片。
- **扫描频率**：每 5 分钟执行一次扫描。
- **扫描条件**：缩略图已完成 + 无 AI 标签关联。
- **批量限制**：每次扫描最多处理 100 张图片入队。
- **配置开关**：`auto_ai_tag_on_import` 配置项，默认 `true`，可在 `config.yaml` 关闭。

### 入队条件判定
- **判定口径**：检查当前 `image_tag` 表是否有 AI 来源的标签关联，而非检查历史记录。
- **数据模型变更**：需要为 `image_tag` 表添加 `source` 字段（或 `ai_generated` 布尔字段），支持区分 AI 生成标签和手动添加标签。
- **手动删除场景**：用户删除 AI 标签后，系统视为"无 AI 标签"，下次扫描时可重新入队生成。
- **重新生成**：已有 AI 标签的图片重新生成，仅通过手动触发入口，不通过自动入队覆盖。

### 大批量导入控制
- **入队批次**：定时任务每次最多创建 100 个 AI 标签任务。
- **并发控制**：使用现有的 `AITagConcurrencyLimiter` 和 RPM 限制器（配置项 `max_concurrency` 和 `requests_per_minute`）。
- **队列积压**：允许队列积压，依靠并发限制自然消费，不主动暂停入队。

### 手动入口整合
- **复用逻辑**：手动触发入口内部调用 `TaskPlatformService`，与自动入队共享同一套任务创建和入队逻辑。
- **保留接口**：保留现有 HTTP 接口 `/api/v1/images/:id/ai-tags`（单图）和 `/api/v1/images/batch-ai-tags`（批量）。
- **批次来源区分**：手动触发的批次来源类型为 `TaskBatchSourceManualSingle` 或 `TaskBatchSourceManualBatch`，与自动入队的 `TaskBatchSourceImportScan` 区分。

### 错误处理
- **入队失败**：AI 标签入队失败时记录日志，不阻塞缩略图任务标记完成。失败的入队可在下次定时扫描时补偿。

### OpenCode's Discretion
- 定时任务的具体实现方式（goroutine + ticker vs 外部 cron）。
- `image_tag.source` 字段的具体命名和枚举值设计。
- 扫描 SQL 查询的优化策略（索引设计、JOIN 方式）。
- 配置项的具体结构和默认值。

</decisions>

<specifics>
## Specific Ideas

- "定时任务扫描补偿"比链式触发更可靠——即使中间环节失败，下次扫描还会补偿。
- 检查"当前标签关联"而非历史记录，尊重用户删除 AI 标签后重新生成的意愿。
- 每批 100 张是平衡及时性和系统负载的经验值，适合单机 SQLite 部署。

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/service/task_platform_service.go`：`PlanBatch()` 创建批次和任务，`QueueTask()` 任务入队。
- `internal/handler/ai_tag_handler.go`：`enqueueAITagBatch()` 现有入队逻辑可复用。
- `internal/worker/ai_tag_handler.go`：`AITagConcurrencyLimiter` 并发限制器已实现。
- `internal/worker/job_manager.go`：Worker pool 和任务队列管理。
- `internal/domain/platform_task.go`：`PlatformTaskTypeAITagGeneration` 任务类型已定义。
- `internal/domain/task_batch.go`：`TaskBatchSourceImportScan` 批次来源类型已定义。

### Established Patterns
- Phase 11 已建立 `TaskPlatformService` 作为任务平台的统一入口，自动入队应复用此服务。
- 去重逻辑使用 `image_version_key + task_type`，由 `FindActiveByDedupeKey()` 实现。
- 现有 AI 标签 HTTP 接口已成熟，手动触发入口应保持不变，内部调用逻辑统一。

### Integration Points
- **定时任务**：需在 `internal/app/bootstrap.go` 或独立模块中启动定时扫描 goroutine。
- **判定查询**：需新增查询接口，查找"缩略图完成 + 无 AI 标签关联"的图片。
- **数据迁移**：需为 `image_tag` 表添加 `source` 字段的 migration。
- **配置扩展**：需在 `config.yaml` 结构中添加 `auto_ai_tag_on_import` 配置项。

### 需要新增的组件
1. **定时扫描服务**：`internal/service/ai_tag_auto_scheduler.go`（或类似命名）
   - 扫描符合条件的图片
   - 调用 `TaskPlatformService.PlanBatch()` 创建任务
   - 调用 `TaskPlatformService.QueueTask()` 入队
2. **判定查询方法**：在 `ImageRepository` 或新增仓库中添加
   - `FindImagesWithoutAITags(limit int)` 或类似方法
3. **数据库迁移**：`image_tag.source` 字段

</code_context>

<deferred>
## Deferred Ideas

- 后台队列监控 UI（暂停/继续/取消/重试）—— Phase 13。
- 失败隔离、重试和恢复闭环 —— Phase 14。
- "未打标签图片批量补入队"运营入口 —— Phase 14。
- 基于系统负载动态调整并发数 —— 未来优化。

</deferred>

---

*Phase: 12-import-task-auto-scheduling*
*Context gathered: 2026-03-26*