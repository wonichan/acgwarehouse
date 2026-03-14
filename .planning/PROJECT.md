# ACGWarehouse（二次元图片库）

## What This Is

ACGWarehouse 是一个面向二次元爱好者的图片库管理系统，支持本地图片扫描入库、AI 角色识别、自动标签生成、相似图片检测去重、以图搜图等功能。用户可以方便地管理和检索大量二次元图片资源。

## Core Value

让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] 图片扫描入库（支持本地文件夹监控）
- [ ] 角色识别（调用 AI 识别动漫角色）
- [ ] 自动打标签（画师、原作、类型）
- [ ] 相似图片检测（去重）
- [ ] 搜图功能（以图搜图）
- [ ] 收藏夹/相册管理

### Out of Scope

- 视频管理 — 专注图片，视频复杂度更高
- 社交功能 — 定位为个人图片库，非社区平台
- 云同步 — v1 专注本地部署

## Context

### 技术架构

**Go 后端:**
- 图片扫描服务
- SQLite/PostgreSQL 数据库
- AI 接口（角色识别、标签生成）
- RESTful API

**Flutter 前端:**
- 图片浏览（瀑布流、网格）
- 标签筛选、搜索
- 批量操作
- 收藏管理

### 可选特性

- Docker 部署
- Web 管理后台

## Constraints

- **技术栈**: Go 后端 + Flutter 前端 — 用户明确指定
- **数据库**: SQLite（轻量）或 PostgreSQL（生产） — 支持两种模式
- **AI 服务**: 需要对接外部 AI API 进行角色识别和标签生成

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Flutter 架构 | Go 高性能适合图片处理，Flutter 跨平台 | — Pending |
| SQLite/PostgreSQL 双支持 | 轻量部署和生产环境兼顾 | — Pending |
| AI 外部服务 | 避免自建模型，降低复杂度 | — Pending |

---
*Last updated: 2026-03-14 after initialization*