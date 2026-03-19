# Phase 8: Windows 桌面端 UI - Context

**Gathered:** 2026-03-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Windows 用户可以使用原生 Fluent Design 界面管理图片库，包括导航、图库浏览、图片详情、搜索、标签管理和窗口控制。

本阶段不包含 Android 移动端 UI 实现（Phase 9）、主题统一与优化（Phase 10）。

</domain>

<decisions>
## Implementation Decisions

### 导航布局
- **显示模式**: 左侧可折叠导航栏（PaneDisplayMode.auto）
- **行为**: 宽屏展开显示图标+文字，窄屏自动折叠为仅图标
- **导航项**: 5 项 — 图库、重复检测、搜索、标签管理、设置
- **窗口标题**: 应用名 + 当前页面名，如 "ACGWarehouse - 图库"

### 页面复用策略
- **适配方式**: 包装适配 — 保留现有 Screen 内容，用 Fluent 容器（ScaffoldPage）包装
- **原则**: 最小改动，复用 v1.0 已验证代码
- **改动范围**: 仅导航容器和包装层，核心页面逻辑不变

### 图库浏览界面
- **布局模式**: 支持两种视图切换 — 网格视图和瀑布流视图
- **默认视图**: 网格视图（更稳定，加载更快）
- **切换方式**: CommandBar 中的视图切换按钮
- **排序**: 保持现有排序选项（导入时间、文件名、文件大小）

### 图片卡片样式
- **样式**: 简约卡片 — 圆角矩形，悬停时显示阴影
- **内容**: 图片缩略图 + 底部文件名（可选显示）
- **交互**: 点击打开详情，悬停显示阴影效果

### 图片详情展示
- **展示方式**: 右侧面板或 Fluent ContentDialog
- **内容**: 核心信息 — 大图、文件名、尺寸、标签列表、操作按钮
- **标签交互**: 点击标签筛选图片，长按编辑标签

### CommandBar 工具栏
- **位置**: 页面级工具栏，每个页面有独立的 CommandBar
- **图库页面**: 视图切换、排序、筛选、标签管理入口
- **搜索页面**: 搜索框、筛选选项
- **标签管理页面**: 新建标签、批量操作

### 搜索功能
- **组织方式**: 独立搜索页面
- **功能**: 文件名搜索、标签搜索、筛选选项
- **结果展示**: 与图库页面一致的网格/瀑布流布局

### 标签管理
- **组织方式**: 独立页面
- **功能**: 标签列表、新建/编辑/删除标签、标签别名管理
- **与详情联动**: 详情面板中可快速添加/移除标签

### 设置页面
- **范围**: 完整设置
- **内容**: 
  - 主题切换（明/暗）
  - API 配置（千问/豆包密钥）
  - 图片存储路径
  - 标签 AI 配置
  - 高级选项
  - 关于页面

### Windows 窗口控制
- **默认大小**: 1280 x 720 像素
- **功能**: 可调整大小、最小化、最大化、关闭
- **实现**: 使用 window_manager 包
- **标题栏**: 系统默认标题栏

### 状态处理
- **空状态**: 简约提示 — 图标 + 简短文案 + 操作按钮（如"添加图片"）
- **加载状态**: Fluent ProgressIndicator
- **错误状态**: 简洁的错误提示 + 重试按钮

### OpenCode's Discretion
- NavigationView 的具体配色细节（主题配色在 Phase 10）
- CommandBar 按钮的具体图标选择
- 详情面板的具体布局（左右分栏还是上下分栏）
- 卡片圆角和阴影的具体数值
- 空状态图标的具体选择

</decisions>

<specifics>
## Specific Ideas

- 左侧可折叠导航栏符合 Windows 用户习惯，Fluent Design 标准
- 包装适配策略最大化代码复用，降低风险
- 两种视图切换满足不同用户偏好
- 简约卡片风格干净利落，适合二次元图片展示
- 设置页面包含完整配置项，方便用户定制

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `flutter_app/lib/screens/gallery_screen.dart`: 图库浏览页面，包含网格/瀑布流切换逻辑，可直接包装适配
- `flutter_app/lib/screens/image_detail_screen.dart`: 图片详情页面，可包装后配合 Fluent Dialog 使用
- `flutter_app/lib/screens/search_screen.dart`: 搜索页面，可直接包装适配
- `flutter_app/lib/screens/duplicate_screen.dart`: 重复检测页面，可直接包装适配
- `flutter_app/lib/screens/tag_management_screen.dart`: 标签管理页面，可直接包装适配
- `flutter_app/lib/widgets/image_grid.dart`: 网格视图组件，可复用
- `flutter_app/lib/widgets/image_masonry.dart`: 瀑布流视图组件，可复用
- `flutter_app/lib/providers/`: 6 个 Provider 可直接复用
- `flutter_app/lib/services/`: 6 个 Service 可直接复用

### Established Patterns
- 项目采用 Go 后端 + Flutter 前端架构
- Flutter 使用 Provider 模式管理状态（v1.0 已验证）
- 现有导航使用底部 NavigationBar（Material 3 style）
- 分页采用 limit/offset 模式
- 图片缓存使用 `cached_network_image`

### Integration Points
- Phase 7 创建的 AdaptiveApp 入口
- Phase 7 创建的 FluentApp shell（NavigationView 骨架）
- Phase 7 创建的 NavigationProvider（管理导航状态）
- 需要添加 `fluent_ui` 和 `window_manager` 依赖（Phase 7 已添加）
- 需要为每个现有 Screen 创建 Fluent 包装器

### Gaps to Address
- 没有 Fluent 风格的图片卡片组件
- 没有 Fluent 风格的详情面板组件
- 没有 Fluent 风格的 CommandBar 实现
- 没有设置页面
- 没有窗口控制配置（window_manager）
- 现有 Screen 需要 Fluent 包装适配

</code_context>

<deferred>
## Deferred Ideas

- Android 移动端 UI 适配 — Phase 9
- 主题统一与优化 — Phase 10
- 键盘快捷键 — Phase 10
- 鼠标悬停效果优化 — Phase 10
- iOS/macOS 支持 — 未来版本
- Linux 桌面端 — 未来版本

</deferred>

---

*Phase: 08-windows-ui*
*Context gathered: 2026-03-20*