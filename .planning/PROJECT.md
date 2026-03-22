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
- ✓ Windows 桌面端 UI（Fluent Design + 二次元风格）— v2.0
- ✓ Android 移动端 UI（Material 3 + 二次元风格）— v2.0
- ✓ 响应式布局系统（多端适配）— v2.0
- ✓ 自适应导航（桌面 NavigationRail / 手机 NavigationBar）— v2.0
- ✓ 二次元定制设计系统（柔和粉紫色系主题）— v2.0

### Active

- [ ] iOS/macOS 支持
- [ ] Linux 桌面端支持
- [ ] 性能优化（大图加载、内存管理）
- [ ] 插件系统

### Out of Scope

- 视频管理 — 专注图片，视频复杂度更高
- 社交功能 — 定位为个人图片库，非社区平台
- 云同步 — v1 专注本地部署
- PostgreSQL 生产路径 — v1 仅 SQLite 主路径

## Context

### 当前状态 (v2.0)

**代码量:** ~32,000 行 (Go 18,450 + Dart 13,550)
**技术栈:** Go 1.23 + Gin + SQLite + Flutter (Windows/Android/Web)
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
- **Windows:** Fluent Design UI（NavigationView 侧边导航）
- **Android:** Material 3 UI（NavigationBar/Rail 自适应导航）
- **Web:** Material 3 UI（响应式布局）
- 统一主题系统（柔和粉紫色系）
- 图片浏览（网格、瀑布流）
- 标签筛选、搜索
- AI 标签复核界面
- 批量选择与操作
- 收藏夹管理

**部署:**
- Docker Compose 单机部署
- Web 管理后台 (`/admin`)
- Windows 桌面端（window_manager）
- Android 移动端
- 宿主机持久化 (data/, library/)

### 已知问题

- Phase 2、4、5 缺少 VERIFICATION.md 文件（v1.0 技术债务）
- Flutter 代码有 9 个 analyze issues（均为 warning/info，无 error）
- ImageGalleryViewer 未集成到导航流（技术债务 TD-04）

## Current Milestone: Planning v2.1 or v3.0

**Shipped v2.0:** UI/UX 重构与多端适配完成

**Goal:** 规划下一里程碑（性能优化、新平台支持、或功能增强）

**Potential Directions:**
- v2.1: 性能优化与稳定性改进
- v2.1: iOS/macOS 支持
- v3.0: 插件系统与扩展架构

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
| **v2.0: Windows Fluent UI** | Microsoft 官方设计系统，专业桌面体验 | ✓ 7/7 requirements complete |
| **v2.0: Android Material 3** | Google 最新设计系统，统一移动端体验 | ✓ 5/5 requirements complete |
| **v2.0: Shared Provider Layer** | 双 UI 框架共享业务逻辑，减少重复 | ✓ 10/10 integration tests pass |
| **v2.0: Pink-Purple Theme** | 柔和粉紫色系契合二次元审美 | ✓ Theme system with persistence |

---
*Last updated: 2026-03-22 after v2.0 milestone shipped*