# ACGWarehouse Vue Gallery Design System

## 1. Atmosphere & Identity

ACGWarehouse Vue Gallery feels like a warm community archive for anime artwork discovery and personal collection care. The signature is a Colorful reading room: a warm yellow canvas, white lifted panels, orange action accents, deep purple-black text, and restrained motion that helps forms and gallery states feel responsive without becoming decorative.

## 2. Color

### Palette

| Role | Token | Light | Usage |
| --- | --- | --- | --- |
| Surface/canvas | `--bg` | `#fff8d7` | Page background and warm community atmosphere |
| Surface/panel | `--surface` | `#ffffff` | Cards, forms, navigation, account panels |
| Surface/warm | `--surface-warm` | `#ffef9f` | Gentle illustration blocks and warm highlights |
| Text/primary | `--fg` | `#1d1836` | Headings, body, high-emphasis labels |
| Text/secondary | `--fg-2` | `#4c426c` | Supporting copy, secondary controls |
| Text/muted | `--muted` | `#796f91` | Helper text, metadata, empty-state descriptions |
| Accent/primary | `--accent` | `#ff6b00` | Primary buttons, focus accents, active account actions |
| Accent/on | `--accent-on` | `#ffffff` | Text over accent surfaces |
| Border/default | `--border` | `#eadfba` | Panel outlines and form controls |
| Border/subtle | `--border-soft` | `#f5eccd` | Internal row dividers and low-emphasis borders |
| Status/success | `--success` | `#2e9d57` | Saved, synced, password-success feedback |
| Status/warning | `--warn` | `#ffb020` | Cautionary helper states |
| Status/error | `--danger` | `#e5484d` | Validation and API errors |

### Rules

- Frontend code uses these CSS variables rather than introducing raw colors.
- Orange is reserved for primary action, active state, focus emphasis, and short status accents.
- Error and success states must be visible inline, not toast-only.

## 3. Typography

### Scale

| Level | Token | Size | Weight | Line Height | Usage |
| --- | --- | --- | --- | --- | --- |
| Display | `--text-4xl` | 76px max via clamp | 900 | `--leading-tight` | Large page identity moments |
| H1 | `--text-3xl` / clamp | 54px max | 900 | `--leading-tight` | Page titles |
| H2 | `--text-2xl` / clamp | 36px max | 800-900 | `--leading-tight` | Panel and section headings |
| H3 | `--text-xl` | 24px | 800-900 | `--leading-tight` | Account form group titles |
| Body | `--text-base` | 16px | 400-700 | `--leading-body` | Default copy |
| Body/sm | `--text-sm` | 14px | 700-900 for labels | 1.52 | Labels, nav, metadata |
| Caption | `--text-xs` | 12px | 700-900 | 1.4 | Helper text, inline errors |

### Font Stack

- Primary: `Inter, system-ui, sans-serif` via `--font-display` and `--font-body`.
- Mono: `"SF Mono", ui-monospace, Menlo, monospace` via `--font-mono` for status counts and compact metrics.

### Rules

- Labels are visible above fields; placeholder text never replaces a label.
- Helper text and validation errors sit below their field and are referenced by `aria-describedby`.
- Account page copy stays specific to profile, preferences, security, and collection sync.

## 4. Spacing & Layout

### Base Unit

The app uses a 4px variable scale while arranging account pages on an 8pt rhythm.

| Token | Value | Usage |
| --- | --- | --- |
| `--space-1` | 4px | Icon-to-label gaps, tiny offsets |
| `--space-2` | 8px | Compact inline groups |
| `--space-3` | 12px | Field gaps, tab padding |
| `--space-4` | 16px | Standard card/form spacing |
| `--space-5` | 20px | Panel header gaps |
| `--space-6` | 24px | Panel padding |
| `--space-8` | 32px | Section intro and grouped content gaps |
| `--space-12` | 48px | Hero and major vertical spacing |

### Grid

- Max content width: `--container-max` (`1180px`).
- Desktop account layout: `320px` profile rail plus flexible content column.
- Tablet and below collapse to a single column.
- Mobile forms keep full-width primary actions and at least `44px` control height.

### Rules

- Forms, profile panels, preferences, security, and recent activity use existing `.panel`, `.panel-raised`, `.stack`, `.form-grid`, `.field`, `.status`, and `.tag` patterns.
- No fixed-width account content; all panes must collapse cleanly below `744px`.

## 5. Components

### Account Tabs

- **Structure**: `.auth-tabs` with `role="tablist"`; each button has stable `id`, `role="tab"`, `aria-selected`, `aria-controls`, and roving `tabindex`.
- **States**: active tab uses dark `--fg` surface; focus uses `--focus-ring`.
- **Accessibility**: left/right and up/down arrows switch tabs and move focus to the selected tab.
- **Motion**: active panel uses a short opacity/translate entry that respects reduced motion.

### Account Forms

- **Structure**: semantic `<form>` with visible labels, helper text, field-level error text, inline status, and submit button.
- **Variants**: login, register, profile, preferences, password.
- **States**: default, loading/disabled, success, error; invalid inputs set `aria-invalid="true"`.
- **Accessibility**: submit status uses `role="status"` or `role="alert"` with `aria-live`.

### Preference Rows

- **Structure**: `.preference-row` with label/hint copy and a native checkbox styled by `.toggle`.
- **States**: checked uses `--accent`; focus uses `--focus-ring`.
- **Accessibility**: toggle has a real label and remains keyboard-operable.

### Header Account Action

- **Structure**: `RouterLink` to `/account` using existing button styles.
- **States**: logged out label is “登录”; logged in label prefers nickname, then username, then “我的”.
- **Accessibility**: current route behavior remains owned by navigation links; account action remains a link.

### Recent Activity Empty State

- **Structure**: `.activity-empty` with inline SVG icon, title, and description.
- **Rules**: no emoji icons; empty state explains that real activity appears after collection, rating, or sync actions.

## 6. Motion & Interaction

### Timing

| Type | Token | Usage |
| --- | --- | --- |
| Fast | `--motion-fast` (`150ms`) | Button, toggle, tab feedback |
| Base | `--motion-base` (`240ms`) | Panel hover, toast, form status |
| Account entry | `420ms` existing account motion | Intro and panel reveal |

### Rules

- Animate only `transform` and `opacity` for account panel entry/hover.
- Do not add looping or decorative account animations.
- Respect the existing `prefers-reduced-motion: reduce` block.
- Buttons, tabs, toggles, inputs, and links must expose hover/focus/disabled/loading states.

## 7. Depth & Surface

### Strategy

The Vue Gallery uses a mixed strategy: default panels use a subtle ring (`--elev-ring`), elevated account/sidebar panels use `--elev-raised`, and internal rows use `--border-soft` dividers.

| Level | Token | Usage |
| --- | --- | --- |
| Flat ring | `--elev-ring` | Standard panels, form cards, result cards |
| Raised | `--elev-raised` | Account sidebar, primary buttons, toast |
| Focus | `--focus-ring` | Keyboard focus and invalid-field emphasis |

### Rules

- Use `--radius-lg` for panels, `--radius-md` for inputs/buttons, and `--radius-pill` for compact badges.
- Do not add new shadow recipes for account UI; reuse the three depth tokens above.
- Status rows and badges must rely on semantic status colors plus borders, not decorative gradients.
