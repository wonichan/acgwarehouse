# Viewer Window Lazy Navigation Design

## Summary

The secondary Flutter viewer window must stop receiving the full result-set payload through `desktop_multi_window`. Instead, the launch payload will carry only the current viewer context, and the subwindow will fetch a local navigation window from the backend on demand.

The approved direction is:

- keep navigation driven by the current `selectedIndex`
- stop passing full `items` arrays or neighboring image IDs
- on viewer startup and on every previous/next navigation, call the backend to fetch up to 10 nearby images
- preserve the original gallery/search ordering by sending a source-specific query snapshot with the launch payload and each viewer-window request

## Problem Statement

The current viewer launch path serializes the entire `ViewerSession.items` array into the window payload passed to `DesktopMultiWindow.createWindow(jsonEncode(...))`. This is fragile and already exceeds safe payload sizes in realistic cases.

Codebase evidence:

- `flutter_app/lib/app/fluent_screens.dart` launches the viewer with `ViewerSession.fromResultSet(...)`
- `flutter_app/lib/services/viewer_window_service.dart` serializes the full session into JSON for the new window
- `flutter_app/lib/models/viewer_session.dart` assumes a complete `items` array and a stable `initialSelectedIndex`
- `flutter_app/lib/screens/viewer/viewer_workspace.dart` and `viewer_filmstrip.dart` assume all items are already present in memory

Observed runtime evidence:

- a 20-image payload already measures well above 8 KB
- the native title can still be correct even when the Flutter viewer content is blank, because the title is set outside the subwindow Dart render path

## Goals

1. Eliminate full-session window payloads when opening the secondary viewer.
2. Preserve existing viewer entry points from gallery and search.
3. Keep navigation centered on `selectedIndex` as requested.
4. Fetch adjacent image information through the backend, with a maximum window size of 10 images per request.
5. Re-query on each previous/next navigation using the current `selectedIndex`.
6. Preserve source-specific ordering and filtering semantics.

## Non-Goals

1. No redesign of the viewer layout.
2. No change to gallery/search result rendering outside the viewer launch path.
3. No attempt to keep the full result-set mirrored inside the subwindow.
4. No new sync channel between the main window and the viewer beyond the initial launch context.

## Existing Constraints

### Flutter-side constraints

- `ViewerWorkspace` currently reads `widget.session.items[_selectedIndex]` directly and uses `items.length` for navigation bounds.
- `ViewerFilmstrip` renders the full `session.items` list and shows `X of total` based on in-memory array length.
- `ViewerWindowService.openSession(...)` currently requires a fully materialized `ViewerSession`.

### Backend constraints

- `GET /api/v1/images` supports `limit`, `offset`, `sort_by`, `sort_dir`, `tag_ids`, and `has_tags`.
- `GET /api/v1/search` supports `q`, `tag_ids`, `sort_by`, `sort_order`, `limit`, and `offset`.
- existing backend list/search APIs are offset-based; there is no current endpoint for “fetch the window around selected index N”.

### Data consistency constraints

- a bare `selectedIndex` is only meaningful when paired with the sort/filter/query context that produced it
- the viewer therefore must carry a source-specific query snapshot into the subwindow

## Recommended Approach

### 1. Replace full viewer session launch payload with a lightweight launch context

Replace the current full-session payload with a compact viewer launch contract.

Proposed launch payload shape:

```json
{
  "kind": "viewer-window",
  "logical_window_id": "viewer-window-1",
  "title": "ACGWarehouse Viewer — example.jpg",
  "context": {
    "source": "gallery",
    "selected_index": 42,
    "selected_image_id": 123,
    "snapshot": {
      "sort_by": "created_at",
      "sort_dir": "desc",
      "tag_ids": [1, 2],
      "has_tags": null
    }
  }
}
```

Notes:

- `selected_index` remains the primary navigation coordinate.
- `selected_image_id` is retained as an identity guard for the selected record.
- `snapshot` differs by source:
  - gallery snapshot: `sort_by`, `sort_dir`, `tag_ids`, `has_tags`
  - search snapshot: `q`, `tag_ids`, `sort_by`, `sort_order`

### 2. Introduce a viewer-specific backend endpoint

Add a dedicated endpoint that returns the neighborhood around the current selected index.

Proposed endpoint:

`POST /api/v1/viewer/window`

Proposed request body:

```json
{
  "source": "gallery",
  "selected_index": 42,
  "selected_image_id": 123,
  "limit": 10,
  "snapshot": {
    "sort_by": "created_at",
    "sort_dir": "desc",
    "tag_ids": [1, 2],
    "has_tags": null
  }
}
```

Proposed response body:

```json
{
  "items": [
    {
      "id": 120,
      "path": "E:/picture/example-120.jpg",
      "filename": "example-120.jpg",
      "source_root": "E:/picture",
      "file_size": 123456,
      "width": 1280,
      "height": 720,
      "format": "jpg",
      "thumbnail_small_url": "https://.../120-small.jpg",
      "thumbnail_large_url": "https://.../120-large.jpg",
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    },
    {
      "id": 121,
      "path": "E:/picture/example-121.jpg",
      "filename": "example-121.jpg",
      "source_root": "E:/picture",
      "file_size": 123457,
      "width": 1280,
      "height": 720,
      "format": "jpg",
      "thumbnail_small_url": "https://.../121-small.jpg",
      "thumbnail_large_url": "https://.../121-large.jpg",
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    },
    {
      "id": 123,
      "path": "E:/picture/example-123.jpg",
      "filename": "example-123.jpg",
      "source_root": "E:/picture",
      "file_size": 123458,
      "width": 1280,
      "height": 720,
      "format": "jpg",
      "thumbnail_small_url": "https://.../123-small.jpg",
      "thumbnail_large_url": "https://.../123-large.jpg",
      "created_at": "2026-04-01T00:00:00Z",
      "updated_at": "2026-04-01T00:00:00Z"
    }
  ],
  "window_start_index": 38,
  "selected_index": 42,
  "selected_index_in_window": 4,
  "total": 615,
  "has_previous": true,
  "has_next": true
}
```

The `items` array must reuse the existing full image JSON contract already returned by image/search APIs so the current viewer stage, metadata sidebar, and filmstrip can render without a second item-shape translation layer.

Server-side logic:

- clamp `limit` to a maximum of 10
- compute `offset` from `selected_index` and requested window size
- reuse existing image/search query semantics under the provided snapshot
- apply deterministic ordering with a stable secondary tie-breaker of `id`
- return a local window plus enough metadata for the subwindow to map global selected index to local array index

This keeps the implementation aligned with the current offset-based backend instead of inventing a cursor protocol the codebase does not already use.

### 3. Refactor the viewer subwindow state around a sliding window

The subwindow should no longer own a full `ViewerSession.items` list. It should own a smaller window state:

- source snapshot
- current global `selectedIndex`
- current selected image id
- currently loaded window items (max 10)
- `windowStartIndex`
- `selectedIndexInWindow`
- `hasPrevious` / `hasNext`

Startup flow:

1. parse launch context from command line
2. initialize viewer runtime bootstrap
3. request the initial window from `POST /api/v1/viewer/window`
4. render the selected item plus the local filmstrip window

Navigation flow:

1. user presses previous or next
2. subwindow updates the global `selectedIndex` by `-1` or `+1`
3. subwindow calls the viewer-window endpoint with the new `selectedIndex`
4. UI re-renders from the returned 10-item window

This matches the requested behavior: each navigation step re-queries using the current `selectedIndex`.

### 4. Keep the filmstrip local to the fetched window

`ViewerFilmstrip` should stop assuming the entire result-set is available.

New behavior:

- render only the currently fetched local window
- derive the selected tile from `selectedIndexInWindow`
- display count text using `selectedIndex + 1` and backend `total`

Example:

- label format: `43 of 615`
- filmstrip tiles: only the current 10-image neighborhood

## Source Snapshot Design

### Gallery source

The gallery snapshot must preserve the exact list semantics currently owned by `ImageListProvider`:

- `sort_by`
- `sort_dir`
- `tag_ids`
- `has_tags`

Gallery ordering must be deterministic. If multiple rows share the same primary sort key, the backend must break ties with `id` using the same direction as the primary sort.

### Search source

The search snapshot must preserve the exact list semantics currently owned by `SearchProvider`:

- `q`
- `tag_ids`
- `sort_by`
- `sort_order`

Search ordering must also be deterministic. For non-relevance sorts, the backend must break ties with `id`. For relevance search, the backend must preserve relevance order first and then break ties with `id`.

The viewer must not infer these values independently; it must receive them from the main-window provider state at launch time.

## Error Handling

### Invalid or stale viewer request

If the backend cannot resolve the selected index under the supplied snapshot:

- return `400 Bad Request` with

```json
{
  "error": "invalid_viewer_request",
  "message": "selected_index is out of range for the supplied snapshot"
}
```

- keep the subwindow open
- render a visible error state instead of a blank surface

### Selected image mismatch

If `selected_image_id` does not match the image found at the requested `selected_index`:

- treat this as snapshot drift
- return `409 Conflict` with

```json
{
  "error": "viewer_snapshot_drift",
  "message": "selected_image_id no longer matches the supplied selected_index"
}
```

- let the viewer show a “result set changed” message

This keeps the first implementation deterministic without silently drifting to another image.

### Image loading failure

If the selected item’s image URL or file path fails to load, the viewer must render an explicit failure state instead of appearing empty.

### Transient backend failure

If the viewer-window endpoint fails unexpectedly:

- return `500 Internal Server Error` with

```json
{
  "error": "viewer_window_failed",
  "message": "failed to load viewer window"
}
```

- keep the subwindow open
- render a retryable error state in the viewer UI

## Window Title Synchronization

The native window title is in scope and must stay synchronized with the selected item.

- initial title comes from the launch payload
- after each successful previous/next fetch, the subwindow must update the native title to match the newly selected item filename
- manual QA must verify title synchronization alongside image, metadata, and filmstrip state

## Testing Strategy

### Backend

- handler tests for `POST /api/v1/viewer/window`
- verify gallery snapshots respect `sort_by`, `sort_dir`, `tag_ids`, `has_tags`
- verify search snapshots respect `q`, `tag_ids`, `sort_by`, `sort_order`
- verify `limit` is capped at 10
- verify boundary cases near index `0` and near the last item
- verify mismatch handling for stale `selected_image_id`

### Flutter

- viewer launch payload tests for lightweight context instead of full session items
- viewer window state tests for initial load and next/previous re-query behavior
- filmstrip tests for local-window rendering with `selectedIndexInWindow` and `total`
- widget tests for explicit loading and error states

### Manual QA

- open viewer from gallery with a large result set
- open viewer from search with active query + tag filters
- navigate previous/next repeatedly and verify the viewer keeps moving without blank windows
- verify title, image, metadata sidebar, and filmstrip stay in sync with the selected item

## Implementation Notes

The smallest implementation path is:

1. add the new backend viewer-window endpoint by reusing existing offset-based query logic
2. replace the full-session launch payload with a lightweight launch context
3. refactor viewer state from `ViewerSession` to fetched window state
4. update filmstrip and navigation to use backend-driven local windows

No git commit is included in this design phase unless explicitly requested.
