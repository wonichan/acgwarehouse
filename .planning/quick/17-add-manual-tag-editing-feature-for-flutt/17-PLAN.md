---
phase: 17-add-manual-tag-editing-feature-for-flutt
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - flutter_app/lib/widgets/edit_tag_dialog.dart
  - flutter_app/lib/screens/image_detail_screen.dart
  - flutter_app/lib/providers/tag_provider.dart
  - flutter_app/lib/services/tag_service.dart
  - flutter_app/test/widgets/edit_tag_dialog_test.dart
autonomous: true
requirements: []
must_haves:
  truths:
    - "User can click edit icon on pending tag"
    - "Dialog shows search input and current tag"
    - "User can select existing tag from search results"
    - "User can create new tag with custom name"
    - "Tag replacement is atomic (old removed, new added)"
    - "Success feedback shown to user"
  artifacts:
    - path: "flutter_app/lib/widgets/edit_tag_dialog.dart"
      provides: "Tag editing dialog widget"
      min_lines: 80
    - path: "flutter_app/lib/screens/image_detail_screen.dart"
      provides: "Edit icon integration"
      contains: "_showEditTagDialog"
  key_links:
    - from: "image_detail_screen._buildPendingTagChip"
      to: "EditTagDialog"
      via: "InkWell.onTap -> showDialog"
      pattern: "_showEditTagDialog"
    - from: "EditTagDialog"
      to: "TagService.mergeImageTag"
      via: "API call"
      pattern: "mergeImageTag"
---

# Plan: Add Manual Tag Editing Feature for Flutter Image Detail Screen

## Context

**User Request:** Add manual tag editing functionality to the Flutter image detail screen. Users can click on a tag and change it to a different tag via a dialog.

**Current State:**
- Pending tags have 3 actions: confirm (✓), reject (✗), merge (merge icon)
- Confirmed tags only have delete action
- Backend already has `POST /api/v1/images/:id/tags/:tag_id/merge` endpoint that supports tag replacement

**Proposed Solution:**
- Add an "edit" (pencil) icon to pending tag chips
- Clicking opens an `EditTagDialog` allowing users to:
  1. Search and select an existing tag
  2. Or type a new tag name to create it
- Backend's merge endpoint handles the replacement atomically

---

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1: Create EditTagDialog widget | None | New widget, no prerequisites |
| Task 2: Add edit icon to pending tag chips | Task 1 | Uses the EditTagDialog |
| Task 3: Integration test & verify | Task 2 | Tests the complete feature |

---

## Parallel Execution Graph

Wave 1 (Start immediately):
├── Task 1: Create EditTagDialog widget (no dependencies)

Wave 2 (After Wave 1 completes):
├── Task 2: Add edit icon to pending tag chips (depends: Task 1)

Wave 3 (After Wave 2 completes):
└── Task 3: Integration test & verify (depends: Task 2)

**Critical Path:** Task 1 → Task 2 → Task 3

---

## Tasks

### Task 1: Create EditTagDialog Widget

**Description:** Create a reusable dialog widget for editing tags, supporting both existing tag selection and new tag creation.

**Delegation Recommendation:**
- Category: `visual-engineering` - Flutter UI component creation with search/autocomplete functionality
- Skills: [`frontend-ui-ux`, `test-driven-development`] - UI component with testable behavior

**Skills Evaluation:**
- INCLUDED `frontend-ui-ux`: UI/UX component creation is the core task
- INCLUDED `test-driven-development`: Widget has testable behavior (search, selection, creation)
- OMITTED `git-master`: Single commit strategy, no complex git operations needed
- OMITTED `systematic-debugging`: No bug to fix, this is new feature development

**Depends On:** None

**Files:**
- `flutter_app/lib/widgets/edit_tag_dialog.dart` (new)
- `flutter_app/test/widgets/edit_tag_dialog_test.dart` (new)

**Behavior (TDD):**
```dart
// Test Case 1: Dialog shows search input with initial tag label
testWidgets('shows search input with initial tag label', (tester) async {
  await tester.pumpWidget(MaterialApp(
    home: Scaffold(body: EditTagDialog(imageId: 1, currentTag: testTag)),
  ));
  expect(find.text('编辑标签'), findsOneWidget);
  expect(find.text('blue hair'), findsOneWidget); // current tag label
});

// Test Case 2: Searching filters tag list
testWidgets('searching filters tag list', (tester) async {
  // Type in search field
  // Verify API call made
  // Verify results displayed
});

// Test Case 3: Selecting existing tag calls onTagSelected
testWidgets('selecting existing tag calls onTagSelected', (tester) async {
  // Tap on a tag from search results
  // Verify callback invoked with selected tag
});

// Test Case 4: Creating new tag calls onNewTagCreated
testWidgets('creating new tag calls onNewTagCreated', (tester) async {
  // Type new tag name
  // Tap "Create" button
  // Verify callback invoked with new tag label
});
```

**Action:**
1. Create `EditTagDialog` StatefulWidget in `flutter_app/lib/widgets/edit_tag_dialog.dart`
2. Follow the pattern from `_MergeTagDialog` and `AddTagDialog` for consistency
3. Include:
   - Text showing current tag label ("Change 'blue hair' to:")
   - Search TextField with autocomplete
   - List of matching tags from TagService.searchTags()
   - "Create new tag" button for custom input
4. Return selected tag or new tag label via Navigator.pop()
5. Create widget tests in `flutter_app/test/widgets/edit_tag_dialog_test.dart`

**Verify:**
```bash
cd flutter_app && flutter test test/widgets/edit_tag_dialog_test.dart
```

**Done:**
- [ ] `EditTagDialog` widget created with search and create functionality
- [ ] Widget tests pass (4+ test cases)
- [ ] Follows existing `AddTagDialog` patterns for consistency

---

### Task 2: Add Edit Icon to Pending Tag Chips

**Description:** Modify the pending tag chip UI to include an edit (pencil) icon, and wire up the EditTagDialog.

**Delegation Recommendation:**
- Category: `visual-engineering` - UI modification with dialog integration
- Skills: [`frontend-ui-ux`] - UI integration work

**Skills Evaluation:**
- INCLUDED `frontend-ui-ux`: UI modification and integration
- OMITTED `test-driven-development`: Integration covered by Task 3
- OMITTED `systematic-debugging`: No bug to fix

**Depends On:** Task 1

**Files:**
- `flutter_app/lib/screens/image_detail_screen.dart` (modify)

**Action:**
1. In `_buildPendingTagChip()` method (line 449-477), add an edit icon:
   ```dart
   InkWell(
     onTap: () => _showEditTagDialog(tag),
     child: const Icon(Icons.edit, size: 18, color: Colors.orange),
   ),
   ```
2. Add `_showEditTagDialog(Tag tag)` method:
   ```dart
   Future<void> _showEditTagDialog(Tag tag) async {
     final result = await showDialog<Map<String, dynamic>>(
       context: context,
       builder: (context) => EditTagDialog(
         imageId: widget.image.id,
         currentTag: tag,
       ),
     );
     
     if (result != null && mounted) {
       try {
         if (result['tagId'] != null) {
           // Selected existing tag
           await _tagProvider.mergeImageTag(
             widget.image.id, 
             tag.id, 
             result['tagId']
           );
         } else if (result['tagLabel'] != null) {
           // Create new tag
           await _tagProvider.mergeImageTag(
             widget.image.id,
             tag.id,
             targetLabel: result['tagLabel']
           );
         }
         await _loadImageTags();
         if (mounted) {
           ScaffoldMessenger.of(context).showSnackBar(
             SnackBar(content: Text('标签已更新为: ${result['label']}')),
           );
         }
       } catch (e) {
         if (mounted) {
           ScaffoldMessenger.of(context).showSnackBar(
             SnackBar(content: Text('更新标签失败: $e')),
           );
         }
       }
     }
   }
   ```
3. Update `TagProvider` and `TagService` if `mergeImageTag` doesn't support `targetLabel` parameter

**Verify:**
```bash
cd flutter_app && flutter analyze lib/screens/image_detail_screen.dart
```

**Done:**
- [ ] Edit icon visible on pending tag chips
- [ ] Clicking edit icon opens EditTagDialog
- [ ] Selecting/creating tag updates the image's tag via merge API
- [ ] Success/error snackbars shown appropriately

---

### Task 3: Integration Test & Verify

**Description:** Write an integration test for the complete edit tag flow and manually verify the feature works end-to-end.

**Delegation Recommendation:**
- Category: `quick` - Simple verification task
- Skills: [`test-driven-development`] - Integration testing

**Skills Evaluation:**
- INCLUDED `test-driven-development`: Integration test creation
- OMITTED `frontend-ui-ux`: No UI creation, just testing

**Depends On:** Task 2

**Files:**
- `flutter_app/integration_test/edit_tag_test.dart` (new, optional)

**Action:**
1. Run existing Flutter tests to ensure no regressions:
   ```bash
   cd flutter_app && flutter test
   ```
2. Manually test the complete flow:
   - Open image detail screen with pending AI tags
   - Click the edit icon on a pending tag
   - Search for an existing tag, select it
   - Verify the tag is replaced
   - Click edit again, type a new tag name
   - Verify the new tag is created and assigned
3. Run Flutter analyzer:
   ```bash
   cd flutter_app && flutter analyze
   ```

**Verify:**
```bash
cd flutter_app && flutter test && flutter analyze
```

**Done:**
- [ ] All existing tests pass
- [ ] Flutter analyzer shows no issues
- [ ] Manual testing confirms edit flow works
- [ ] Success snackbar shows correct tag name
- [ ] Error handling works (network failure simulation)

---

## Commit Strategy

**Atomic commits by task completion:**

1. **After Task 1:**
   ```bash
   git add flutter_app/lib/widgets/edit_tag_dialog.dart flutter_app/test/widgets/edit_tag_dialog_test.dart
   git commit -m "feat(flutter): add EditTagDialog widget for manual tag editing"
   ```

2. **After Task 2:**
   ```bash
   git add flutter_app/lib/screens/image_detail_screen.dart flutter_app/lib/providers/tag_provider.dart flutter_app/lib/services/tag_service.dart
   git commit -m "feat(flutter): integrate EditTagDialog into pending tag chips"
   ```

3. **After Task 3:**
   ```bash
   git add flutter_app/integration_test/edit_tag_test.dart  # if created
   git commit -m "test(flutter): add integration test for edit tag feature"
   ```

---

## Success Criteria

- [ ] User can click edit icon on any pending tag
- [ ] EditTagDialog opens with search functionality
- [ ] User can select existing tag from search results
- [ ] User can create new tag by typing custom name
- [ ] Original tag is replaced with selected/created tag
- [ ] Success/error feedback shown via snackbar
- [ ] All Flutter tests pass
- [ ] Flutter analyzer shows no issues

---

## Must-Haves (Goal-Backward Verification)

```yaml
must_haves:
  truths:
    - "User can click edit icon on pending tag"
    - "Dialog shows search input and current tag"
    - "User can select existing tag from search results"
    - "User can create new tag with custom name"
    - "Tag replacement is atomic (old removed, new added)"
    - "Success feedback shown to user"
  artifacts:
    - path: "flutter_app/lib/widgets/edit_tag_dialog.dart"
      provides: "Tag editing dialog widget"
      min_lines: 80
    - path: "flutter_app/lib/screens/image_detail_screen.dart"
      provides: "Edit icon integration"
      contains: "_showEditTagDialog"
  key_links:
    - from: "image_detail_screen._buildPendingTagChip"
      to: "EditTagDialog"
      via: "InkWell.onTap -> showDialog"
      pattern: "_showEditTagDialog"
    - from: "EditTagDialog"
      to: "TagService.mergeImageTag"
      via: "API call"
      pattern: "mergeImageTag"
```

---

## Output

After completion, create `.planning/quick/17-add-manual-tag-editing-feature-for-flutt/17-SUMMARY.md`