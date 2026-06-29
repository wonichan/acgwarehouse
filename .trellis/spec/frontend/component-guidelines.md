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

```vue
<style scoped>
.picker-panel { margin-top: var(--space-4); }
</style>
```

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
