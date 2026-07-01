# Component Guidelines

> How components are built in this project.

---

## Overview

<!--
Document your project's component conventions here.

Questions to answer:
- What component patterns do you use?
- How are props defined?
- How do you handle composition?
- What accessibility standards apply?
-->

(To be filled by the team)

---

## Component Structure

<!-- Standard structure of a component file -->

(To be filled by the team)

---

## Props Conventions

<!-- How props should be defined and typed -->

(To be filled by the team)

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
