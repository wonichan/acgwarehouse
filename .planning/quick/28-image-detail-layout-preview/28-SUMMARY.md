---
phase: quick-28
plan: 01
subsystem: ui
tags: [flutter, image-detail, lightbox, hero-animation, layout]
requirements: [UI-01]
dependency_graph:
  requires: []
  provides: [fullscreen-image-preview, hero-transition]
  affects: [image-detail-screen]
tech_stack:
  added: []
  patterns: [hero-animation, gesture-detector, show-general-dialog]
key_files:
  created:
    - flutter_app/test/screens/image_detail_screen_test.dart
  modified:
    - flutter_app/lib/screens/image_detail_screen.dart
    - flutter_app/test/widgets/image_lightbox_test.dart
decisions:
  - Use showGeneralDialog instead of showDialog for custom fade transition
  - Hero tag format: 'image-{imageId}' for consistent animation
  - Increase maxHeight from 0.6 to 0.75 for better image visibility
  - Reduce padding from 16 to 12 for compact layout
metrics:
  duration: ~15 minutes
  tasks_completed: 2
  files_modified: 3
  tests_added: 11
  completed_date: 2026-03-20
---

# Quick Task 28: Image Detail Layout Preview Summary

## One-liner

Added fullscreen image preview/lightbox with Hero animation and reduced whitespace in image detail screen for Weibo/Bilibili-style viewing experience.

## Changes Made

### Task 1: ImageLightbox Widget (Pre-existing)

The `ImageLightbox` widget was already created in a previous quick task. It provides:
- Fullscreen image preview with dark background (0.9 opacity)
- Pinch-to-zoom (0.5x to 3.0x) and pan via `ExtendedImageMode.gesture`
- Swipe-down to dismiss gesture
- Hero animation support for smooth transitions
- Close button in top-right corner
- Loading and error states

### Task 2: ImageDetailScreen Integration & Layout Improvements

**File:** `flutter_app/lib/screens/image_detail_screen.dart`

1. **Hero Animation Integration**
   - Added `Hero` widget wrapper around `ExtendedImage.network`
   - Hero tag: `'image-${widget.image.id}'`
   - Enables smooth transition from detail screen to fullscreen preview

2. **Tap-to-Fullscreen**
   - Added `GestureDetector` with `onTap` to trigger `ImageLightbox.show()`
   - Added visual hint overlay: "点击全屏" with fullscreen icon
   - Passes hero tag for consistent animation

3. **Layout Improvements**
   - Increased `maxHeight` from `0.6` to `0.75` of screen height
   - Reduced padding in metadata section: `16 → 12`
   - Reduced metadata row vertical padding: `4 → 2`
   - Reduced label width: `80 → 70`
   - Added smaller font sizes (`13`) for compact appearance
   - Reduced AI tag section margin/padding
   - Reduced tags section padding: `16 → 12`
   - Reduced section spacing: `16 → 12`, `8 → 6`

### Test Improvements

**Files:**
- `flutter_app/test/screens/image_detail_screen_test.dart` (new - 6 tests)
- `flutter_app/test/widgets/image_lightbox_test.dart` (fixed - 5 tests)

Fixed `pumpAndSettle` timeout issue by using `pump(Duration(milliseconds: 300))` for tests involving network images.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Blocking Issue] Pre-existing test timeout**
- **Found during:** test execution
- **Issue:** `image_lightbox_test.dart` tests using `pumpAndSettle()` timed out because network images never settle
- **Fix:** Replaced `pumpAndSettle()` with `pump(const Duration(milliseconds: 300))`
- **Files modified:** `flutter_app/test/widgets/image_lightbox_test.dart`
- **Commit:** 4d73bc0

### Deferred Issues

- `flutter_app/test/widget_test.dart` contains default Flutter counter test that fails - out of scope for this task

## Verification Steps

1. Run Flutter tests:
   ```bash
   cd flutter_app && flutter test test/widgets/image_lightbox_test.dart test/screens/image_detail_screen_test.dart
   ```
   Result: All 11 tests passed

2. Manual verification:
   ```bash
   cd flutter_app && flutter run -d chrome
   ```
   - Navigate to any image in the gallery
   - Verify image detail page has less whitespace
   - Tap image to open fullscreen preview
   - Test pinch-to-zoom in preview
   - Test swipe-down to dismiss
   - Verify Hero animation is smooth

## Success Criteria Met

- [x] ImageLightbox widget created with fullscreen preview functionality (pre-existing)
- [x] Image detail screen has reduced whitespace (image takes more vertical space)
- [x] Tapping image opens fullscreen lightbox with smooth Hero transition
- [x] Pinch-to-zoom and swipe-to-dismiss work in lightbox
- [x] All existing functionality (tags, AI, metadata) still works
- [x] No new dependencies added

## Commits

| Commit | Message |
|--------|---------|
| 8786c89 | docs(quick-28): create image detail layout preview plan |
| 9375a6e | feat(ui): integrate lightbox into image detail screen with layout improvements |
| 4d73bc0 | test(ui): fix image_lightbox_test pumpAndSettle timeout |

## Self-Check

- [x] Files created: `flutter_app/test/screens/image_detail_screen_test.dart` exists
- [x] Files modified: `flutter_app/lib/screens/image_detail_screen.dart` has changes
- [x] Commits exist: 9375a6e, 4d73bc0 verified in git log
- [x] Tests pass: 11/11 relevant tests pass