# Phase 10: 主题统一与优化 - Context

**Gathered:** 2026-03-22
**Status:** Ready for planning

<domain>
## Phase Boundary

用户在双平台体验一致的二次元风格主题设计。实现统一的粉紫色动漫配色方案、明暗主题切换（跟随系统偏好）、Windows 键盘快捷键支持、Windows 鼠标悬停效果。

本阶段是 v2.0 UI/UX 重构的最后一个阶段，为前两个阶段（Phase 8 Windows UI、Phase 9 Android UI）添加统一的主题系统和桌面增强功能。

</domain>

<decisions>
## Implementation Decisions

### 配色系统
- **主色调**: 使用 `0xFFED79B5` (柔和粉紫/桃红色) 作为 Material 3 的 seedColor
- **配色生成**: Material 3 自动生成完整的颜色系统（primary、secondary、surface 等）
- **Fluent UI 配色**: 基于 Material 3 生成的颜色系统，映射到 FluentThemeData 的 accentColor 和其他颜色
- **暗色主题**: 使用"深色粉紫"变体 — 在暗色模式下使用更深沉、饱和度略低的粉紫色调，减少夜间视觉疲劳
- **代码组织**: 集中式主题配置（`lib/theme/app_theme.dart`），统一定义所有颜色、主题数据，两个平台引用同一个源

### 明暗主题切换机制
- **切换方式**: 默认跟随系统，用户在设置中可手动选择"浅色"/"深色"/"跟随系统"三种模式
- **存储方式**: 使用 shared_preferences 持久化存储用户偏好
- **生效时机**: 实时切换，无需重启。通过 Provider 通知所有页面更新
- **UI 入口**: 双平台都添加设置页面（Windows: FluentSettingsPage，Android: 新的设置页面），使用各自平台的组件实现一致的切换功能

### 键盘快捷键设计（Windows）
- **快捷键列表**:
  - `Ctrl+N` — 新建/导入图片
  - `Delete` — 删除选中图片
  - 方向键 — 在图片网格中导航（上下左右移动选择）
  - `Ctrl+A` — 全选图片
  - `Ctrl+S` — 搜索
  - `Esc` — 取消选择/关闭对话框
- **实现方式**: Flutter 原生 Focus 系统（FocusNode + RawKeyboardListener 或 Shortcuts widget）
- **作用范围**: 页面级快捷键 — 仅在特定页面生效（如图库页面才响应方向键）
- **反馈方式**: 纯视觉反馈 — 仅通过视觉变化（如选中态变化）反馈，不额外显示文字提示

### 鼠标悬停效果（Windows Fluent UI）
- **作用组件**: 导航栏项目（NavigationPane 中的项）、按钮（Button、IconButton）、标签 Chip（FluentTagChip）
- **实现方式**: 使用 Fluent UI 内置的悬停效果（通过 HoverButton、Button 等组件自带），保持 Fluent Design 系统的一致性
- **高亮颜色**: 使用主题色的浅色变体（如 `seedColor.withOpacity(0.1)` 或 Material 3 生成的 Container/Card 颜色）
- **动画时长**: 标准动画时长 200ms（Flutter 默认的 kThemeAnimationDuration）

### OpenCode's Discretion
- 主题 Provider 的具体字段设计（ThemeProvider 是否需要存储更多状态）
- 快捷键的具体实现细节（FocusNode 还是 Shortcuts widget）
- 设置页面的具体布局和组织方式
- 暗色主题的具体颜色微调数值

</decisions>

<specifics>
## Specific Ideas

- 粉紫色系符合二次元/动漫风格，`0xFFED79B5` 已在 STATE.md 中记录为设计方向
- Windows 快捷键设计参考标准 Windows 应用习惯（Ctrl+N 新建、Delete 删除等）
- 悬停效果保持 Fluent Design 原生风格，不做过度的自定义动画
- 设置页面在 Phase 8 已预留占位，Phase 10 完成主题切换等功能

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- **main.dart**: 现有双平台架构（AdaptiveApp + FluentApp/MaterialApp），主题配置在此添加
- **fluent_app_shell.dart**: Fluent UI 导航框架，键盘快捷键和悬停效果在此实现
- **material_app_shell.dart**: Material 3 导航框架，主题切换状态监听在此添加
- **fluent_settings_page.dart**: 现有占位页面，Phase 10 在此实现完整设置功能
- **navigation_provider.dart**: 导航状态管理，主题 Provider 可参照此模式创建

### Established Patterns
- **平台检测**: `!kIsWeb && defaultTargetPlatform == TargetPlatform.windows` 判断 Fluent/Material
- **状态管理**: Provider + ChangeNotifier 模式（NavigationProvider 已验证）
- **响应式布局**: BreakpointObserver + ResponsiveBreakpoint 已在 Phase 9 实现
- **主题访问**: `FluentTheme.of(context)` 和 `Theme.of(context)` 已在组件中使用

### Integration Points
- **AdaptiveApp**: 需要添加 ThemeProvider，根据当前主题模式构建对应的 ThemeData
- **FluentApp**: 主题切换需要更新 `FluentThemeData` 的 accentColor 和 brightness
- **MaterialApp**: 主题切换需要更新 `ThemeData` 的 colorScheme 和 brightness
- **设置页面**: Windows (FluentSettingsPage) 和 Android (新建) 都需要添加主题切换控件
- **快捷键**: 需要在 Fluent 页面（FluentGalleryPage 等）添加 Focus 系统支持

### Gaps to Address
- 没有 ThemeProvider 或类似的主题状态管理
- 没有集中的主题配置文件（colors.dart, app_theme.dart 等）
- 设置页面是占位符，需要实现完整功能
- 没有键盘快捷键的实现
- 悬停效果仅依赖 Fluent UI 默认，需要确保与粉紫主题协调

</code_context>

<deferred>
## Deferred Ideas

- 动画过渡效果增强 — v2.x 迭代阶段考虑
- 国际化 (i18n) 支持 — 当前仅中文用户，暂不需要
- Linux 桌面端主题支持 — 用户群体较小，推迟到后续版本
- macOS/iOS 主题支持 — 需要额外测试设备和开发环境

</deferred>

---

*Phase: 10-theme-unification-optimization*
*Context gathered: 2026-03-22*
