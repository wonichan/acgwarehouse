# ACGWarehouse（二次元图片库）

## Current State

ACGWarehouse 已于 `v3.0` 完成“导入后任务平台化”里程碑，当前产品已经具备图片扫描入库、AI 自动标签、重复检测、搜索、收藏夹、多端访问，以及导入后自动入队、批次监控、队列控制、补跑恢复与失败诊断能力。项目当前处于 `v4.0` 执行期，`Phase 15` 已完成计算侧车基础设施（Go 侧生命周期编排、运行时地址 manifest 发现、降级可用与可观测性边界），下一步进入重复检测计算迁移阶段。

## Current Milestone: v4.0 Windows Photos 风格重构与计算层拆分

**Goal:** 在保留现有 `Go + Flutter + SQLite` 主路径可运行的前提下，同时完成 Windows Photos 风格桌面体验重构、关键内部边界整理，并把图像计算能力正式拆入 `Python` 侧车。

**Target features:**
- Windows 桌面主体验重构：侧边导航、顶部工具栏、图库网格 / 瀑布流、查看器与 filmstrip 风格收敛
- 后端 / 前端边界整理：围绕图库浏览、导入后任务、重复检测、AI 标签等主链路做结构性重整
- `Python` 计算侧车落地：先承接重复检测 / 图像哈希 / 相似度类计算，并由 `Go` 负责编排与生命周期管理
- 为后续计算任务扩展预留统一计算接口、打包路径和进程治理策略

## What This Is

ACGWarehouse 是一个面向二次元爱好者的本地图片库管理系统，支持图片扫描入库、AI 自动标签生成、相似图片检测去重、以图搜图、收藏夹管理，以及 Windows / Android / Web 多端访问。系统现在还提供导入后任务平台，用统一批次与后台运营工具把 AI 标签等异步整理能力收敛到同一条可观测、可控制、可恢复的工作流里。

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
- ✓ 自适应导航（桌面 `NavigationRail` / 手机 `NavigationBar`）— v2.0
- ✓ 二次元定制设计系统（柔和粉紫色系主题）— v2.0
- ✓ 导入后任务平台基础（统一批次、任务与生命周期模型）— v3.0
- ✓ 导入完成后的 AI 标签自动入队（默认只覆盖无 AI 标签图片）— v3.0
- ✓ 后台队列监控与控制（按批次监控、暂停 / 继续 / 重试 / 取消 / 清空）— v3.0
- ✓ “未打 AI 标签图片”批量补跑 — v3.0
- ✓ 导入后任务失败隔离、分组失败摘要与恢复提示 — v3.0
- ✓ Go ↔ Python 计算侧车基础设施（生命周期、manifest 发现、降级可用边界）— v4.0 Phase 15

### Active

- [ ] Windows 桌面端导航、工具栏、图库与查看器重构为更接近 Windows Photos 的主交互
- [ ] 梳理图库浏览、导入后任务、AI 标签与重复检测相关的前后端结构边界
- [ ] 引入 Python 计算侧车，承接重复检测 / 图像哈希 / 相似度等计算任务
- [ ] 建立 Go ↔ Python 生命周期、打包、错误恢复与回退策略
- [ ] 为后续计算任务扩展保留统一任务接口

### Out of Scope

- 视频管理 — 专注图片，视频处理链路与成本更高
- 社交功能 — 定位为个人图片库，非社区平台
- 云同步 — 当前仍聚焦本地部署与单机流程
- PostgreSQL 生产路径 — 当前主路径仍为 SQLite
- 分布式多机任务调度 — 先继续验证单机任务平台与运营模型
- 第三方任务插件市场 — 先把内建任务平台扩展能力做稳，再考虑开放生态
- iOS / macOS 客户端支持 — 本里程碑聚焦 Windows 桌面重构，不扩展新端
- Linux 桌面端支持 — 先完成 Windows 主路径和 Python 侧车治理，再评估跨端复制

## Context

### 当前状态（v3.0 shipped）

**代码量:** ~32,000 行（Go 18,450 + Dart 13,550）  
**技术栈:** Go 1.23 + Gin + SQLite + Flutter（Windows / Android / Web）  
**部署:** Docker Compose 单机部署

### 技术架构

**Go 后端:**
- 图片扫描服务（支持文件夹监控）
- SQLite 数据库
- AI 接口（千问 / 豆包多模态标签生成）
- RESTful API（Gin）
- 导入后任务平台（批次、平台任务、调度与恢复）
- 重复检测服务（SHA256 + pHash）
- 搜索服务（FTS + 标签）

**Flutter / Web 前端:**
- **Windows:** Fluent Design UI（NavigationView 侧边导航）
- **Android:** Material 3 UI（NavigationBar / NavigationRail 自适应导航）
- **Web / Admin:** 响应式管理与运营后台
- 统一主题系统（柔和粉紫色系）
- 图片浏览（网格、瀑布流）
- 标签筛选、搜索
- AI 标签复核界面
- 批量选择与操作
- 收藏夹管理

### 已知问题

- v1.0 的 Phase 2 / 4 / 5 仍缺少独立 `VERIFICATION.md` 文档（历史技术债）
- Flutter 代码仍有少量 `analyze` warning / info（无 error）
- v3.0 的里程碑审计是在归档阶段基于 phase-level verification 汇总生成，而不是在里程碑完成当天单独执行
- 当前仓库尚未存在 Python 服务实现，相关重复检测 / 图像哈希能力仍由 Go 直接承担

## Current Milestone Goals

- 完成 Windows Photos 风格桌面信息架构与核心浏览 / 查看流程重构
- 完成 Python 计算侧车首期落地与 Go 编排 / 生命周期治理
- 重构重复检测 / 图像哈希等计算链路，并为后续计算任务外移建立边界
- 保持现有单机部署路径可运行，并补齐 Windows 打包、诊断与回退策略

<details>
<summary>Archived milestone brief: v3.0 导入后任务平台化</summary>

**Goal:** 把导入后的 AI 标签等后台处理任务统一纳入可观测、可控制、可恢复的任务平台，消除海量导入后的逐张人工触发。

**Delivered:**
- 导入批次与导入后任务的统一平台模型
- 导入完成后的 AI 标签自动入队
- 后台按批次监控任务进度与状态
- 队列暂停 / 继续 / 重试 / 取消 / 清空
- “未打 AI 标签图片”批量补加入队
- 单图失败隔离与失败可见性

</details>

## Constraints

- **技术栈**: Go 主控 + Flutter 前端 + Python 计算侧车 — 在现有主路径基础上演进，不推翻单机架构
- **数据库**: SQLite 单机主路径 — 下一里程碑继续以单机部署为默认
- **AI 服务**: 千问 / 豆包多模态 API（OpenAI 兼容格式）
- **部署模式**: 单机 Docker Compose 部署
- **运营目标**: 面向 10k+ 图片导入后的后台整理、补跑与恢复，而不是实时交互式推理
- **交付边界**: 本期允许引入 Python 侧车，但必须保证 Windows 单机可打包、可诊断、可回退

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
| **v3.0: 导入后任务统一平台** | 解决海量导入后任务分散、手动触发和不可观测的问题 | ✓ 已发版 |
| **v3.0: 默认仅无 AI 标签图片自动入队** | 控制成本与重复处理风险，优先覆盖真正缺失标签的图片 | ✓ Phase 12 验证通过 |
| **v3.0: `async_jobs` 仅保留执行层角色** | 保留现有 worker 引擎，同时把产品语义提升到 batch/task 模型 | ✓ Phase 11 验证通过 |
| **v3.0: 去重按 `image_version_key + task_type`** | 防止未变更图片因重复触发或路径变化而重复入队 | ✓ Phase 11 验证通过 |
| **v3.0: 后台主入口采用批次优先监控台** | 运营排障应先看批次，再下钻任务明细 | ✓ Phase 13 验证通过 |
| **v3.0: 平台概览使用独立 overview 契约** | 顶部监控必须直接反映 queue / batches / tasks，而不是继续拼旧 summary | ✓ Phase 13 验证通过 |
| **v3.0: 破坏性动作必须强确认并返回影响数量** | pause / resume / cancel / clear / retry 需要可控且可解释的运营反馈 | ✓ Phase 13 验证通过 |
| **v3.0: failed retry 形成新批次而不是复活旧任务** | 保留失败历史可追踪性，并保持“每次触发动作 = 一个新批次”的平台语义 | ✓ Phase 13 验证通过 |
| **v3.0: 回填采用 preview-first 流程** | 让运营在执行前先看到可入队数量、跳过原因和 no-op 反馈 | ✓ Phase 14 验证通过 |
| **v3.0: 失败摘要按 grouped failure reasons 暴露** | 直接给运营提供 reason/count/retry hint，而不是只看一堆单任务错误 | ✓ Phase 14 验证通过 |
| **v4.0: 桌面体验以 Windows Photos 为主要参考** | 优先重构用户可感知的浏览、查看与导航体验，但不做像素级复刻 | — Pending |
| **v4.0: Python 仅承担计算层职责** | 保持 Go 继续负责编排、任务平台、持久化与进程治理，避免职责混乱 | ✓ Phase 15 建立基础边界 |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-04 after Phase 15 completion*
