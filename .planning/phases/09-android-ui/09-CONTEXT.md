# Phase 9: Android 移动端 UI - Context

**Gathered:** 2026-03-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Android 用户使用自适应 Material 3 界面管理图片库。实现 NavigationBar 底部导航（手机端）和 NavigationRail 侧边导航（平板端）的自适应切换，响应式网格布局，以及触摸手势优化。不包含主题配色（Phase 10）和 Windows 桌面端功能。

</domain>

<decisions>
## Implementation Decisions

### 导航结构
- 底部导航栏显示 **3 个项**：图库、搜索、标签管理
- 导航顺序：**图库 → 搜索 → 标签**（从左到右）
- 图库为主页（第 1 位），搜索居中，标签在右
- 重复检测和设置通过图库页面 AppBar 溢出菜单（三个点）访问
- 精简导航，保持底部栏简洁

### 响应式网格列数
- **断点策略**：Material Design 标准三断点
  - Compact (≤600px)：手机
  - Medium (600-840px)：中等平板
  - Expanded (>840px)：大平板
- **手机端**：竖屏 2 列，横屏 3 列
- **平板端**：中等屏幕 3 列，大屏幕 4 列
- **间距策略**：动态间距
  - 手机：4px
  - 中等：8px
  - 大屏幕：12px

### 触摸手势交互
- **图片详情页**：完整缩放切换
  - 双击：放大/缩小
  - 双指捏合：缩放
  - 左右滑动：切换图片
- **返回手势**：从屏幕左侧边缘向右滑动返回上一页
- **图库网格**：无双击操作（避免与单击冲突）
- **刷新**：下拉刷新（Pull-to-refresh）
- **长按图片**：仅进入选择模式（不显示上下文菜单）
- **列表项**：无滑动操作（避免误触）
- **网格缩放手势**：不支持捏合改变列数

### NavigationBar ↔ NavigationRail 切换
- **切换动画**：淡入淡出过渡效果
- **导航状态**：切换时保持当前选中页面
- **Rail 显示模式**：72px 图标模式（标准宽度）
- **横屏手机**：屏幕宽度 >600px 时切换到 NavigationRail
- **断点一致性**：与响应式网格使用相同的 600px 断点

### 页面布局适配
- **图片详情页**：底部 Sheet 式布局
  - 上方大图，下滑显示元数据和标签
  - 类似手机系统图库应用
- **标签管理页**：单列列表布局
  - 每个标签占一行
  - 显示名称、数量、操作按钮
- **标签筛选**：左侧滑出抽屉（Drawer）
  - 全屏抽屉显示所有筛选选项
  - 符合 Material Design 规范
- **批量操作**：底部固定操作栏
  - 显示已选数量
  - 显示操作按钮（收藏、删除、标签等）
  - 选择模式下始终可见

### OpenCode's Discretion
- 导航切换动画的具体时长和曲线
- 图片详情页底部 Sheet 的高度比例
- 标签列表项的具体高度和间距
- 底部操作栏的具体按钮样式
- 响应式断点的具体实现方式（MediaQuery 或 LayoutBuilder）

</decisions>

<specifics>
## Specific Ideas

- 图片详情页参考系统相册应用的行为
- 底部导航栏保持 Material 3 标准样式
- 标签筛选抽屉参考 Gmail 的侧边抽屉设计
- 批量操作栏参考 Google Photos 的多选操作

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- **adaptive_app.dart**: 平台检测入口，已区分 Windows 和 Material 平台
- **material_app_shell.dart**: 基础 Material 应用壳，含 NavigationBar（当前只有3项）
- **navigation_provider.dart**: 共享导航状态管理，定义了 5 个导航项
- **ImageGrid**: 支持 crossAxisCount 参数，需添加响应式逻辑
- **ImageMasonry**: 瀑布流组件，可复用
- **screens/**: GalleryScreen、SearchScreen、DuplicateScreen、TagManagementScreen、ImageDetailScreen 已存在
- **Providers/Services/Models**: Phase 7 已完成共享层，全部可复用

### Established Patterns
- **状态管理**: Provider + ChangeNotifier 模式
- **导航模式**: NavigationProvider 管理选中索引，页面通过 Consumer 响应
- **平台检测**: `!kIsWeb && defaultTargetPlatform == TargetPlatform.windows` 判断是否使用 Fluent
- **Material 3**: 已在 main.dart 中启用 `useMaterial3: true`

### Integration Points
- **MaterialAppShell**: 需要扩展为支持 NavigationBar/Rail 自适应切换
- **GalleryScreen**: 已有 RefreshIndicator 支持下拉刷新，需扩展 AppBar 菜单
- **ImageGrid**: 需包装响应式逻辑，根据屏幕宽度动态计算 crossAxisCount
- **main.dart**: 需要为 MaterialApp 添加主题配置（Phase 10 处理）
- **ImageDetailScreen**: 需要移动端布局变体

### Phase 8 Reference
- FluentAppShell 使用 NavigationView 作为参考
- 页面结构保持一致：Gallery、Search、Duplicate、Tags、Settings
- 共享 Provider 模式已验证可行

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 09-android-ui*
*Context gathered: 2026-03-21*
