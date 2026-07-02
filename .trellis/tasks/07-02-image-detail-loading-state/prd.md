# Improve image detail loading state

## Goal

When a user opens an image detail page while image data is still loading, the page should present a polished, user-facing loading state instead of exposing backend/API implementation details.

## User Value

Users stay oriented while the detail page loads, and the product feels finished rather than showing developer/debug text such as `正在通过 /api/v1/images/3947 获取图片、标签、评分和相似推荐`.

## Confirmed Facts

- The frontend is a Vue 3 + TypeScript + Vite app under `frontend/vue-gallery`.
- The affected page is `frontend/vue-gallery/src/pages/DetailPage.vue`.
- `DetailPage.vue:199-204` currently renders the loading state for valid image IDs.
- `DetailPage.vue:201-203` currently shows:
  - eyebrow: `正在加载`
  - heading: `读取真实作品详情`
  - lead: `正在通过 /api/v1/images/{{ imageId }} 获取图片、标签、评分和相似推荐。`
- `DetailPage.vue:61-84` controls the loading lifecycle around `getImage(imageId.value)`.
- `DetailPage.vue:207-217` has the existing error and retry UI that should remain functionally intact.
- The successful detail viewer begins at `DetailPage.vue:219`, with the main image inside `.viewer-art` at `DetailPage.vue:221-223`.
- Project dependency inspection found no suitable UI component library for this task; `frontend/vue-gallery/package.json` only has `vue` and `vue-router` as runtime dependencies.
- Component inspection found no existing Skeleton/Spinner/Loading component; the nearest existing loading styles are button/status variants, not an image-area loader.
- Existing CSS already has design tokens, motion tokens, and `prefers-reduced-motion` patterns, so a local CSS implementation fits the project style.
- `.viewer-art` already has decorative placeholder styling via pseudo-elements and should be reused for the image loading animation where practical.

## Requirements

- Remove user-visible API route/loading-debug copy from the image detail page loading state.
- Replace the current plain loading panel with an animation-only visual loading state that includes an image-area skeleton or loader.
- Do not show visible loading text in the normal loading state; loading status may remain accessible to assistive technology if needed.
- Preserve existing image detail success, error, and retry flows.
- Match existing Vue gallery styling and CSS token usage; use local CSS animation rather than a UI-library component because no suitable project dependency/component exists.
- Use GPU-friendly animation (`opacity`, `transform`, or gradient-position`) and respect reduced-motion preferences.

## Acceptance Criteria

- [ ] Entering `/detail?id=<id>` while loading no longer displays `/api/v1/...` or internal API-fetch wording.
- [ ] The normal loading state shows no visible explanatory loading text; the loading state is communicated visually through animation/skeleton.
- [ ] The loading state reserves a credible main-image viewing area instead of showing only a text panel.
- [ ] The main image loading placeholder has a subtle animation in normal motion settings.
- [ ] Users with reduced-motion preference see a stable placeholder without continuous shimmer/motion.
- [ ] Existing image detail success, error, and retry flows still work.
- [ ] Frontend type/build validation covering `frontend/vue-gallery` passes.

## Out of Scope

- Backend API changes.
- Changing image detail data contracts.
- Redesigning unrelated pages that also expose API-route loading text.
- Adding new dependencies or UI libraries.
- Changing the route shape (`/detail?id=<id>`).

## Planning Notes

This is lightweight and can remain PRD-only. Implementation should edit `DetailPage.vue` directly after task activation, replacing the text loading panel with an animation-only layout that mirrors the success-state viewer grid. Use local CSS built from existing design/motion tokens and `.viewer-art` placeholder styling; do not add a UI library or new dependency. Then run targeted diagnostics and the frontend build/typecheck path.
