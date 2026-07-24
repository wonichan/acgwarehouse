# Component Guidelines

> How components are built in this project.

---

## Overview

- 栈：Vue 3 SFC + TypeScript + Vite；页面在 `src/pages/`，可复用 UI 在 `src/components/`，逻辑在 `src/composables/`。
- 视觉身份：根目录 `DESIGN.md` + `src/assets/app.css` tokens；旗舰页（首页 hero、详情观影）可更沉浸，工具页保持清晰（见根目录 `AGENTS.md` frontend 段）。
- 图标：统一 Lucide（见下节）；禁止 emoji / Unicode 符号当图标。
- 无障碍：交互控件保留可见 focus；仅图标按钮必须有 `aria-label`；折叠菜单见「Mobile Disclosure Menu」。

---

## Icons (Lucide)

**What**: 用户可见交互图标统一使用 `lucide-vue-next`，经薄封装 `src/components/AppIcon.vue` 引用。

**Why**: 避免 emoji 渲染不一致与字符图标（如 `✓`）的 a11y/对齐问题；按名 import 便于 tree-shake。

**Rules**:
- 依赖：`lucide-vue-next`（禁止 `import * as Icons`）。
- 用法：`import { Search } from 'lucide-vue-next'` + `<AppIcon :icon="Search" :size="16" />`。
- 邻接可见文案时图标 `aria-hidden`（`AppIcon` 默认）；仅图标按钮由父级提供 `aria-label`。
- 禁止：界面 emoji；用 `✓` / `×` 等字符冒充图标。

```vue
<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import AppIcon from '@/components/AppIcon.vue'
</script>

<template>
  <button type="button" aria-label="选择图片">
    <AppIcon :icon="Check" :size="16" />
  </button>
</template>
```

---

## Cinema Backdrop (详情沉浸衬)

**What**: 详情主图区默认「影院衬」——局部更深背景，不是全站 dark theme。

**Why**: 旗舰观影与暖色档案侧栏形成双轨；改 `:root` 会污染 Header 与其它路由。

**Rules**:
- 作用域：详情 viewer / loading 主图区的 scoped 样式（或明确的局部 class），**不**改 `:root` 为 dark。
- 颜色：仅用现有 token 的 `color-mix`（如 `--fg` / `--accent` / `--surface`），禁止新 raw 色板。
- 侧栏元数据/评分/标签/收藏保持可读 surface；主图用真实 `width`/`height` + `object-fit: contain`。
- Loading 骨架（`DetailLoadingState`）须与影院衬同调，避免亮骨架闪白。
- 动效仅 `transform` / `opacity`，尊重 `prefers-reduced-motion`。

```css
/* Correct: local cinema surface */
.cinema-viewer {
  background: linear-gradient(
    160deg,
    color-mix(in oklab, var(--fg), transparent 10%),
    color-mix(in oklab, var(--fg), transparent 4%)
  );
}

/* Wrong: global dark theme for one page */
:root { --bg: #111; }
```

---

## Mobile Disclosure Menu (Header)

**What**: ≤1180px 时桌面 `nav-links` 隐藏，改用菜单按钮展开的 disclosure 面板。

**Why**: 仅用 CSS 隐藏桌面导航会导致无入口；仅用 `hidden` 时若媒体查询写了 `display:block`，关闭态链接仍可被 Tab 聚焦。

**Required**:
- 触发器：`aria-expanded`、`aria-controls`、明确 `aria-label`（打开/关闭）。
- Esc 关闭；路由变化时关闭。
- 关闭态：对面板使用 `:inert`（或等价），避免 `display:block` 覆盖 `hidden` 后仍可键盘聚焦。
- 动效：`max-height`/`opacity`/`transform` + reduced-motion 全局契约。
- Focus trap 非 MVP 必做；打开后至少保证 Esc 与 inert-when-closed。

```vue
<button
  type="button"
  :aria-expanded="menuOpen"
  aria-controls="mobile-nav-panel"
  :aria-label="menuOpen ? '关闭导航菜单' : '打开导航菜单'"
  @click="toggleMenu"
/>
<div
  id="mobile-nav-panel"
  :hidden="!menuOpen"
  :inert="!menuOpen || undefined"
  :class="{ 'is-open': menuOpen }"
>
  <!-- links -->
</div>
```

---

## Component Structure

- 单文件组件：`<script setup lang="ts">` + template + 可选 `<style scoped>`。
- 页面级路由组件放 `pages/`；跨页复用放 `components/`。
- 类型从 `@/api/client` 等处以 `import type` 引入。

---

## Props Conventions

- 使用 `defineProps` + TypeScript 泛型或接口；可选 props 用 `withDefaults`。
- 布尔 props 默认值显式写出；事件用 `defineEmits` 声明载荷类型。

---

## Styling Patterns

- Prefer existing design-system classes from `frontend/vue-gallery/DESIGN.md` and `src/assets/app.css` (`panel`, `form-grid`, `field`, `status`, `tag`, `btn`, spacing tokens).
- Task-specific component affordances belong in the component's `<style scoped>` block. Do not add narrow feature selectors to the oversized global `app.css` unless the class is intentionally reusable across the whole app.
- Scoped styles must use existing CSS variables such as `--space-*`, `--radius-*`, `--border`, `--accent`, and motion tokens; do not introduce one-off colors or hard-coded pixel systems.
- Frontend layout changes must remain responsive for desktop, tablet, and mobile. Prefer fluid grids/flex, `minmax()`, `clamp()`, wrapping, and existing spacing tokens over fixed-width layouts that only fit one viewport.
- Components rendered inside narrow sidebars or cards should respond to their own container width, not only the page viewport. Use `container-type: inline-size` plus `@container` for local layout switches when a desktop viewport can still provide a narrow component slot.
- Image-led UI should carry backend `width` / `height` through its presentation type and use the real aspect ratio in CSS. Prefer `object-fit: contain` when users need to inspect the whole image; add min/max height bounds for extreme ratios instead of forcing one fixed crop.

```vue
<style scoped>
.picker-panel { margin-top: var(--space-4); }
</style>
```

### Loading States

- Do not expose backend route names, proxy details, or fetch internals in user-visible loading UI. Keep API paths such as `/api/v1/...` out of normal loading copy.
- For image-led pages, prefer a layout-preserving skeleton that reserves the eventual media area (`.viewer-art`, `.thumb`, or a component-specific equivalent) so loading does not collapse into a text-only panel.
- If the product decision is animation-only loading, visible explanatory text should be omitted; use `role="status"` with an `aria-label` or `.sr-only` text for assistive technology instead.
- Prefer local CSS using existing design tokens and motion tokens before adding a UI library or dependency for a single loading state.
- Loading animation must communicate loading state and respect `prefers-reduced-motion`; shimmer, pulse, and float effects should disable continuous motion under reduced-motion settings.

### Design Skills (UI/UX Work)

When writing or redesigning UI/UX code (new pages, major layout changes, visual rework, component restyling), consider loading one or more of these skills before writing the visual code:

- `/redesign-existing-projects` — Upgrades existing UI to premium quality without breaking functionality. Use when redesigning a page that already exists (e.g., restyling `CollectionsPage.vue` or building a new detail page that should match premium design standards).
- `/design-taste-frontend-v1` — The original taste-skill. Enforces anti-slop design rules and blocks generic AI defaults. Use as the baseline taste gate for any visible UI work.
- `/high-end-visual-design` — Defines exact fonts, spacing, shadows, card structures, and animations that make a site feel expensive. Use when adding new visual components that need polish (e.g., the collection cover card, the detail page image grid).

**When to load**:
- New page or major section → load at least `/design-taste-frontend-v1`.
- Redesigning an existing page → load `/redesign-existing-projects` + `/design-taste-frontend-v1`.
- New visual component needing polish (cards, grids, covers, modals) → load `/high-end-visual-design`.

**Not required for**:
- Pure logic changes with no visible UI delta.
- Adding a type or API method.
- Backend-only work.

Load via the `skill` tool. Do not skip loading when the work is visibly user-facing — these skills block generic AI defaults that the project's design system exists to prevent.

### Responsive Layout Contract

**What**: When editing pages, panels, lists, forms, modals, or navigation, verify the layout at desktop, tablet, and mobile widths.

**Why**: A UI that fits desktop only often breaks search/list density, action buttons, and forms on smaller screens.

**Required checks**:
- Desktop: primary content uses available width without oversized empty columns.
- Tablet: grids and side-by-side controls wrap cleanly; no horizontal scrolling unless the content is an intentional data table.
- Mobile: controls remain tappable, text wraps inside its container, and sticky/fixed elements do not cover content.

**Wrong**:
```css
.toolbar {
  width: 1180px;
  display: flex;
}
```

**Correct**:
```css
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
```

### Image Feed Layout Contract

**What**: Infinite image feeds must reserve image space before the image file loads.

**Why**: Lazy-loaded images without known dimensions cause layout shift. CSS multi-column masonry also lets the browser rebalance old cards when new cards arrive, which makes the visible scroll position jump.

**Required**:
- Carry backend `width` / `height` through the presentation type used by image cards.
- Set card preview space with `aspect-ratio` or equivalent stable dimensions before the image loads.
- For infinite masonry feeds, append new cards to explicit stable columns or a position cache. Do not use CSS `columns` as the primary layout for paginated image feeds.
- Rebuild columns only for explicit layout changes such as viewport column-count changes or filter resets; next-page append must not redistribute already-rendered cards.

**Wrong**:
```css
.masonry {
  columns: 4 240px;
}
```

**Correct**:
```vue
<div class="masonry" :style="{ '--masonry-columns': String(columnCount) }">
  <div v-for="column in columns" class="masonry-column">
    <ArtCard v-for="item in column" :key="item.id" :item="item" />
  </div>
</div>
```

---

### Autoplay / Motion Contract (Carousels, Marquees, Auto-Advance)

**What**: Any component that automatically advances slides, rows, or media on a timer must respect user context: hover, focus, viewport visibility, motion preferences, and Vue lifecycles.

**Why**: Autoplay that ignores these signals steals attention, drains battery in background tabs, breaks screen readers, and leaks timers across `<KeepAlive>` reactivation.

**Required in the composable that owns the timer** (see `useCarousel.ts` for the reference implementation):

- Expose `pause()` / `resume()` for pointer interaction and `pauseByFocus()` / `resumeByFocus()` for keyboard focus. The component template binds `@mouseenter/leave` and `@focusin/focusout` on the root.
- Any user-driven `next` / `prev` / `goto` must **reset** the pending timer, not stack a new one on top.
- Listen to `document.visibilitychange` and stop the timer while `document.visibilityState === 'hidden'`.
- Listen to `window.matchMedia('(prefers-reduced-motion: reduce)')`, including its `change` event, and refuse to autoplay while it matches.
- Bail out when `slides.length <= 1`.
- Register the listeners in `onMounted` **and** `onActivated`; tear them down in `onBeforeUnmount` **and** `onDeactivated`. Never leave a `setTimeout` handle live across `<KeepAlive>` deactivation.

**Required in the SFC**:

- Fade-based transitions (`opacity` + `transform` + `filter`) are preferred over sliding `translateX` because they compose better with `position: grid` overlays and don't require known viewport widths.
- Custom cubic-bezier only (e.g. `cubic-bezier(0.32, 0.72, 0, 1)`). Do not use `linear` or `ease-in-out` for the main transition. Under `prefers-reduced-motion`, a short (≤ 120ms) opacity-only fade with `linear` is acceptable because it minimises perceived motion.
- Non-active slides must be `aria-hidden="true"` and set `pointer-events: none` (or `inert`) so keyboard users cannot focus into off-screen content.

---

## Accessibility

- Mutation actions that require login must render durable visible state, not toast-only feedback.
- Use `role="alert"` for login-required or mutation error states that appear after user action, and include a `/account` link when login resolves the state.

```vue
<div class="status status--error is-visible" role="alert">
  <span>请先登录后再管理标签</span>
  <RouterLink class="btn btn-secondary btn-small" to="/account">去登录</RouterLink>
</div>
```

---

## Common Mistakes

- Do not replace real mutation workflows with success toasts. If the backend route exists, call it and only show success after it resolves.
- Do not keep feature-specific picker/list styles in `app.css` just to share them between two components; extract a small component or duplicate a few scoped token-backed rules.
- Do not use emoji or Unicode glyphs (`✓`, `×`) as UI icons; use Lucide via `AppIcon`.
- Do not implement page-local dark mode by changing `:root` tokens; use scoped cinema backdrop `color-mix` instead.
- Do not hide desktop nav with CSS alone without a mobile disclosure entry; when the open panel uses `display:block` in a media query, closed state must use `:inert` (or remove from tab order), not only `hidden`/`opacity`/`pointer-events`.
