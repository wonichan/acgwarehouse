# Vue.js 图库原型转换 - 实现计划

## Execution Checklist

### Phase 1: 项目初始化

- [ ] 1.1 创建 Vue 项目
  ```bash
  cd /opt/acgwarehouse/frontend
  npm create vite@latest vue-gallery -- --template vue-ts
  cd vue-gallery
  npm install
  npm install vue-router@4
  ```

- [ ] 1.2 配置项目结构
  ```bash
  mkdir -p src/{composables,components,pages,router,types}
  ```

- [ ] 1.3 复制 CSS 资源
  ```bash
  cp ../example/assets/app.css src/assets/
  ```

- [ ] 1.4 配置路径别名 (vite.config.ts, tsconfig.json)

- [ ] 1.5 验证项目可运行
  ```bash
  npm run dev
  ```

### Phase 2: 类型定义与 Composables

- [ ] 2.1 创建类型定义 `src/types/index.ts`
  - ArtItem, CarouselSlide, Album 类型

- [ ] 2.2 实现 `useToast` composable
  - 全局 toast 状态管理
  - 验证: 单元测试或手动验证

- [ ] 2.3 实现 `useSelection` composable
  - 批量选择状态管理
  - 验证: 选择/取消/清空功能

- [ ] 2.4 实现 `useCarousel` composable
  - 轮播索引管理
  - 验证: next/prev/goto 功能

- [ ] 2.5 实现 `useZoom` composable
  - 图片缩放状态
  - 验证: zoomIn/zoomOut/reset 功能

- [ ] 2.6 实现 `useForm` composable
  - 表单验证与提交状态
  - 验证: 必填校验、邮箱格式

- [ ] 2.7 实现 `useTabs` composable
  - 标签页切换逻辑
  - 验证: 键盘导航

- [ ] 2.8 实现 `useMotion` composable
  - 入场动效逻辑
  - 验证: IntersectionObserver 触发

### Phase 3: 公共组件

- [ ] 3.1 实现 `AppHeader.vue`
  - 复用原导航 HTML 结构
  - 使用 RouterLink 替代 a 标签
  - 验证: 导航链接高亮正确

- [ ] 3.2 实现 `AppFooter.vue`
  - 复用原页脚 HTML
  - 验证: 响应式显示正常

- [ ] 3.3 实现 `Toast.vue`
  - 使用 useToast composable
  - 验证: 消息显示/自动隐藏

- [ ] 3.4 实现 `BatchPanel.vue`
  - 使用 useSelection composable
  - 验证: 选择计数/面板显示隐藏

- [ ] 3.5 实现 `ArtCard.vue`
  - 作品卡片组件
  - 验证: 选择功能/链接跳转

- [ ] 3.6 实现 `Carousel.vue`
  - 使用 useCarousel composable
  - 验证: 轮播切换/键盘导航/圆点指示

- [ ] 3.7 实现 `SegmentedControl.vue`
  - 分段切换组件
  - 验证: 选项切换/状态反馈

- [ ] 3.8 实现 `Panel.vue`
  - 面板容器组件
  - 验证: raised 样式变体

### Phase 4: 路由配置

- [ ] 4.1 创建路由配置 `src/router/index.ts`
  - 6 个页面路由
  - 路由懒加载

- [ ] 4.2 配置 App.vue
  - RouterView
  - AppHeader / AppFooter / Toast / BatchPanel

- [ ] 4.3 验证路由跳转正常

### Phase 5: 页面实现

- [ ] 5.1 实现 `GalleryPage.vue`
  - Hero 区域 + 轮播 + 瀑布流卡片
  - 验证: 轮播功能/批量选择/导航跳转

- [ ] 5.2 实现 `DetailPage.vue`
  - 图片查看器 + 缩放控制 + 详情面板
  - 验证: 缩放功能/表单提交/相似推荐

- [ ] 5.3 实现 `SearchPage.vue`
  - 搜索表单 + 结果列表
  - 验证: 表单提交/结果摘要更新

- [ ] 5.4 实现 `TrendingPage.vue`
  - 热榜列表 + 分段切换
  - 验证: 周期切换/列表显示

- [ ] 5.5 实现 `CollectionsPage.vue`
  - 相册创建 + 相册网格 + 批量操作
  - 验证: 表单提交/批量选择

- [ ] 5.6 实现 `AccountPage.vue`
  - 登录注册标签页 + 资料编辑 + 偏好设置
  - 验证: 标签页切换/表单验证/开关交互

### Phase 6: 集成验证

- [ ] 6.1 响应式布局验证
  - 桌面端 (> 1180px)
  - 平板端 (744px - 1180px)
  - 移动端 (< 744px)
  - 工具: Chrome DevTools 设备模拟

- [ ] 6.2 可访问性验证
  - 键盘导航完整
  - aria 属性正确
  - 工具: axe DevTools 或手动检查

- [ ] 6.3 交互完整性验证
  - 所有交互与原型一致
  - Toast 消息显示正确
  - 表单验证反馈正常

- [ ] 6.4 构建验证
  ```bash
  npm run build
  npm run preview
  ```

## Validation Commands

```bash
# 开发服务器
npm run dev

# 类型检查
npm run type-check  # 需在 package.json 添加: "vue-tsc --noEmit"

# 构建生产版本
npm run build

# 预览生产版本
npm run preview
```

## Risky Files / Rollback Points

| 风险点 | 回滚方案 |
|--------|----------|
| CSS 类名冲突 | 检查 `app.css` 中是否有与 Vue 内置类名冲突 |
| 路由懒加载失败 | 回退到静态导入 |
| Composable 状态丢失 | 确保 composable 在组件树顶层调用 |

## Pre-Start Checklist

- [ ] Node.js >= 18 已安装
- [ ] npm >= 9 已安装
- [ ] `/opt/acgwarehouse/frontend/example` 目录存在且完整
- [ ] 网络连接正常（用于 npm install）