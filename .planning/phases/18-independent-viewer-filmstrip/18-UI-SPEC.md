---
phase: 18
slug: independent-viewer-filmstrip
status: approved
shadcn_initialized: false
preset: none
created: 2026-04-05
reviewed_at: 2026-04-05T00:00:00Z
---

# Phase 18 — UI Design Contract

> Visual and interaction contract for frontend phases. Generated for Phase 18 viewer planning and consumed by the planner/executor.

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
| xs | 4px | Inline icon gaps, compact metadata rows, thumbnail border insets |
| sm | 8px | Filmstrip item spacing, button gaps, compact sidebar groups |
| md | 16px | Default control spacing, sidebar padding, viewer chrome insets |
| lg | 24px | Section spacing between main canvas, sidebar groups, and command clusters |
| xl | 32px | Major desktop workspace gutters |
| 2xl | 48px | Large empty/loading-state rhythm only |

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
- Viewer metadata labels, filmstrip status text, and compact actions use `Label`.
- Sidebar section titles and viewer chrome titles use `Heading`.
- Empty-state hero text inside the viewer uses `Display` only when no image can render.

---

## Color

| Role | Value | Usage |
|------|-------|-------|
| Dominant (60%) | `#0F1115` | Viewer background, image stage surround, dark desktop workspace base |
| Secondary (30%) | `#1B1F27` | Right metadata sidebar surface, filmstrip surface, title/control bands |
| Accent (10%) | `#2563EB` | Selected filmstrip border, active focus outline, current-item emphasis |
| Subtle Text | `#C9D1D9` | Secondary metadata text, helper copy, inactive position labels |

Accent reserved for: selected thumbnail emphasis, keyboard focus ring, and viewer-active affordances — never all controls at once.

---

## Visual Hierarchy

- **Primary focal point:** the central image canvas. The opened image must dominate the window immediately.
- **Secondary focal point:** the bottom filmstrip with a clearly highlighted current item and visible adjacent context.
- **Tertiary focal point:** the persistent right metadata sidebar. It supports inspection without overtaking the image.
- Viewer chrome should feel like a desktop workspace, not a modal sheet, overlay, or mobile lightbox.
- The sidebar remains visually quieter than the image canvas, while the filmstrip remains more glanceable than the sidebar.

---

## Interaction Contract

### Window Behavior
- Double-clicking an image from gallery or search opens a separate non-blocking viewer window.
- Multiple viewer windows may exist simultaneously.
- Phase 18 does not persist viewer window size or position; each opens from a stable default desktop size and centered placement.
- Closing a viewer window affects only that viewer instance, not the main gallery shell or sibling viewer windows.

### Viewer Workspace
- Layout order: top viewer chrome (lightweight) → central image canvas + right metadata sidebar → bottom filmstrip.
- The central image canvas must remain dominant at standard desktop widths.
- The right metadata sidebar is visible by default and remains docked, not modal or overlay-based.
- Sidebar width target: `320px` default, with enough room for long file paths and tag wraps.
- The sidebar shows the extended metadata set: filename, format, resolution, size, path, imported time, and tags.

### Filmstrip
- The filmstrip follows the current result set context rather than folder-only scope, per Phase 18 decision D-03.
- Thumbnail density is medium: enough thumbnails for continuous scanning without becoming too small to identify.
- The selected thumbnail requires a clearly visible active state using border, surface contrast, and position context — not color alone.
- Thumbnail taps switch the main image and scroll the filmstrip enough to keep nearby context visible.

### Image Interaction
- Initial image state is fit-to-window.
- Double-click toggles fit-to-window ↔ `2x` zoom.
- Pan/drag is enabled while zoomed.
- Moving to a different image by filmstrip or keyboard resets the image state to fit-to-window.
- Loading and failure states use centered feedback and do not collapse the surrounding workspace layout.

### Keyboard Contract
- `←` and `→` switch to previous/next image within the viewer session.
- `Esc` closes only the current viewer window.
- Keyboard navigation should function without requiring focus inside the sidebar or filmstrip first.

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Viewer window title pattern | ACGWarehouse Viewer — {filename} |
| Filmstrip position label | {current} of {total} |
| Sidebar metadata heading | Image Details |
| Sidebar tags heading | Tags |
| Loading state | Loading image… |
| Missing image state | Image preview is unavailable. |
| Missing metadata fallback | Metadata unavailable |

Additional copy rules:
- Metadata labels should use concise desktop terminology: `Filename`, `Format`, `Resolution`, `Size`, `Path`, `Imported`, `Tags`.
- Do not introduce editing, deleting, sharing, or slideshow copy in this phase.
- Window titles should help distinguish multiple viewer windows without adding noisy status text.

---

## Accessibility Contract

- Filmstrip selection must not rely on accent color alone; use visible border and active-state contrast.
- Metadata labels and values must maintain readable contrast against the dark sidebar surface.
- Keyboard navigation must work for previous/next image and close-window flows.
- Thumbnails, sidebar fields, and any top-chrome controls require accessible labels or readable text equivalents.
- Loading and failure states must present text-first feedback, not icon-only feedback.

---

## Desktop-Specific Constraints

- Phase 18 targets true multi-window Windows desktop behavior for shipped execution, even if early implementation slices use route-based or surface-only harnesses for testability.
- The viewer shell should be container-agnostic enough to render under a test harness before native secondary-window bootstrap is wired.
- Existing `ExtendedImage` interaction semantics are the baseline unless implementation research proves a blocking limitation.

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

**Approval:** approved 2026-04-05
