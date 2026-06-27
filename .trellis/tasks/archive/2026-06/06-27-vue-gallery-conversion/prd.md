# Vue.js 图库原型转换

## Goal

将 `/opt/acgwarehouse/frontend/example` 的静态 HTML 原型转换为 Vue.js 项目，保持现有视觉设计、布局、交互和素材不变，生成可运行的 Vue.js SPA。

## Background

这是一个面向二次元爱好者的在线图库高保真原型，使用 Colorful 设计系统（暖黄背景、白色面板、橙色强调色）。

### 现有资源

| 类型 | 内容 |
|------|------|
| 页面 | 6 个静态 HTML 页面 |
| 样式 | `assets/app.css` (370 行)，包含完整设计令牌和组件样式 |
| 脚本 | `assets/app.js` (334 行)，实现轮播、批量选择、缩放、表单验证等交互 |
| 设计规范 | `DESIGN.md` 定义视觉系统、组件原则、动效和响应式规则 |

### 页面清单

| 页面 | 路径 | 功能 |
|------|------|------|
| 图库首页 | `/` | 瀑布流浏览、本周社区焦点轮播、批量选择 |
| 图片详情 | `/detail` | 图片放大查看、标签评分、收藏操作 |
| 智能搜索 | `/search` | 关键词/标签/评分筛选、结果列表 |
| 热榜 | `/trending` | 每日/每周/每月榜单切换 |
| 收藏夹 | `/collections` | 相册创建、相册列表、批量整理 |
| 账户中心 | `/account` | 登录注册、资料编辑、偏好设置、安全操作 |

### 已实现的交互逻辑 (app.js)

- **轮播**: `bindCommunityCarousel()` - 上一张/下一张/分页圆点/键盘导航
- **批量选择**: `bindSelectableCards()` - 卡片选择、批量操作面板、取消选择
- **图片缩放**: `bindZoomViewer()` - 放大/缩小/复位
- **分段切换**: `bindSegments()` - 热榜周期切换
- **搜索筛选**: `bindSearchFilters()` - 表单提交、结果更新
- **认证标签页**: `bindAuthTabs()` - 登录/注册切换、键盘导航
- **表单验证**: `bindForms()` - 必填校验、邮箱格式、错误提示、提交状态
- **入场动效**: `bindAccountMotion()` - IntersectionObserver 渐显
- **Toast 通知**: `showToast()` - 全局状态反馈

## Requirements

### 功能需求

1. **路由配置**: 使用 Vue Router 实现 6 个页面的 SPA 路由
2. **组件拆分**: 将公共部分（顶部导航、Toast、批量操作面板）提取为可复用组件
3. **状态管理**: 使用 Composition API 的 reactive/ref 管理全局状态
4. **样式迁移**: 保持 `app.css` 设计令牌，确保视觉一致性
5. **交互复现**: 所有 `app.js` 中的交互逻辑使用 Vue Composables 实现

### 技术约束

- 目标框架: Vue 3 + Composition API + TypeScript
- 构建工具: Vite
- 样式方案: 全局 CSS（复用现有 `app.css`）
- 无外部图片素材，所有图片示意使用 CSS 渐变生成

### 非功能需求

- 保持响应式布局 (桌面/平板/移动端)
- 保持可访问性 (aria-* 属性、键盘导航)
- 保持 `prefers-reduced-motion` 支持
- 首屏加载性能优化 (代码分割)

## Confirmed Decisions

- **Vue 版本**: Vue 3 + Composition API
- **TypeScript**: 使用 TypeScript
- **构建工具**: Vite
- **样式方案**: 全局 CSS（复用现有 `app.css`）
- **UI 库**: 不使用，保持现有 CSS 组件样式
- **状态管理**: Composition API (reactive/ref)，无需 Pinia
- **项目位置**: `/opt/acgwarehouse/frontend/vue-gallery`

## Acceptance Criteria

- [ ] Vue 项目可运行，通过 `npm run dev` 启动开发服务器
- [ ] 6 个页面路由正确，导航链接工作正常
- [ ] 首页轮播功能完整：上一张/下一张/分页圆点/键盘导航
- [ ] 图库卡片批量选择功能正常，批量操作面板显示/隐藏正确
- [ ] 图片详情页缩放功能正常 (放大/缩小/复位)
- [ ] 搜索页表单提交后更新结果摘要
- [ ] 热榜页分段切换工作正常
- [ ] 收藏夹页相册创建表单提交正常
- [ ] 账户页登录/注册标签页切换正常，表单验证工作正常
- [ ] Toast 通知在各页面正常显示
- [ ] 响应式布局在各断点正常显示
- [ ] 所有 aria-* 属性和键盘导航保持不变

## Extension Strategy

后续扩展路径：
1. **Phase 1 (当前)**: 纯 Vue + 自定义 CSS + Composables
2. **Phase 2 (如需复杂交互)**: 引入 Headless UI Vue (无样式组件)
3. **Phase 3 (如需 API)**: 引入 Pinia + Axios
4. **Phase 4 (如需管理后台)**: 独立子项目 + Naive UI
