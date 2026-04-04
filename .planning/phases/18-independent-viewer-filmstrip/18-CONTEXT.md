# Phase 18: Independent Viewer & Filmstrip - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

在桌面端为图片浏览补齐独立查看器：用户可以从图库或搜索结果中打开非阻塞查看器窗口，通过底部 filmstrip 快速切换图片，进行缩放/拖拽/双击放大等基础查看操作，并在右侧侧栏查看图片元信息与标签。

本阶段不包含：标签治理（Phase 19）、导入后任务监控与 sidecar 诊断（Phase 20）、Windows 打包（Phase 21）、大图库性能专项与 sidecar 恢复提示（Phase 22）。

</domain>

<decisions>
## Implementation Decisions

### 窗口策略
- **D-01:** 双击图片后打开**非模态独立查看器窗口**，允许同时存在多个查看器窗口，主图库保持可继续浏览与搜索。
- **D-02:** Phase 18 第一版**不做查看器窗口大小/位置记忆**，每次按统一默认尺寸与位置打开。

### Filmstrip 范围与密度
- **D-03:** 底部 filmstrip **跟随当前结果集**进行切换，而不是严格限定为“同一文件夹图片”；这是对 ROADMAP 原始 wording 的用户覆盖。
- **D-04:** filmstrip 采用**中等密度缩略图**，当前项需要有明显高亮，同时保留前后文缩略图以支持连续扫图。

### 查看器布局与信息
- **D-05:** 查看器采用**主图区域 + 右侧固定元信息栏 + 底部 filmstrip** 的三段式布局，右侧元信息栏默认展开。
- **D-06:** 元信息栏采用**扩展信息集**：除文件名、分辨率、大小、标签外，还显示格式、路径与导入时间。

### 查看交互
- **D-07:** 双击放大采用**适应窗口 ↔ 2x 放大**切换语义。
- **D-08:** 用户通过 filmstrip 或左右切图切到新图片时，查看器状态**重置为适应窗口**，不继承上一张图片的缩放比例与拖拽位置。
- **D-09:** 键盘交互第一版锁定为**基础快捷键**：`←/→` 切图，`Esc` 关闭当前查看器。

### the agent's Discretion
- 多窗口查看器实例的内部标识、窗口标题格式与生命周期同步细节
- filmstrip 缩略图的精确尺寸、间距、选中态与滚动动画
- 元信息栏内字段分组顺序与视觉排版方式
- 鼠标滚轮 / 触控板缩放的具体灵敏度，只要不违背 D-07 / D-08

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 18 的目标、依赖、4 条 success criteria 与 UI hint
- `.planning/REQUIREMENTS.md` — `VIEW-01`、`VIEW-02`、`VIEW-03`、`VIEW-04` 的正式定义
- `.planning/PROJECT.md` — v4.0 里程碑目标、Windows Photos 参考原则，以及 Go 主控 / Flutter 前端 / Python 计算侧车边界
- `.planning/STATE.md` — 当前阶段位置、Phase 17 已完成并切换到 Phase 18 的状态记录

### Prior Phase Decisions
- `.planning/phases/17-desktop-shell-foundation/17-CONTEXT.md` — 桌面主壳层、图库工作区、顶栏/右侧面板等已锁定 UI 基线
- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md` — Flutter 继续只依赖 Go 契约、桌面启动与窗口治理的前序边界
- `.planning/phases/16-duplicate-detection-migration/16-CONTEXT.md` — Go / Python 职责边界维持不变，查看器阶段不下沉到 Python 侧车

### Research & Product References
- `.planning/research/SUMMARY.md` — v4.0 总体研究结论，含独立查看器 / 胶片条风险提示与建议依赖
- `.planning/research/FEATURES.md` — Windows Photos 风格下独立查看器、filmstrip 与非模态窗口的产品研究
- `.planning/research/ARCHITECTURE.md` — `flutter_app/lib/screens/viewer/` 作为查看器承载位置的架构建议
- `.planning/research/STACK.md` — `window_manager`、`photo_view` 等桌面查看器相关技术栈建议
- `Windows11-Photos-App-ACG-Gallery-Research.md` — Windows Photos 参考体验，涵盖查看器、胶片条与桌面感目标
- `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md` — Flutter / Go / Python 三层职责与桌面 UI 模块边界

### Existing Code Anchors
- `flutter_app/lib/app/fluent_screens.dart` — 图库/搜索当前通过 `Navigator.push` 打开页内详情，是 Phase 18 的主接入点
- `flutter_app/lib/widgets/fluent_gallery_content.dart` — 图库结果集与图片点击回调入口
- `flutter_app/lib/widgets/fluent_image_card.dart` — 当前桌面图库图片卡片交互入口
- `flutter_app/lib/screens/image_detail_screen.dart` — 已有元信息与标签详情区块，可复用到查看器侧栏
- `flutter_app/lib/screens/image_gallery_viewer.dart` — 已有缩放、拖拽、双击放大等查看能力基线
- `flutter_app/lib/utils/window_manager.dart` — 当前桌面窗口初始化与基础窗口控制工具
- `flutter_app/lib/models/image.dart` — 查看器元信息字段来源（文件名、路径、尺寸、大小、格式、时间）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `flutter_app/lib/screens/image_detail_screen.dart`：已有“元数据 + 标签”详情区块，可直接复用到查看器右侧固定栏
- `flutter_app/lib/screens/image_gallery_viewer.dart`：已有 `ExtendedImage` 驱动的缩放、拖拽、双击放大与多图切页逻辑，可作为查看器主图交互基线
- `flutter_app/lib/widgets/fluent_gallery_content.dart` + `flutter_app/lib/widgets/fluent_image_card.dart`：现有图库结果集与图片点击入口，可承接“从结果集打开独立查看器”的入口变更
- `flutter_app/lib/utils/window_manager.dart`：已有桌面窗口初始化、尺寸与基础窗口控制工具，适合作为多窗口查看器的起点
- `flutter_app/lib/models/image.dart`：已经具备文件名、路径、分辨率、文件大小、格式、时间、缩略图 URL 等字段

### Established Patterns
- 当前桌面图库/搜索打开详情的模式是**页内导航 push 到详情页**，Phase 18 需要把这一路径改造成**独立窗口查看器**而不是简单扩展详情页
- 现有图片查看交互建立在 `ExtendedImage` 之上，说明第一版更适合沿用现有查看栈而不是整阶段切换到全新查看器框架
- 桌面主壳层已经在 Phase 17 固化为 `FluentAppShell + FluentScreens` 结构，查看器需要作为其外部独立工作区延伸，而不是重做壳层
- 当前图库结果集由 `ImageListProvider` 持有，这为“filmstrip 跟随当前结果集”提供了天然状态来源

### Integration Points
- `flutter_app/lib/app/fluent_screens.dart` 中的 `_showImageDetail(...)` 是图库页和搜索页跳入查看器的主要接入点
- `flutter_app/lib/providers/image_provider.dart` 持有当前结果集、排序与筛选状态，可作为 filmstrip 数据来源
- `flutter_app/lib/screens/image_detail_screen.dart` 中的元信息与标签区块可拆分复用到查看器右侧侧栏
- `flutter_app/lib/utils/window_manager.dart` 与现有桌面启动路径需要扩展到“新建查看器窗口/管理当前查看器窗口”能力

</code_context>

<specifics>
## Specific Ideas

- 查看器要像一个**独立桌面工作区**，而不是模态覆盖层或只会阻塞主图库的预览弹窗
- 用户明确选择让 filmstrip **跟随当前结果集**，即使这与 ROADMAP 中“同一文件夹” wording 不完全一致，也应视为已锁定产品决策
- 第一版优先追求**稳定清晰的桌面查看流程**，因此不做窗口状态记忆，也不扩展高级快捷键体系

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 18-independent-viewer-filmstrip*
*Context gathered: 2026-04-05*
