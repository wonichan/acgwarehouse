# 技术设计：前端视觉升级（切片 C）

## 1. 边界

| 层 | 改什么 | 不改什么 |
|----|--------|----------|
| 依赖 | 增加 `lucide-vue-next` | 不引入 UI 框架、动画库 |
| 全局 | `AppHeader`；必要时微调 `app.css` 中 **共享** art-card/header/detail 基础类 | 不为单页堆 feature 选择器 |
| 页面 | `GalleryPage`、`DetailPage` | Search/Trending/Collections/Account 仅承受全局副作用 |
| 组件 | `ArtCard`、`DetailLoadingState`（对齐影院衬）、可选 `AppIcon` | 不重写 Carousel 业务逻辑（可加图标按钮） |
| 数据 | 无 | 全部 API / 类型 / 排序 / 分页契约 |

## 2. 架构与模块

```
lucide-vue-next
    └── (optional) components/AppIcon.vue  — size/stroke 统一
AppHeader.vue     — 导航 + 搜索 + 账户 + 小屏折叠
ArtCard.vue       — 选择勾 Lucide + hover 层次（scoped 增强）
DetailPage.vue    — 沉浸布局 + 影院衬 + zoom 控件图标
DetailLoadingState.vue — 骨架与影院衬协调
GalleryPage.vue   — hero/区块节奏/toolbar 层次（逻辑尽量不动）
app.css           — 仅共享 token 派生或极小共享规则
```

### 2.1 图标

- 包：`lucide-vue-next`（与 Vue 3 SFC 兼容，tree-shake 按名导入）。
- 约定：
  - `import { Search, Menu, X, Check, ZoomIn, ZoomOut, ... } from 'lucide-vue-next'`
  - 或薄封装 `AppIcon`：`name`/`size`/`strokeWidth`，默认 20px、stroke 2，`aria-hidden` 当邻接可见文案时。
- 禁止：emoji；用 Unicode 符号冒充图标（如 `✓`）。
- 无障碍：仅图标按钮必须有 `aria-label`（现有选择按钮已有 label，保留并增强）。

### 2.2 Header（响应式）

- Desktop：品牌 | nav links | 搜索 + 账户（现状结构增强当前态/图标）。
- Tablet/Mobile：
  - 搜索可缩为图标展开或全宽第二行；
  - nav 用菜单按钮 + 展开面板 / drawer（`transform`+`opacity`），焦点陷阱非必须 MVP，但须可键盘关闭（Esc）与 `aria-expanded`。
- Sticky topnav 已有；折叠面板 `z-index` 高于 toolbar（toolbar sticky top ~88px）。

### 2.3 ArtCard

- 保留：`aspect-ratio` from width/height、RouterLink 进详情、`useSelection` 多选。
- 视觉：默认干净；hover/focus-visible lift（已有 translateY）+ 可选 meta 对比增强；`select-check` 内 Lucide `Check`。
- 样式：优先 `ArtCard.vue` scoped；若必须改全局 `.art-card`，保持全站一致且不破坏 Collection 等复用处。

### 2.4 Detail 沉浸 + 影院衬

**布局**

- 保持 `detail-stage` 双栏（主图 | 侧栏）；桌面主图列更大、侧栏固定宽度量级（现 360px 可微调）。
- 窄屏：单列，主图在上、操作在下（已有媒体查询须核对并补强）。
- 主图容器：真实 `width`/`height` 驱动 `aspect-ratio` 或 max 高度，避免固定 3/4 裁切导致大图体验差（在「可检视整图」方向靠拢 `object-fit: contain` + min/max 高度，符合 component-guidelines）。

**影院衬（默认 B）**

- 作用域：详情 section / viewer 面板背景，**不是** `:root` 或全站 shell。
- 示例策略（实现时可微调数值，原则不变）：
  - viewer 背景：`color-mix(in oklab, var(--fg), transparent 8~18%)` 或更深混合
  - 控件：secondary 按钮在暗衬上用高对比 surface/accent
  - 侧栏可保持暖色 surface，形成「影院银幕 + 档案侧栏」双轨，避免整页变黑难读
- Loading：`DetailLoadingState` 主图占位与 viewer 同衬，避免亮骨架闪一下。

**Zoom**

- 继续 CSS `transform: scale(var(--zoom))`；按钮换 Lucide + 保留文案或 `aria-label`。
- 可选增强（本任务允许、非必须全做）：滚轮缩放需 `preventDefault` 谨慎（仅 hover 在 viewer 时）；双击复位。若时间紧，图标化现有三按钮即可。

### 2.5 Gallery 策展

- **不改**数据加载、filter map、masonry 算法、IntersectionObserver 参数语义。
- **改**呈现：
  - hero：字号层次、间距、CTA 与 carousel 对齐；空/错状态保持 panel 契约
  - feed 区 section 标题/eyebrow（「全部作品」类）与 sticky toolbar 视觉层级
  - DailyRecommendations 上下节奏（margin/section 标题），组件内部大改非必须

## 3. 数据流与契约

无 API 变更。继续：

- Gallery：`getImages` + `getCommunityFocusSlides` + `useDailyRecommendations`
- Detail：`getImage` / `rateImage` / pickers
- 路由：`/detail?id=`、`/?tag=`

## 4. 兼容与回归风险

| 风险 | 缓解 |
|------|------|
| masonry 回归 | 不改列算法；只改卡片视觉 |
| sticky 层叠 | 测 header 菜单 vs toolbar `z-index` |
| 影院衬对比度 | 侧栏保持亮色 panel；控件 focus-ring 保留 |
| 图标包体积 | 按名 import，禁止 `import *` |
| 工具页被 Header 挤坏 | 小屏菜单方案在 Account/Search 路由各扫一眼 |
| `app.css` 膨胀 | feature 进 scoped |

## 5. 动效

- 仅 `transform` / `opacity`
- 复用 `--motion-fast` / `--motion-base` / `--ease-standard`
- `@media (prefers-reduced-motion: reduce)`：关闭连续动画与大位移（与现有 carousel/account 模式一致）

## 6. 回滚

- 依赖：移除 `lucide-vue-next` 与图标引用即可回退图标层
- 页面：按文件 git revert（`AppHeader` / `ArtCard` / `DetailPage` / `GalleryPage` / loading）
- 无数据迁移

## 7. 验证

见 `implement.md` 验证命令与检查清单。
