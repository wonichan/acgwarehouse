# Favorites and Tag Management Frontend Design

## Scope

Expose backend-supported collection/favorite and ordinary-user tag-management workflows in `frontend/vue-gallery`. Replace placeholder toasts in image detail and batch selection flows with real API-backed interactions while preserving the existing Colorful Vue Gallery design system.

## Boundaries

- Frontend-only implementation under `frontend/vue-gallery/src` plus task/spec artifacts.
- Backend routes and authorization remain unchanged.
- Admin tag rename/delete is out of scope.
- Collection item image cards are not faked. If collection item images are displayed, they must be fetched from backend image endpoints by ID.
- UI uses `frontend/vue-gallery/DESIGN.md` tokens and existing classes (`panel`, `panel-raised`, `form-grid`, `field`, `tag`, `status`, `btn`) rather than introducing a new design system.

## Backend Contracts

### Collections

- `GET /api/v1/collections` returns the current user's collections and requires auth.
- `POST /api/v1/collections` body: `{name, visibility}`.
- `GET /api/v1/collections/:id` returns collection metadata plus `items` containing `image_id` rows.
- `PUT /api/v1/collections/:id` body: `{name, visibility}`.
- `DELETE /api/v1/collections/:id` deletes a collection.
- `POST /api/v1/collections/:id/items` body: `{image_id}`.
- `DELETE /api/v1/collections/:id/items/:imageId` removes an image from a collection.

### Tags

- `GET /api/v1/tags` lists tags.
- `GET /api/v1/tags/suggest?q=&limit=` suggests tags.
- `POST /api/v1/tags` body: `{name}` and requires auth.
- `POST /api/v1/images/tags` body: `{image_ids, tag_ids}` and requires auth.
- `DELETE /api/v1/images/tags` body: `{image_ids, tag_ids}` and requires auth.
- `PUT/DELETE /api/v1/tags/:id` are admin-only and not exposed in this task.

## API Client Changes

Add typed helpers in `src/api/client.ts` and corresponding types in `src/api/types.ts`:

- `createTag(name): Promise<TagResponse>`.
- `assignTagsToImages(imageIds, tagIds): Promise<readonly ImageTagResponse[]>`.
- `unassignTagsFromImages(imageIds, tagIds): Promise<readonly ImageTagResponse[]>`.

All request helpers continue to use `apiCall` / `unwrapResponse`, relative `/api/v1/*` paths, and `import type` for types.

## Component Boundaries

Picker state should be implemented as local component/composable logic, not a global store. Create small reusable composables only if both detail and batch flows need the same async state:

- `useCollectionPicker()` may own collection load/create/select/add status.
- `useTagPicker()` may own tag load/suggest/create/assign/unassign status.
- No Pinia/global store and no localStorage persistence for backend-owned collection/tag state.
- Collection rename/delete, collection item removal, and admin tag CRUD are intentionally excluded from these picker boundaries.

## UI Flow

### Detail Page Favorite Flow

1. User opens `/detail?id=<imageId>`.
2. If logged out, the favorite action shows login-required state and does not call mutation APIs.
3. If logged in, clicking “收藏到相册” opens an inline panel or modal using existing panel/form classes.
4. The panel loads real collections via `getCollections()`.
5. User selects a collection, optionally creates a new collection, then calls `addImageToCollection(collectionId, imageId)`.
6. On success, show specific feedback and refresh detail data so `is_collected` / `favorite_count` can update.

### Detail Page Tag Flow

1. Existing image tags from `ImageDetailResponse.tags` render as real current tags.
2. Clicking tag management opens an inline panel or modal.
3. The panel loads/suggests tags from `getTags()` / `suggestTags()`.
4. User selects tags to add or remove. Creating a new tag calls `createTag(name)` before assignment.
5. Add calls `assignTagsToImages([imageId], tagIds)`; remove calls `unassignTagsFromImages([imageId], tagIds)`.
6. On success, refresh detail data and show inline/toast confirmation.

### Batch Panel Flow

1. `useSelection()` exposes selected image IDs as strings; parse and filter to positive integers before backend calls.
2. If no valid image IDs remain, show an honest error and do not call backend.
3. Batch “加入收藏夹” reuses the collection picker; selected image IDs are added sequentially or with `Promise.all` using `addImageToCollection`.
4. Batch “批量打标签” reuses the tag picker and calls `assignTagsToImages(validImageIds, tagIds)`.
5. After successful batch operation, clear selection only after backend success.

### Collections Page Management

Keep the current real-data list/create behavior and improve actionability without mock fallbacks:

- Add visibility choice to creation.
- Show collection metadata, item counts, and real IDs clearly.
- Do not add collection rename/delete or item-removal UI in this task.
- Do not render fake thumbnails for collection items.

## State And Accessibility

- Each new workflow must explicitly handle: logged-out, loading, empty, error-with-retry, in-flight mutation, success, and no-valid-target states.
- Detail collection picker: logged-out; loading collections; no collections with create option; load error + retry; adding disabled/in-flight; success refreshes `getImage()`.
- Detail tag picker: logged-out; loading tags; no tags with create option; suggest/load error + retry; add/remove disabled/in-flight; success refreshes `getImage()`.
- Batch collection picker: no valid selected image IDs; logged-out; loading collections; no collections with create option; per-image add failures reported without fake success; clear selection only after full success.
- Batch tag picker: no valid selected image IDs; logged-out; loading/suggesting tags; no tags with create option; add/remove disabled/in-flight; clear selection only after full success.
- New panels/modals expose loading, empty, error, and success states.
- Form controls have visible labels and `aria-describedby` for helper/error/status text.
- Buttons become disabled while their mutation is in flight.
- Status feedback uses existing `status` or toast patterns; error and success are not toast-only when the user must make a follow-up decision.
- Escape/close controls should not discard in-flight operations.

## Visual Direction

Use taste-skill polish inside the existing Colorful system:

- Warm yellow canvas and white panels stay unchanged.
- Important pickers use `panel-raised`, clear section eyebrows, concise helper copy, and compact tags.
- Avoid generic CRUD table styling; model collection/tag actions as “organize into素材库” workflows.
- Motion uses existing `--motion-fast`, `--motion-base`, and `--ease-standard`; no new decorative animation loop.

## Rollback Points

- API helper additions can be reverted independently of UI panels.
- Detail page favorite/tag panels can be reverted independently.
- Batch panel integration can be reverted independently.
- Collections page management additions can be reverted independently.

## Validation Strategy

- Static build/type check: `npm run build` in `frontend/vue-gallery`.
- API smoke where backend is available: `/api/v1/tags`, `/api/v1/collections` unauthenticated 401, and authenticated mutation flows if a test user is available.
- Browser validation through Vite: detail page logged-out states, collection picker empty/login states, tag picker list/suggest state, and batch panel no fake success.
