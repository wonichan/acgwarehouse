# Image Detail Tag Picker Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the image-detail Add/Edit Tag dialogs show existing tags immediately, sorted by existing usage priority, support scrolling to load more tags, and still keep fuzzy search plus create-new behavior.

**Architecture:** Reuse the existing `/api/v1/tags` pagination contract already exposed through `TagService.fetchTags()`, and add a shared Flutter widget/state helper for the dialog result area so Add/Edit dialogs do not fork into two separate pagination/search state machines. Keep search mode separate from default-list mode: empty query shows paginated default tags with infinite scroll; non-empty query shows fuzzy search results only.

**Tech Stack:** Flutter Web, Provider, existing `TagService`, existing Go `/api/v1/tags` backend, Flutter widget tests, `flutter test`

---

## File Map

- **Modify:** `flutter_app/lib/widgets/add_tag_dialog.dart`
  - Replace input-only flow with shared default-list/search-results UI.
- **Modify:** `flutter_app/lib/widgets/edit_tag_dialog.dart`
  - Same behavior as add dialog, but preserve replace-tag return contract.
- **Create:** `flutter_app/lib/widgets/tag_picker_results_panel.dart`
  - Shared list panel for default tags, search results, load-more spinner, empty/error states, and row rendering.
- **Modify:** `flutter_app/lib/services/tag_service.dart`
  - Keep API contract stable; only extend/clarify fetch behavior if tests reveal gaps.
- **Create:** `flutter_app/test/widgets/add_tag_dialog_test.dart`
  - Add dialog regression coverage for default list, search mode, load-more, and create-new fallback.
- **Modify:** `flutter_app/test/widgets/edit_tag_dialog_test.dart`
  - Extend current test suite to cover default list and load-more behavior.
- **Modify:** `flutter_app/test/services/tag_service_test.dart`
  - Add fetchTags contract coverage for `limit`, `offset`, and `search` query parameters if not already covered.
- **Modify:** `flutter_app/tool/manual_qa_image_detail.dart`
  - Extend fake HTTP client so manual QA can exercise default tag list + scrolling + search.

## Constraints / Existing Patterns

- `internal/repository/tag_repository.go` already sorts `FindAll()` and `FindByLabelLike()` by `usage_count DESC, id ASC`.
- `internal/handler/tag_handler.go` already exposes `GET /api/v1/tags?limit=&offset=&search=`.
- `flutter_app/lib/services/tag_service.dart` already exposes `fetchTags({search, limit, offset})` and `searchTags(query)`.
- `AddTagDialog` currently mutates directly on selection; keep that behavior.
- `EditTagDialog` currently returns a structured result map; keep that behavior unchanged.
- Do **not** add new backend endpoints unless tests prove the existing pagination/search contract is insufficient.

## Chunk 1: Shared default-list/search UI foundation

### Task 1: Add service-level contract coverage for paginated tag fetching

**Files:**
- Modify: `flutter_app/test/services/tag_service_test.dart`
- Modify: `flutter_app/lib/services/tag_service.dart` (only if needed to make tests pass)

- [ ] **Step 1: Write the failing test**

Add tests that verify:

```dart
test('fetchTags sends limit and offset for default tag paging', () async {
  when(mockClient.get(any)).thenAnswer(
    (_) async => http.Response('{"tags":[],"total":0}', 200),
  );

  await tagService.fetchTags(limit: 30, offset: 60);

  final captured = verify(mockClient.get(captureAny)).captured.single as Uri;
  expect(captured.path, '/api/v1/tags');
  expect(captured.queryParameters['limit'], '30');
  expect(captured.queryParameters['offset'], '60');
  expect(captured.queryParameters.containsKey('search'), isFalse);
});

test('fetchTags includes search query when provided', () async {
  when(mockClient.get(any)).thenAnswer(
    (_) async => http.Response('{"tags":[],"total":0}', 200),
  );

  await tagService.fetchTags(search: 'hair', limit: 20, offset: 0);

  final captured = verify(mockClient.get(captureAny)).captured.single as Uri;
  expect(captured.queryParameters['search'], 'hair');
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test test/services/tag_service_test.dart`
Expected: FAIL only if query parameter behavior is missing or incorrectly asserted. If it already passes, keep the tests as regression coverage and move on.

- [ ] **Step 3: Write minimal implementation**

If failure occurs, adjust only `TagService.fetchTags()` so it preserves existing response parsing while sending the expected query parameters. Do not change its public return type.

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test test/services/tag_service_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/test/services/tag_service_test.dart flutter_app/lib/services/tag_service.dart
git commit -m "test: cover tag paging contract"
```

### Task 2: Introduce shared tag results panel for dialogs

**Files:**
- Create: `flutter_app/lib/widgets/tag_picker_results_panel.dart`
- Test: `flutter_app/test/widgets/add_tag_dialog_test.dart`
- Test: `flutter_app/test/widgets/edit_tag_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

First write widget tests that assert the new shared behavior from the dialog surface, not private internals:

```dart
testWidgets('shows default tags immediately when add dialog opens', (tester) async {
  when(() => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 0))
      .thenAnswer((_) async => [popularTag, anotherTag]);

  await tester.pumpWidget(createAddDialogHost());
  await tester.tap(find.text('Open Dialog'));
  await tester.pump();

  expect(find.text(popularTag.preferredLabel), findsOneWidget);
  expect(find.text(anotherTag.preferredLabel), findsOneWidget);
});
```

Add parallel tests for edit dialog.

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test test/widgets/add_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart`
Expected: FAIL because dialogs currently show nothing until typing.

- [ ] **Step 3: Write minimal implementation**

Create a focused shared widget that accepts:

- current mode (`defaultList` vs `search`)
- items to render
- loading flags (`isInitialLoading`, `isLoadingMore`, `isSearching`)
- `hasMore`
- row tap callback
- retry callback / empty-state copy
- optional `ScrollController`

Keep it presentational; dialog state should remain in the parent dialog widgets.

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test test/widgets/add_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart`
Expected: PASS for the new default-list visibility assertions after dialogs are wired.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/tag_picker_results_panel.dart flutter_app/test/widgets/add_tag_dialog_test.dart flutter_app/test/widgets/edit_tag_dialog_test.dart
git commit -m "feat: add shared tag picker results panel"
```

## Chunk 2: Add dialog default tags + scroll loading

### Task 3: Implement AddTagDialog default tag loading and search mode switching

**Files:**
- Modify: `flutter_app/lib/widgets/add_tag_dialog.dart`
- Create: `flutter_app/test/widgets/add_tag_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

Add tests for these behaviors one by one:

```dart
testWidgets('loads default tags on open before typing', ...)
testWidgets('tapping default tag immediately adds it', ...)
testWidgets('typing switches to search results', ...)
testWidgets('clearing query restores previously loaded default tags', ...)
testWidgets('shows create-new path when no matching tag is selected', ...)
```

Use a mock `TagService` and verify calls to:

- `fetchTags(limit: pageSize, offset: 0)` on dialog open
- `searchTags('hair')` after text entry
- `addImageTag(imageId, tagId: ...)` on existing-tag selection
- `addImageTag(imageId, tagLabel: ...)` on create-new

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test test/widgets/add_tag_dialog_test.dart`
Expected: FAIL because dialog lacks default fetch, mode switching, and restore behavior.

- [ ] **Step 3: Write minimal implementation**

In `AddTagDialog`:

- add dialog-owned state for:
  - `_defaultTags`
  - `_searchResults`
  - `_isInitialLoading`
  - `_isLoadingMore`
  - `_isSearching`
  - `_hasMoreDefaultTags`
  - `_defaultOffset`
  - `ScrollController`
- trigger initial `fetchTags(limit: pageSize, offset: 0)` from `initState()`
- on empty query, render default-list mode
- on non-empty query, render search mode using existing `searchTags()`
- when query becomes empty, do **not** discard loaded default tags; switch back to them
- preserve current `Navigator.pop(...)` result contract and current error handling pattern

Suggested default page size: `20`

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test test/widgets/add_tag_dialog_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/add_tag_dialog.dart flutter_app/test/widgets/add_tag_dialog_test.dart
git commit -m "feat: add default tag picker to add dialog"
```

### Task 4: Add AddTagDialog incremental load-more behavior

**Files:**
- Modify: `flutter_app/lib/widgets/add_tag_dialog.dart`
- Create/Modify: `flutter_app/test/widgets/add_tag_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

Add a widget test that opens the add dialog with two paged responses:

```dart
testWidgets('loads more default tags when scrolled near bottom', (tester) async {
  when(() => mockTagService.fetchTags(limit: 20, offset: 0))
      .thenAnswer((_) async => firstPageTags);
  when(() => mockTagService.fetchTags(limit: 20, offset: 20))
      .thenAnswer((_) async => secondPageTags);

  await tester.pumpWidget(createAddDialogHost());
  await tester.tap(find.text('Open Dialog'));
  await tester.pumpAndSettle();

  await tester.drag(find.byType(ListView).first, const Offset(0, -1000));
  await tester.pump();

  expect(find.text(secondPageTags.first.preferredLabel), findsOneWidget);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test test/widgets/add_tag_dialog_test.dart`
Expected: FAIL because no load-more path exists.

- [ ] **Step 3: Write minimal implementation**

Add scroll listener logic that:

- only runs in default-list mode
- exits early when already loading or `hasMore == false`
- requests the next page with `offset = _defaultTags.length`
- appends the results
- sets `hasMore = fetched.length == pageSize`

Do not implement infinite scroll for search mode.

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test test/widgets/add_tag_dialog_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/add_tag_dialog.dart flutter_app/test/widgets/add_tag_dialog_test.dart
git commit -m "feat: support add dialog tag load more"
```

## Chunk 3: Edit dialog parity + manual QA

### Task 5: Implement EditTagDialog default tags and load-more behavior

**Files:**
- Modify: `flutter_app/lib/widgets/edit_tag_dialog.dart`
- Modify: `flutter_app/test/widgets/edit_tag_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

Add/extend tests for:

```dart
testWidgets('shows default tags immediately when edit dialog opens', ...)
testWidgets('tapping default tag returns existing-tag replacement payload', ...)
testWidgets('scrolling default list loads additional tags', ...)
testWidgets('typing uses search results and clearing restores default list', ...)
```

Keep the existing return contract:

```dart
{
  'tagId': 2,
  'tagLabel': null,
  'label': 'red hair',
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test test/widgets/edit_tag_dialog_test.dart`
Expected: FAIL because edit dialog is still input-first only.

- [ ] **Step 3: Write minimal implementation**

Mirror the add-dialog state machine in `EditTagDialog`, but keep:

- current-label context text
- existing structured result map
- create-new behavior unchanged

Use the same page size and shared results panel to keep behavior consistent.

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test test/widgets/edit_tag_dialog_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/edit_tag_dialog.dart flutter_app/test/widgets/edit_tag_dialog_test.dart
git commit -m "feat: add default tag picker to edit dialog"
```

### Task 6: Manual QA harness and final verification

**Files:**
- Modify: `flutter_app/tool/manual_qa_image_detail.dart`
- Modify: `flutter_app/lib/widgets/add_tag_dialog.dart`
- Modify: `flutter_app/lib/widgets/edit_tag_dialog.dart`
- Modify: `flutter_app/lib/widgets/tag_picker_results_panel.dart`

- [ ] **Step 1: Write the failing test**

If the manual QA harness lacks `/api/v1/tags` fake responses, add a focused test or smoke assertion in widget tests that expects the default list to render from paged API data. If existing widget tests already cover this, skip new automated coverage here and use the manual QA harness task as validation only.

- [ ] **Step 2: Run test to verify it fails**

Run the most relevant widget suite if a new failing test was added:

`flutter test test/widgets/add_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart`

Expected: FAIL only if the harness/data path is still incomplete.

- [ ] **Step 3: Write minimal implementation**

Update `manual_qa_image_detail.dart` fake client so it can return:

- first page of `/api/v1/tags?limit=20&offset=0`
- second page for `/api/v1/tags?limit=20&offset=20`
- search results for `/api/v1/tags?search=...`

If useful, add a QA launcher button or host widget that opens `AddTagDialog` and `EditTagDialog` directly; keep this limited to manual QA tooling.

- [ ] **Step 4: Run test to verify it passes**

Automated:

`flutter test test/services/tag_service_test.dart test/widgets/add_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart`

Expected: PASS

Manual QA:

`flutter run -d chrome tool/manual_qa_image_detail.dart`

Verify in the browser:

- Add dialog opens with default tags visible
- Scrolling loads more tags
- Searching replaces the default list with filtered results
- Clearing the query restores the loaded default list
- Existing-tag click still adds/replaces immediately
- Create-new still works

- [ ] **Step 5: Commit**

```bash
git add flutter_app/tool/manual_qa_image_detail.dart flutter_app/lib/widgets/add_tag_dialog.dart flutter_app/lib/widgets/edit_tag_dialog.dart flutter_app/lib/widgets/tag_picker_results_panel.dart flutter_app/test/services/tag_service_test.dart flutter_app/test/widgets/add_tag_dialog_test.dart flutter_app/test/widgets/edit_tag_dialog_test.dart
git commit -m "feat: improve image detail tag picker dialogs"
```
