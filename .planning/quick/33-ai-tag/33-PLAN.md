---
phase: 33-ai-tag
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - flutter_app/lib/widgets/batch_operation_sheet.dart
  - flutter_app/lib/screens/gallery_screen.dart
  - flutter_app/lib/widgets/image_grid.dart
  - flutter_app/lib/widgets/image_masonry.dart
autonomous: true
requirements: [BATCH-AI-TAGS]
user_setup: []
must_haves:
  truths:
    - "User can long-press an image to enter selection mode"
    - "User can select multiple images with checkboxes"
    - "User can trigger batch AI tag generation from selection"
    - "Generated tags appear as pending for user confirmation"
    - "Progress visible in admin dashboard job list"
  artifacts:
    - path: flutter_app/lib/widgets/batch_operation_sheet.dart
      provides: "Batch operation UI with AI tags button"
      contains: "onGenerateAITags"
    - path: flutter_app/lib/screens/gallery_screen.dart
      provides: "Gallery with selection mode"
      contains: "SelectionProvider"
  key_links:
    - from: flutter_app/lib/screens/gallery_screen.dart
      to: flutter_app/lib/services/tag_service.dart
      via: "batchTriggerAITags call"
---

<objective>
Add batch AI tag generation functionality to the gallery UI.

Purpose: Allow users to select multiple images and trigger AI tag generation in bulk, with progress visible in admin dashboard.
Output: Working batch AI tag generation from gallery selection.
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

## Existing Infrastructure

**Backend (Already Complete):**
- `POST /api/v1/images/batch-ai-tags` - Creates individual jobs per image
- Jobs visible in admin dashboard at `/admin`

**Frontend Services:**
- `tag_service.dart` has `batchTriggerAITags(List<int> imageIds)` method
- `SelectionProvider` manages selection state (not yet integrated)

**Configuration:**
- `WorkerPool.WorkerCount` controls AI concurrency (default 4, configurable via config.yaml)
- Set to 3 for AI concurrency limit if desired

## Key Interfaces

From flutter_app/lib/services/tag_service.dart:
```dart
Future<Map<String, dynamic>> batchTriggerAITags(List<int> imageIds) async {
  final response = await _client.post(
    Uri.parse(ApiConfig.batchAITags),
    headers: {'Content-Type': 'application/json'},
    body: jsonEncode({'image_ids': imageIds}),
  );
  // Returns { "job_ids": [...] }
}
```

From flutter_app/lib/providers/selection_provider.dart:
```dart
class SelectionProvider extends ChangeNotifier {
  Set<int> get selectedImageIds;
  bool get isSelectionMode;
  int get selectedCount;
  bool isSelected(int imageId);
  void toggleSelection(int imageId);
  void enterSelectionMode();
  void exitSelectionMode();
  bool handleImageTap(int imageId, {bool longPress, int? index});
}
```

From flutter_app/lib/config/api_config.dart:
```dart
static String get batchAITags => '$baseUrl/images/batch-ai-tags';
```
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add AI Generate Tags button to BatchOperationSheet</name>
  <files>flutter_app/lib/widgets/batch_operation_sheet.dart, flutter_app/test/widgets/batch_operation_sheet_test.dart</files>
  <behavior>
    - Test 1: Widget accepts onGenerateAITags callback parameter
    - Test 2: Button labeled "AI生成标签" appears in the sheet
    - Test 3: Tapping button calls onGenerateAITags callback
  </behavior>
  <action>
    Modify BatchOperationSheet to add AI tag generation capability:
    
    1. Add new callback parameter:
       ```dart
       final VoidCallback? onGenerateAITags;
       ```
    
    2. Add new button row with "AI生成标签" button:
       - Use icon: Icons.auto_awesome
       - Place after the existing tag buttons row
       - Blue/purple accent color for AI theme
    
    3. Update the static `show` method to accept the new callback
    
    4. Add basic widget test for the new button
    
    Reference: Look at how onAddTags and onRemoveTags are implemented for consistency.
  </action>
  <verify>
    <automated>flutter test flutter_app/test/widgets/batch_operation_sheet_test.dart</automated>
  </verify>
  <done>
    - BatchOperationSheet has onGenerateAITags parameter
    - "AI生成标签" button visible in the sheet
    - Widget tests pass
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Integrate selection mode into GalleryScreen</name>
  <files>
    flutter_app/lib/screens/gallery_screen.dart,
    flutter_app/lib/widgets/image_grid.dart,
    flutter_app/lib/widgets/image_masonry.dart,
    flutter_app/test/screens/gallery_screen_test.dart
  </files>
  <behavior>
    - Test 1: Long-press on image enters selection mode
    - Test 2: In selection mode, tap toggles checkbox
    - Test 3: Selection count displays in app bar or bottom sheet
    - Test 4: BatchOperationSheet appears with AI tags option
    - Test 5: AI tags button triggers batchTriggerAITags API call
  </behavior>
  <action>
    Integrate batch selection and AI tag generation into the gallery:
    
    1. **GalleryScreen modifications:**
       - Wrap content with SelectionProvider at the top of widget tree
       - Add selection mode state management
       - In app bar, show selection count when in selection mode
       - Add "Done" button in app bar to exit selection mode
       - When in selection mode, show floating action button or bottom bar to trigger BatchOperationSheet
    
    2. **ImageGrid modifications:**
       - Add selectionProvider parameter
       - Add onLongPress callback for entering selection mode
       - Show checkbox overlay when in selection mode
       - Use semi-transparent overlay on selected images
       
    3. **ImageMasonry modifications:**
       - Same as ImageGrid - add selection support
       
    4. **Wire BatchOperationSheet:**
       - Show sheet when user taps the batch action button
       - Pass onGenerateAITags callback that:
         - Gets selected image IDs from SelectionProvider
         - Calls tag_service.batchTriggerAITags(imageIds)
         - Shows success snackbar with job count
         - Exits selection mode
         
    5. **Import SelectionProvider and TagService:**
       - Import selection_provider.dart
       - Import tag_service.dart
       - Import batch_operation_sheet.dart
    
    Reference: Look at ImageDetailScreen for how TagService is used for AI tag operations.
    
    Minimal approach: Start with ImageGrid only, ImageMasonry can be added later if needed.
  </action>
  <verify>
    <automated>flutter test flutter_app/test/screens/gallery_screen_test.dart</automated>
  </verify>
  <done>
    - Long-press on image enters selection mode with checkbox
    - Multiple images can be selected
    - Selection count visible in UI
    - "AI生成标签" button triggers batch API call
    - Success/error feedback shown to user
    - Selection mode exits after operation
  </done>
</task>

</tasks>

<verification>
1. Run `flutter test flutter_app/test/widgets/batch_operation_sheet_test.dart`
2. Run `flutter test flutter_app/test/screens/gallery_screen_test.dart`
3. Manual test:
   - Launch app
   - Long-press an image to enter selection mode
   - Select multiple images
   - Tap batch action button
   - Tap "AI生成标签" button
   - Verify success snackbar appears
   - Check admin dashboard for new jobs
</verification>

<success_criteria>
- [ ] BatchOperationSheet has AI Generate Tags button
- [ ] Long-press on image enters selection mode
- [ ] Checkboxes visible on selected images
- [ ] BatchOperationSheet shows when action button pressed
- [ ] AI tags API called with selected image IDs
- [ ] User sees feedback (snackbar) after triggering
- [ ] Jobs appear in admin dashboard
</success_criteria>

<output>
After completion, create `.planning/quick/33-ai-tag/33-SUMMARY.md`
</output>