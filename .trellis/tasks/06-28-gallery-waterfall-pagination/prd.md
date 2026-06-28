# Fix frontend gallery waterfall pagination

## Goal

Fix the frontend gallery page so the waterfall/masonry layout automatically continues loading additional images when the user scrolls to the bottom, and pagination query parameters are applied correctly.

## User-Reported Problem

- On the frontend gallery interface, the waterfall layout reaches the bottom and then stops.
- Pagination query does not appear to take effect.

## Confirmed Facts

- Trellis task created for this bug: `.trellis/tasks/06-28-gallery-waterfall-pagination`.
- Gallery frontend route is `frontend/vue-gallery/src/pages/GalleryPage.vue`.
- Gallery rendering uses `ArtCard` in a masonry container; the CSS waterfall layout is `.masonry { columns: 4 240px; }` in `frontend/vue-gallery/src/assets/app.css`.
- Initial gallery load calls `getImages({ limit: 20 })` in `GalleryPage.vue:83-85`; no `page` parameter is passed, so the backend receives the default `page=1`.
- `GalleryPage.vue:88-90` assigns `artItems.value = imagesData.items.map(imageToArtItem)`, which replaces the list during `loadGallery()` rather than appending a next page.
- `GalleryPage.vue` currently has no pagination state such as current page, has-more, or loading-more.
- `GalleryPage.vue` currently has no infinite-scroll trigger: no `IntersectionObserver`, scroll listener, bottom sentinel, or load-more control.
- `handleFilter()` currently calls `loadGallery()` again and is annotated as future sort behavior in `GalleryPage.vue:102-106`.
- Frontend API client `getImages()` accepts `page`/`limit`/`sort`/`order` query options and appends them to `/images` when provided.
- Backend image listing is pagination-ready: `ImageHandler.List()` parses `page`/`size`, the service receives the query, and the repository applies SQL `LIMIT`/`OFFSET`.
- User decision: use automatic infinite scroll rather than a manual “load more” button.
- Root cause: frontend gallery pagination/infinite-scroll behavior is missing; backend pagination does not need to change for this bug based on current evidence.

## Requirements

- Add frontend pagination state for the gallery page, including current page, total/has-more, and loading-more guard.
- Initial gallery load must request page 1 with the expected page-size parameter.
- When more data is available, reaching the bottom of the masonry list must automatically request the next page with a `page` parameter.
- Newly fetched pages must append to `artItems` instead of replacing already rendered items.
- Prevent duplicate concurrent next-page requests for the same page.
- Preserve existing gallery behavior outside pagination/loading, including carousel ranking loading and existing filter labels.
- Fix the root cause without visual redesign, backend contract changes, or broad data-fetching refactors.

## Acceptance Criteria

- [ ] Scrolling the gallery waterfall layout to the bottom automatically requests the next page when more data is available.
- [ ] The outgoing gallery request includes the expected pagination parameters for subsequent pages.
- [ ] Newly fetched gallery items append to the existing waterfall list instead of replacing it unexpectedly or stopping prematurely.
- [ ] The UI does not issue duplicate concurrent next-page requests for the same page.
- [ ] The UI stops requesting more pages when the API response indicates all items have been loaded.
- [ ] Relevant frontend checks/tests pass, or any unrelated pre-existing failures are documented.

## Out of Scope

- Visual redesign of the gallery.
- Backend API contract changes unless later verification proves the frontend sends correct pagination and the backend is the failing layer.
- Broad data-fetching refactors unrelated to this pagination defect.
- New sorting/filtering semantics beyond preserving the existing filter interaction.

## Open Questions

- None.
