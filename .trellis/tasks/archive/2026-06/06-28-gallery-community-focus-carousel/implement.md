# Implement — Gallery community focus carousel redesign

## Preconditions

- Load context in this order before implementation: `implement.jsonl` entries, `prd.md`, `design.md`, this file.
- Use `trellis-before-dev` before editing frontend files.
- Use `frontend` / `redesign-existing-projects` guidance for visual changes.
- Do not run `task.py start` until the user approves these artifacts.

## Files

- Modify: `frontend/vue-gallery/src/pages/GalleryPage.vue`
- Modify: `frontend/vue-gallery/src/components/Carousel.vue`
- May modify: `frontend/vue-gallery/src/composables/useCarousel.ts`
- May modify: `frontend/vue-gallery/src/types/index.ts`
- Modify: `frontend/vue-gallery/src/assets/app.css`
- Read/verify: `DESIGN.md`, `.trellis/spec/frontend/index.md`, `.trellis/spec/frontend/api-client.md`

## Implementation Checklist

### 1. Weekly ranking data request

- [ ] In `GalleryPage.vue`, introduce named constants for community focus period, request size, and display size, e.g. weekly + a request buffer such as 20 + 10 displayed works.
- [ ] Change the ranking request to weekly rankings with at least 10 items.
- [ ] Filter non-displayable ranking entries before mapping to carousel slides; zero-size/dimension directory-like entries must not render as broken slides.
- [ ] Keep `getImages({ limit: 20 })` unchanged unless build/type checks require otherwise.
- [ ] Keep `rankingToSlide()` backed by `RankingResponse`; do not create mock slides.

### 2. Carousel state and markup

- [ ] In `Carousel.vue`, keep the carousel region semantics and ArrowLeft/ArrowRight keyboard handling.
- [ ] Replace the compact one-card + dot footer layout with an active feature area and a readable weekly navigation rail.
- [ ] Ensure previous/next buttons, direct rail controls, slide labels, and status text remain visible and accessible.
- [ ] If helper data is needed, add type-safe computed values in `Carousel.vue` or `useCarousel.ts`; avoid over-generalizing the composable.
- [ ] If `CarouselSlide` needs extra display fields, update `src/types/index.ts` and `rankingToSlide()` together using `import type` rules.

### 3. Carousel CSS

- [ ] Update only carousel-related styles in `frontend/vue-gallery/src/assets/app.css` unless responsive hero balance requires a small adjacent layout tweak.
- [ ] Use existing `DESIGN.md` tokens: colors, spacing, radius, focus rings, and motion variables.
- [ ] Build responsive behavior for desktop, tablet, and mobile: no horizontal page overflow; rail may scroll inside its own container.
- [ ] Preserve `prefers-reduced-motion` behavior and only animate transform/opacity/filter.

### 4. Loading, empty, and error states

- [ ] In `GalleryPage.vue`, distinguish loading/empty/error copy for the community focus panel where practical.
- [ ] Do not render static fallback slides when rankings are empty.
- [ ] Keep the page-level retry action functional.

## Validation Commands

- [ ] Run `npm run build` in `frontend/vue-gallery`; expected: exit 0 from `vue-tsc -b && vite build`.
- [ ] If backend is available, run the weekly rankings curl matching the implementation request size; expected: backend envelope with list data and enough displayable image entries for 10 slides when data exists.
- [ ] Launch the Vite app and visit the gallery route `/`; expected network request includes `/api/v1/rankings?period=week&size=<request limit>` through the Vite proxy.
- [ ] Browser QA at 375px, 768px, and 1280px; expected: carousel is readable, rail usable, no horizontal page overflow, focus states visible.
- [ ] Keyboard QA: focus carousel region, press ArrowRight and ArrowLeft; expected status and active slide update.

## Review Gates

- [ ] After edits, read all changed files and compare against `prd.md` acceptance criteria.
- [ ] Run `trellis-check` after implementation.
- [ ] Use visual QA before reporting completion because this is a UI redesign.
- [ ] Do not commit unless explicitly requested by the user during finish phase.

## Rollback Points

- After Task 1: reverting `GalleryPage.vue` restores daily 3-item request.
- After Task 2: reverting `Carousel.vue` restores old carousel markup while keeping data request changes separate.
- After Task 3: reverting carousel CSS restores old visual layout.
