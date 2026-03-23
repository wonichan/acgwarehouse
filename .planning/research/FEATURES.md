# Feature Research: ACGWarehouse v3.0 导入后任务平台化

**Domain:** 导入后任务平台 / AI 标签队列
**Milestone:** v3.0 导入后任务平台化
**Researched:** 2026-03-23
**Confidence:** HIGH

## Table Stakes（本次必须具备）

| Feature | Why It Matters | Complexity |
|---------|----------------|------------|
| 导入批次视图 | 让用户知道“这次导入”后面发生了什么 | MEDIUM |
| 自动入队 AI 标签任务 | 解决“导入后还要逐张点”的核心痛点 | MEDIUM |
| 队列状态监控 | 区分待处理、执行中、成功、失败、已取消 | MEDIUM |
| 队列控制动作 | 暂停、继续、重试、取消、清空 | MEDIUM |
| 未打标签图片批量补跑 | 历史图片也能补齐 AI 标签 | LOW-MEDIUM |
| 单图失败隔离 | 避免一张图失败拖垮整个批次 | MEDIUM |

## Differentiators（做得好会明显更顺手）

| Feature | User Value |
|---------|------------|
| 按批次展示进度 | 用户更容易把导入和后处理关联起来 |
| 失败原因摘要 | 用户知道是 API、数据还是图片本身的问题 |
| 自动过滤已打 AI 标签图片 | 减少重复处理和成本浪费 |
| 补跑入口直接面向“无 AI 标签图片” | 运营动作更直接，避免手工筛图 |

## Anti-Features（本次不该扩张）

| Feature | Why Avoid |
|---------|-----------|
| 先做多队列优先级系统 | 会把 v3.0 从平台化首版拖成调度系统重构 |
| 先做多机 worker 编排 | 现有部署模式不需要，验证成本高 |
| 默认允许所有图片反复自动重跑 AI | 成本与重复处理风险都高 |
| 把任务平台立即做成开放插件市场 | 边界不清，会让实现和运营复杂度失控 |

## Requirement Shaping Notes

- 本次需求重点是“导入后后台任务统一平台”，不是仅做 AI 标签按钮升级
- AI 标签是首个重点任务类型，但平台边界应允许后续纳入更多导入后任务
- 后台能力必须覆盖“看得见 + 控得住 + 失败可恢复”三件事

## Sources

- Immich Discussion / Jobs Docs — 大批量作业的可见性诉求
- digiKam Face / Maintenance Docs — 长任务后台进度与恢复模式
- Wistia Bulk Tag API — 批量标签的后台作业模型

---
*Feature research for milestone v3.0*
