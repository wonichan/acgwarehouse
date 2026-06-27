# ACG Warehouse Design System

## 1. Atmosphere & Identity

ACG Warehouse feels like a warm community archive: playful enough for illustration discovery, structured enough for sorting, ranking, search, and collection workflows. The signature is warm editorial surfaces with orange interaction accents, rounded panels, and image-first cards that keep metadata secondary to visual browsing.

## 2. Color

### Palette

| Role | Token | Value | Usage |
|------|-------|-------|-------|
| Background | `--bg` | `#fff8d7` | Page background |
| Surface | `--surface` | `#ffffff` | Panels, cards, controls |
| Warm surface | `--surface-warm` | `#ffef9f` | Decorative gradients and image placeholders |
| Text primary | `--fg` | `#1d1836` | Headlines, primary text, active dark chips |
| Text secondary | `--fg-2` | `#4c426c` | Secondary copy and inactive navigation |
| Text muted | `--muted` | `#796f91` | Metadata and helper copy |
| Meta/accent | `--meta` | `#ff6b00` | Metadata highlight, legacy accent alias |
| Border | `--border` | `#eadfba` | Panels, cards, inputs, dividers |
| Border soft | `--border-soft` | `#f5eccd` | Subtle separations |
| Accent | `--accent` | `#ff6b00` | Primary actions, active controls, section labels |
| Accent text | `--accent-on` | `#ffffff` | Text on accent surfaces |
| Success | `--success` | `#2e9d57` | Success status |
| Warning | `--warn` | `#ffb020` | Warning status |
| Danger | `--danger` | `#e5484d` | Error and destructive status |

### Rules

- Keep interactive emphasis orange via `--accent`.
- Use `color-mix()` only from declared tokens.
- Do not introduce raw colors outside this document and `src/assets/app.css` token declarations.

## 3. Typography

### Scale

| Level | Token | Value | Usage |
|-------|-------|-------|-------|
| Caption | `--text-xs` | `12px` | Metadata, helper text, overline labels |
| Small | `--text-sm` | `14px` | Secondary body, controls, tags |
| Body | `--text-base` | `16px` | Default text |
| Lead | `--text-lg` | `18px` | Lead paragraphs |
| Card title | `--text-xl` | `24px` | Cards and small section titles |
| Section title | `--text-2xl` | `36px` | Section headers and avatars |
| Page title | `--text-3xl` | `54px` | Large headings |
| Display | `--text-4xl` | `76px` | Hero display headings |

### Font Stack

- Display: `Inter, system-ui, sans-serif`
- Body: `Inter, system-ui, sans-serif`
- Mono: `"SF Mono", ui-monospace, Menlo, monospace`

### Rules

- Headings use `--font-display`, `--tracking-display`, and `--leading-tight`.
- Body and metadata use `--font-body` with `--leading-body`.
- Numeric ranking/count values may use `--font-mono` and tabular numerals.

## 4. Spacing & Layout

### Base Unit

All spacing derives from a 4px base.

| Token | Value | Usage |
|-------|-------|-------|
| `--space-1` | `4px` | Tight inline gaps |
| `--space-2` | `8px` | Control groups, tags |
| `--space-3` | `12px` | Compact padding, form gaps |
| `--space-4` | `16px` | Standard card/form spacing |
| `--space-5` | `20px` | Panel inner rhythm |
| `--space-6` | `24px` | Panel padding, grid gaps |
| `--space-8` | `32px` | Hero actions and larger groups |
| `--space-12` | `48px` | Hero top spacing |
| `--section-y-phone` | `48px` | Mobile section vertical padding |
| `--section-y-tablet` | `68px` | Tablet hero lower padding |
| `--section-y-desktop` | `96px` | Desktop section vertical padding |

### Grid

- Max content width: `--container-max` (`1180px`).
- Gutters: desktop `36px`, tablet `24px`, phone `16px`.
- Main two-column layouts collapse at `1180px`; dense card grids collapse to one column below `744px`.

### Rules

- Prefer existing utility classes (`.container`, `.section`, `.grid-main`, `.stack`, `.row`, `.row-between`) over one-off layout CSS.
- New spacing should use existing tokens or multiples of 4px.

## 5. Components

### Button

- Structure: `.btn` plus variant `.btn-primary`, `.btn-secondary`, `.btn-ghost`, and optional `.btn-small`.
- States: hover lifts with `translateY(-1px)`, active resets transform, disabled uses opacity and `cursor: not-allowed`.
- Accessibility: preserve semantic `<button>` or `<a>`/`RouterLink` based on action vs navigation.

### Panel / Card

- Structure: `.panel` or `.card`, optional `.panel-raised`.
- Spacing: `.panel` uses `--space-6`, reduced to `--space-4` at very small screens.
- Usage: loading, error, empty, and data states should use panels rather than unstyled text.

### Art Card

- Structure: `.art-card` with `.art-preview` and `.art-body`.
- Variants: `default`, `tall`, `wide` preview variants.
- Data: real image URLs render with `<img>`; placeholders only for missing backend URLs, not as mock data fallback.

### Tags / Pills

- Structure: `.tag` or `.pill`; active/hot uses `.is-hot` or `.is-active`.
- Usage: metadata chips, filters, score/count summaries.

### Status

- Structure: `.status`, `.status--loading`, `.status--success`, `.status--error`, or panel-based equivalents.
- Usage: API loading, error, empty, login-required states.

## 6. Motion & Interaction

### Timing

| Token | Value | Usage |
|-------|-------|-------|
| `--motion-fast` | `150ms` | Hover, active, control changes |
| `--motion-base` | `240ms` | Cards, carousel, panels |
| `--ease-standard` | `cubic-bezier(0.2, 0, 0, 1)` | Standard easing |

### Rules

- Animate `transform` and `opacity`; avoid layout animation.
- Preserve `prefers-reduced-motion` behavior from `app.css`.
- Every interactive element must keep hover/focus/disabled states.

## 7. Depth & Surface

### Strategy

The project uses mixed border-and-shadow depth: most panels are bordered and ringed, while key cards/actions use warm raised shadows.

| Token | Value | Usage |
|-------|-------|-------|
| `--elev-flat` | `none` | Flat surfaces |
| `--elev-ring` | `0 0 0 1px var(--border)` | Default panel/card outline |
| `--elev-raised` | `0 18px 44px color-mix(in oklab, var(--fg), transparent 86%)` | Primary buttons, raised panels, hover cards |
| `--focus-ring` | `0 0 0 4px color-mix(in oklab, var(--accent), transparent 74%)` | Focus and selected states |

### Rules

- Use `.panel-raised` sparingly for focal surfaces.
- Preserve visible focus rings when adding buttons, links, inputs, or toggles.
