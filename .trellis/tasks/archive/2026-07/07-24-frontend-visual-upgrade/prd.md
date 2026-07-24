# 前端视觉升级：双轨旗舰体验

## Goal

在不破坏图库产品可用性的前提下，把 `frontend/vue-gallery` 的视觉与交互品质抬到「精品社区档案」水准：全站保持 `DESIGN.md` 的 warm community archive 身份；旗舰页（首页 hero、详情观影）允许更强叙事与沉浸；本任务不重做工具页，但全局 Header/图标变化不得破坏其可用性。

用户价值：浏览更像策展与画廊；核心路径（看图、评分、标签、收藏）仍一眼可达。

## Scope

**切片 C — 基础层 + 双旗舰**

| 纳入 | 不纳入 |
|------|--------|
| Lucide 图标体系 + 统一用法 | Search / Trending / Collections / Account 专门 redesign |
| `AppHeader` 升级（含小屏） | 全站 dark theme / 可切换观影模式 |
| `ArtCard` + Gallery 图流 polish | 后端 API / 新业务功能 |
| `GalleryPage` 策展感（hero / 日推节奏 / 筛选区） | 换路由、部署、推荐算法 |
| `DetailPage` 沉浸观影 + **默认影院衬** | 换 display 字体 / webfont 加载 |
| 清理无用脚手架（如确认无引用的 `HelloWorld.vue`） | 装置艺术级全站 redesign |

**已决产品决策**

1. 切片：**C**
2. 详情：**默认影院衬（B）** — 局部 token `color-mix`，非全站 dark，无切换记忆
3. 字体：**不换字（A）** — 继续现有 `Inter, system-ui` 声明；用字号/字重/tracking 做层次

**建议实现顺序**：图标+Header+ArtCard → Detail 沉浸+影院衬 → Gallery 策展。

## Background

- 设计源：`DESIGN.md` + `frontend/vue-gallery/src/assets/app.css` tokens；`.trellis/spec/frontend/` 契约。
- Agent 约束：`AGENTS.md` frontend 双轨（主身份 + 旗舰例外；Lucide；transform/opacity + reduced-motion；scoped）。
- 栈：Vue 3 + TS + Vite + Vue Router；**无图标库**；`index.html` **未加载 webfont**。
- 现状亮点：stable masonry、详情 loading skeleton、carousel reduced-motion/KeepAlive、auth-required。
- 现状缺口：无 Lucide；Header 纯文字；ArtCard 选择勾为 `✓` 字符；详情 `detail-stage` 双栏但 viewer 仍偏亮色 panel；hero 已有结构但策展层次可加强。
- 体量：`GalleryPage` ~381 行、`DetailPage` ~280 行、`ArtCard` ~80 行、`app.css` ~398 行。

## Requirements

### R1 — 身份与工程边界

- Token / 色板 / 间距 / motion 以 `DESIGN.md` + `app.css` 为准；新色仅 `color-mix` 自现有 token。
- 禁止 emoji；图标统一 Lucide（Vue 适配）。
- 动效仅 `transform` / `opacity`；尊重 `prefers-reduced-motion` 与 `--motion-*`。
- 新样式优先组件 `<style scoped>`；不向 `app.css` 堆 feature 选择器（共享 design-system 类可微调）。
- desktop / tablet / mobile 均可用。
- 不把评分/标签/收藏藏进难发现手势。

### R2 — 图标与 Header

- 依赖：`lucide-vue-next`（或项目选定的官方 Vue 适配包）。
- 统一用法：薄封装（如 `AppIcon`）或一致的 import 约定；禁止散落 emoji/字符当图标。
- 替换范围内可见字符图标（至少 `ArtCard` 的 `✓` → Lucide Check）。
- `AppHeader`：品牌 / 主导航 / 快速搜索 / 账户；当前页可辨；小屏不挤爆（折叠 nav 或等价）。
- 删除确认无引用的脚手架（`HelloWorld.vue`）。

### R3 — ArtCard 与图流

- 默认图优先；hover/focus：克制 lift + 元数据层次（可在现有 `.art-card:hover` 上增强，scoped 优先）。
- 保留 width/height、`aspect-ratio`、stable masonry 列追加；禁止 CSS `columns` 作为主图流。
- 不破坏多选 / batch 行为。

### R4 — Gallery 策展

- 强化 hero 文案层级与区块节奏（eyebrow / display 字号 / CTA / carousel）。
- `DailyRecommendations` 与 feed 之间节奏清晰；筛选 toolbar sticky 可用且不遮挡关键内容。
- 筛选、tag 深链、无限滚动、KeepAlive 行为保持正确。
- 空/错/加载不暴露 API 路径。

### R5 — Detail 沉浸 + 默认影院衬

- 主图主导；侧栏元数据/评分/标签/收藏可见可操作。
- **默认影院衬**：主图区（及必要局部 chrome）更深衬底；token `color-mix` only。
- 侧栏与控件对比度可读；loading skeleton 与影院衬协调（`DetailLoadingState` 对齐）。
- `useZoom` 增强可发现性（图标按钮等）；缩放仍仅 transform。
- `AuthRequiredStatus` / picker 成功路径保持。

## Acceptance Criteria

- [x] AC1：本任务触及 UI 无 emoji；交互图标来自 Lucide；`frontend/vue-gallery` 下 `npm run build` 通过。
- [x] AC2：`AppHeader` 三端可用；当前导航可辨；搜索与账户可达。
- [x] AC3：`ArtCard` 图优先 + hover/focus polish；masonry 稳定列与 aspect-ratio 无回归；选择控件用 Lucide 而非 `✓`。
- [x] AC4：`GalleryPage` hero/策展 vs 列表层次强于改前；筛选与无限滚动仍工作。
- [x] AC5：`DetailPage` 沉浸布局 + **默认影院衬**；评分/标签/收藏可达；影院色仅 token 派生。
- [x] AC6：reduced-motion 遵守；无新 raw 色板；scoped 为主。
- [x] AC7：Header / Gallery masonry / Detail 主图+侧栏 三端无布局回归。

## Out of Scope

- 工具页专门视觉 redesign
- 全站 dark theme；可切换观影模式与偏好记忆
- webfont / 换 display 字体
- 后端、路由模式、部署、新业务能力
- 完整 redesign 每一个非旗舰页

## Technical Notes（摘要，细节见 design.md）

- 图标包：`lucide-vue-next`
- 影院衬：Detail 局部 class + scoped/`color-mix`，不改 `:root` 为 dark
- 字体：不改 `DESIGN.md` 字体表；仅用现有 scale 做层次
- 实现顺序与验证命令见 `implement.md`

## Open Questions

无阻塞项。规划可进入 design/implement 评审后 `task.py start`。
