# Use Backend API Data In Frontend Implementation Plan

> **For agentic workers:** REQUIRED: Use `trellis-before-dev` before editing, then use `trellis-check` after each implementation pass. Keep all frontend requests on `/api/v1/*`; do not hardcode `localhost:2018` in source.

**Goal:** Replace remaining frontend mock/hard-coded dynamic data with real backend API responses from the local service behind Vite proxy.

**Architecture:** Treat `src/api/client.ts` as the contract boundary. Page components consume typed client methods and display mappers; they do not know backend port, raw envelope shape, or unsupported query params.

**Tech Stack:** Vue 3, TypeScript, Vite, existing fetch-based API client, local Go backend on port `2018`.

---

## Pre-Start Gates

- [x] User confirmed scoring display decision: backend percent scale (`0-100`, e.g. `62.3/100`).
- [ ] Read `.trellis/spec/frontend/api-client.md` before editing.
- [ ] Read `.trellis/spec/frontend/index.md` before editing.
- [ ] Confirm local backend still responds: `curl -sS -m 5 'http://localhost:2018/api/v1/images?size=1'`.
- [ ] Confirm frontend dependencies are installed or install only if missing using the project package manager.

## Task 1: Align API Client With Backend Contract

**Files:**
- Modify: `frontend/vue-gallery/src/api/client.ts`

- [ ] Update `ApiResponse<T>` to use `msg`, not `message`.
- [ ] Update `apiCall` / unwrap behavior so `code !== 0` throws `ApiError` with `msg`.
- [ ] Update error parsing to read `msg`, not only `message`.
- [ ] Update `ImageDetailResponse` to match backend `{image,tags,avg_score,rating_count,favorite_count,my_rating,is_collected,similar_images}`.
- [ ] Update `TagResponse` to `{id,name,usage_count,created_at,updated_at}`.
- [ ] Update `CollectionResponse` / `CollectionDetailResponse` to backend `{id,user_id,name,visibility,created_at,items}` and item `{collection_id,image_id,created_at}`.
- [ ] Update `ImageQuery` and `getImages()` so `limit` maps to query `size`; support `filename/sort/order`; remove unsupported `category` emission.
- [ ] Update `SearchQuery` and `searchImages()` so `keyword` maps to `q` and `limit` maps to `size`; do not send unsupported `tags/min_score`.
- [ ] Update `suggestTags()` to send `q`, not `prefix`.
- [ ] Update `getRankings()` to accept `{period?: 'day'|'week'|'month', page?: number, limit?: number}` and send `period/page/size`.
- [ ] Update `createCollection()` to send `{name, visibility}` with default `private`.
- [ ] Run `npm run build` in `frontend/vue-gallery`; fix TypeScript errors caused by API type changes.

## Task 2: Replace TrendingPage Hardcoded Rankings

**Files:**
- Modify: `frontend/vue-gallery/src/pages/TrendingPage.vue`
- May modify: `frontend/vue-gallery/src/types/index.ts` if obsolete display types need removal or adjustment

- [ ] Import `getRankings` and `ApiError` from API client; import ranking types with `import type`.
- [ ] Add reactive `rankings`, `loading`, `error` state.
- [ ] Map UI periods to backend periods: `daily -> day`, `weekly -> week`, `monthly -> month`.
- [ ] Fetch rankings on mount and whenever the period changes.
- [ ] Render real ranking rows with image filename/title, rank, score, favorite/view/rating metadata.
- [ ] Render empty state when backend returns no rankings.
- [ ] Render retryable error state on `ApiError` or network failure.
- [ ] Run `npm run build` in `frontend/vue-gallery`.

## Task 3: Replace DetailPage Static Image Data

**Files:**
- Modify: `frontend/vue-gallery/src/pages/DetailPage.vue`

- [ ] Import `useRoute` from `vue-router` and `getImage`, `rateImage`, `ApiError` from API client.
- [ ] Read numeric image ID from route query `id`.
- [ ] Show clear empty/error state when no valid ID exists instead of rendering a fixed sample.
- [ ] Fetch image detail on mount and whenever query ID changes.
- [ ] Render `detail.image` filename, image URL, size, dimensions, tags, average score, rating/favorite/view counts, and `similar_images`.
- [ ] Replace static viewer art with the actual image URL while preserving existing zoom controls.
- [ ] Implement rating save only if logged in; use confirmed scoring scale and send backend `0-100` integer.
- [ ] Do not fake favorite/download/tag success; if collection selection or download endpoint is missing, show an honest unavailable/login-required message.
- [ ] Run `npm run build` in `frontend/vue-gallery`.

## Task 4: Replace CollectionsPage Mock Data

**Files:**
- Modify: `frontend/vue-gallery/src/pages/CollectionsPage.vue`
- May modify: `frontend/vue-gallery/src/types/index.ts`

- [ ] Import `useAuth`, `getCollections`, `createCollection`, and `ApiError`.
- [ ] Remove `Mock albums` and `Mock collection items` arrays.
- [ ] Add `collections`, `loading`, `error` state.
- [ ] If not logged in, render a login-required panel and do not call `/collections`.
- [ ] If logged in, fetch `/collections` and render real collection metadata.
- [ ] Compute collection count from `items.length` because backend does not return `item_count`.
- [ ] Wire create form to `createCollection(albumName, 'private')`, then refresh collection list.
- [ ] Remove or neutralize mock masonry items; do not show fake collection images.
- [ ] Run `npm run build` in `frontend/vue-gallery`.

## Task 5: Verify Existing API Pages After Client Fix

**Files:**
- Inspect/modify if needed: `frontend/vue-gallery/src/pages/GalleryPage.vue`
- Inspect/modify if needed: `frontend/vue-gallery/src/pages/SearchPage.vue`

- [ ] Confirm `GalleryPage` still compiles with updated `getImages` and `getRankings` signatures.
- [ ] Confirm `SearchPage` sends backend-supported search query `q` via `searchImages`.
- [ ] Remove or clarify unsupported score/tag filters if they are no longer backed by backend query params.
- [ ] Ensure router links from cards include real `id`, e.g. `/detail?id=<id>`.
- [ ] Replace “查看示例详情” with a route that uses a real selected/latest item when possible.
- [ ] Run `npm run build` in `frontend/vue-gallery`.

## Task 6: Runtime Verification

**Commands:**
- `curl -sS -m 5 'http://localhost:2018/api/v1/images?size=2'`
- `curl -sS -m 5 'http://localhost:2018/api/v1/rankings?period=day&size=3'`
- `curl -sS -m 5 'http://localhost:2018/api/v1/search?q=682325&size=2'`
- `curl -sS -m 5 'http://localhost:2018/api/v1/tags'`
- `npm run build` from `frontend/vue-gallery`

- [ ] Start Vite dev server for browser verification.
- [ ] Visit gallery page and confirm network requests use `/api/v1/images` and `/api/v1/rankings`.
- [ ] Visit trending page and confirm real ranking data changes by period.
- [ ] Visit detail page with a known ID, e.g. `/detail?id=5149`, and confirm real image detail renders.
- [ ] Visit collections page while logged out and confirm login-required state, not mock albums.
- [ ] If user approves creating/logging in with a test user, verify collection creation and authenticated collection list.

## Rollback Points

- After Task 1: API client contract changes can be reverted independently if build breaks broadly.
- After Task 2: Trending page is independent and can be reverted without affecting gallery/search.
- After Task 3: Detail page is independent except shared client types.
- After Task 4: Collections page authenticated flow is independent except shared collection client types.

## Known Non-Goals

- Do not add backend endpoints.
- Do not introduce mock fallback data.
- Do not create or mutate real backend users/collections without explicit user approval.
- Do not redesign visuals or change layout except where needed for loading/error/empty states.
