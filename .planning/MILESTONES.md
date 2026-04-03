# Milestones

## v3.0 导入后任务平台化 (Shipped: 2026-03-29)

**Phases completed:** 4 phases (11-14), 15 plans, 28 tasks

**Key accomplishments:**

- ✓ 统一导入后任务平台：批次模型、平台任务语义、去重规则与 `async_jobs` 执行层接线落地。
- ✓ 导入后 AI 自动入队：建立“缩略图已完成 + 无 AI 标签 + 无活动 AI 任务”的自动调度闭环。
- ✓ 后台批次优先监控台：支持概览统计、批次 / 任务明细、暂停 / 恢复 / 取消 / 清空 / 重试控制。
- ✓ 运营恢复闭环：支持过滤感知的补跑 preview/execute、单图失败隔离、分组失败摘要与 retry hint。
- ✓ 重试语义升级：失败任务重试统一创建新批次，避免破坏历史追踪。

**Stats:**

- 时间线: 6 天 (2026-03-24 → 2026-03-29)
- 提交数: 254 (`v2.0..HEAD`)
- 代码变化: 162 files changed, +23,134 / -5,321
- 需求覆盖: 15/15（依据 phase-level verification 回填归档）

---

**Archives:**

- `.planning/milestones/v3.0-ROADMAP.md`
- `.planning/milestones/v3.0-REQUIREMENTS.md`
- `.planning/milestones/v3.0-MILESTONE-AUDIT.md`

---

## v2.0 UI/UX 重构与多端适配 (Shipped: 2026-03-22)

**Phases completed:** 4 phases (7-10), 20 plans, 61+ tests passing

**Key accomplishments:**

- ✓ 双平台架构基础（AdaptiveApp + NavigationProvider）
- ✓ Windows Fluent Design UI（NavigationView、CommandBar、窗口控制）
- ✓ Android Material 3 UI（NavigationBar/Rail、响应式网格）
- ✓ 统一主题系统（柔和粉紫色系、明暗切换、持久化）
- ✓ 响应式断点系统（600px/900px 自适应布局）
- ✓ 触摸手势优化（滑动浏览、双击缩放、下拉刷新）

**Stats:**

- 代码量: ~32,000 行 (Go 18,450 + Dart 13,550)
- 新增 Dart 代码: ~5,760 行
- 提交数: 42 (v2.0 期间)
- 时间线: 3 天 (2026-03-20 → 2026-03-22)
- 需求覆盖: 22/22 (100%)

**Archives:**

- `.planning/milestones/v2.0-ROADMAP.md`
- `.planning/milestones/v2.0-REQUIREMENTS.md`
- `.planning/milestones/v2.0-MILESTONE-AUDIT.md`

---

## v1.0 MVP (Shipped: 2026-03-19)

**Phases completed:** 6 phases, 28 plans

**Key accomplishments:**

- ✓ Go 后端项目骨架与 SQLite 数据库 Schema 初始化
- ✓ 图片扫描服务、文件夹监控与异步任务基础设施
- ✓ 缩略图生成、感知哈希计算与 Flutter 图片浏览界面
- ✓ 千问/豆包 AI 标签集成与标签治理能力
- ✓ 重复检测、以图搜图与搜索功能
- ✓ Docker Compose 单机部署与 Web 管理后台

**Stats:**

- 代码量: ~26,240 行 (Go 18,450 + Dart 7,790)
- 提交数: 189
- 时间线: 6 天 (2026-03-14 → 2026-03-20)

**Archives:**

- `.planning/milestones/v1.0-ROADMAP.md`
- `.planning/milestones/v1.0-REQUIREMENTS.md`
- `.planning/milestones/v1.0-MILESTONE-AUDIT.md`

---
