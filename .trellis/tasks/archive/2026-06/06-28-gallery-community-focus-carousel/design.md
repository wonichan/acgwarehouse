# Design — Gallery community focus carousel redesign

## Scope

Modify the Vue gallery home page community focus area only:

- `frontend/vue-gallery/src/pages/GalleryPage.vue` for the weekly ranking request and state-specific carousel placement.
- `frontend/vue-gallery/src/components/Carousel.vue` for the carousel structure and accessible 10-item navigation.
- `frontend/vue-gallery/src/composables/useCarousel.ts` only if the redesigned navigation needs derived helpers beyond `currentIndex`, `next`, `prev`, and `goto`.
- `frontend/vue-gallery/src/types/index.ts` only if display metadata needs a type-safe field added.
- `frontend/vue-gallery/src/assets/app.css` for design-system-compliant layout, responsive behavior, and states.

Do not modify backend ranking behavior, the API client contract, or the gallery masonry feed unless build/type checks reveal a direct compile dependency.

## Data Flow

`GalleryPage.vue` remains the data boundary for this area:

1. On load, request images and rankings in parallel.
2. Change the ranking request from `getRankings({ period: 'day', limit: 3 })` to weekly rankings with a request limit above 10; requesting a buffer such as 20 is allowed so non-displayable backend entries can be skipped while still showing 10 real works.
3. Filter out non-displayable ranking entries at the page boundary before mapping: zero-size images, zero dimensions, empty URLs, or URLs ending in `/` must not become visible carousel slides.
4. Map remaining `RankingResponse` values through `rankingToSlide()` into `CarouselSlide[]` and display the first 10 valid slides.
5. Pass real slides to `<Carousel />` only when `carouselSlides.length > 0`.
6. Keep loading/error/empty states explicit; do not synthesize fake slides if `/rankings` returns fewer than 10 displayable items.

The backend API client already maps `limit` to `size`, so no API client change is planned.

## UX Structure

The current compact right-column card was designed for 3 slides and becomes weak with 10. The redesigned component should use a richer "feature + rail" pattern:

- A larger active feature panel with image, tag, title, description, heat score, favorites, and current position.
- Previous/next controls near the heading for linear browsing.
- A compact horizontal or wrapped navigation rail for 10 weekly items, using readable numbered/thumb labels instead of 10 tiny dots.
- The rail must remain keyboard and pointer friendly on mobile; allow horizontal overflow if needed instead of shrinking controls below readable size.
- `aria-live` status continues to announce the current slide.
- ArrowLeft and ArrowRight continue to work while the carousel region is focused.

Recommended desktop layout: keep the hero text in the left column, but make the carousel panel visually denser and more editorial: active artwork dominates the panel, metadata layers beneath/alongside it, and the 10-item rail acts as the weekly queue. At tablet/mobile widths, stack content and make the rail horizontally scrollable.

## Visual System

Follow root `DESIGN.md`:

- Use existing CSS variables for all colors, spacing, radius, motion, and focus rings.
- Preserve the warm community archive identity: `--bg`, `--surface`, `--surface-warm`, `--accent`, and raised panel shadows.
- Use transform/opacity transitions only; do not animate layout properties.
- Keep meaningful images as `<img>` with descriptive `alt` text from the slide title.
- New repeated component patterns, if any, should be reflected in `DESIGN.md` during implementation only if reused beyond this carousel.

## State Handling

- Loading: keep a raised panel placeholder that says weekly community focus is loading. It may include skeleton-like blocks using design tokens, but no mock artwork/text.
- Empty: if rankings return an empty list, show an intentional empty weekly-focus panel that points users to the trending page or explains that rankings will appear after activity accumulates.
- Error: existing page-level error handling may remain, but the carousel area must not silently disappear into blank space.

## Accessibility Contracts

- Carousel wrapper remains a named `role="region"` with `aria-roledescription="carousel"`.
- Each slide remains `role="group"`, `aria-roledescription="slide"`, and a positional label.
- Non-active slides are not keyboard traps.
- Direct navigation controls use semantic `<button>` elements with clear `aria-label` text.
- Focus-visible styles are preserved through `:focus-visible` / `--focus-ring`.
- The status text uses `aria-live="polite"`.

## Compatibility and Rollback

The change is local to the Vue frontend. Rollback is straightforward:

- Revert `GalleryPage.vue` ranking request and carousel state copy.
- Revert `Carousel.vue` markup.
- Revert carousel-related CSS in `app.css`.
- Revert any optional `useCarousel.ts` / type changes.

No data migration or backend deployment ordering is required.

## Risks

- The backend may return fewer than 10 displayable weekly rankings in a sparse environment. The UI must handle the actual returned count while requesting at least 10 and skipping non-displayable entries.
- Ten direct controls can crowd small screens. Use a rail or numbered chips instead of a simple dot row.
- Changing the hero right column can affect first viewport balance. Browser QA at 375px, 768px, and 1280px is required.
