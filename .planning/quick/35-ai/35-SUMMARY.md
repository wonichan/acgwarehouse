# Plan 35: Async AI Tag Generation for Batch Operations - Summary

**Status:** ✅ COMPLETE  
**Completed:** 2026-03-20  
**Phase:** Quick Task  
**Plan:** 35-ai  

---

## Overview

Refactored the AI tag generation feature in the gallery to execute asynchronously, providing immediate UI feedback and preventing blocking during batch operations.

## What Changed

### 1. Gallery Screen (`gallery_screen.dart`)

**Before:** Synchronous blocking operation
```dart
Future<void> _generateAITags(BuildContext context) async {
  // ... get dependencies
  try {
    final result = await tagService.batchTriggerAITags(imageIds);  // BLOCKS UI
    // ... show snackbar after API completes
    selectionProvider.exitSelectionMode();
  } catch (e) {
    // ... show error
  }
}
```

**After:** Non-blocking async operation
```dart
Future<void> _generateAITags(BuildContext context) async {
  // ... get dependencies
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

### 2. Batch Operation Sheet (`batch_operation_sheet.dart`)

Fixed layout overflow by wrapping the Column in `SingleChildScrollView`:

```dart
return Container(
  padding: const EdgeInsets.all(16),
  child: SingleChildScrollView(  // Added for scrollability
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        // ... buttons
      ],
    ),
  ),
);
```

### 3. Test Coverage (`gallery_screen_test.dart`)

Added comprehensive test suite for async behavior:

- ✅ `closes bottom sheet immediately when AI generate tapped`
- ✅ `shows snackbar immediately with correct message`
- ✅ `exits selection mode immediately`
- ✅ `fires API call without blocking UI`

## User Experience Flow

**Before:**
1. User taps "批量操作" → Opens bottom sheet
2. User taps "AI生成标签" → UI freezes
3. Wait for API response (2-10 seconds)
4. Snackbar appears
5. Bottom sheet closes
6. Selection mode exits

**After:**
1. User taps "批量操作" → Opens bottom sheet
2. User taps "AI生成标签" → 
   - Bottom sheet closes **immediately**
   - Snackbar shows: "AI标签生成任务已在后台启动 (N张图片)" **immediately**
   - Selection mode exits **immediately**
3. User can continue using gallery while AI processes in background

## Test Results

```
$ flutter test test/screens/gallery_screen_test.dart

00:01 +9: All tests passed!

Test Groups:
- GalleryScreen: 5 tests passing
- Async AI Tag Generation: 4 tests passing
```

## Commits

| Commit | Description |
|--------|-------------|
| `25db59c` | test(35-ai): add failing tests for async AI tag generation |
| `bb8ce9e` | feat(35-ai): make AI tag generation async with immediate feedback |

## Files Modified

1. `flutter_app/lib/screens/gallery_screen.dart` - Async implementation
2. `flutter_app/lib/widgets/batch_operation_sheet.dart` - Overflow fix
3. `flutter_app/test/screens/gallery_screen_test.dart` - Test coverage

## Technical Details

### Why Async?

The AI tag generation API call can take 2-10 seconds depending on:
- Number of images selected
- AI model response time
- Network latency

Blocking the UI during this time creates a poor user experience. By making it async:
- Users get immediate feedback
- UI remains responsive
- Background processing continues even if user navigates away

### Error Handling

Since the operation is now async, errors are handled silently:
- Success: No additional message (initial snackbar is sufficient)
- Failure: Logged to debug console only
- Rationale: User already received "tasks started" confirmation; error during processing is background concern

### Test Strategy

Tests verify:
1. **Immediate feedback**: Widget state changes happen before API call
2. **Non-blocking**: UI remains responsive
3. **API invocation**: The service method is actually called
4. **Correct messaging**: Snackbar shows image count

## Deviations from Plan

**None** - Plan executed exactly as written.

### Auto-discovered Fix

During test execution, discovered the batch operation sheet had a layout overflow (46px over). Fixed by wrapping Column in SingleChildScrollView. This was not in the original plan but was required for tests to pass.

**File:** `batch_operation_sheet.dart`  
**Change:** Added SingleChildScrollView wrapper  
**Commit:** Included in GREEN commit

## Verification

To verify the implementation:

```bash
# Run unit tests
cd flutter_app
flutter test test/screens/gallery_screen_test.dart

# Manual testing
cd ..
go run cmd/server/main.go  # Start backend
cd flutter_app
flutter run -d chrome

# Test flow:
# 1. Long-press images to select
# 2. Tap 批量操作
# 3. Tap AI生成标签
# 4. Verify immediate close + snackbar
```

## Next Steps

This quick task is complete. The async AI tag generation is now active and tested.

---

**Summary:** Successfully refactored AI tag generation from synchronous blocking to asynchronous fire-and-forget, improving UX by providing immediate feedback while processing continues in the background. All 4 new tests pass, and the existing gallery functionality remains intact.
