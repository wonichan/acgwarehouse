# Phase 17: Desktop Shell Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `17-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-04-04T15:34:30+08:00
**Phase:** 17-desktop-shell-foundation
**Areas discussed:** 壳层布局、搜索入口、图库网格、筛选面板

---

## 壳层布局

| Option | Description | Selected |
|--------|-------------|----------|
| 渐进重组 | 保留 `NavigationView` 主骨架，把顶部工具栏收拢进统一壳层 | |
| 自定义顶栏 | 用更桌面化的自定义顶栏承接窗口标题区与主操作区 | ✓ |
| 最小改动 | 基本维持现有左侧导航 + 页面内 Header/CommandBar | |

**User's choice:** 自定义顶栏
**Notes:** 进一步确认采用“顶栏主导 + 侧栏保留”，也就是顶部承载主操作，左侧导航继续保留但退居视图切换层。

---

## 搜索入口

| Option | Description | Selected |
|--------|-------------|----------|
| 常驻搜索框 | 顶栏直接放常驻搜索框 | ✓ |
| 搜索按钮跳转 | 顶栏只保留搜索按钮，点击后再进入搜索页 | |
| 展开式搜索 | 顶栏放紧凑搜索入口，点击后在当前页展开搜索区域 | |

**User's choice:** 常驻搜索框
**Notes:** 进一步确认“统一入口 + 独立结果”，即顶栏常驻搜索框，但搜索结果仍由独立搜索视图承载。

---

## 图库网格

| Option | Description | Selected |
|--------|-------------|----------|
| 纯方块网格 | 只交付方块网格，不暴露其他模式 | |
| 方块为主，保留切换 | 方块网格为主，但保留已有网格/瀑布流切换 | ✓ |
| 双模式并行 | 把网格和瀑布流都作为本阶段主壳层设计中心 | |

**User's choice:** 方块为主，保留切换
**Notes:** 进一步确认方块网格的视觉目标是“固定磁贴感优先”，窗口变化只做有限响应式调整。

---

## 筛选面板

| Option | Description | Selected |
|--------|-------------|----------|
| 右侧面板 | 右侧可开合筛选面板，持续服务图库上下文 | ✓ |
| 继续用弹窗 | 维持 `ContentDialog` 弹窗筛选 | |
| 左侧抽屉 | 左侧次级筛选抽屉，与导航并列 | |

**User's choice:** 右侧面板
**Notes:** 进一步确认筛选交互采用“即时生效”，不再要求用户点“应用筛选”。

---

## the agent's Discretion

- 自定义顶栏的拖拽区域、按钮密度和窄窗口溢出策略
- 顶栏中导入入口的具体按钮形式（普通按钮 / 下拉按钮 / 带状态复合按钮）
- 方块网格的具体列数阈值、间距和 tile 尺寸细节
- 右侧筛选面板的宽度、动画和默认展开策略

## Deferred Ideas

- 独立查看器与 filmstrip — Phase 18
- 桌面端标签治理 — Phase 19
- 导入任务监控与 sidecar 诊断入口 — Phase 20
- 大图库性能专项优化 — Phase 22
