# Stack Research: ACGWarehouse v3.0 导入后任务平台化

**Domain:** 导入后后台任务平台
**Milestone:** v3.0 导入后任务平台化
**Researched:** 2026-03-23
**Confidence:** HIGH

## Executive Summary

本次里程碑不需要引入新的分布式基础设施，推荐沿用现有 Go + Gin + SQLite + Flutter + Admin 的单机主路径，在现有后端里补齐统一任务平台层。外部案例普遍说明：10k+ 图片场景的关键不是“再加一个批量按钮”，而是把导入后处理建模成**批次 + 任务 + 状态 + 控制动作**。

## Recommended Additions

| Addition | Purpose | Why Now |
|----------|---------|---------|
| 统一导入批次模型 | 把一次导入后的任务归拢到同一个批次 | 后台需要按批次监控，而不是逐图触发 |
| 统一任务状态机 | 管理待处理 / 执行中 / 成功 / 失败 / 已取消 | 队列控制与失败恢复都依赖一致状态 |
| 后台管理接口 | 暴露统计、明细、暂停、继续、重试、取消、清空 | 用户明确要求在后台监控和操作这些队列 |
| AI 任务入队规则 | 只把无 AI 标签图片自动入队 | 控制成本，降低重复处理 |
| 批量补跑入口 | 将“未打 AI 标签图片”补加入队 | 覆盖历史存量图片整理场景 |

## Recommended Reuse

- **Go 后端服务与现有异步能力**：优先在现有进程内统一任务模型，而不是新增独立调度服务
- **SQLite 主路径**：先用数据库持久化批次 / 任务 / 状态，不为 v3.0 提前引入消息中间件
- **现有 Admin 后台**：把队列监控与控制放到已有后台，而不是新建单独运维面板
- **现有 AI 服务适配层**：继续复用千问 / 豆包集成，只改任务触发方式与状态反馈

## What NOT to Add Yet

| Avoid | Why |
|-------|-----|
| 分布式消息队列 / 多机 worker 集群 | 超出单机部署边界，会扩大运维复杂度 |
| 通用插件执行平台 | 任务平台边界还未稳定，过早泛化风险高 |
| 自动重跑所有已有 AI 标签图片 | 直接放大推理成本，也会制造重复整理 |

## Sources

- Immich Jobs / Workers — 队列、worker 与后台作业监控模式
- LibrePhotos Job System — 长任务状态、日志与恢复模式
- PhotoPrism Import Docs — 批量导入安全与自动处理节奏
- Chevereto Bulk Importer — 批量导入状态与失败隔离

---
*Stack research for milestone v3.0*
