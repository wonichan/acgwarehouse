# Gallery Tag Filter Lazy Tree Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace gallery filter's eager full-tree load with lazy tree loading plus a separate orphan-tag section, while preserving subtree matching for root/parent selections and exact matching for orphan/child selections.

**Architecture:** Split tag filtering into two lightweight data sources: hierarchical tree endpoints for real root/parent/child browsing and paged orphan endpoints for `parent_id IS NULL AND level != root`. Keep expensive descendant expansion out of filter initialization; only apply subtree expansion when image queries execute with `subtree_root_tag_ids`.

**Tech Stack:** Go + Gin + SQLite, Flutter desktop/web, Provider state management, existing repository/handler tests, Flutter widget/service/provider tests.

---

## File Structure

### Backend files
- Modify: `internal/handler/routes.go`
  - Register new gallery-friendly tag endpoints without removing governance endpoints.
- Modify: `internal/handler/tag_handler.go`
  - Add lightweight handlers for tree roots, tree children, orphan list/search, and any shared response structs.
- Modify: `internal/service/tag_admin_service.go`
  - Add focused read methods for roots, children, and orphan paging/search.
- Modify: `internal/repository/tag_repository.go`
  - Add repository methods for root queries, child queries, orphan paging/search, and `has_children` lookups.
- Modify: `internal/repository/image_repository.go`
  - Add exact-vs-subtree filter methods for gallery image queries and pending-tag variants.
- Modify: `internal/repository/tag_hierarchy_query.go`
  - Refactor hierarchy clause helpers so exact tags and subtree-root tags can be combined without over-expanding exact child/orphan selections.
- Modify: `internal/handler/image_handler.go`
  - Parse `exact_tag_ids` and `subtree_root_tag_ids`, validate incompatibilities, and route to the new repository methods.
- Modify: `internal/service/search_service.go`
  - Keep search/viewer options aligned with any new filter fields if gallery snapshot semantics are affected.
- Test: `internal/handler/tag_handler_test.go`
- Test: `internal/repository/tag_repository_test.go`
- Test: `internal/repository/image_repository_test.go`
- Create: `internal/handler/image_handler_test.go` (if no existing handler coverage fits the new query-param matrix cleanly)

### Frontend files
- Modify: `flutter_app/lib/services/tag_service.dart`
  - Replace `getTree()`-only gallery dependency with lazy tree/orphan/search methods.
- Modify: `flutter_app/lib/providers/tag_provider.dart`
  - Replace single `_tagTree` state with roots / children / orphans / loading / search buckets.
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
  - Render lazy tree section, orphan section, selected chips, search groups, and localized error boundaries.
- Modify: `flutter_app/lib/providers/image_provider.dart`
  - Forward both `exactTagIds` and `subtreeRootTagIds` to API layer and expose both in UI state.
- Modify: `flutter_app/lib/services/api_service.dart`
  - Serialize `exact_tag_ids` and `subtree_root_tag_ids` distinctly.
- Modify: `flutter_app/lib/screens/gallery_screen.dart`
  - Stop preloading unrelated full tag lists for the gallery flow if no longer needed.
- Possibly modify: `flutter_app/lib/models/gallery_filter_state.dart`
  - Only if helper methods need stronger normalization for subtree/exact coexistence.
- Test: `flutter_app/test/services/tag_service_test.dart`
- Test: `flutter_app/test/providers/tag_provider_test.dart`
- Test: `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`
- Test: `flutter_app/test/services/api_service_test.dart`
- Test: `flutter_app/test/providers/image_provider_filter_state_test.dart`

---

## Chunk 1: Backend tag browsing endpoints

### Task 1: Add focused repository queries for lazy tree + orphan tags

**Files:**
- Modify: `internal/repository/tag_repository.go`
- Test: `internal/repository/tag_repository_test.go`

- [ ] **Step 1: Write failing repository tests for roots, children, and orphans**
  - Add tests that prove:
    - roots query returns only `level = root`
    - children query returns direct children for a parent with stable ordering
    - orphan query returns `parent_id IS NULL AND level != root`
    - orphan search + pagination do not include true roots

- [ ] **Step 2: Run repository tests to verify failure**

Run:
```bash
go test ./internal/repository -run "TagRepository|ImageRepository" -count=1
```
Expected: FAIL because the new query methods and test expectations do not exist yet.

- [ ] **Step 3: Implement minimal repository methods**
  - Add methods such as:
    - `FindTreeRoots(ctx)`
    - `FindTreeChildren(ctx, parentID)`
    - `ListOrphanTags(ctx, search string, limit, offset int)`
    - `CountOrphanTags(ctx, search string)`
  - Each result should return enough data for handler/service mapping plus a `has_children` signal.
  - Prefer SQL that computes `has_children` with `EXISTS (SELECT 1 FROM tags child WHERE child.parent_id = tags.id)`.

- [ ] **Step 4: Re-run repository tests**

Run:
```bash
go test ./internal/repository -run "TagRepository" -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/tag_repository.go internal/repository/tag_repository_test.go
git commit -m "feat: add lazy tag tree repository queries"
```

### Task 2: Add service + handler endpoints for roots, children, and orphan tags

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Modify: `internal/handler/tag_handler.go`
- Modify: `internal/handler/routes.go`
- Test: `internal/service/tag_admin_service_test.go`
- Test: `internal/handler/tag_handler_test.go`

- [ ] **Step 1: Write failing service/handler tests**
  - Cover:
    - `GET /api/v1/tags/tree/roots`
    - `GET /api/v1/tags/tree/children?parent_id=...`
    - `GET /api/v1/tags/orphans?limit=...&offset=...&search=...`
    - response includes `items`, `total`, `has_more` for orphan paging
    - response includes `has_children` for tree nodes

- [ ] **Step 2: Run tests to verify failure**

Run:
```bash
go test ./internal/handler ./internal/service -run "Tag" -count=1
```
Expected: FAIL because the endpoints and DTOs are not wired yet.

- [ ] **Step 3: Implement minimal service/handler wiring**
  - Add service DTO(s) for gallery tree nodes and orphan page responses.
  - Keep old `GetTagTree()` endpoint intact for governance pages unless separately retired later.
  - Add route registration in `routes.go`.
  - Ensure handler validation returns `400` for missing/invalid `parent_id` on children endpoint.

- [ ] **Step 4: Re-run service/handler tests**

Run:
```bash
go test ./internal/handler ./internal/service -run "Tag" -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/tag_admin_service.go internal/service/tag_admin_service_test.go internal/handler/tag_handler.go internal/handler/tag_handler_test.go internal/handler/routes.go
git commit -m "feat: add lazy gallery tag browsing endpoints"
```

---

## Chunk 2: Backend image filtering semantics

### Task 3: Split exact-tag vs subtree-root filtering in repository layer

**Files:**
- Modify: `internal/repository/tag_hierarchy_query.go`
- Modify: `internal/repository/image_repository.go`
- Test: `internal/repository/image_repository_test.go`

- [ ] **Step 1: Write failing repository tests for mixed exact/subtree filters**
  - Add coverage for:
    - exact child/orphan tag should **not** auto-expand descendants
    - subtree root/parent tag **does** include descendants
    - exact + subtree together keep AND semantics
    - pending-tag variants honor the same split semantics

- [ ] **Step 2: Run tests to verify failure**

Run:
```bash
go test ./internal/repository -run "ImageRepository" -count=1
```
Expected: FAIL because repository methods only accept one `tagIDs` bucket today.

- [ ] **Step 3: Implement minimal repository refactor**
  - Introduce repository methods such as:
    - `FindByGalleryFilter(ctx, exactTagIDs, subtreeRootTagIDs []int64, ...)`
    - `CountByGalleryFilter(ctx, exactTagIDs, subtreeRootTagIDs []int64)`
    - pending-tag equivalents if needed by handler branching
  - Refactor hierarchy helper to expand descendants only for `subtreeRootTagIDs`.
  - Preserve current ordering / pagination behavior.

- [ ] **Step 4: Re-run repository tests**

Run:
```bash
go test ./internal/repository -run "ImageRepository" -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/tag_hierarchy_query.go internal/repository/image_repository.go internal/repository/image_repository_test.go
git commit -m "feat: split gallery exact and subtree tag filters"
```

### Task 4: Accept new gallery filter params in image handler

**Files:**
- Modify: `internal/handler/image_handler.go`
- Modify: `internal/service/search_service.go` (only if snapshot/viewer filter structs must stay in sync)
- Create/Modify: `internal/handler/image_handler_test.go`

- [ ] **Step 1: Write failing handler tests for new query parameters**
  - Cover:
    - `exact_tag_ids=1,2`
    - `subtree_root_tag_ids=5`
    - both parameters together
    - invalid combinations with `has_tags=false`
    - backward-compatible request with no new params

- [ ] **Step 2: Run tests to verify failure**

Run:
```bash
go test ./internal/handler -run "ImageHandler|ListImages" -count=1
```
Expected: FAIL because handler parsing still expects only `tag_ids`.

- [ ] **Step 3: Implement minimal handler changes**
  - Parse the two new query params.
  - Normalize empty lists.
  - Route list/count operations through the new repository methods.
  - Keep existing `has_tags` and `has_pending_tags` semantics intact.
  - If viewer snapshot code serializes gallery filters, extend it in the smallest possible way.

- [ ] **Step 4: Re-run handler tests**

Run:
```bash
go test ./internal/handler -run "ImageHandler|ListImages" -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/image_handler.go internal/handler/image_handler_test.go internal/service/search_service.go
git commit -m "feat: accept exact and subtree gallery filters"
```

---

## Chunk 3: Frontend service + provider state

### Task 5: Add lazy tree/orphan methods to Flutter tag service

**Files:**
- Modify: `flutter_app/lib/services/tag_service.dart`
- Test: `flutter_app/test/services/tag_service_test.dart`

- [ ] **Step 1: Write failing service tests**
  - Add tests for:
    - roots endpoint path
    - children endpoint query param
    - orphan endpoint path + paging/search params
    - error propagation for each new method

- [ ] **Step 2: Run Flutter service tests to verify failure**

Run:
```bash
cd flutter_app; flutter test test/services/tag_service_test.dart
```
Expected: FAIL because the new service methods do not exist yet.

- [ ] **Step 3: Implement minimal service methods**
  - Add typed helpers for roots, children, orphan page, and grouped search if the plan keeps search split client-side.
  - Do **not** delete legacy `getTree()` until all callers are migrated.

- [ ] **Step 4: Re-run Flutter service tests**

Run:
```bash
cd flutter_app; flutter test test/services/tag_service_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/services/tag_service.dart flutter_app/test/services/tag_service_test.dart
git commit -m "feat: add lazy gallery tag service methods"
```

### Task 6: Refactor TagProvider to support lazy tree + orphan state

**Files:**
- Modify: `flutter_app/lib/providers/tag_provider.dart`
- Possibly modify: `flutter_app/lib/models/gallery_filter_state.dart`
- Test: `flutter_app/test/providers/tag_provider_test.dart`

- [ ] **Step 1: Write failing provider tests**
  - Cover:
    - initial roots/orphans load
    - child-node load per parent
    - orphan paging append behavior
    - isolated error state for roots vs children vs orphans
    - search results grouped without destroying loaded state

- [ ] **Step 2: Run provider tests to verify failure**

Run:
```bash
cd flutter_app; flutter test test/providers/tag_provider_test.dart
```
Expected: FAIL because provider currently only supports `_tagTree` + `_allTags`.

- [ ] **Step 3: Implement minimal provider refactor**
  - Introduce separate state buckets for roots, children-by-parent, orphan page, node loading, node errors, and grouped search state.
  - Keep governance and image-tag behavior intact.
  - Avoid cross-contaminating governance tree state with gallery filter state unless there is already a safe shared abstraction.

- [ ] **Step 4: Re-run provider tests**

Run:
```bash
cd flutter_app; flutter test test/providers/tag_provider_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/providers/tag_provider.dart flutter_app/lib/models/gallery_filter_state.dart flutter_app/test/providers/tag_provider_test.dart
git commit -m "refactor: support lazy gallery tag provider state"
```

---

## Chunk 4: Frontend filter pane + image query wiring

### Task 7: Update gallery image API wiring for exact + subtree params

**Files:**
- Modify: `flutter_app/lib/services/api_service.dart`
- Modify: `flutter_app/lib/providers/image_provider.dart`
- Test: `flutter_app/test/services/api_service_test.dart`
- Test: `flutter_app/test/providers/image_provider_filter_state_test.dart`

- [ ] **Step 1: Write failing API/provider tests**
  - Add coverage for:
    - `exact_tag_ids`
    - `subtree_root_tag_ids`
    - combined serialization
    - `has_tags=false` normalization still clears both buckets
    - `ImageListProvider` forwards exact + subtree separately

- [ ] **Step 2: Run tests to verify failure**

Run:
```bash
cd flutter_app; flutter test test/services/api_service_test.dart test/providers/image_provider_filter_state_test.dart
```
Expected: FAIL because only `tag_ids` is serialized today.

- [ ] **Step 3: Implement minimal API/provider changes**
  - Extend `fetchImages()` signature.
  - Update `ImageListProvider.loadImages()` to pass both sets.
  - Preserve current sort / pagination behavior.

- [ ] **Step 4: Re-run tests**

Run:
```bash
cd flutter_app; flutter test test/services/api_service_test.dart test/providers/image_provider_filter_state_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/services/api_service.dart flutter_app/lib/providers/image_provider.dart flutter_app/test/services/api_service_test.dart flutter_app/test/providers/image_provider_filter_state_test.dart
git commit -m "feat: send exact and subtree gallery filters"
```

### Task 8: Rebuild FluentTagFilterPane around lazy tree + orphan sections

**Files:**
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
- Modify: `flutter_app/lib/screens/gallery_screen.dart`
- Test: `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`

- [ ] **Step 1: Write failing widget tests**
  - Cover:
    - roots and orphan section render independently
    - expanding a root loads children lazily
    - selecting root/parent marks subtree semantics
    - selecting orphan/child marks exact semantics
    - search groups results into hierarchical/orphan sections
    - tree load failure does not hide orphan section, and vice versa

- [ ] **Step 2: Run widget tests to verify failure**

Run:
```bash
cd flutter_app; flutter test test/widgets/fluent_tag_filter_pane_test.dart
```
Expected: FAIL because the pane still expects a full `tree` payload.

- [ ] **Step 3: Implement minimal UI rewrite**
  - Keep the draft/apply filter workflow introduced in recent commits.
  - Add selected-chip summary area for subtree vs exact labels.
  - Remove gallery dependence on `loadTagTree()` / `loadTags()` at panel init if no longer necessary.
  - Keep UI honest: orphan tags stay in their own collapsible section.

- [ ] **Step 4: Re-run widget tests**

Run:
```bash
cd flutter_app; flutter test test/widgets/fluent_tag_filter_pane_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/fluent_tag_filter_pane.dart flutter_app/lib/screens/gallery_screen.dart flutter_app/test/widgets/fluent_tag_filter_pane_test.dart
git commit -m "feat: add lazy tree gallery filter pane"
```

---

## Chunk 5: Verification and integration

### Task 9: Run focused backend verification

**Files:**
- No code changes

- [ ] **Step 1: Run focused backend suites**

Run:
```bash
go test ./internal/repository ./internal/handler ./internal/service -count=1
```
Expected: PASS.

- [ ] **Step 2: If failures appear, fix only regressions caused by this plan**

- [ ] **Step 3: Commit regression fixes (if any)**

```bash
git add <affected-files>
git commit -m "test: fix gallery tag filter regressions"
```

### Task 10: Run focused Flutter verification

**Files:**
- No code changes

- [ ] **Step 1: Run targeted Flutter tests**

Run:
```bash
cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/services/api_service_test.dart test/providers/image_provider_filter_state_test.dart test/widgets/fluent_tag_filter_pane_test.dart
```
Expected: PASS.

- [ ] **Step 2: Run analyzer on changed frontend files**

Run:
```bash
cd flutter_app; flutter analyze lib/services/tag_service.dart lib/providers/tag_provider.dart lib/services/api_service.dart lib/providers/image_provider.dart lib/widgets/fluent_tag_filter_pane.dart lib/screens/gallery_screen.dart
```
Expected: PASS.

- [ ] **Step 3: Commit regression fixes (if any)**

```bash
git add <affected-files>
git commit -m "test: verify lazy gallery tag filtering"
```

### Task 11: End-to-end manual check

**Files:**
- No code changes unless a bug is found

- [ ] **Step 1: Start backend and Flutter app**

Run:
```bash
go run cmd/server/main.go
```

In a second terminal:
```bash
cd flutter_app; flutter run -d windows
```

- [ ] **Step 2: Verify the workflow manually**
  - Open gallery filter drawer.
  - Confirm no `/api/v1/tags/tree` full-load request fires for the gallery path.
  - Expand a root; verify only child endpoint fires.
  - Scroll/load more orphan tags.
  - Select a root and confirm subtree matching returns descendants.
  - Select an orphan and confirm exact matching only.
  - Force one endpoint failure and confirm only the affected panel section errors.

- [ ] **Step 3: Document any manual verification notes before handoff**

---

Plan complete and saved to `docs/superpowers/plans/2026-04-18-gallery-tag-filter-lazy-tree.md`. Ready to execute?
