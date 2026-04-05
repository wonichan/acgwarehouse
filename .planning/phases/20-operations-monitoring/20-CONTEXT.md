# Phase 20: Operations Monitoring - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

管理员可以在桌面端监控导入后任务的批次与任务状态，并诊断 Python sidecar 的运行健康。Phase 20 的交付边界包括：在桌面导航中提供运营监控入口、展示批次列表与任务下钻、展示 sidecar 运行状态与错误摘要、提供手动重启 sidecar 的操作入口。管理员不需要检查外部日志即可完成诊断。

本阶段不包含：完整 Windows 打包（Phase 21）、大图库性能优化（Phase 22）、Python 不可用时的自动降级计算路径（COMP-05 / Phase 22）。

</domain>

<decisions>
## Implementation Decisions

### 页面结构与导航入口
- **D-01:** 运营监控作为桌面侧栏中的**单个独立导航项**，不拆成多个入口。导航项从当前的 5 个扩展到 6 个。
- **D-02:** 监控页内部采用**单页双区布局**：上半区展示批次任务监控，下半区展示 sidecar 诊断面板。两个区域在同一页面内滚动展示，而不是 Tab 切换。
- **D-03:** 监控页继续沿用 Phase 17/19 已验证的独立导航页 + Fluent 壳层工作区模式。

### 批次任务监控视图
- **D-04:** 批次列表采用**批次优先 + 点击下钻任务明细**的两层视角，复用 Phase 13 已建立的后端批次优先监控模型。
- **D-05:** 每个批次行展示以下关键信息：状态标签（pending/running/completed/failed）、进度百分比或进度条、时间信息（创建时间/完成时间）、错误统计与分组摘要。
- **D-06:** 任务明细层展示单任务状态、错误原因、重试提示等，复用 Phase 14 已建立的 grouped failure reasons 契约。

### Sidecar 诊断面板
- **D-07:** Sidecar 诊断采用**状态卡片 + 错误摘要列表**形式：顶部状态卡片显示运行状态（ready 绿/degraded 黄/stopped 红）、运行时长、最后探测时间；下方列表显示最近错误摘要（错误类型 + 时间 + 消息）。
- **D-08:** 诊断范围**扩展到 Go 运行时**：不仅展示 Python sidecar 状态，还展示 Go 层的队列状态、Worker 活跃数等内部运营指标。复用 Phase 13 admin overview 已建立的队列/批次/任务概览契约。

### 手动重启与操作交互
- **D-09:** Sidecar 手动重启按钮**嵌入在 sidecar 诊断卡片内部**，与状态展示在同一视觉上下文中。
- **D-10:** 重启操作采用**二次确认 + 影响说明**流程：确认对话框说明重启将中断正在进行的计算任务，用户确认后才执行。

### 数据刷新与实时性
- **D-11:** 监控页数据使用**WebSocket 实时推送**：批次状态变更、sidecar 状态变化通过 WebSocket 主动推送到前端，而不是依赖定时轮询。
- **D-12:** 进入监控页时自动建立 WebSocket 连接，离开时断开；连接断开时 UI 展示断连提示并提供手动重试。

### API 契约边界
- **D-13:** 前端监控页**复用现有 admin API 端点**（如 /admin/api/overview、/admin/api/batches 等），不新建产品级 API 路径。后端改动最小化。
- **D-14:** Sidecar 诊断信息复用 Phase 15 已建立的 admin overview sidecar 诊断端点，不额外新建诊断 API。

### 边缘状态处理
- **D-15:** 当监控数据加载失败（Go 后端不可达、admin 端点超时等）时，UI 展示**服务不可用提示 + 重试按钮**，不尝试推断或缓存旧状态。
- **D-16:** 监控页自身不应因 sidecar 停止而无法加载——批次列表依赖 Go 后端，即使 sidecar 不可用也应正常展示批次信息。

### the agent's Discretion
- 双区布局的具体高度分配、滚动行为与视觉分隔方式
- 批次列表的排序方式（默认按创建时间倒序还是按状态分组）
- Sidecar 诊断卡片中运行时指标的具体展示数量和格式
- WebSocket 连接失败时的退避策略与最大重试次数
- 进度条的视觉风格（线性 vs 环形）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 20 的目标、依赖关系与 3 条 success criteria（批次任务监控、sidecar 状态与重启、无外部日志诊断）
- `.planning/REQUIREMENTS.md` — `OPS-01`（导入后任务监控入口）、`OPS-02`（sidecar 运行状态 + 错误摘要 + 手动重启）的正式定义
- `.planning/PROJECT.md` — v4.0 里程碑原则、Go 主控 / Flutter 前端 / Python 计算侧车的职责边界、Windows 单机可诊断/可回退约束
- `.planning/STATE.md` — 当前项目状态、v4.0 执行位置与已知 Flutter 基线问题

### Prior Phase Decisions
- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md` — Sidecar runtime 状态机、分层健康语义、admin overview 诊断端点、降级可用边界
- `.planning/phases/16-duplicate-detection-migration/16-CONTEXT.md` — Python 不可用时硬拒绝 + 前置检查机制、计算迁移完成
- `.planning/phases/17-desktop-shell-foundation/17-CONTEXT.md` — 桌面壳层导航结构已收口、独立导航页模式已验证
- `.planning/phases/19-tag-management/19-CONTEXT.md` — 最近一个独立导航页落地经验，可复用页面结构模式

### Existing Backend Monitoring Infrastructure
- `internal/service/admin_service.go` — 管理概览聚合：sidecar 诊断、队列状态、批次概览、失败重试
- `internal/service/task_platform_service.go` — 任务平台：批次规划、任务排队、状态管理与去重
- `internal/service/task_read_service.go` — 任务读取：批次/任务模型、失败分组与重试推荐
- `internal/domain/task_batch.go` — TaskBatch 领域模型与状态常量（pending/running/completed/failed 等）
- `internal/domain/platform_task.go` — PlatformTask 领域模型与任务类型常量
- `internal/sidecar/runtime.go` — Sidecar Runtime 状态机（not_started → starting → ready → degraded → stopped）
- `internal/sidecar/client.go` — Python sidecar HTTP 通信客户端
- `internal/handler/health_handler.go` — Go 健康检查端点（/health, /ready）

### Existing Flutter Desktop Shell
- `flutter_app/lib/app/fluent_app_shell.dart` — 桌面 NavigationView 壳层与导航项定义
- `flutter_app/lib/providers/navigation_provider.dart` — 导航索引与页面标题管理
- `flutter_app/lib/app/fluent_screens.dart` — Fluent 页面实现模式参考
- `flutter_app/lib/config/api_config.dart` — API 端点配置与 baseUrl 管理

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/service/admin_service.go`：已有完整的 admin overview 聚合能力，包括 sidecar 状态、队列信息、批次统计。Phase 20 可直接复用为数据源。
- `internal/service/task_read_service.go`：已封装批次列表查询、任务明细查询、失败分组摘要。前端监控页可直接消费这些契约。
- `internal/sidecar/runtime.go`：已有完整的状态机与探活逻辑，前端只需读取状态即可展示。
- `flutter_app/lib/app/fluent_app_shell.dart`：已有 5 项导航结构，添加第 6 项监控入口的改动边界明确。
- Phase 19 刚完成独立导航页模式验证（标签管理页），监控页可以复用相同模式。

### Established Patterns
- 桌面端独立导航页模式已在 Phase 17/18/19 连续验证三次，Phase 20 继续沿用无需额外探索。
- 后端已有"破坏性动作必须强确认并返回影响数量"的运营设计传统，与本阶段 sidecar 重启的二次确认需求一致。
- Phase 13 已锁定"批次优先监控台"的运营视角，Phase 20 直接继承。
- Phase 15 已锁定"admin overview 暴露 sidecar 诊断"的边界，Phase 20 将其从 Web admin 扩展到桌面端。

### Integration Points
- 监控页导航入口在 `fluent_app_shell.dart` 的 NavigationPane items 中添加，`NavigationProvider` 增加监控页索引。
- 批次/任务数据消费 `admin_service.go` / `task_read_service.go` 已有端点，无需新建后端路由。
- Sidecar 重启操作需要确认后端是否已有 restart 端点（可能在 Phase 15 已预留），否则需要补齐。
- WebSocket 实时推送需要 Go 后端新增 WebSocket endpoint，这是本阶段最显著的后端新增能力。

</code_context>

<specifics>
## Specific Ideas

- 运营监控应该是管理员在桌面端的"一站式仪表盘"——批次任务和 sidecar 诊断在同一视野内，不需要在多个页面间跳转。
- 双区布局让管理员可以同时感知"任务进展"和"基础设施健康"两个维度，更接近真正的运营视角。
- WebSocket 推送让监控数据保持实时，管理员不需要手动刷新来获取最新状态。
- Sidecar 重启是一个敏感操作，确认对话框必须清楚说明"正在进行的计算任务将被中断"。
- 即使 Python sidecar 完全停止，批次监控区仍应正常工作——批次/任务信息存在 Go/SQLite 中。

</specifics>

<deferred>
## Deferred Ideas

- Python 不可用时自动降级到 Go 本地后备计算路径 — COMP-05 / Phase 22 范围
- 完整日志查看器（在桌面端直接查看 Go/Python 日志文件）— 可作为后续运营增强
- 监控数据的历史趋势图表与告警阈值设置 — 超出本阶段 success criteria
- 侧车远程管理（非本地单机场景下的 sidecar 管理）— 超出当前部署形态

---

*Phase: 20-operations-monitoring*
*Context gathered: 2026-04-05*
