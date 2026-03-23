# ACGWarehouse（二次元图片库）

## What This Is

ACGWarehouse 是一个面向二次元爱好者的本地图片库管理系统，支持图片扫描入库、AI 自动标签生成、相似图片检测去重、以图搜图、收藏夹管理，以及 Windows / Android / Web 多端访问。当前项目正从“功能已具备”推进到“导入后自动整理”，重点补上大批量导入后的后台任务平台能力。

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

- [ ] 导入后任务平台化（统一批次、任务与生命周期模型）
- [ ] AI 标签自动入队与“未打标签图片”批量补跑
- [ ] 后台队列监控与控制（暂停 / 继续 / 重试 / 取消 / 清空）
- [ ] 导入后任务失败隔离、重试与恢复能力

### Out of Scope

- 视频管理 — 专注图片，视频处理链路与成本更高
- 社交功能 — 定位为个人图片库，非社区平台
- 云同步 — 当前仍聚焦本地部署与单机流程
- PostgreSQL 生产路径 — 当前主路径仍为 SQLite
- 分布式多机任务调度 — v3.0 先验证单机任务平台模型
- 第三方任务插件市场 — 先完成内建导入后任务平台，再评估开放扩展

## Context

### 当前状态 (v2.0 shipped)

**代码量:** ~32,000 行 (Go 18,450 + Dart 13,550)
**技术栈:** Go 1.23 + Gin + SQLite + Flutter (Windows/Android/Web)
**部署:** Docker Compose 单机部署

### 技术架构

**Go 后端:**
- 图片扫描服务（支持文件夹监控）
- SQLite 数据库
- AI 接口（千问 / 豆包多模态标签生成）
- RESTful API (Gin)
- 异步任务队列与管理后台接口
- 重复检测服务（SHA256 + pHash）
- 搜索服务（FTS + 标签）

**Flutter 前端:**
- **Windows:** Fluent Design UI（NavigationView 侧边导航）
- **Android:** Material 3 UI（NavigationBar / NavigationRail 自适应导航）
- **Web:** Material 3 UI（响应式布局）
- 统一主题系统（柔和粉紫色系）
- 图片浏览（网格、瀑布流）
- 标签筛选、搜索
- AI 标签复核界面
- 批量选择与操作
- 收藏夹管理

### 已知问题

- 一次性导入上万张图片后，AI 标签仍需要逐图或逐次手动触发，缺少“导入后自动整理”的闭环
- 导入后的后台处理能力缺少统一任务批次、统一队列监控和统一恢复控制模型
- Phase 2、4、5 缺少 VERIFICATION.md 文件（v1.0 技术债务）
- Flutter 代码仍有少量 analyze warning / info（无 error）

### Post-v3.0 候选方向

- iOS / macOS 支持
- Linux 桌面端支持
- 更细粒度的任务优先级与资源配额
- 插件系统与可扩展任务类型

## Current Milestone: v3.0 导入后任务平台化

**Goal:** 把导入后的 AI 标签等后台处理任务统一纳入可观测、可控制、可恢复的任务平台，消除海量导入后的逐张人工触发。

**Target features:**
- 导入批次与导入后任务的统一平台模型
- 导入完成后的 AI 标签自动入队
- 后台按批次监控任务进度与状态
- 队列暂停 / 继续 / 重试 / 取消 / 清空
- “未打 AI 标签图片”批量补加入队
- 单图失败隔离与失败可见性

## Constraints

- **技术栈**: Go 后端 + Flutter 前端 — 保持现有主架构不变
- **数据库**: SQLite 单机主路径 — v3.0 不引入分布式队列基础设施
- **AI 服务**: 千问 / 豆包多模态 API（OpenAI 兼容格式）
- **部署模式**: 单机 Docker Compose 部署
- **运营目标**: 面向 10k+ 图片导入后的后台整理场景，而不是实时交互式推理

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Flutter 架构 | Go 高性能适合图片处理，Flutter 跨平台 | ✓ 验证通过，性能良好 |
| SQLite 主路径 | 简化部署，适合个人图库 | ✓ v1.0 已验证 |
| AI 外部服务 | 避免自建模型，降低复杂度 | ✓ 千问 / 豆包集成完成 |
| OpenAI 兼容格式 | 千问 / 豆包都支持，简化实现 | ✓ 统一接口，易扩展 |
| Docker Compose 部署 | 一次命令启动全部服务 | ✓ 验证通过 |
| 标签归并采用精确匹配优先 | 避免过早引入模糊治理误判 | ✓ 待后续优化 |
| 双重哈希检测策略 | SHA256 检测完全相同，pHash 检测相似 | ✓ Union-Find 分组算法 |
| **v2.0: Windows Fluent UI** | Microsoft 官方设计系统，专业桌面体验 | ✓ 验证通过 |
| **v2.0: Android Material 3** | Google 最新设计系统，统一移动端体验 | ✓ 验证通过 |
| **v2.0: Shared Provider Layer** | 双 UI 框架共享业务逻辑，减少重复 | ✓ 验证通过 |
| **v3.0: 导入后任务统一平台** | 解决海量导入后任务分散、手动触发和不可观测的问题 | — Pending |
| **v3.0: 默认仅无 AI 标签图片自动入队** | 控制成本与重复处理风险，优先覆盖真正缺失标签的图片 | — Pending |

---
*Last updated: 2026-03-23 after milestone v3.0 start*
