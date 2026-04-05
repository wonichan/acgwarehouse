# Phase 20: Operations Monitoring - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 20-operations-monitoring
**Areas discussed:** 页面结构与导航入口、批次任务监控视图、Sidecar 诊断面板、手动重启与操作交互、数据刷新与实时性、诊断范围、API 契约边界、边缘状态处理

---

## 页面结构与导航入口

| Option | Description | Selected |
|--------|-------------|----------|
| 单页双区布局 | 一个导航项，监控页内部上下分区：批次监控 + sidecar 诊断 | ✓ |
| Tab 切换双页 | 一个导航项，监控页内用 Tab 切换两个子视图 | |
| 两个独立导航项 | 拆成"任务监控"和"系统诊断"两个侧栏入口 | |

**User's choice:** 单页双区布局（推荐）
**Notes:** 用户选择推荐方案。让管理员同时看到任务进展和基础设施健康，更接近真正的运营视角。

---

## 批次任务监控视图

| Option | Description | Selected |
|--------|-------------|----------|
| 批次列表 + 任务下钻 | 主区显示批次列表，点击某批次展开任务明细 | ✓ |
| 纯批次列表 | 只显示批次级别信息，不下钻到任务 | |
| 双栏并列 | 左栏批次列表，右栏选中批次的任务列表 | |

**User's choice:** 批次列表 + 任务下钻（推荐）
**Notes:** 复用 Phase 13 已建立的批次优先视角。用户还选择了批次行应展示：状态标签、进度百分比/进度条、时间信息、错误统计与分组摘要。

---

## Sidecar 诊断面板

| Option | Description | Selected |
|--------|-------------|----------|
| 状态卡片 + 错误摘要列表 | 顶部状态卡片显示运行状态/时长/最后探测；下方错误列表 | ✓ |
| 紧凑状态行 + 展开详情 | 默认只占一行，点击展开详细错误列表 | |

**User's choice:** 状态卡片 + 错误摘要列表（推荐）
**Notes:** 采用颜色编码（ready=绿，degraded=黄，stopped=红）让状态一目了然。

---

## 手动重启与操作交互

| Option | Description | Selected |
|--------|-------------|----------|
| 诊断卡片内嵌入按钮 | 重启按钮直接在 sidecar 状态卡片内，与状态同一视觉上下文 | ✓ |
| 页面级操作栏 | 重启按钮在页面顶部工具栏中 | |
| 浮动操作按钮 | 独立浮动按钮，不与任何区域绑定 | |

**User's choice:** 诊断卡片内嵌入按钮（推荐）

**确认流程选择：**

| Option | Description | Selected |
|--------|-------------|----------|
| 二次确认 + 影响说明 | 确认对话框说明重启将中断正在进行的计算任务 | ✓ |
| 直接执行 | 点击即执行，不需要确认 | |

**User's choice:** 二次确认 + 影响说明（推荐）

---

## 数据刷新与实时性

| Option | Description | Selected |
|--------|-------------|----------|
| 手动刷新 + 自动初始化 | 进入页面加载一次，后续手动刷新 | |
| 定时轮询 | 固定间隔（如 5-30 秒）自动刷新 | |
| WebSocket 实时推送 | 服务端主动推送状态变更 | ✓ |

**User's choice:** WebSocket 实时推送
**Notes:** 用户明确选择 WebSocket，而不是推荐的简单方案。这是本阶段最显著的后端新增能力。

---

## 诊断范围

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 Python sidecar | 只展示 sidecar 状态和错误 | |
| 扩展到 Go 运行时 | 同时展示 Go 队列状态、Worker 数等内部运营指标 | ✓ |

**User's choice:** 扩展到 Go 运行时
**Notes:** 不只是看 Python 侧车，还要看 Go 层的运行时指标，复用 Phase 13 admin overview 契约。

---

## API 契约边界

| Option | Description | Selected |
|--------|-------------|----------|
| 复用现有 admin 端点 | 直接消费 /admin/api/... 已有端点，后端改动最小 | ✓ |
| 新建产品向 API 端点 | 新建 /api/v1/operations/... 产品级契约 | |

**User's choice:** 复用现有 admin 端点（推荐）
**Notes:** 后端改动最小化，前端直接消费已有 admin 契约。

---

## 边缘状态处理

| Option | Description | Selected |
|--------|-------------|----------|
| 服务不可用提示 + 重试 | 清楚提示哪个服务不可用，提供重试按钮 | ✓ |
| 分级错误诊断 | 区分 Go 不可用 vs sidecar 不可用 vs 端点超时 | |
| 简单超时提示 | 只显示"加载失败" | |

**User's choice:** 服务不可用提示 + 重试（推荐）
**Notes:** 不尝试缓存旧状态或推断状态。即使 sidecar 停止，批次列表（依赖 Go）仍应正常工作。

---

## the agent's Discretion

- 双区布局的具体高度分配、滚动行为与视觉分隔方式
- 批次列表的排序方式（默认按创建时间倒序还是按状态分组）
- Sidecar 诊断卡片中运行时指标的具体展示数量和格式
- WebSocket 连接失败时的退避策略与最大重试次数
- 进度条的视觉风格（线性 vs 环形）

## Deferred Ideas

- 完整日志查看器（桌面端直接查看 Go/Python 日志文件）— 超出本阶段 scope
- 监控数据历史趋势图表与告警阈值设置 — 超出 success criteria
- 侧车远程管理（非本地单机场景）— 超出当前部署形态
