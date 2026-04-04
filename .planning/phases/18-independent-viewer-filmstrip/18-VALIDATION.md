# Phase 18 Validation: Independent Viewer with Filmstrip

## Overview
This document outlines the repeatable smoke test procedures and automated test verification for the Phase 18 Independent Viewer with Filmstrip feature. It ensures that the desktop multi-window viewer is correctly launched, isolated, and navigable, without regressions to the main shell.

## 1. Automated Verification
The following commands were run and passed successfully to guarantee coverage.

```bash
cd flutter_app
# Focused Suite Verification
flutter test test/app/fluent_screens_test.dart test/widgets/fluent_gallery_content_test.dart test/widgets/fluent_search_content_test.dart test/widgets/fluent_image_card_test.dart test/services/viewer_window_service_test.dart test/screens/viewer/viewer_workspace_test.dart test/screens/viewer/viewer_stage_test.dart test/screens/viewer/viewer_metadata_sidebar_test.dart

# Full Suite Verification
flutter test
```

## 2. Desktop Smoke Test Procedure

### 2.1 Multi-Window Isolation
1. Launch the desktop application (`flutter run -d windows`).
2. Navigate to the Gallery view.
3. Double-click on `Image A`.
   - **Expectation**: A new independent viewer window opens displaying `Image A`.
4. Return to the main shell without closing the first viewer window.
5. Search for a different tag/keyword, and double-click on `Image B` from the search results.
   - **Expectation**: A second, distinct viewer window opens displaying `Image B`.
6. Verify the main shell remains fully interactive (can scroll, filter, navigate) while both viewer windows are open.

### 2.2 Navigation and Filmstrip
1. In the first viewer window (opened from Gallery), verify the bottom filmstrip displays the images surrounding `Image A` from the Gallery result set.
2. Click an adjacent image in the filmstrip or press the `Right Arrow` key.
   - **Expectation**: The main stage updates to the new image. The window title updates to reflect the new image's original filename.
3. In the second viewer window (opened from Search), verify its filmstrip displays the images surrounding `Image B` from the Search result set.
4. Press the `Right Arrow` key in the second viewer window.
   - **Expectation**: The second viewer window updates its stage and title independently of the first viewer window.

### 2.3 Keyboard Controls
1. In any open viewer window, double-click the main image.
   - **Expectation**: The image scales to 2x (actual size). Double-click again to return to fit-to-window.
2. Press the `Esc` key while focused on a viewer window.
   - **Expectation**: Only the currently focused viewer window closes. The main shell and other viewer windows remain open.

### 2.4 Guardrail Checks (Out-of-Scope Items)
1. Verify there are no "Delete", "Share", "Edit", or "Slideshow" buttons in the viewer window UI.
2. Verify the metadata sidebar accurately displays read-only properties (dimensions, file size, hash, date).
3. Verify there are no interactive UI elements to add/remove tags directly from the viewer (reserved for Phase 19/20).

## 3. Sign-off
Phase 18 validation criteria have been met. The desktop application effectively orchestrates multiple isolated viewer windows with correct snapshot context and keyboard interaction semantics.
