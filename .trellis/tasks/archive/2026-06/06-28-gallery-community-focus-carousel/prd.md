# Redesign gallery community focus carousel

## Goal

Redesign the gallery home page's "本周社区焦点" carousel so the community focus area feels intentional, image-led, and usable with at least 10 real backend ranking items instead of the current compact 3-item carousel.

User value: visitors should see a richer community discovery surface on the gallery page, with enough items to feel like a weekly focus collection rather than a small teaser.

## Confirmed Facts

- Frontend package: `frontend/vue-gallery`, using TypeScript + Vue 3 + Vite (`frontend/vue-gallery/package.json`).
- Gallery route component: `frontend/vue-gallery/src/pages/GalleryPage.vue`.
- Existing carousel component: `frontend/vue-gallery/src/components/Carousel.vue`.
- Existing carousel state composable: `frontend/vue-gallery/src/composables/useCarousel.ts`.
- Existing carousel display type: `frontend/vue-gallery/src/types/index.ts` `CarouselSlide`.
- Current slide count is capped by `GalleryPage.vue:75`: `getRankings({ period: 'day', limit: 3 })`.
- `getRankings()` maps `limit` to backend query param `size` and calls `/api/v1/rankings` (`frontend/vue-gallery/src/api/client.ts:147`).
- Product decision: "本周社区焦点" must use weekly ranking data (`period: 'week'`) rather than the current daily data.
- Live weekly ranking data can include non-displayable directory entries such as `filename=thumbnails`, `size=0`, `width=0`, `height=0`, and a URL ending in `/`; these must not become visible carousel slides.
- Existing design system is root `DESIGN.md`; UI changes must preserve the warm community archive palette, existing tokens, focus states, and transform/opacity motion rules.
- Frontend API spec requires real backend data only: empty backend ranking lists must show an intentional empty/loading state and must not fall back to mock carousel slides.
- Current worktree already has unrelated untracked paths: `.env`, `bin/`, and `frontend/example/`; this task must not modify or clean them.

## Requirements

- R1: The gallery page must request at least 10 weekly ranking items for the community focus carousel.
- R2: The redesigned carousel must stay on the gallery home hero/community focus area and continue to use real backend ranking data with non-displayable image entries filtered out.
- R3: The carousel layout must be redesigned to work well with 10+ slides; it should not simply add more dots to the existing cramped 3-slide card layout.
- R4: Navigation must remain accessible: previous/next controls, direct slide selection or an equivalent overview control, keyboard support, visible focus states, and `aria-live` status.
- R5: The design must use tokens/patterns from `DESIGN.md` and `frontend/vue-gallery/src/assets/app.css`; no ad-hoc palette or framework migration.
- R6: Loading, empty, and error states must remain intentional and must not display mock community focus data.
- R7: The implementation must remain Vue 3 + TypeScript + Vite with existing dependencies unless the user explicitly approves a dependency addition.
- R8: This task must not add a ranking-period selector; the carousel is a weekly-focus component, not a general ranking filter.

## Acceptance Criteria

- [ ] `GalleryPage.vue` requests rankings with `period: 'week'` and `limit >= 10` for the community focus carousel.
- [ ] The carousel renders 10 slides when the backend returns at least 10 displayable ranking items.
- [ ] Directory-like or zero-size ranking entries do not render as broken carousel images or titles such as `thumbnails`.
- [ ] The redesigned layout is visually more balanced than the current right-column compact carousel at desktop width and remains usable at mobile width.
- [ ] Previous/next controls, keyboard ArrowLeft/ArrowRight navigation, visible focus, and current-slide status still work.
- [ ] Direct slide selection or an equivalent 10-item navigation pattern works without producing an unreadable row of cramped dots on small screens.
- [ ] Empty/loading/error states do not use mock slides and communicate what is happening.
- [ ] `npm run build` passes in `frontend/vue-gallery`.
- [ ] Browser QA covers at least mobile 375px, tablet 768px, and desktop 1280px on the gallery page.

## Out of Scope

- Backend ranking algorithm changes.
- Adding a day/week/month period switcher to the carousel.
- Replacing the whole gallery page or gallery masonry feed.
- Adding fake or static fallback carousel content.
- Cleaning unrelated untracked worktree paths.
