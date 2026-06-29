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

```vue
<style scoped>
.picker-panel { margin-top: var(--space-4); }
</style>
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
