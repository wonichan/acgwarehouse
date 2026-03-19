# Requirements: ACGWarehouse v2.0 UI/UX 重构与多端适配

**Defined:** 2026-03-20
**Core Value:** 让用户能够高效地管理和检索二次元图片库，通过 AI 自动化减少手动整理的工作量，实现"存入即整理"的体验。

## v2.0 Requirements

### Architecture (架构层)

- [ ] **ARCH-01**: 实现 AdaptiveApp 平台检测入口，根据平台自动选择 FluentApp 或 MaterialApp
- [ ] **ARCH-02**: 创建 FluentApp shell（Windows 桌面端），包含 NavigationView 导航框架
- [ ] **ARCH-03**: 创建 MaterialApp shell（Android/Web），包含 NavigationBar/NavigationRail 导航框架
- [ ] **ARCH-04**: 提取共享业务逻辑层，确保 Provider/Services/Models 与双 UI 框架兼容

### Windows Desktop (Windows 桌面端)

- [ ] **WIN-01**: 实现 NavigationView 侧边导航栏，支持 PaneDisplayMode.auto 自适应
- [ ] **WIN-02**: 实现图库浏览界面，使用 Fluent 风格的图片网格和瀑布流
- [ ] **WIN-03**: 实现图片详情界面，包含元数据显示和标签管理
- [ ] **WIN-04**: 实现搜索界面，支持文件名/标签搜索
- [ ] **WIN-05**: 实现标签管理界面，支持标签审核和治理
- [ ] **WIN-06**: 实现 Windows 窗口控制（最小化/最大化/关闭）通过 window_manager
- [ ] **WIN-07**: 实现 Fluent 主题，使用柔和粉紫色系配色

### Android Mobile (Android 移动端)

- [ ] **ANDROID-01**: 实现 NavigationBar 底部导航栏（手机端 < 600px）
- [ ] **ANDROID-02**: 实现 NavigationRail 侧边导航栏（平板/大屏 >= 600px）
- [ ] **ANDROID-03**: 实现响应式网格布局，根据屏幕宽度自动调整列数
- [ ] **ANDROID-04**: 实现 Material 3 主题，使用柔和粉紫色系配色
- [ ] **ANDROID-05**: 优化触摸手势交互（滑动、缩放、长按）

### Cross-Platform (跨平台)

- [ ] **CROSS-01**: 建立统一配色系统，柔和粉紫色系在双平台保持一致
- [ ] **CROSS-02**: 实现明暗主题切换，跟随系统设置
- [ ] **CROSS-03**: 建立响应式断点系统（600px 手机/平板分界，900px 扩展导航）

### Desktop Enhancements (桌面增强 - v2.x)

- [ ] **ENH-01**: 实现键盘快捷键（Ctrl+N 新建, Delete 删除, 方向键导航）
- [ ] **ENH-02**: 实现鼠标悬停效果（按钮高亮、卡片阴影）
- [ ] **ENH-03**: 实现 CommandBar 工具栏（页面级操作按钮）

## v1.0 Requirements (Validated - Already Shipped)

### Authentication
- ✓ **AUTH-01**: 图片扫描入库（本地文件夹监控）
- ✓ **AUTH-02**: AI 自动标签生成（千问/豆包多模态）
- ✓ **AUTH-03**: 相似图片检测与去重
- ✓ **AUTH-04**: 搜索功能（文件名/标签/以图搜图）
- ✓ **AUTH-05**: 收藏夹/相册管理
- ✓ **AUTH-06**: 批量操作（选择/标签/移动/删除）
- ✓ **AUTH-07**: Docker 部署
- ✓ **AUTH-08**: Web 管理后台

## Out of Scope

| Feature | Reason |
|---------|--------|
| iOS/macOS 支持 | 优先 Windows 和 Android，iOS/macOS 需要额外测试设备和开发环境 |
| Linux 桌面端 | 用户群体较小，推迟到后续版本 |
| 自定义手势 | 增加 UI 复杂度，保持平台标准手势 |
| 动画过渡效果 | 核心功能优先，动画可在 v2.x 迭代 |
| 国际化 (i18n) | 当前仅中文用户，暂不需要 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ARCH-01 | Phase 7 | Pending |
| ARCH-02 | Phase 7 | Pending |
| ARCH-03 | Phase 7 | Pending |
| ARCH-04 | Phase 7 | Pending |
| WIN-01 | Phase 8 | Pending |
| WIN-02 | Phase 8 | Pending |
| WIN-03 | Phase 8 | Pending |
| WIN-04 | Phase 8 | Pending |
| WIN-05 | Phase 8 | Pending |
| WIN-06 | Phase 8 | Pending |
| WIN-07 | Phase 10 | Pending |
| ANDROID-01 | Phase 9 | Pending |
| ANDROID-02 | Phase 9 | Pending |
| ANDROID-03 | Phase 9 | Pending |
| ANDROID-04 | Phase 10 | Pending |
| ANDROID-05 | Phase 9 | Pending |
| CROSS-01 | Phase 10 | Pending |
| CROSS-02 | Phase 10 | Pending |
| CROSS-03 | Phase 9 | Pending |
| ENH-01 | Phase 10 | Pending |
| ENH-02 | Phase 10 | Pending |
| ENH-03 | Phase 8 | Pending |

**Coverage:**
- v2.0 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-20*
*Last updated: 2026-03-20 after milestone v2.0 start*