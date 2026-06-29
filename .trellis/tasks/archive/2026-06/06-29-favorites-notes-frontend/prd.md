# Expose favorites and tag management frontend

## Goal

Complete the Vue frontend exposure for backend-supported collection/favorite and tag-management workflows so users can create and use collections, add images to collections, and manage image tags from the existing gallery UI instead of seeing placeholder toasts.

## Corrected Scope

- The requested second feature is **tag management**, not notes/memos.
- The task directory slug still contains `notes` because it was created before this correction; all planning and implementation artifacts use the corrected `favorites + tags` scope.

## Confirmed Facts

- Frontend project: `frontend/vue-gallery`, using TypeScript + Vue 3 + Vite.
- Existing original prototype/design plan included six SPA routes, including `/detail` for image detail with tag/rating/favorite actions and `/collections` for album creation and batch organization.
- Frontend API calls must go through `frontend/vue-gallery/src/api/client.ts` and relative `/api/v1/*` paths; Vite proxy owns the local backend port.
- Collection backend routes exist: `GET/POST /api/v1/collections`, `GET/PUT/DELETE /api/v1/collections/:id`, `POST /api/v1/collections/:id/items`, and `DELETE /api/v1/collections/:id/items/:imageId`.
- Tag backend routes exist: `GET /api/v1/tags`, `GET /api/v1/tags/suggest`, authenticated `POST /api/v1/tags`, admin `PUT/DELETE /api/v1/tags/:id`, and authenticated batch assignment routes `POST /api/v1/images/tags` / `DELETE /api/v1/images/tags`.
- Current frontend client already has `getTags()` and `suggestTags()`, but does not expose tag create, batch assign, or batch unassign helpers.
- `DetailPage.vue` currently shows placeholder toasts for `handleFavorite()` and `handleTagging()` instead of calling collection/tag APIs.
- `BatchPanel.vue` currently shows success-like placeholder toasts for “加入收藏夹” and “批量打标签”, without executing real backend operations.
- `useSelection()` stores selected image IDs as strings; selected gallery IDs can be parsed into backend `image_ids` for batch collection/tag workflows.
- `CollectionsPage.vue` already reads and creates real collections, but only creates private collections and only displays collection metadata/item counts. It does not expose collection detail, visibility editing, deletion, item removal, or image-card display from collection items.
- `CollectionDetailResponse.items` contains `{collection_id,image_id,created_at}` only, not full image cards; full image display requires fetching image detail/list data by image IDs or deliberately staying metadata-only.
- `frontend/vue-gallery/DESIGN.md` exists and defines the active Colorful Vue Gallery design system. UI work must reuse its CSS tokens/components and apply taste-skill polish within that system.

## Decisions

- Tag management scope is ordinary user image tagging and untagging, plus creating a new tag inside the tagging flow when needed.
- Admin-only tag rename/delete is out of scope for this task and should be split into a separate admin surface if needed later.

## Requirements

- R1. Replace placeholder success toasts for favorite/tag actions with real workflows or honest disabled/unavailable states when backend support is insufficient.
- R2. Expose collection selection from image detail so a logged-in user can add the current image to one of their real collections through `/collections/:id/items`.
- R3. Collection management in this task is limited to real collection list/create, visibility choice during create, and adding selected/detail images to an existing or newly created collection.
- R4. Expose ordinary-user tag management using existing backend tag APIs: list/suggest tags, create a tag when needed, and apply/remove tags on selected images or the detail image.
- R5. Preserve auth boundaries: unauthenticated users see login-required states for collection mutation and tag assignment; admin-only tag rename/delete must not be exposed to ordinary users as if it will work.
- R6. Preserve existing TypeScript/API-client contracts: type imports use `import type`, no `as any`, no `@ts-ignore`, no fake localStorage persistence for backend-owned state.
- R7. Use the existing Colorful/gallery visual language plus the requested taste-skill polish; do not switch to an unrelated visual system or dark SaaS redesign.
- R8. Keep loading, empty, error, login-required, and retry states visible for every new API-backed panel or modal.

## Acceptance Criteria

- [ ] Detail page no longer shows “收藏夹选择流程尚未接入” for logged-in users; it loads real collections and can add the current image to a selected collection.
- [ ] Detail page no longer shows “个人标签保存接口尚未接入” for logged-in users; it exposes a real tag workflow aligned to backend capabilities.
- [ ] Batch panel no longer emits fake success for collection/tag actions; selected images can enter real collection/tag workflows or show a precise login/backend-limit reason.
- [ ] API client exposes typed helpers for every backend route used by the new frontend workflows.
- [ ] Users can create a new tag from the tag workflow and apply it to the target image(s) without leaving the current context.
- [ ] Users can remove selected existing tags from the detail image or selected batch target set when backend assignment state allows it.
- [ ] Collection UI uses real collection data only; no mock album or mock image-card fallback is introduced.
- [ ] Tag UI uses real tag data from `/tags` or `/tags/suggest`; no hard-coded tag catalog is introduced except transient input placeholders.
- [ ] Auth-required, loading, empty, API error, and success states are visible and accessible.
- [ ] UI additions follow the existing design language and pass the project frontend build/type checks.
- [ ] Browser-level verification covers at least detail page favorite/tag flows, collections page, and batch panel states.

## Out of Scope

- Adding backend endpoints or changing backend authorization rules unless planning discovers a blocking contract bug.
- Implementing a full admin tag-management console unless explicitly approved; current backend rename/delete routes are admin-only.
- Collection rename/delete UI.
- Collection item removal UI.
- Faking collection item image cards from static data when only `image_id` is available.
- Replacing the entire visual design system or restyling unrelated pages.
