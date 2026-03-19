---
phase: quick-28
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - flutter_app/lib/widgets/image_lightbox.dart
  - flutter_app/lib/screens/image_detail_screen.dart
autonomous: true
requirements: [UI-01]
user_setup: []

must_haves:
  truths:
    - "User can tap the image to open a fullscreen preview"
    - "User can pinch-zoom and pan in the fullscreen preview"
    - "User can dismiss the preview with a swipe-down gesture"
    - "The image detail layout has reduced whitespace"
  artifacts:
    - path: "flutter_app/lib/widgets/image_lightbox.dart"
      provides: "Fullscreen image preview component"
      exports: ["ImageLightbox"]
    - path: "flutter_app/lib/screens/image_detail_screen.dart"
      provides: "Updated image detail screen with lightbox integration"
      contains: "ImageLightbox"
  key_links:
    - from: "image_detail_screen.dart"
      to: "ImageLightbox"
      via: "GestureDetector onTap"
      pattern: "ImageLightbox\\.show"
---

<objective>
Improve the image detail page layout by reducing whitespace and adding a Weibo/Bilibili-style fullscreen image preview/lightbox feature.

Purpose: Enhance user experience when viewing images in detail
Output: Updated image detail screen with lightbox functionality
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

## Current Implementation Analysis

**ImageDetailScreen** (`flutter_app/lib/screens/image_detail_screen.dart`):
- Uses `ExtendedImage.network` for image display
- Has `maxHeight: MediaQuery.of(context).size.height * 0.6` constraint
- Uses `BoxFit.contain` which preserves aspect ratio
- Already has `ExtendedImageMode.gesture` for pinch/zoom

**Available Packages**:
- `extended_image: ^8.2.1` - Supports Hero animations, gesture mode, fullscreen preview
- `cached_network_image: ^3.3.1` - Network image caching

**Key Files to Modify**:
- `flutter_app/lib/screens/image_detail_screen.dart` - Main detail screen
- `flutter_app/lib/widgets/image_lightbox.dart` - New widget (to be created)

**Weibo/Bilibili Style Preview Features**:
1. Tap image to enter fullscreen preview
2. Pinch to zoom, drag to pan
3. Swipe down to dismiss
4. Smooth Hero animation transition
5. Dark background overlay
</context>

<tasks>

<task type="auto" tdd="true">
  <name>task 1: Create ImageLightbox widget for fullscreen preview</name>
  <files>flutter_app/lib/widgets/image_lightbox.dart</files>
  <behavior>
    - Test 1: Widget displays image fullscreen with dark background
    - Test 2: Pinch gesture zooms the image (min 0.5x, max 3x)
    - Test 3: Swipe down dismisses the preview
    - Test 4: Hero animation transitions smoothly from source image
  </behavior>
  <action>
Create a new `ImageLightbox` widget in `flutter_app/lib/widgets/image_lightbox.dart`:

1. **Widget Structure**:
   - `ImageLightbox` - A stateless widget with static `show()` method
   - Uses `showDialog()` or `showGeneralDialog()` for fullscreen overlay
   - Dark background (Colors.black with 0.9 opacity)

2. **Image Display**:
   - Use `ExtendedImage.network` with `ExtendedImageMode.gesture`
   - Configure `GestureConfig`:
     - minScale: 0.5
     - maxScale: 3.0
     - speed: 1.0
     - inertialSpeed: 100.0
   - `BoxFit.contain` to preserve aspect ratio

3. **Dismissible Gestures**:
   - Wrap in `GestureDetector` with `onVerticalDragEnd`
   - If drag velocity exceeds threshold, dismiss the dialog
   - Alternative: Use `Dismissible` widget with swipe-down direction

4. **Hero Animation Support**:
   - Accept `heroTag` parameter for Hero animation
   - Wrap image in `Hero` widget with the tag

5. **Static show() Method**:
   ```dart
   static Future<void> show(
     BuildContext context, {
     required String imageUrl,
     String? heroTag,
   })
   ```
   - Uses `showGeneralDialog()` with fade transition
   - Barrier color: transparent
   - Barrier dismissible: true

6. **UI Polish**:
   - Add close button in top-right corner (IconButton with white icon)
   - Semi-transparent status bar overlay
   - Smooth fade-in/fade-out animation (200ms duration)

DO NOT add additional features like:
- Image info overlay (not requested)
- Share buttons (not requested)
- Multiple image swipe (single image preview only)
  </action>
  <verify>
    <automated>flutter test flutter_app/test/widgets/image_lightbox_test.dart 2>/dev/null || echo "Test file will be created by TDD cycle"</automated>
  </verify>
  <done>
    - ImageLightbox widget created in flutter_app/lib/widgets/image_lightbox.dart
    - Widget accepts imageUrl and optional heroTag
    - Static show() method displays fullscreen preview
    - Pinch-zoom and swipe-down-to-dismiss work correctly
  </done>
</task>

<task type="auto" tdd="true">
  <name>task 2: Integrate lightbox into ImageDetailScreen and reduce whitespace</name>
  <files>flutter_app/lib/screens/image_detail_screen.dart</files>
  <behavior>
    - Test 1: Tapping the image opens the fullscreen lightbox
    - Test 2: Hero animation transitions smoothly
    - Test 3: Image viewer takes more vertical space (reduced whitespace)
    - Test 4: Layout is more compact and visually appealing
  </behavior>
  <action>
Update `flutter_app/lib/screens/image_detail_screen.dart`:

1. **Import the lightbox widget**:
   - Add import: `import '../widgets/image_lightbox.dart';`

2. **Update _buildImageViewer() method**:
   - Remove the `maxHeight` constraint (or increase to 0.75)
   - Replace Container with `LayoutBuilder` for responsive sizing
   - Wrap `ExtendedImage.network` in `GestureDetector` with `onTap`:
     ```dart
     onTap: () => ImageLightbox.show(
       context,
       imageUrl: largeUrl,
       heroTag: 'image-${widget.image.id}',
     ),
     ```
   - Add Hero widget wrapper with unique tag:
     ```dart
     Hero(
       tag: 'image-${widget.image.id}',
       child: ExtendedImage.network(...),
     )
     ```
   - Add visual feedback on tap (InkWell or splash effect)

3. **Layout Adjustments**:
   - Reduce padding in metadata section (from 16 to 12)
   - Make image viewer expand to fit available space
   - Use `Flexible` or `Expanded` for better space utilization
   - Consider using `InteractiveViewer` for the main image as well

4. **Improve _buildMetadataSection()**:
   - Use more compact row layout for metadata
   - Reduce vertical padding from 4 to 2
   - Consider using a DataTable or more compact design

5. **Visual Improvements**:
   - Add subtle shadow/border around the image
   - Add tap hint (small overlay or ripple effect on image)
   - Consider adding a "View Full Size" tooltip

DO NOT:
- Remove existing functionality (tags, AI section, metadata)
- Change the navigation structure
- Add new dependencies (use existing packages only)
  </action>
  <verify>
    <automated>flutter test flutter_app/test/screens/image_detail_screen_test.dart 2>/dev/null || echo "Test file will be created by TDD cycle"</automated>
  </verify>
  <done>
    - ImageDetailScreen shows image with reduced whitespace
    - Tapping image opens fullscreen lightbox preview
    - Hero animation provides smooth transition
    - Layout is more compact and visually appealing
  </done>
</task>

</tasks>

<verification>
1. Run Flutter app: `cd flutter_app && flutter run -d chrome`
2. Navigate to any image in the gallery
3. Verify the image detail page has less whitespace
4. Tap the image to open fullscreen preview
5. Test pinch-to-zoom in the preview
6. Test swipe-down to dismiss
7. Verify Hero animation is smooth
</verification>

<success_criteria>
- [ ] ImageLightbox widget created with fullscreen preview functionality
- [ ] Image detail screen has reduced whitespace (image takes more vertical space)
- [ ] Tapping image opens fullscreen lightbox with smooth Hero transition
- [ ] Pinch-to-zoom and swipe-to-dismiss work in lightbox
- [ ] All existing functionality (tags, AI, metadata) still works
- [ ] No new dependencies added
</success_criteria>

<output>
After completion, create `.planning/quick/28-image-detail-layout-preview/28-SUMMARY.md`
</output>

<commit_strategy>
## Atomic Commit Strategy

**Commit 1**: `feat(ui): add ImageLightbox widget for fullscreen image preview`
- Creates `flutter_app/lib/widgets/image_lightbox.dart`
- Implements fullscreen preview with gesture support
- Implements swipe-to-dismiss functionality

**Commit 2**: `feat(ui): integrate lightbox into image detail screen with layout improvements`
- Updates `flutter_app/lib/screens/image_detail_screen.dart`
- Adds Hero animation support
- Reduces whitespace in layout
- Improves overall visual appearance
</commit_strategy>