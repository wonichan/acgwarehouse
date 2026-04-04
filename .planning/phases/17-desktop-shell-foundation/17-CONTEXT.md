# Phase 17: Desktop Shell Foundation - Context

**Gathered:** 2026-04-04
**Status:** Ready for planning

<domain>
## Phase Boundary

在 Windows Photos 风格桌面界面里完成图库主壳层：用户可以在图库主上下文中通过顶部工具栏访问搜索、导入和设置，使用方块网格浏览图库，并通过可访问的标签筛选面板筛选内容。

本阶段不包含：独立查看器与 filmstrip（Phase 18）、桌面端标签治理（Phase 19）、运营监控入口（Phase 20）、完整打包与性能治理（Phase 21/22）。

</domain>

<decisions>
## Implementation Decisions

### 壳层布局
- **D-01:** Phase 17 采用**自定义顶栏**作为桌面主壳层的核心，而不是继续仅依赖现有页面级 `PageHeader + CommandBar` 的松散组合。
- **D-02:** 顶栏采取**顶栏主导 + 侧栏保留**的结构：顶部承担搜索、导入、设置等主操作；左侧导航继续保留，但退居为视图切换层而不是主操作区。
- **D-03:** 顶栏需要同时承担窗口级桌面壳层语义与产品级主操作语义，目标是更接近 Windows Photos 的桌面感，但不追求像素级复刻。

### 搜索入口
- **D-04:** 搜索入口采用**顶栏常驻搜索框**，作为桌面主壳层的一等入口，而不是只保留按钮跳转。
- **D-05:** 搜索结果继续落在**独立搜索视图**中承载；Phase 17 统一的是入口位置，不把图库列表与搜索结果数据模型强行合并。

### 图库网格
- **D-06:** Phase 17 以**方块网格**作为主浏览模式，满足 `DSK-02` 的 square grid 成功标准。
- **D-07:** 现有 `masonry` 能力**暂时保留但不作为本阶段主目标**；方块网格优先，切换能力不作为主壳层设计中心。
- **D-08:** 方块网格的视觉目标是**固定磁贴感优先**：用户首先感知为稳定、整齐、统一尺寸的 tile，只在窗口宽度变化时做有限响应式调整。

### 筛选面板
- **D-09:** 标签筛选采用**右侧可开合面板**，不再以临时弹窗作为主交互形态，以满足 `DSK-03` 的可访问筛选面板语义。
- **D-10:** 筛选交互采用**即时生效**：标签勾选与“未打标签”开关变更后立即刷新图库，不要求用户额外点击“应用筛选”。
- **D-11:** 当前图库页标题区域里展示的已选标签 chips 仍可保留，但角色变为“当前筛选状态反馈 / 快速取消”，而不是主要筛选入口。

### 导入入口
- **D-12:** 顶栏必须提供**导入入口**，与搜索和设置并列为主操作。
- **D-13:** Phase 17 对导入的目标是“在桌面主壳层里可访问现有导入能力”，而不是在本阶段扩展成完整的新导入中心或新运营模块。
- **D-14:** 规划时优先复用现有后端手动扫描链路与已有导入状态契约线索；若桌面前端缺少直接入口，则在本阶段补齐 UI 接入层，而不改变 Go / Python 已锁定的职责边界。

### the agent's Discretion
- 自定义顶栏中窗口拖拽区域、标题文本与主操作区的精确布局比例
- 顶栏在窄窗口下的溢出策略、图标文案与按钮收纳方式
- 方块网格在不同桌面宽度下的具体列数阈值与间距数值
- 右侧筛选面板的动画、默认宽度与折叠/展开触发细节
- 导入入口最终采用按钮、下拉按钮还是带状态提示的复合按钮，只要仍满足“顶栏可访问导入”

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 17 的目标、依赖关系与 3 条 success criteria
- `.planning/REQUIREMENTS.md` — `DSK-01`、`DSK-02`、`DSK-03` 的正式定义，以及 `DSK-04` 明确属于 Phase 19 的边界
- `.planning/PROJECT.md` — v4.0 里程碑目标、Windows Photos 参考原则，以及 Go 主控 / Flutter 前端 / Python 计算侧车的职责边界
- `.planning/STATE.md` — 当前项目位置、v4.0 执行顺序、Phase 17 作为当前焦点的状态记录

### Prior Phase Decisions
- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md` — Go 唯一主控、Flutter 只依赖 Go 契约、Runtime Manifest、分层健康与降级可用等前序约束
- `.planning/phases/16-duplicate-detection-migration/16-CONTEXT.md` — 计算迁移已完成、Phase 17 不改重复检测计算链路，只消费既有服务边界

### Research & Architecture
- `.planning/research/SUMMARY.md` — v4.0 研究总结，特别是桌面 UI 重构风险、Windows Photos 参考方向与范围控制提醒
- `.planning/research/ARCHITECTURE.md` — Flutter / Go / Python 三层职责边界与推荐的桌面 UI 演进结构

### Existing Shell & Gallery Implementation
- `flutter_app/lib/app/fluent_app_shell.dart` — 当前桌面端 `NavigationView + TitleBar + NavigationPane` 壳层实现基线
- `flutter_app/lib/app/fluent_screens.dart` — 当前图库页 / 搜索页 / Fluent 标签筛选对话框实现，规划时需要据此决定迁移与复用边界
- `flutter_app/lib/widgets/fluent_gallery_content.dart` — 当前图库网格 / 瀑布流、分页加载、刷新与空态行为
- `flutter_app/lib/providers/image_provider.dart` — 图库视图模式、排序、标签筛选与未打标签筛选状态
- `flutter_app/lib/providers/navigation_provider.dart` — 现有桌面导航索引与页面标题约定
- `flutter_app/lib/widgets/fluent_settings_page.dart` — 现有设置页，可作为顶栏设置入口承接目标

### Existing Import/Scan Integration
- `internal/service/admin_service.go` — 现有手动扫描触发入口 `TriggerScan()`，通过 `manual_scan` job 复用后端扫描链路
- `internal/worker/scan_handler.go` — `manual_scan` 的执行处理器，支持使用配置扫描根或 payload 指定路径
- `internal/service/scanner_service.go` — 实际扫描、导入、批次规划与缩略图任务排队逻辑
- `internal/handler/routes.go` — 当前后端路由：`/admin/api/actions/scan` 已存在，`/api/v1/images/scan` 仍是 placeholder
- `flutter_app/lib/config/api_config.dart` — Flutter 侧已有 `importStatus` 契约线索，但尚未形成完整桌面导入入口

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `flutter_app/lib/app/fluent_app_shell.dart`：已有桌面端 Fluent 壳层，可作为“自定义顶栏 + 保留侧栏”的改造起点，而不是从零重写窗口结构。
- `flutter_app/lib/app/fluent_screens.dart`：图库页已有 `CommandBar`、标题区 chips、搜索页与现成标签筛选对话框，适合拆解并迁移到新顶栏 / 右侧面板结构。
- `flutter_app/lib/widgets/fluent_gallery_content.dart`：已有方块网格、瀑布流、分页加载、刷新与空态逻辑，Phase 17 可以在此基础上收敛默认模式而不是重做图库数据流。
- `flutter_app/lib/providers/image_provider.dart`：已经封装 `viewMode`、排序、标签筛选与 `hasTagsFilter`，能直接支撑“方块为主、筛选即时生效”的目标。
- `flutter_app/lib/widgets/fluent_settings_page.dart`：设置页已可直接承接顶栏设置入口。
- `internal/service/admin_service.go` + `internal/worker/scan_handler.go` + `internal/service/scanner_service.go`：后端已具备手动扫描/导入链路，可作为顶栏导入入口的后端基础。

### Established Patterns
- 桌面 Fluent 页面当前采用 `ScaffoldPage + PageHeader + CommandBar` 模式，说明 Phase 17 更适合“收口成统一壳层”而不是彻底推翻 Fluent 组件体系。
- `NavigationView + NavigationPane` 已经是当前桌面信息架构中心，因此“顶栏主导 + 侧栏保留”比“完全移除侧栏”更符合现状演进路径。
- 标签筛选当前由 `TagProvider` 与 `ImageListProvider` 协同完成，且标签筛选与未打标签筛选互斥，这为右侧面板即时生效提供了现成状态模型。
- 图库滚动与分页逻辑已经在 `FluentGalleryContent` 里稳定工作，说明 Phase 17 可以聚焦壳层和交互重组，不需要同时重写列表数据流。
- Windows 桌面启动已经接入 `window_manager` 与 runtime manifest，引入自定义顶栏时仍应沿用现有桌面启动路径，而不是新增独立入口。

### Integration Points
- 自定义顶栏的主要接入点在 `flutter_app/lib/app/fluent_app_shell.dart`，需要把当前 `TitleBar` 与页面级 `CommandBar` 的职责重新分配。
- 顶栏常驻搜索框可以复用 `SearchProvider` 与 `FluentSearchPage`，通过 `NavigationProvider` 切换到独立搜索视图承载结果。
- 右侧筛选面板应直接复用 `TagProvider` + `ImageListProvider`，并替换当前 `ContentDialog` 式标签筛选为侧边面板形态。
- 方块网格主模式应继续落在 `FluentGalleryContent` / `ImageListProvider` 现有组合上，规划时只需明确默认模式、密度与切换暴露策略。
- 导入入口需要在桌面顶栏补 UI 接入层；后端已有 `/admin/api/actions/scan` 手动扫描能力，但产品向 `/api/v1/images/scan` 仍未实现，说明当前状态是“后端基础已有、桌面前端入口缺失且产品向 API 未完成”。

</code_context>

<specifics>
## Specific Ideas

- 顶栏不是单纯换皮，而是把“窗口标题区 + 主操作区”合成一条更桌面化的壳层。
- 搜索要前置到顶栏，但结果不强求在图库原地展示，优先复用现有独立搜索视图承载结果。
- 方块网格要先给用户稳定、整齐、像照片库磁贴的感觉，再考虑更激进的布局切换。
- 筛选入口从“弹窗”升级为“右侧持续可见/可开合面板”，让筛选成为浏览过程的一部分。
- 导入入口需要出现在顶栏，但本阶段更强调“接得上现有导入能力”，不把范围扩成完整导入中心或运营工作台。

</specifics>

<deferred>
## Deferred Ideas

- 独立查看器窗口、filmstrip 与查看器交互 — 属于 Phase 18
- 桌面端标签治理（查看/编辑/整理） — 属于 Phase 19
- 导入后任务监控与 sidecar 诊断入口 — 属于 Phase 20
- 大图库性能专项优化与侧车不可用恢复提示 — 属于 Phase 22

</deferred>

---

*Phase: 17-desktop-shell-foundation*
*Context gathered: 2026-04-04*
