---
phase: 17
slug: desktop-shell-foundation
status: approved
shadcn_initialized: false
preset: none
created: 2026-04-04
reviewed_at: 2026-04-04T00:00:00Z
---

# Phase 17 — UI Design Contract

> Visual and interaction contract for frontend phases. Generated for Phase 17 desktop shell planning and consumed by the planner/executor.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | none |
| Preset | not applicable |
| Component library | fluent_ui |
| Icon library | FluentIcons |
| Font | Segoe UI, falling back to system sans-serif |

---

## Spacing Scale

Declared values (must be multiples of 4):

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | Icon gaps, compact chip padding, inline status spacing |
| sm | 8px | Button/icon separation, tile gaps, compact list rows |
| md | 16px | Default control spacing, panel padding, content insets |
| lg | 24px | Section padding, shell group spacing, gallery workspace gutters |
| xl | 32px | Major layout gaps between gallery workspace regions |
| 2xl | 48px | Empty-state vertical rhythm and wide panel breathing room |
| 3xl | 64px | Large empty-state top/bottom spacing only |

Exceptions: none

---

## Typography

| Role | Size | Weight | Line Height |
|------|------|--------|-------------|
| Body | 16px | 400 | 1.5 |
| Label | 14px | 600 | 1.3 |
| Heading | 20px | 600 | 1.2 |
| Display | 28px | 600 | 1.15 |

Type rules:
- Use only the four sizes above.
- Use only `400` and `600` weights.
- Top-bar action labels, filter section titles, and chips use `Label`.
- Page titles and panel headings use `Heading`.
- Empty-state hero heading uses `Display`.

---

## Color

| Role | Value | Usage |
|------|-------|-------|
| Dominant (60%) | `#F5F7FB` | Main shell background, gallery canvas, search field surface |
| Secondary (30%) | `#E7ECF5` | Navigation pane surface, right filter panel surface, cards, title bar contrast layer |
| Accent (10%) | `#2563EB` | Primary import button, active navigation indicator, focused search border, selected filter checkbox, selected state chip emphasis |
| Destructive | `#C42B1C` | Only destructive confirmations and destructive warning text if introduced |

Accent reserved for: primary import button, active navigation indicator, focused search box outline, selected filter checkbox state, selected filter chip emphasis — never all interactive elements.

---

## Visual Hierarchy

- **Primary focal point:** the persistent top-bar search box. It is the first visual anchor when the shell opens.
- **Secondary focal point:** the square gallery tile field. Users should read the gallery as a stable photo wall immediately below the shell chrome.
- **Tertiary focal point:** the right filter panel when open; it should support refinement without overpowering the gallery.
- Left navigation is present but visually quieter than the top bar. It exists for view switching, not as the primary action surface.
- Icon actions in the shell must always have visible text labels at standard widths. If the shell compresses at narrow widths, icons may collapse only after preserving tooltip + accessible label text.

---

## Interaction Contract

### Shell Top Bar
- Height target: `52px` visual rhythm.
- Layout order, left to right: drag/title region → persistent search box → import action → settings action.
- Search box remains visible in all standard desktop widths and keeps a text placeholder.
- Import action is the only primary button in the shell.
- Settings action is a secondary button with icon + text, not icon-only by default.

### Gallery Workspace
- Default gallery mode is square grid.
- Square tiles should read as fixed, even units; width changes may alter column count, not tile aspect ratio.
- Masonry mode may remain in code, but it must not dominate shell copy, controls, or affordances in this phase.

### Filter Panel
- The right filter panel opens alongside the gallery content, not as a modal dialog.
- Panel width target: `320px` default.
- Tag selection and untagged toggle apply immediately.
- Selected-tag chips in the gallery header reflect current filters and allow quick removal.
- Filter controls must be keyboard reachable in predictable top-to-bottom order.

### Import Feedback
- Import action feedback is lightweight: queued / success / failure messaging only.
- No task monitoring dashboard, live operations board, or sidecar diagnostics belong in this phase.

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Primary CTA | Import Library |
| Empty state heading | Your gallery is ready for its first import |
| Empty state body | Import a library folder to start browsing images in the desktop gallery. |
| Error state | Gallery refresh failed. Check the backend connection, then refresh the gallery. |
| Destructive confirmation | Clear Filters: Remove all selected tags and show the full gallery again. |

Additional copy rules:
- Search placeholder: `Search images and tags`
- Filter panel heading: `Filter by Tags`
- Untagged toggle label: `Show untagged images only`
- Settings button label: `Open Settings`
- Import success info bar: `Library import queued`
- Import failure info bar: `Library import could not start`

---

## Accessibility Contract

- Search box, import button, settings button, filter toggle, and all panel controls require visible labels or an accessible text equivalent.
- Focus order moves left-to-right across the top bar, then into page content, then into the filter panel.
- Selected filter state must not rely on color alone; use checkbox state, chip text, and count/selection text where applicable.
- Empty states and import feedback use text-first messaging rather than icon-only cues.

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| shadcn official | none | not required |
| third-party registries | none | not applicable |

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS
- [x] Dimension 2 Visuals: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Typography: PASS
- [x] Dimension 5 Spacing: PASS
- [x] Dimension 6 Registry Safety: PASS

**Approval:** approved 2026-04-04
