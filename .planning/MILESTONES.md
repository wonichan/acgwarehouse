# Milestones

## v4.0 Windows Photos 风格重构与计算层拆分 (Shipped: 2026-04-10)

**Phases completed:** 7 phases, 22 plans, 45 tasks

**Key accomplishments:**

- Go-owned sidecar runtime now enforces bounded startup, degraded transition, and shutdown reap semantics while app lifecycle and base health boundaries remain stable.
- Go now atomically publishes runtime `go.base_url`, and Flutter consumes it at startup so API traffic no longer relies on fixed Go ports in product behavior.
- Admin overview now exposes bounded sidecar diagnostics while degraded sidecar scenarios keep Go serving paths usable and verifiable across Go+Flutter checks.
- Python 侧车已完整提供 SHA256 + 256-bit pHash 计算、Union-Find 分组、多维推荐评分，以及可轮询的异步检测任务 API。
- Go 数据层已完成 256-bit pHash 与结构化推荐依据扩展，能够稳定接收并持久化 Python 侧返回的 phash 与推荐结果。
- Go 已完成重复检测链路迁移：由 Python 侧车负责计算，Go 负责任务编排与持久化，并在侧车不可用时提供可诊断 503。
- Delivered a tested desktop shell contract where top-level search/import/settings are persistent and shell-owned, while gallery/search pages stay focused on page content.
- Delivered a tested desktop gallery workspace that is grid-first, keeps tiles square-oriented, and applies right-panel filters immediately without modal flow.
- Delivered a real desktop import action path from top-bar button to backend queue trigger, with bounded queued/failure user feedback and test coverage on both Go and Flutter sides.
- Windows desktop now has a real secondary viewer-window bootstrap path with serializable viewer sessions, a dedicated launch coordinator, and a placeholder spawned-window host ready for Phase 18-02 workspace mounting.
- Phase 19 desktop UI now exposes a Fluent-native, list-first governance workspace with explicit merge/bulk tools and verified gallery drilldown behavior.
- Real-time monitoring backend now ships with an overview event bus, an authenticated monitoring WebSocket, and a sidecar restart endpoint that reports interrupted running-task impact before execution.
- Typed Flutter monitoring contracts now cover admin overview, batch/task drilldown, sidecar restart impact, and provider-managed realtime websocket state.
- Desktop operations monitoring now ships as a real 6th Fluent shell page with batch drilldown, sidecar diagnostics, reconnect/retry handling, and restart confirmation UX.
- Portable runtime layout resolution, packaged sidecar bootstrap wiring, and Python CLI startup safety for the Windows bundle path.
- Flutter-first packaged startup orchestration with manifest wait, classified startup failure UI, manifest path handoff, and bounded Go shutdown.
- Windows x64 portable packaging now builds the Flutter launcher, Go runtime, Python sidecar, and ZIP artifact through one PowerShell command with smoke-checked operator docs.

---

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
