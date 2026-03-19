# Phase 7: 架构基础层 - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

建立平台感知的应用入口，实现根据运行平台自动选择 Fluent UI（Windows）或 Material UI（Android/Web），同时确保共享业务逻辑层与双 UI 框架兼容。

本阶段不包含完整的 Windows 桌面端 UI 实现（Phase 8）、Android 移动端 UI 实现（Phase 9）、主题统一与优化（Phase 10）。

</domain>

<decisions>
## Implementation Decisions

### 平台检测策略

- **检测逻辑**: Windows 平台使用 Fluent UI，其他平台（Android、iOS、Web）使用 Material 3
- **实现方式**: 使用 Flutter 的 `Theme.of(context).platform` 或 `defaultTargetPlatform` 检测当前平台
- **调试支持**: 添加调试开关（环境变量或设置项），允许在 Windows 上预览 Material UI，或在 Android 上预览 Fluent UI，方便开发测试

### 应用入口架构

- **入口结构**: 创建统一的 `AdaptiveApp` widget，内部根据平台返回 `FluentApp` 或 `MaterialApp`
- **Provider 初始化**: 所有 Provider 在 `AdaptiveApp` 外层的 `MultiProvider` 初始化一次，两个 UI 框架共享同一套 Provider 实例
- **现有入口**: 将现有 `main.dart` 的 `MaterialApp` 和 `MultiProvider` 逻辑重构为 `AdaptiveApp` 模式

### 共享业务逻辑层

- **目录结构**: 保持现有结构，不创建新的 `shared/` 目录
- **现有资产**: `providers/`、`services/`、`models/` 目录本身就是平台无关的，只需确保新 UI 层正确引用
- **原则**: 最小改动，风险最低，复用已验证的 v1.0 代码

### 导航状态持久化

- **状态管理**: 创建全局 `NavigationProvider` 管理 `selectedIndex`
- **跨平台同步**: `FluentApp` 和 `MaterialApp` 都从该 Provider 读取导航状态，切换平台时状态自动保持
- **实现方式**: `NavigationProvider` 继承 `ChangeNotifier`，存储当前选中的导航项索引

### 页面映射策略

- **共享 Widget**: `FluentApp` 和 `MaterialApp` 使用相同的页面 Widget（如 `GalleryScreen`、`SearchScreen`）
- **导航容器**: 只有导航容器不同（Windows 使用 `NavigationView`，Android/Web 使用 `NavigationBar`）
- **复用原则**: 代码复用最大化，最小改动

### FluentApp Shell 范围

- **Phase 7 范围**: 只完成 `FluentApp` shell 骨架，包含 `NavigationView` + 空页面容器
- **页面内容**: 页面内容（如 Gallery 网格、详情页）在 Phase 8 实现
- **验证目标**: 确保 shell 结构正确、导航切换正常、Provider 正常工作

### OpenCode's Discretion

- `AdaptiveApp` 的具体实现细节（如使用 `Theme.of(context).platform` 还是 `defaultTargetPlatform`）
- 调试开关的实现方式（环境变量、配置文件、还是隐藏设置项）
- `NavigationProvider` 的具体字段设计（是否需要存储更多导航状态）
- `FluentApp` 的初始主题配置（基础主题，详细配色在 Phase 10）

</decisions>

<specifics>
## Specific Ideas

- 平台检测逻辑应该简单直接，不引入复杂的条件判断
- 调试开关对开发测试很重要，可以快速验证两个 UI 框架的表现
- 保持现有目录结构可以减少风险，Phase 7 的目标是验证架构可行性
- 共享页面 Widget 可以最大化代码复用，只有导航容器需要平台适配

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets

- `flutter_app/lib/main.dart`: 现有应用入口，使用 `MaterialApp` + `MultiProvider` + `NavigationBar`，可重构为 `AdaptiveApp` 模式
- `flutter_app/lib/providers/`: 6 个 Provider（ImageListProvider, TagProvider, CollectionProvider, SelectionProvider, SearchProvider, DuplicateProvider），可直接复用
- `flutter_app/lib/services/`: 6 个 Service（ApiService, TagService, CollectionService, BatchService, SearchService, DuplicateService），可直接复用
- `flutter_app/lib/screens/`: 5 个 Screen（GalleryScreen, ImageDetailScreen, SearchScreen, DuplicateScreen, TagManagementScreen），可在两个 UI 框架中共享
- `flutter_app/lib/widgets/`: 11 个 Widget，可在两个 UI 框架中共享

### Established Patterns

- 项目采用 Go 后端 + Flutter 前端架构
- Flutter 使用 Provider 模式管理状态（v1.0 已验证）
- 导航使用底部 `NavigationBar`（Material 3 style）
- 分页采用 limit/offset 模式
- 图片缓存使用 `cached_network_image`

### Integration Points

- 需要添加 `fluent_ui` 和 `window_manager` 依赖到 `pubspec.yaml`
- 需要创建 `AdaptiveApp` widget 替代现有 `MyApp`
- 需要创建 `FluentAppShell`（Windows）和保持现有 `MaterialAppShell`（Android/Web）
- 需要创建 `NavigationProvider` 管理全局导航状态
- 需要修改 `main.dart` 使用新的 `AdaptiveApp` 入口

### Gaps to Address

- 没有平台检测逻辑
- 没有 `fluent_ui` 依赖
- 没有 `window_manager` 依赖（Windows 窗口控制）
- 没有 `AdaptiveApp` widget
- 没有 `FluentAppShell`
- 没有 `NavigationProvider`

</code_context>

<deferred>
## Deferred Ideas

- iOS/macOS 支持 — 优先 Windows 和 Android，iOS/macOS 需要额外测试设备和开发环境
- Linux 桌面端 — 用户群体较小，推迟到后续版本
- 完整的 Windows 桌面端 UI — Phase 8
- Android 移动端 UI 适配 — Phase 9
- 主题统一与优化 — Phase 10

</deferred>

---

*Phase: 07-architecture-foundation*
*Context gathered: 2026-03-20*