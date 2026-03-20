# Roadmap: ACGWarehouse v2.0 UI/UX 重构与多端适配

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-03-19)
  - 详见: `.planning/milestones/v1.0-ROADMAP.md`
- 🚧 **v2.0 UI/UX 重构** - Phases 7-10 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) - SHIPPED 2026-03-19</summary>

### Phase 1: 基础架构、图片扫描与标签基础层
**Goal**: Go 后端项目骨架与 SQLite 数据库 Schema 初始化
**Plans**: 3 plans (complete)

### Phase 2: 缩略图、基础浏览与 AI 复核界面底座
**Goal**: 缩略图生成、感知哈希计算与 Flutter 图片浏览界面
**Plans**: 5 plans (complete)

### Phase 3: AI 开放标签与治理
**Goal**: 千问/豆包 AI 标签集成与标签治理能力
**Plans**: 6 plans (complete)

### Phase 4: 重复检测与搜索
**Goal**: 重复检测、以图搜图与搜索功能
**Plans**: 6 plans (complete)

### Phase 5: 收藏夹与批量操作
**Goal**: 收藏夹/相册管理与批量操作功能
**Plans**: 4 plans (complete)

### Phase 6: 优化与部署
**Goal**: Docker Compose 单机部署与 Web 管理后台
**Plans**: 4 plans (complete)

</details>

### 🚧 v2.0 UI/UX 重构与多端适配 (In Progress)

**Milestone Goal:** 为 ACGWarehouse 添加 Windows 桌面端和 Android 移动端支持，打造二次元风格的精美界面

#### Phase 7: 架构基础层
**Goal**: 建立平台感知的应用入口，为双 UI 框架提供统一架构支撑
**Depends on**: Phase 6 (v1.0 complete)
**Requirements**: ARCH-01, ARCH-02, ARCH-03, ARCH-04
**Success Criteria** (what must be TRUE):
  1. App detects platform and displays Fluent UI on Windows startup
  2. App detects platform and displays Material UI on Android/Web startup
  3. Shared Providers/Services/Models work with both FluentApp and MaterialApp
  4. Navigation state persists when switching between navigation modes
**Plans**: 4 plans

Plans:
- [ ] 07-01: AdaptiveApp 平台检测入口
- [ ] 07-02: FluentApp shell (Windows)
- [ ] 07-03: MaterialApp shell (Android/Web)
- [ ] 07-04: 共享业务逻辑层验证

#### Phase 8: Windows 桌面端 UI
**Goal**: Windows 用户可以使用原生 Fluent Design 界面管理图片库
**Depends on**: Phase 7
**Requirements**: WIN-01, WIN-02, WIN-03, WIN-04, WIN-05, WIN-06, ENH-03
**Success Criteria** (what must be TRUE):
  1. User can navigate between Gallery, Search, Tags, and Settings using NavigationView sidebar
  2. User can browse images in grid layout with Fluent-styled cards
  3. User can view image details, metadata, and manage tags in Fluent dialog
  4. User can minimize, maximize, and close the application window using native controls
  5. User can access common page actions via CommandBar toolbar
**Plans**: 7 plans

Plans:
- [x] 08-01: NavigationView 侧边导航栏
- [x] 08-02: 图库浏览界面 (Fluent 网格)
- [x] 08-03: 图片详情与标签管理界面
- [x] 08-04: 搜索界面
- [x] 08-05: 标签管理界面
- [x] 08-06: Windows 窗口控制
- [x] 08-07: CommandBar 工具栏

#### Phase 9: Android 移动端 UI
**Goal**: Android 用户可以使用自适应 Material 3 界面管理图片库
**Depends on**: Phase 8 (Windows UI patterns established)
**Requirements**: ANDROID-01, ANDROID-02, ANDROID-03, ANDROID-05, CROSS-03
**Success Criteria** (what must be TRUE):
  1. User sees NavigationBar on phones (< 600px) and NavigationRail on tablets (≥ 600px)
  2. User can browse images with responsive grid that auto-adjusts columns by screen width
  3. User can navigate between Gallery, Search, and Settings sections
  4. User can select images via long-press and navigate with swipe gestures
  5. Navigation switches smoothly between Bar and Rail when resizing screen
**Plans**: 5 plans

Plans:
- [ ] 09-01: NavigationBar 底部导航栏 (手机端)
- [ ] 09-02: NavigationRail 侧边导航栏 (平板端)
- [ ] 09-03: 响应式网格布局
- [ ] 09-04: 触摸手势交互优化
- [ ] 09-05: 响应式断点系统

#### Phase 10: 主题统一与优化
**Goal**: 用户在双平台体验一致的二次元风格主题设计
**Depends on**: Phase 9
**Requirements**: WIN-07, ANDROID-04, CROSS-01, CROSS-02, ENH-01, ENH-02
**Success Criteria** (what must be TRUE):
  1. User sees consistent pink-purple anime color scheme on Windows and Android
  2. User can switch between light and dark themes following system preference
  3. Windows user can use keyboard shortcuts for common actions (Ctrl+N, Delete, arrow keys)
  4. Windows user sees hover highlight and shadow effects on buttons and cards
  5. Theme changes apply immediately without app restart
**Plans**: 6 plans

Plans:
- [ ] 10-01: Fluent 主题配色 (Windows)
- [ ] 10-02: Material 3 主题配色 (Android)
- [ ] 10-03: 统一配色系统
- [ ] 10-04: 明暗主题切换
- [ ] 10-05: 键盘快捷键 (Windows)
- [ ] 10-06: 鼠标悬停效果 (Windows)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 基础架构、图片扫描与标签基础层 | v1.0 | 3/3 | Complete | 2026-03-14 |
| 2. 缩略图、基础浏览与 AI 复核界面底座 | v1.0 | 5/5 | Complete | 2026-03-15 |
| 3. AI 开放标签与治理 | v1.0 | 6/6 | Complete | 2026-03-15 |
| 4. 重复检测与搜索 | v1.0 | 6/6 | Complete | 2026-03-17 |
| 5. 收藏夹与批量操作 | v1.0 | 4/4 | Complete | 2026-03-18 |
| 6. 优化与部署 | v1.0 | 4/4 | Complete | 2026-03-19 |
| 7. 架构基础层 | v2.0 | 0/4 | Planned | - |
| 8. Windows 桌面端 UI | v2.0 | 0/7 | Planned | - |
| 9. Android 移动端 UI | v2.0 | 0/5 | Not started | - |
| 10. 主题统一与优化 | v2.0 | 0/6 | Not started | - |

---
*Roadmap created: 2026-03-14*
*Last updated: 2026-03-20 after Phase 8 planned*