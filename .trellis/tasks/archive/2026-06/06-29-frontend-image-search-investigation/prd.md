# Investigate frontend image search

## Goal

Determine why the frontend image-search experience is not working: whether the product lacks a frontend search-image feature, whether the frontend is not wired to the existing backend search API, or whether an implemented flow has a bug.

## Requirements

- Inspect the Vue frontend search entry points, API client usage, routing, and image listing behavior.
- Inspect the backend search API contract only as needed to decide whether frontend behavior is missing or incorrectly integrated.
- Produce an evidence-backed classification: missing feature, frontend bug, backend/API bug, data/indexing issue, or ambiguous.
- Do not change implementation during planning unless the task is explicitly started for execution.

## Confirmed Facts

- `frontend/vue-gallery/src/api/client.ts:157` exports `searchImages(params: SearchQuery)`, maps `keyword` to the backend `q` parameter, and calls `/search` with `page` and `size`.
- `frontend/vue-gallery/src/pages/SearchPage.vue:20` implements `handleSearch()`, calls `searchImages({ keyword: keyword.value.trim() || undefined, limit: 20 })`, and renders returned image rows.
- `frontend/vue-gallery/src/router/index.ts:18` registers `/search` and `frontend/vue-gallery/src/components/AppHeader.vue:17` links the main nav item labeled `搜索` to that page.
- `frontend/vue-gallery/src/components/AppHeader.vue:44` also renders a top-nav input labeled `快速搜索`, but the input has no `v-model`, submit handler, route navigation, or API call. Typing there cannot trigger search.
- `internal/handler/image.go:58` handles backend search by reading query parameter `q` and calling `ImageService.Search`.
- `internal/service/image.go:137` calls the configured searcher, fetches active images by result IDs, and returns a list response.
- `internal/infra/search/bleve.go:103` executes the Bleve query and returns matched image IDs and total count.

## Current Classification

The search capability exists, so this is not a wholly missing feature. The most likely frontend defect is the header quick-search affordance: the UI suggests users can search from the top bar, but that input is currently non-functional. The dedicated `/search` page is implemented; if that page returns no results, the next likely category would be backend index/data quality rather than missing frontend feature.

## Recommended Next Step

If the user expects the top navigation input to work, fix `frontend/vue-gallery/src/components/AppHeader.vue` so entering a keyword routes to `/search` with a query parameter, then update `SearchPage.vue` to initialize from that route query and rerun search when the query changes. If the user only cares about the dedicated `/search` page returning empty results, verify the backend search index/data path before changing frontend UI.

## Implementation Result

- `frontend/vue-gallery/src/components/AppHeader.vue` now treats the top-nav quick search as a real search form. Submitting a non-empty keyword routes to `/search?q=<keyword>`; submitting an empty value routes to `/search`.
- `frontend/vue-gallery/src/pages/SearchPage.vue` now reads `route.query.q`, initializes the input from it, and automatically performs `/api/v1/search?q=<keyword>&size=20` for direct links or header submissions.
- The dedicated search-page button still supports manual search, including the prior empty-keyword behavior that requests all search results without fabricating unsupported filters.

## Validation Evidence

- `npm run build` in `frontend/vue-gallery` passed after the implementation (`vue-tsc -b && vite build`).
- Browser smoke with Vite preview: typing `miku` into the header quick-search field and pressing Enter navigated to `http://127.0.0.1:4173/search?q=miku`.
- Browser network evidence: the search page requested `GET http://127.0.0.1:4173/api/v1/search?q=miku&size=20` and received `{code:0,data:{total:0,page:1,size:20,list:[]},msg:""}`.
- Browser DOM evidence: header input, page input, and `data-search-summary` all reflected `miku`; console warning/error count was zero.
- TypeScript no-excuse script was attempted, but it reports `No TypeScript files found` for the changed `.vue` files; `vue-tsc` through `npm run build` is the applicable type gate in this project.
- Trellis Phase 2.2 `trellis-check` fixed one additional header synchronization edge case: the global header now watches both `route.path` and `route.query.q`, so entering `/search` without a `q` change does not keep stale quick-search text.
- Trellis Phase 3.3 updated `.trellis/spec/frontend/api-client.md` with the route-driven search contract and the forbidden static quick-search-input pattern.

## Test Coverage Note

- `frontend/vue-gallery/package.json` has no test script or Vue test dependencies. No automated regression test was added to avoid introducing a new test stack in this focused bug fix; the behavior is covered by build/type checking and browser smoke evidence above.

## Acceptance Criteria

- [x] The task documents whether frontend image search exists in code and where it is exposed to users.
- [x] The task documents the relevant frontend-to-backend request path, including file paths and endpoint names.
- [x] The task identifies the most likely failure category with code evidence.
- [x] If implementation is needed, the task has a clear next-step recommendation before Phase 2 starts.

## Notes

- User request: "前端搜索图片的功能未生效，是功能没有还是bug？"
- Repository README says the frontend is `frontend/vue-gallery` and backend image APIs include `GET /images`, `GET /images/:id`, and `GET /search` under `/api/v1`.
- Frontend spec index confirms `/api/v1/search` is an expected image endpoint and `/search` must be covered by SPA fallback rules.
