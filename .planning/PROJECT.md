# ACGWarehouse（二次元图片库）

## What This Is

ACGWarehouse 是一个面向二次元爱好者的本地图片库管理系统，支持图片扫描入库、AI 自动标签生成、相似图片检测去重、以图搜图、收藏夹管理等功能。通过 Docker Compose 一键部署，实现"存入即整理"的体验。

## Core Value

让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## Requirements

### Validated

- ✓ 图片扫描入库（本地文件夹监控）— v1.0
- ✓ AI 自动标签生成（千问/豆包多模态）— v1.0
- ✓ 相似图片检测与去重 — v1.0
- ✓ 搜索功能（文件名/标签/以图搜图）— v1.0
- ✓ 收藏夹/相册管理 — v1.0
- ✓ 批量操作（选择/标签/移动/删除）— v1.0
- ✓ Docker 部署 — v1.0
- ✓ Web 管理后台 — v1.0

### Active

- [ ] Windows 桌面端 UI 重构（Fluent Design + 二次元风格）
- [ ] Android 移动端 UI 适配（Material 3 + 二次元风格）
- [ ] 响应式布局系统（多端适配）
- [ ] 自适应导航（桌面 NavigationRail / 手机 NavigationBar）
- [ ] 二次元定制设计系统（柔和粉紫色系主题）

### Out of Scope

- 视频管理 — 专注图片，视频复杂度更高
- 社交功能 — 定位为个人图片库，非社区平台
- 云同步 — v1 专注本地部署
- PostgreSQL 生产路径 — v1 仅 SQLite 主路径

## Context

### 当前状态 (v1.0)

**代码量:** ~26,240 行 (Go 18,450 + Dart 7,790)
**技术栈:** Go 1.23 + Gin + SQLite + Flutter Web
**部署:** Docker Compose 单机部署

### 技术架构

**Go 后端:**
- 图片扫描服务（支持文件夹监控）
- SQLite 数据库
- AI 接口（千问/豆包多模态标签生成）
- RESTful API (Gin)
- 异步任务队列
- 重复检测服务（SHA256 + pHash）
- 搜索服务（FTS + 标签）

**Flutter 前端:**
- 图片浏览（网格、瀑布流）
- 标签筛选、搜索
- AI 标签复核界面
- 批量选择与操作
- 收藏夹管理

**部署:**
- Docker Compose 单机部署
- Web 管理后台 (`/admin`)
- 宿主机持久化 (data/, library/)

### 已知问题

- Phase 2、4、5 缺少 VERIFICATION.md 文件（技术债务）
- 部分计划缺少 SUMMARY.md 文件
- Flutter 代码有 9 个 analyze issues（均为 warning/info，无 error）

## Current Milestone: v2.0 UI/UX 重构与多端适配

**Goal:** 为 ACGWarehouse 添加 Windows 桌面端和 Android 移动端支持，打造二次元风格的精美界面

**Target features:**
- Windows 桌面端 Fluent Design UI（柔和粉紫色系）
- Android 移动端 Material 3 UI（柔和粉紫色系）
- 自适应导航系统（桌面 NavigationRail / 手机 NavigationBar）
- 响应式布局系统（多端适配）
- 二次元定制设计系统

**Architecture decisions:**
- Windows UI: fluent_ui 包（Microsoft Fluent Design）
- Android UI: Material 3
- Shared layer: Provider 状态管理、API 服务、数据模型
- Development priority: Windows 优先 → Android 跟进

## Constraints

- **技术栈**: Go 后端 + Flutter 前端 — 用户明确指定
- **数据库**: SQLite（v1 主路径）— PostgreSQL 已移出 v1 范围
- **AI 服务**: 千问/豆包多模态 API（OpenAI 兼容格式）
- **部署模式**: 单机 Docker Compose 部署

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Flutter 架构 | Go 高性能适合图片处理，Flutter 跨平台 | ✓ 验证通过，性能良好 |
| SQLite 主路径 | 简化部署，适合个人图库 | ✓ v1.0 已验证 |
| AI 外部服务 | 避免自建模型，降低复杂度 | ✓ 千问/豆包集成完成 |
| OpenAI 兼容格式 | 千问/豆包都支持，简化实现 | ✓ 统一接口，易扩展 |
| Docker Compose 部署 | 一次命令启动全部服务 | ✓ 验证通过 |
| 标签归并采用精确匹配优先 | 避免过早引入模糊治理误判 | ✓ 待后续优化 |
| 双重哈希检测策略 | SHA256 检测完全相同，pHash 检测相似 | ✓ Union-Find 分组算法 |

---
*Last updated: 2026-03-20 after v2.0 milestone started*