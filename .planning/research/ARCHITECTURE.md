# Architecture Research: ACGWarehouse v3.0 导入后任务平台化

**Domain:** 导入后任务平台架构
**Milestone:** v3.0 导入后任务平台化
**Researched:** 2026-03-23
**Confidence:** HIGH

## Recommended Architecture Shape

导入后的处理链路建议统一为：

`导入批次 -> 导入后任务批次 -> 任务分发 -> worker 执行 -> 状态回写 -> 后台监控 / 控制`

关键点不是换技术栈，而是把现有分散的导入后动作收敛到同一模型中。

## New / Modified Components

| Component | Type | Purpose |
|-----------|------|---------|
| 导入后批次模型 | NEW | 标识一次导入后产生的处理批次 |
| 平台任务模型 | NEW | 统一表示 AI 标签等后台任务 |
| 任务状态机 | NEW | 规范等待、执行、成功、失败、取消等状态 |
| 调度入口 | NEW / MODIFY | 统一负责创建并分发导入后任务 |
| Admin 队列视图 | MODIFY | 展示批次、任务状态和控制动作 |
| 控制动作接口 | NEW | 暂停、继续、重试、取消、清空 |

## Integration Points

- **导入流程**：需要有明确的“导入已可进入后处理”时机，避免文件未稳定就启动下游任务
- **AI 标签服务**：继续复用现有 provider / adapter，只改变触发入口与执行反馈
- **后台管理页面**：接入新的批次列表、状态统计和控制动作
- **数据库**：持久化批次、任务、尝试次数和当前状态，保证重启后可恢复

## Suggested Build Order

1. 先收敛数据模型与状态机
2. 再挂接导入流程，形成自动入队
3. 然后补后台监控与控制动作
4. 最后补历史未打标签图片的补跑与恢复体验

## Main Architectural Risks

- 同时保留旧入口和新平台入口，容易形成双轨逻辑
- 没有批次模型时，后台只能看到零散任务，无法解释“这次导入进展如何”
- 没有幂等 / 去重规则时，重复触发会导致任务堆积和重复推理

## Sources

- Immich Jobs / Workers
- Adobe AEM Asset Microservices
- LibrePhotos Job System

---
*Architecture research for milestone v3.0*
