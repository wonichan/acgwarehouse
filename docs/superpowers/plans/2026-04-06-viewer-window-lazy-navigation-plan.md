# Viewer Window Lazy Navigation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace full-result-set secondary viewer payloads with a lightweight launch context and backend-fetched 10-item navigation windows driven by `selectedIndex`.

**Architecture:** The backend gains a viewer-specific window endpoint that reuses existing gallery/search ordering semantics with deterministic offset-based queries. The Flutter viewer launch path sends only source context, selected position, and snapshot state; the subwindow owns a small fetched window state, re-queries on every previous/next navigation, and keeps title, metadata, and filmstrip synchronized from the returned window.

**Tech Stack:** Go + Gin + SQLite repository layer; Flutter desktop + fluent_ui + provider + http; desktop_multi_window; flutter_test + go test.

---

## File Structure

### Backend

- Modify: `internal/handler/routes.go`
  - Register the new viewer endpoint under `/api/v1`.
- Modify: `internal/handler/image_handler.go`
  - Add request/response DTOs, accept search-service access, and implement the new `ViewerWindow` handler method.
- Modify: `internal/repository/image_repository.go`
  - Add deterministic ordering helpers and reusable query logic needed by the viewer endpoint.
- Modify: `internal/service/search_service.go`
  - Add a viewer-window search path or reusable helper that can resolve a deterministic search window by snapshot + offset.
- Test: `internal/handler/image_handler_test.go`
  - Add handler coverage for the new endpoint and error/status behavior.
- Test: `internal/handler/routes_test.go`
  - Assert the new route is wired.
- Test: `internal/service/search_service_test.go`
  - Add coverage for deterministic search ordering and viewer-window search resolution if search logic is extended there.

### Flutter launch contract and API client

- Modify: `flutter_app/lib/services/viewer_window_service.dart`
  - Replace full-session payloads with a lightweight viewer launch context.
- Modify: `flutter_app/lib/app/fluent_screens.dart`
  - Build launch context from gallery/search provider state instead of materializing `ViewerSession.fromResultSet(...)`.
- Modify: `flutter_app/lib/providers/image_provider.dart`
  - Expose current gallery snapshot fields needed for viewer launch.
- Modify: `flutter_app/lib/providers/search_provider.dart`
  - Expose current search snapshot fields needed for viewer launch.
- Modify: `flutter_app/lib/services/api_service.dart`
  - Add a typed client for `POST /api/v1/viewer/window`.
- Create: `flutter_app/lib/models/viewer_window_context.dart`
  - Typed launch-context DTOs and snapshot models.
- Create: `flutter_app/lib/models/viewer_window_result.dart`
  - Typed API response model for the fetched 10-item window.
- Test: `flutter_app/test/services/viewer_window_service_test.dart`
  - Update payload encoding/parsing tests for lightweight context.

### Flutter subwindow state and UI

- Modify: `flutter_app/lib/main.dart`
  - Parse the new bootstrap context instead of a full session.
- Modify: `flutter_app/lib/app/viewer_window_app.dart`
  - Pass launch context into a fetched-state viewer workspace and keep native title in sync after navigation.
- Modify: `flutter_app/lib/screens/viewer/viewer_workspace.dart`
  - Replace full-session assumptions with fetched window state and API-driven prev/next re-queries.
- Modify: `flutter_app/lib/screens/viewer/viewer_filmstrip.dart`
  - Render only the local fetched window and global count text.
- Modify: `flutter_app/lib/screens/viewer/viewer_stage.dart`
  - Continue rendering selected image from fetched window data; preserve explicit error handling.
- Modify: `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`
  - Continue reading selected item from fetched window state.
- Create: `flutter_app/lib/services/viewer_window_api_service.dart`
  - Small wrapper around `ApiService` for viewer-window requests if keeping `ApiService` focused is cleaner.
- Test: `flutter_app/test/app/viewer_window_app_test.dart`
  - Update app bootstrap expectations for context-driven startup.
- Test: `flutter_app/test/screens/viewer/viewer_workspace_test.dart`
  - Add navigation/re-query/state-sync coverage.
- Create: `flutter_app/test/services/viewer_window_api_service_test.dart`
  - Add API result parsing/error handling coverage if a separate service file is introduced.

## Chunk 1: Backend viewer-window endpoint

### Task 1: Add deterministic ordering helpers for image lists

**Files:**
- Modify: `internal/repository/image_repository.go`
- Test: `internal/repository/image_repository_test.go`

- [ ] **Step 1: Write the failing repository tests for stable tie-break ordering**

Add tests that seed images sharing the same `created_at`, `filename`, or `file_size`, then assert repeated `FindAll(...)` / `FindByTagIDs(...)` / `FindUntagged(...)` calls return a stable order broken by `id` in the same direction.

- [ ] **Step 2: Run the repository tests to verify they fail**

Run: `go test ./internal/repository -run "TestImageRepository.*(Stable|TieBreak|Deterministic)"`
Expected: FAIL because current SQL orders by the primary column only.

- [ ] **Step 3: Implement minimal ordering helper changes**

Refactor the SQL builders in `image_repository.go` so the generated `ORDER BY` clause always appends `, id <same_dir>` (or `, i.id <same_dir>` for joined queries).

- [ ] **Step 4: Run the repository tests to verify they pass**

Run: `go test ./internal/repository -run "TestImageRepository.*(Stable|TieBreak|Deterministic)"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/image_repository.go internal/repository/image_repository_test.go
git commit -m "test: stabilize image ordering tie breaks"
```

### Task 2: Add viewer-window endpoint contract and gallery query path

**Files:**
- Modify: `internal/handler/image_handler.go`
- Modify: `internal/handler/routes.go`
- Test: `internal/handler/image_handler_test.go`
- Test: `internal/handler/routes_test.go`

- [ ] **Step 1: Write the failing handler tests for gallery viewer windows**

Add tests covering:
- `POST /api/v1/viewer/window` returns 200 with a 10-item max window
- response includes `items`, `window_start_index`, `selected_index`, `selected_index_in_window`, `total`, `has_previous`, `has_next`
- `selected_index` out of range returns `400` with `{"error":"invalid_viewer_request",...}`
- `selected_image_id` mismatch returns `409` with `{"error":"viewer_snapshot_drift",...}`
- route registration exists in `routes_test.go`

- [ ] **Step 2: Run the handler tests to verify they fail**

Run: `go test ./internal/handler -run "TestImageHandler.*ViewerWindow|TestRoutes"`
Expected: FAIL because the endpoint and route do not exist.

- [ ] **Step 3: Implement the minimal gallery viewer-window handler**

In `image_handler.go`:
- define request DTO with `source`, `selected_index`, `selected_image_id`, `limit`, and `snapshot`
- clamp `limit` to `10`
- validate gallery snapshots against existing image-list query semantics
- compute `offset` window around `selected_index`
- fetch items through repository methods already used by `/api/v1/images`
- build the structured response and the `400` / `409` / `500` error contracts

In `routes.go`:
- register `api.POST("/viewer/window", imageHandler.ViewerWindow)` when image dependencies are present

- [ ] **Step 4: Run the handler tests to verify they pass**

Run: `go test ./internal/handler -run "TestImageHandler.*ViewerWindow|TestRoutes"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/image_handler.go internal/handler/image_handler_test.go internal/handler/routes.go internal/handler/routes_test.go
git commit -m "feat: add gallery viewer window endpoint"
```

### Task 3: Extend search service for viewer-window queries

**Files:**
- Modify: `internal/handler/routes.go`
- Modify: `internal/handler/image_handler.go`
- Modify: `internal/service/search_service.go`
- Test: `internal/service/search_service_test.go`
- Test: `internal/handler/image_handler_test.go`

- [ ] **Step 1: Write the failing tests for search-source viewer windows**

Add tests that verify a search snapshot (`q`, `tag_ids`, `sort_by`, `sort_order`) can resolve a local window around the requested `selected_index`, preserves deterministic order, and returns the same error contract as gallery mode.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service ./internal/handler -run "Test(SearchService|ImageHandler).*ViewerWindow.*Search"`
Expected: FAIL because no viewer-window search resolution exists.

- [ ] **Step 3: Implement the minimal search viewer-window helper**

Extend `SearchService` with a viewer-window helper that:
- reuses existing search semantics
- resolves the local window via offset/limit under the search snapshot
- preserves relevance ordering first, then `id` tie-breaks when needed
- correctly enforces tag filters for query+tag snapshots instead of relying on the current `imageHasAllTags` stub

Wire `image_handler.go` to call that helper when `source == "search"`.

In `routes.go`, update the `ImageHandler` construction so the new viewer-window handler can access `deps.SearchSvc` for search-backed requests.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service ./internal/handler -run "Test(SearchService|ImageHandler).*ViewerWindow.*Search"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/search_service.go internal/service/search_service_test.go internal/handler/image_handler.go internal/handler/image_handler_test.go internal/handler/routes.go
git commit -m "feat: add search-backed viewer window queries"
```

## Chunk 2: Flutter launch context and API client

### Task 4: Replace full-session viewer payloads with lightweight launch context

**Files:**
- Create: `flutter_app/lib/models/viewer_window_context.dart`
- Modify: `flutter_app/lib/services/viewer_window_service.dart`
- Modify: `flutter_app/lib/app/fluent_screens.dart`
- Modify: `flutter_app/lib/providers/image_provider.dart`
- Modify: `flutter_app/lib/providers/search_provider.dart`
- Test: `flutter_app/test/services/viewer_window_service_test.dart`

- [ ] **Step 1: Write the failing Flutter tests for the new launch context**

Add tests asserting that viewer launch payloads now encode:
- `context.source`
- `context.selected_index`
- `context.selected_image_id`
- source-specific snapshot fields

Also update bootstrap parsing expectations to read `context` instead of `session`.

- [ ] **Step 2: Run the Flutter tests to verify they fail**

Run: `flutter test test/services/viewer_window_service_test.dart`
Expected: FAIL because the code still serializes full `session.items`.

- [ ] **Step 3: Implement the minimal launch-context refactor**

In `viewer_window_context.dart` define:
- context DTO
- gallery snapshot DTO
- search snapshot DTO

In providers expose read-only snapshot builders for current state.

In `fluent_screens.dart` replace `ViewerSession.fromResultSet(...)` with lightweight launch-context creation.

In `viewer_window_service.dart`:
- replace `ViewerSession` payload encoding/parsing with `ViewerWindowContext`
- keep `buildWindowTitle(...)` unchanged

- [ ] **Step 4: Run the Flutter tests to verify they pass**

Run: `flutter test test/services/viewer_window_service_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/models/viewer_window_context.dart flutter_app/lib/services/viewer_window_service.dart flutter_app/lib/app/fluent_screens.dart flutter_app/lib/providers/image_provider.dart flutter_app/lib/providers/search_provider.dart flutter_app/test/services/viewer_window_service_test.dart
git commit -m "feat: send lightweight viewer launch context"
```

### Task 5: Add typed client models for `POST /api/v1/viewer/window`

**Files:**
- Modify: `flutter_app/lib/services/api_service.dart`
- Create: `flutter_app/lib/models/viewer_window_result.dart`
- Create: `flutter_app/test/services/viewer_window_api_service_test.dart`

- [ ] **Step 1: Write the failing API client tests**

Add tests for:
- successful parsing of the viewer-window response
- `400`, `409`, and `500` error mapping
- item JSON reusing the existing image shape

- [ ] **Step 2: Run the Flutter tests to verify they fail**

Run: `flutter test test/services/viewer_window_api_service_test.dart`
Expected: FAIL because the typed request/response client does not exist.

- [ ] **Step 3: Implement the minimal API client additions**

Add typed request/response models and a method on `ApiService` (or a small focused wrapper service if that keeps responsibilities clearer) that posts the viewer-window request and returns parsed typed data.

- [ ] **Step 4: Run the Flutter tests to verify they pass**

Run: `flutter test test/services/viewer_window_api_service_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/services/api_service.dart flutter_app/lib/models/viewer_window_result.dart flutter_app/test/services/viewer_window_api_service_test.dart
git commit -m "feat: add viewer window API client"
```

## Chunk 3: Subwindow fetched state, filmstrip, and title sync

### Task 6: Refactor subwindow bootstrap from full session to fetched viewer state

**Files:**
- Modify: `flutter_app/lib/main.dart`
- Modify: `flutter_app/lib/app/viewer_window_app.dart`
- Modify: `flutter_app/lib/screens/viewer/viewer_workspace.dart`
- Test: `flutter_app/test/app/viewer_window_app_test.dart`
- Test: `flutter_app/test/screens/viewer/viewer_workspace_test.dart`

- [ ] **Step 1: Write the failing widget/state tests for context-driven startup**

Add tests asserting:
- subwindow startup accepts launch context
- initial render performs an API-backed load
- loading state is visible before the response
- selected item metadata appears after the response

- [ ] **Step 2: Run the tests to verify they fail**

Run: `flutter test test/app/viewer_window_app_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: FAIL because `ViewerWorkspace` still depends on `ViewerSession`.

- [ ] **Step 3: Implement the minimal fetched-state refactor**

In `main.dart` and `viewer_window_service.dart`, pass parsed launch context into the viewer app.

In `viewer_workspace.dart`:
- replace full-session state with fetched local window state
- load the initial window on startup
- update global `selectedIndex` on prev/next and re-query every navigation step
- keep tag loading tied to the selected item returned by the current window

In `viewer_window_app.dart`:
- keep title updates after each successful selected item change
- preserve escape-to-close behavior

- [ ] **Step 4: Run the tests to verify they pass**

Run: `flutter test test/app/viewer_window_app_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/main.dart flutter_app/lib/app/viewer_window_app.dart flutter_app/lib/screens/viewer/viewer_workspace.dart flutter_app/test/app/viewer_window_app_test.dart flutter_app/test/screens/viewer/viewer_workspace_test.dart
git commit -m "feat: load viewer state from backend windows"
```

### Task 7: Refactor filmstrip to local-window rendering and global count text

**Files:**
- Modify: `flutter_app/lib/screens/viewer/viewer_filmstrip.dart`
- Modify: `flutter_app/lib/screens/viewer/viewer_workspace.dart`
- Test: `flutter_app/test/screens/viewer/viewer_workspace_test.dart`
- Create or Modify: `flutter_app/test/screens/viewer/viewer_filmstrip_test.dart`

- [ ] **Step 1: Write the failing filmstrip tests**

Add tests asserting:
- filmstrip renders only the returned local window
- count text uses global `selectedIndex + 1` and backend `total`
- tapping a tile maps to the correct global selected index

- [ ] **Step 2: Run the filmstrip tests to verify they fail**

Run: `flutter test test/screens/viewer/viewer_filmstrip_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: FAIL because the current filmstrip still uses `session.items` and `items.length`.

- [ ] **Step 3: Implement the minimal filmstrip refactor**

Change the filmstrip contract to accept:
- local window items
- `selectedIndexInWindow`
- `windowStartIndex`
- global `selectedIndex`
- backend `total`

Keep scrolling and selection visuals, but scope them to the local 10-item window.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `flutter test test/screens/viewer/viewer_filmstrip_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/screens/viewer/viewer_filmstrip.dart flutter_app/lib/screens/viewer/viewer_workspace.dart flutter_app/test/screens/viewer/viewer_filmstrip_test.dart flutter_app/test/screens/viewer/viewer_workspace_test.dart
git commit -m "feat: render local viewer filmstrip windows"
```

### Task 8: Verify end-to-end title, image, metadata, and navigation synchronization

**Files:**
- Modify: `flutter_app/lib/screens/viewer/viewer_stage.dart`
- Modify: `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart` (only if selected item contract changes require it)
- Modify: `flutter_app/test/app/viewer_window_app_test.dart`
- Modify: `flutter_app/test/screens/viewer/viewer_workspace_test.dart`

- [ ] **Step 1: Write the failing synchronization tests**

Add tests asserting that after a successful next/previous fetch:
- selected image content updates
- metadata sidebar updates
- native title update callback receives the new filename
- explicit error UI appears for API failure or image failure instead of blank space

- [ ] **Step 2: Run the tests to verify they fail**

Run: `flutter test test/app/viewer_window_app_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: FAIL because synchronization is incomplete.

- [ ] **Step 3: Implement the minimal synchronization changes**

Ensure the selected item is the single source of truth for:
- `ViewerStage`
- `ViewerMetadataSidebar`
- title updates via `AppWindowManager.setTitle(...)`
- visible loading/error states

- [ ] **Step 4: Run the tests to verify they pass**

Run: `flutter test test/app/viewer_window_app_test.dart test/screens/viewer/viewer_workspace_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/screens/viewer/viewer_stage.dart flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart flutter_app/test/app/viewer_window_app_test.dart flutter_app/test/screens/viewer/viewer_workspace_test.dart
git commit -m "feat: sync viewer title and selected item state"
```

## Final Verification

- [ ] **Step 1: Run backend tests for touched packages**

Run: `go test ./internal/handler ./internal/repository ./internal/service`
Expected: PASS.

- [ ] **Step 2: Run Flutter tests for touched viewer files**

Run: `flutter test test/services/viewer_window_service_test.dart test/services/viewer_window_api_service_test.dart test/app/viewer_window_app_test.dart test/screens/viewer/viewer_workspace_test.dart test/screens/viewer/viewer_filmstrip_test.dart`
Expected: PASS.

- [ ] **Step 3: Run Flutter diagnostics on modified files**

Run diagnostics for:
- `flutter_app/lib/services/viewer_window_service.dart`
- `flutter_app/lib/models/viewer_window_context.dart`
- `flutter_app/lib/models/viewer_window_result.dart`
- `flutter_app/lib/app/viewer_window_app.dart`
- `flutter_app/lib/screens/viewer/viewer_workspace.dart`
- `flutter_app/lib/screens/viewer/viewer_filmstrip.dart`
- `flutter_app/lib/services/api_service.dart`

Expected: zero new errors.

- [ ] **Step 4: Manual QA the actual feature**

Run the desktop app and verify:
- opening the viewer from gallery no longer depends on full launch payload size
- opening the viewer from search with active filters works
- each previous/next action triggers a fresh viewer-window API request
- the filmstrip shows only the local fetched neighborhood
- the window title, selected image, metadata sidebar, and count label stay synchronized

- [ ] **Step 5: Final commit**

```bash
git add internal/handler internal/repository internal/service flutter_app/lib flutter_app/test
git commit -m "feat: lazy load viewer window navigation"
```
