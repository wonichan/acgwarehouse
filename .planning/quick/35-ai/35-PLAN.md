# Plan 35: Async AI Tag Generation for Batch Operations

## Context

**User Request**: When users long-press images and select batch operations, clicking "AI生成标签" (AI Generate Tags) should immediately close the bottom sheet and return to the gallery. The backend executes tasks asynchronously. Users should be notified that tasks are in progress.

**Current Behavior**: 
- User clicks "AI生成标签" → waits for API response → then sees snackbar → then returns
- This is synchronous and blocks the UI

**Desired Behavior**:
- User clicks "AI生成标签" → bottom sheet closes immediately → snackbar shows "任务正在后台运行" (Tasks running in background) → API call fires asynchronously → user can continue using the app

**Backend Status**: Already supports async execution via `POST /api/v1/images/batch-ai-tags` which returns 202 Accepted with job IDs.

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1: Modify Gallery Screen | None | Frontend UI change only |
| Task 2: Add Unit Tests | Task 1 | Tests verify the async behavior |

## Parallel Execution Graph

Wave 1 (Start immediately):
├── Task 1: Modify gallery_screen.dart async behavior (no dependencies)

Wave 2 (After Wave 1 completes):
└── Task 2: Add unit tests for async AI tag generation

Critical Path: Task 1 → Task 2
No parallel speedup possible (tests depend on implementation)

## Tasks

### Task 1: Modify Gallery Screen for Async AI Tag Generation

**Description**: Refactor `_generateAITags` method in `gallery_screen.dart` to close the bottom sheet immediately and fire the API call asynchronously without blocking the UI.

**Delegation Recommendation**:
- Category: `quick` - Simple Flutter widget modification, single file change
- Skills: [`test-driven-development`] - Need to write tests first

**Skills Evaluation**:
- INCLUDED `test-driven-development`: Required to follow TDD and write tests before implementation
- OMITTED `frontend-ui-ux`: Not a UI design task, just behavior modification
- OMITTED `systematic-debugging`: No bugs to debug, feature modification

**Depends On**: None

**Acceptance Criteria**:
1. Bottom sheet closes immediately when "AI生成标签" is clicked
2. Snackbar shows "AI标签生成任务已在后台启动 (${count}张图片)" immediately
3. API call fires without blocking (use `unawaited` or fire-and-forget pattern)
4. Selection mode exits immediately
5. User can continue interacting with gallery while AI processes

**Implementation Details**:

Current code in `_generateAITags` (lines 306-326):
```dart
Future<void> _generateAITags(BuildContext context) async {
  final selectionProvider = context.read<SelectionProvider>();
  final tagService = context.read<TagProvider>().tagService;
  final imageIds = selectionProvider.selectedImageIds.toList();

  try {
    final result = await tagService.batchTriggerAITags(imageIds);
    final jobIds = (result['job_ids'] as List?) ?? const [];
    if (!context.mounted) return;

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('已触发 ${jobIds.length} 个 AI 标签任务')),
    );
    selectionProvider.exitSelectionMode();
  } catch (e) {
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('AI 生成失败: $e')),
    );
  }
}
```

New implementation should be:
```dart
Future<void> _generateAITags(BuildContext context) async {
  final selectionProvider = context.read<SelectionProvider>();
  final tagService = context.read<TagProvider>().tagService;
  final imageIds = selectionProvider.selectedImageIds.toList();
  final count = imageIds.length;

  // Close bottom sheet immediately
  Navigator.pop(context);
  
  // Show immediate feedback
  if (context.mounted) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('AI标签生成任务已在后台启动 ($count张图片)')),
    );
  }
  
  // Exit selection mode
  selectionProvider.exitSelectionMode();

  // Fire API call asynchronously without blocking
  _triggerAITagsAsync(tagService, imageIds);
}

void _triggerAITagsAsync(TagService tagService, List<int> imageIds) async {
  try {
    await tagService.batchTriggerAITags(imageIds);
    // Success - no need to show another message
  } catch (e) {
    // Log error but don't show to user since we're async
    debugPrint('AI tag generation error: $e');
  }
}
```

**TDD Approach**:
1. Write test expecting `Navigator.pop` to be called immediately
2. Write test expecting snackbar to show immediately with correct message
3. Write test expecting selection mode to exit immediately
4. Write test expecting API call to be made (can verify via mock)
5. Implement the changes
6. All tests pass

---

### Task 2: Add Unit Tests for Async AI Tag Generation

**Description**: Add comprehensive unit tests for the new async behavior in gallery_screen_test.dart or create dedicated tests.

**Delegation Recommendation**:
- Category: `quick` - Unit tests for Flutter widgets
- Skills: [`test-driven-development`] - Primary skill for writing tests

**Skills Evaluation**:
- INCLUDED `test-driven-development`: Core skill for writing tests
- OMITTED `verification-before-completion`: Tests ARE the verification

**Depends On**: Task 1

**Acceptance Criteria**:
1. Test verifies bottom sheet closes immediately (Navigator.pop called)
2. Test verifies snackbar shows with correct message format
3. Test verifies selection provider's exitSelectionMode is called
4. Test verifies API call is made with correct image IDs
5. Test verifies UI is not blocked (async behavior)

**Test Cases to Add**:

```dart
group('Async AI Tag Generation', () {
  testWidgets('closes bottom sheet immediately when AI generate tapped', (tester) async {
    // Setup: Show batch operation sheet with selected images
    // Action: Tap AI generate tags button
    // Verify: Bottom sheet is dismissed immediately
  });

  testWidgets('shows snackbar immediately with correct message', (tester) async {
    // Setup: 5 images selected
    // Action: Trigger AI generate
    // Verify: Snackbar shows "AI标签生成任务已在后台启动 (5张图片)"
  });

  testWidgets('exits selection mode immediately', (tester) async {
    // Setup: Selection mode active
    // Action: Trigger AI generate
    // Verify: Selection mode exited
  });

  testWidgets('fires API call without blocking UI', (tester) async {
    // Setup: Mock tag service with delay
    // Action: Trigger AI generate
    // Verify: UI remains responsive (pump between checks)
  });
});
```

## Commit Strategy

**Atomic Commits** (TDD-style):

1. **Commit 1 (RED)**: Add failing tests for async AI tag behavior
   ```
   test(35-ai): add failing tests for async AI tag generation
   
   - Test immediate bottom sheet close
   - Test immediate snackbar display
   - Test immediate selection mode exit
   - Test non-blocking API call
   ```

2. **Commit 2 (GREEN)**: Implement async behavior to pass tests
   ```
   feat(35-ai): make AI tag generation async with immediate feedback
   
   - Close bottom sheet immediately on AI generate tap
   - Show snackbar immediately: "AI标签生成任务已在后台启动"
   - Fire API call asynchronously without blocking UI
   - Exit selection mode immediately
   ```

3. **Commit 3 (REFACTOR)**: Clean up if needed (optional)
   ```
   refactor(35-ai): extract async helper method
   
   - Move async tag triggering to separate method
   - Add error logging for background failures
   ```

## Success Criteria

**Functional Verification**:
1. ✅ Open gallery, long-press images to enter selection mode
2. ✅ Select multiple images
3. ✅ Tap "批量操作" → Tap "AI生成标签"
4. ✅ Bottom sheet closes immediately
5. ✅ Snackbar appears: "AI标签生成任务已在后台启动 (N张图片)"
6. ✅ Gallery remains interactive while AI processes
7. ✅ Check admin dashboard to verify jobs are queued

**Automated Verification**:
```bash
# Run unit tests
cd flutter_app
flutter test test/screens/gallery_screen_test.dart

# All tests should pass
```

**Manual Verification**:
```bash
# Start backend
go run cmd/server/main.go

# Start Flutter app
cd flutter_app
flutter run -d chrome

# Test the flow:
# 1. Long-press images to select
# 2. Tap 批量操作
# 3. Tap AI生成标签
# 4. Verify immediate close + snackbar
```

## Files Modified

1. `flutter_app/lib/screens/gallery_screen.dart` - Modify `_generateAITags` method
2. `flutter_app/test/screens/gallery_screen_test.dart` - Add async AI tag tests

## Output

After completion, create `.planning/quick/35-ai/35-SUMMARY.md` with:
- Implementation summary
- Test coverage report
- Verification results
