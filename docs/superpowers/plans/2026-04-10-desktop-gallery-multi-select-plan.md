# Desktop Gallery Multi-Select Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an explicit selection mode to the Windows Fluent desktop gallery so users can select multiple images, select all / deselect all, batch add tags, and batch delete, while preserving the existing batch AI tag behavior unchanged.

**Architecture:** Keep the existing `SelectionProvider` as the single source of truth for selection state, but move the Fluent desktop path to an explicit two-toolbar model: browse mode vs selection mode. Wire `FluentGalleryPage` to enter/exit selection mode, `FluentGalleryContent` to route card clicks into selection toggles when active, and `FluentImageCard` to render clear selected/unselected selection affordances. Reuse existing batch add and batch delete services, but normalize success/failure behavior around awaited refreshes and explicit `exitSelectionMode()`.

**Tech Stack:** Flutter, Fluent UI, Provider, existing `BatchService`, existing gallery/tag providers, Flutter widget tests.

---

## File Map

- Modify: `flutter_app/lib/app/fluent_screens.dart`
  - Add browse-mode vs selection-mode command bar switching.
  - Add explicit `选择` entry point.
  - Add selection-mode actions: `全不选`, `全选`, `批量添加标签`, `批量删除`, `退出选择模式`.
  - Keep existing `批量AI标签` only in browse mode.
  - Fix batch-add success path to await refresh then exit selection mode.
  - Add batch-delete confirmation + execution flow.
- Modify: `flutter_app/lib/widgets/fluent_gallery_content.dart`
  - Pass selection-mode state into cards and ensure selection mode prevents image-detail navigation.
- Modify: `flutter_app/lib/widgets/fluent_image_card.dart`
  - Keep existing selected overlay and add unselected selection affordance in selection mode.
- Modify: `flutter_app/lib/providers/selection_provider.dart`
  - Preserve existing API, but add tests and only minimal helpers if implementation friction requires them.
- Create: `flutter_app/test/providers/selection_provider_test.dart`
  - Lock selection-mode semantics: `enterSelectionMode`, `clearSelection`, `exitSelectionMode`, `selectAll`.
- Modify: `flutter_app/test/widgets/fluent_image_card_test.dart`
  - Cover selection-mode visuals and click behavior.
- Modify: `flutter_app/test/widgets/fluent_gallery_content_test.dart`
  - Cover selection-mode click handling and retention of right-click menu behavior.
- Modify: `flutter_app/test/app/fluent_screens_test.dart`
  - Cover toolbar switching, `选择` entry, `全选/全不选/退出选择模式`, browse-mode-only `批量AI标签`, batch add exit semantics, and batch delete confirm flow.
- Create: `flutter_app/test/services/batch_service_test.dart`
  - Add explicit API contract coverage for `batchDeleteImages()` if absent.

## Chunk 1: Selection state and card behavior

### Task 1: Lock `SelectionProvider` semantics with tests

**Files:**
- Create: `flutter_app/test/providers/selection_provider_test.dart`
- Modify: `flutter_app/lib/providers/selection_provider.dart` (only if helper changes are truly required)

- [ ] **Step 1: Write failing provider tests for selection-mode semantics**

```dart
test('clearSelection keeps selection mode active', () {
  final provider = SelectionProvider();
  provider.enterSelectionMode();
  provider.toggleSelection(1);

  provider.clearSelection();

  expect(provider.isSelectionMode, isTrue);
  expect(provider.selectedImageIds, isEmpty);
});

test('exitSelectionMode clears selection and disables mode', () {
  final provider = SelectionProvider();
  provider.enterSelectionMode();
  provider.selectAll([1, 2, 3]);

  provider.exitSelectionMode();

  expect(provider.isSelectionMode, isFalse);
  expect(provider.selectedImageIds, isEmpty);
});

test('selectAll selects the loaded image ids only', () {
  final provider = SelectionProvider();
  provider.enterSelectionMode();

  provider.selectAll([10, 11, 12]);

  expect(provider.selectedImageIds, {10, 11, 12});
});
```

- [ ] **Step 2: Run provider tests to verify initial failures or missing coverage**

Run: `flutter test test/providers/selection_provider_test.dart`
Expected: FAIL until the semantics are explicitly covered.

- [ ] **Step 3: Make only the minimal provider changes needed**

Implementation rules:
- Keep `clearSelection()` = clear ids only.
- Keep `exitSelectionMode()` = disable mode + clear ids.
- Do not add Ctrl/Shift behavior.
- Add helpers only if they directly simplify Fluent toolbar/card wiring, for example:

```dart
bool get hasAnySelection => _selectedImageIds.isNotEmpty;
```

- [ ] **Step 4: Re-run provider tests**

Run: `flutter test test/providers/selection_provider_test.dart`
Expected: PASS.

### Task 2: Make `FluentImageCard` visibly selectable in selection mode

**Files:**
- Modify: `flutter_app/lib/widgets/fluent_image_card.dart:10-153`
- Modify: `flutter_app/test/widgets/fluent_image_card_test.dart:1-102`

- [ ] **Step 1: Write failing widget tests for selection-mode visuals and click routing**

```dart
testWidgets('shows unselected selection affordance in selection mode', (tester) async { /* ... */ });
testWidgets('tapping card in selection mode calls onSelect instead of onTap', (tester) async { /* ... */ });
testWidgets('shows selected overlay and checkmark when selected', (tester) async { /* ... */ });
```

Required assertions:
- In selection mode, an unselected card still shows a weak checkbox marker.
- In selection mode, tapping the card toggles selection through `onSelect`.
- In browse mode, tapping still calls `onTap`.

- [ ] **Step 2: Run the card tests**

Run: `flutter test test/widgets/fluent_image_card_test.dart`
Expected: FAIL until new visual state/API is implemented.

- [ ] **Step 3: Implement the minimal `FluentImageCard` API change**

Add an explicit selection-mode flag instead of inferring everything from `onSelect != null`:

```dart
final bool isSelectionMode;

const FluentImageCard({
  super.key,
  required this.image,
  this.onTap,
  this.onSecondaryTapDown,
  this.borderRadius = 8.0,
  this.isSelected = false,
  this.isSelectionMode = false,
  this.onSelect,
});
```

Rendering rules:
- `isSelectionMode && isSelected` -> current strong overlay + checkmark.
- `isSelectionMode && !isSelected` -> add weak top-right selection affordance.
- `!isSelectionMode` -> preserve existing browse behavior and hover style.

- [ ] **Step 4: Re-run the card tests**

Run: `flutter test test/widgets/fluent_image_card_test.dart`
Expected: PASS.

### Task 3: Wire `FluentGalleryContent` so selection mode suppresses detail-open

**Files:**
- Modify: `flutter_app/lib/widgets/fluent_gallery_content.dart:18-336`
- Modify: `flutter_app/test/widgets/fluent_gallery_content_test.dart:1-284`

- [ ] **Step 1: Write failing widget tests for selection-mode click behavior**

```dart
testWidgets('clicking a card in selection mode toggles selection instead of opening detail', (tester) async { /* ... */ });
testWidgets('right click still opens context menu while selection mode is active', (tester) async { /* ... */ });
```

Test setup requirements:
- Provide `SelectionProvider` explicitly.
- Put the widget under a `FluentApp` + `Material` wrapper, matching current test patterns.

- [ ] **Step 2: Run the content tests**

Run: `flutter test test/widgets/fluent_gallery_content_test.dart`
Expected: FAIL until selection mode is fully wired for Fluent cards.

- [ ] **Step 3: Implement Fluent gallery selection routing**

Required wiring in `_buildImageList(...)`:

```dart
final isSelectionMode = selection.isSelectionMode;

return FluentImageCard(
  image: image,
  isSelectionMode: isSelectionMode,
  isSelected: selection.isSelected(image.id),
  onTap: isSelectionMode ? null : widget.onImageTap,
  onSelect: (img, selected) => selection.toggleSelection(img.id),
  onSecondaryTapDown: (img, details) => _showImageContextMenu(img, details.globalPosition),
);
```

Do not add long-press entry on Fluent desktop.

- [ ] **Step 4: Re-run the content tests**

Run: `flutter test test/widgets/fluent_gallery_content_test.dart`
Expected: PASS.

## Chunk 2: Fluent toolbar switching and batch operations

### Task 4: Convert `FluentGalleryPage` to browse-mode vs selection-mode toolbars

**Files:**
- Modify: `flutter_app/lib/app/fluent_screens.dart:55-117, 149-168, 478-538`
- Modify: `flutter_app/test/app/fluent_screens_test.dart:61-458`

- [ ] **Step 1: Write failing page tests for toolbar switching**

```dart
testWidgets('browse mode shows 选择 and 批量AI标签 but not selection actions', (tester) async { /* ... */ });
testWidgets('entering selection mode hides browse toolbar items and shows selection toolbar', (tester) async { /* ... */ });
testWidgets('全不选 keeps selection mode active while clearing selected count', (tester) async { /* ... */ });
testWidgets('退出选择模式 clears selection and restores browse toolbar', (tester) async { /* ... */ });
```

Critical assertions:
- Browse mode: `排序`, `刷新`, `批量AI标签`, `选择` are visible.
- Selection mode: `全不选`, `全选`, `批量添加标签`, `批量删除`, `退出选择模式` are visible.
- Selection mode: `批量AI标签` is hidden.
- Zero-selection state: `批量添加标签` and `批量删除` are disabled; `全选` and `退出选择模式` remain enabled.

- [ ] **Step 2: Run the page tests**

Run: `flutter test test/app/fluent_screens_test.dart --plain-name "selection mode"`
Expected: FAIL until the toolbar split is implemented.

- [ ] **Step 3: Implement explicit toolbar builders**

Refactor `FluentGalleryPage` to use two helpers:

```dart
List<CommandBarItem> _buildBrowseModeItems(...)
List<CommandBarItem> _buildSelectionModeItems(...)
```

Browse-mode items must include:
- `排序`
- `刷新`
- `批量AI标签`
- `选择`

Selection-mode items must include:
- `全不选`
- `全选`
- `批量添加标签`
- `批量删除`
- `退出选择模式`

Behavior mapping:
- `选择` -> `selection.enterSelectionMode()`
- `全不选` -> `selection.clearSelection()`
- `全选` -> `selection.selectAll(imageProvider.images.map((e) => e.id).toList())`
- `退出选择模式` -> `selection.exitSelectionMode()`

- [ ] **Step 4: Re-run the page tests**

Run: `flutter test test/app/fluent_screens_test.dart`
Expected: PASS for toolbar-switching coverage.

### Task 5: Normalize batch add flow around awaited refresh + exit mode

**Files:**
- Modify: `flutter_app/lib/app/fluent_screens.dart:478-538`
- Modify: `flutter_app/test/app/fluent_screens_test.dart`

- [ ] **Step 1: Add a failing test for batch-add success semantics**

```dart
testWidgets('successful batch add refreshes, exits selection mode, and restores browse toolbar', (tester) async { /* ... */ });
```

Required assertions:
- Uses selected ids from `SelectionProvider`.
- Waits for refresh completion before returning to browse mode.
- Restores browse-mode toolbar items after success.
- Failure preserves selection mode and selected ids.

- [ ] **Step 2: Run the failing batch-add test**

Run: `flutter test test/app/fluent_screens_test.dart --plain-name "batch add"`
Expected: FAIL because current code only calls `selection.clearSelection()` and does not await refresh.

- [ ] **Step 3: Implement minimal batch-add flow correction**

Change the success path from:

```dart
selection.clearSelection();
imageProvider.loadImages(refresh: true);
```

to:

```dart
await imageProvider.loadImages(refresh: true);
if (!context.mounted) return;
selection.exitSelectionMode();
```

Do not change the existing tag picker dialog contract.

- [ ] **Step 4: Re-run the batch-add test**

Run: `flutter test test/app/fluent_screens_test.dart --plain-name "batch add"`
Expected: PASS.

### Task 6: Add batch delete service coverage and Fluent delete flow

**Files:**
- Create: `flutter_app/test/services/batch_service_test.dart`
- Modify: `flutter_app/lib/app/fluent_screens.dart`
- Modify: `flutter_app/test/app/fluent_screens_test.dart`

- [ ] **Step 1: Write failing service + page tests for batch delete**

```dart
test('batchDeleteImages posts selected image ids to /batch/images/delete', () async { /* ... */ });
testWidgets('batch delete confirms, calls service, refreshes, and exits selection mode', (tester) async { /* ... */ });
testWidgets('batch delete failure keeps selection mode active', (tester) async { /* ... */ });
```

The widget tests must also verify:
- Delete confirmation dialog shows selected-count-aware copy.
- Actions are `取消` and `确认删除`.
- `批量AI标签` remains untouched and absent from selection mode.
- If backend still only supports all-or-nothing delete responses, assert the current full-failure path and leave a TODO comment in the implementation for richer partial-result messaging.

- [ ] **Step 2: Run the failing delete tests**

Run: `flutter test test/services/batch_service_test.dart test/app/fluent_screens_test.dart --plain-name "batch delete"`
Expected: FAIL until the delete flow exists.

- [ ] **Step 3: Implement minimal batch-delete flow**

Add a helper in `FluentGalleryPage` such as:

```dart
Future<void> _confirmAndDeleteSelectedImages(
  BuildContext context,
  SelectionProvider selection,
  ImageListProvider imageProvider,
) async
```

Implementation rules:
- Use `selection.selectedImageIds.toList()` as payload.
- Instantiate `BatchService(baseUrl: context.read<ConfigProvider>().baseUrl)` unless a test seam is added.
- Show a destructive `ContentDialog` before the API call.
- On success: `await imageProvider.loadImages(refresh: true)` then `selection.exitSelectionMode()`.
- On failure: show error dialog and keep selection state intact.
- If backend returns only `images_deleted` today, do not invent partial-success parsing on the client; document the limitation inline and keep the UI message honest.

- [ ] **Step 4: Re-run the delete tests**

Run: `flutter test test/services/batch_service_test.dart test/app/fluent_screens_test.dart`
Expected: PASS.

## Chunk 3: Regression and desktop verification

### Task 7: Run targeted verification for touched Flutter surfaces

**Files:**
- Verify only (no new files)

- [ ] **Step 1: Run targeted Flutter tests**

Run: `flutter test test/providers/selection_provider_test.dart test/widgets/fluent_image_card_test.dart test/widgets/fluent_gallery_content_test.dart test/app/fluent_screens_test.dart test/services/batch_service_test.dart`
Expected: PASS.

- [ ] **Step 2: Run analyzer on touched files**

Run: `flutter analyze lib/app/fluent_screens.dart lib/widgets/fluent_gallery_content.dart lib/widgets/fluent_image_card.dart lib/providers/selection_provider.dart test/providers/selection_provider_test.dart test/widgets/fluent_image_card_test.dart test/widgets/fluent_gallery_content_test.dart test/app/fluent_screens_test.dart test/services/batch_service_test.dart`
Expected: No analyzer errors.

- [ ] **Step 3: Manual QA on Windows desktop runtime**

Run: `flutter run -d windows`

Manual QA checklist:
- Browse mode shows `排序 / 刷新 / 批量AI标签 / 选择`.
- Clicking `选择` enters selection mode and hides `批量AI标签`.
- Clicking cards in selection mode toggles visible selection state instead of opening detail.
- `全选` selects all currently loaded cards.
- `全不选` clears selection but remains in selection mode.
- `退出选择模式` returns to browse mode.
- `批量添加标签` succeeds, refreshes, and returns to browse mode.
- `批量删除` shows confirmation, succeeds, refreshes, and returns to browse mode.
- Existing browse-mode `批量AI标签` still works exactly as before.

- [ ] **Step 4: Capture final regression notes in the implementation PR/summary**

Include:
- Selection mode scope = currently loaded images only.
- Batch AI tag behavior intentionally unchanged.
- No Ctrl/Shift multi-select included in this phase.
