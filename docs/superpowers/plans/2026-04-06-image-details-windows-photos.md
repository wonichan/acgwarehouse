# Image Details Windows Photos Pane Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the desktop image details page and viewer metadata sidebar look like one Windows Photos-inspired dark information pane, removing hardcoded light surfaces and keeping metadata workflows unchanged.

**Architecture:** Introduce one shared metadata-pane theme helper for surfaces, dividers, and text roles, then apply it from the shared `ImageMetadataPanel` into both hosts (`ImageDetailScreen` and `ViewerMetadataSidebar`). Verify the redesign with widget tests that explicitly protect against the old white-card-in-dark-mode regression.

**Tech Stack:** Flutter 3.9, Material `ThemeData`/`ColorScheme`, `provider`, `flutter_test`

---

## File Structure

### Create
- `flutter_app/lib/widgets/image_metadata_pane_theme.dart` — shared pane palette + text-role helper derived from `Theme.of(context)`
- `flutter_app/test/screens/image_detail_screen_test.dart` — widget regression coverage for the dedicated image details page in dark mode

### Modify
- `flutter_app/lib/widgets/image_metadata_panel.dart` — remove forced light surfaces, replace white cards with pane sections, theme AI/tag controls, add stable test keys
- `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart` — convert host shell into a theme-aware right info pane with divider and stable key
- `flutter_app/lib/screens/image_detail_screen.dart` — make the metadata side read as an edge-aligned info pane, use shared pane palette, add stable key
- `flutter_app/test/screens/viewer/viewer_metadata_sidebar_test.dart` — replace old “always light” assertions with theme-aware dark-pane assertions

### Existing References To Read Before Editing
- `docs/superpowers/specs/2026-04-06-image-details-windows-photos-design.md`
- `flutter_app/lib/widgets/image_metadata_panel.dart`
- `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`
- `flutter_app/lib/screens/image_detail_screen.dart`
- `flutter_app/test/screens/viewer/viewer_metadata_sidebar_test.dart`

### Design Decisions Locked In For This Plan
- Use **one fixed desktop metadata row pattern**: label above value, not adaptive left-column logic.
- Use `Theme.of(context).textTheme` for hierarchy instead of introducing new hardcoded text colors.
- Use low-contrast dividers, but keep them visible in dark mode.
- Do **not** add backend changes, routing changes, or metadata field changes.
- Do **not** commit unless the user explicitly requests it.

## Chunk 1: Regression Tests + Shared Pane Tokens

### Task 1: Replace the old light-theme regression tests with dark-pane expectations

**Files:**
- Modify: `flutter_app/test/screens/viewer/viewer_metadata_sidebar_test.dart`
- Create: `flutter_app/test/screens/image_detail_screen_test.dart`

- [ ] **Step 1: Rewrite the viewer sidebar regression test expectations first**

Replace the old assertions that require `#F3F3F3`, `Card`, and `Colors.white` in dark mode with assertions that enforce the new design.

Use stable keys in the tests that the implementation will add later:

```dart
expect(find.byKey(const ValueKey('viewer-metadata-sidebar')), findsOneWidget);
expect(find.byKey(const ValueKey('metadata-pane-root')), findsOneWidget);
expect(find.byKey(const ValueKey('metadata-section-basic')), findsOneWidget);
```

Add assertions that the sidebar does **not** expose the old hardcoded light treatment:

```dart
final sidebar = tester.widget<Container>(
  find.byKey(const ValueKey('viewer-metadata-sidebar')),
);
final decoration = sidebar.decoration! as BoxDecoration;

expect(decoration.color, isNot(equals(const Color(0xFFF3F3F3))));

final cards = tester.widgetList<Card>(find.byType(Card));
expect(
  cards.where((card) => card.color == Colors.white),
  isEmpty,
);
```

- [ ] **Step 2: Add a new dark-mode details-page regression test**

Create `flutter_app/test/screens/image_detail_screen_test.dart` with a minimal `ImageModel` fixture and assert that the details pane is rendered as a dark, theme-aware host.

Use this fixture shape:

```dart
const image = ImageModel(
  id: 1,
  path: '/library/demo/image.jpg',
  filename: 'image.jpg',
  sourceRoot: '/library/demo',
  fileSize: 1024 * 1024,
  width: 1920,
  height: 1080,
  format: 'jpeg',
  phash: 123,
  thumbnailSmallUrl: 'http://small.jpg',
  thumbnailLargeUrl: 'http://large.jpg',
  createdAt: DateTime.utc(2023, 1, 1),
  updatedAt: DateTime.utc(2023, 1, 1),
);
```

Core assertions:

```dart
expect(find.byKey(const ValueKey('image-detail-metadata-pane')), findsOneWidget);
expect(find.byKey(const ValueKey('metadata-pane-root')), findsOneWidget);
expect(find.text('元数据'), findsOneWidget);
expect(find.byType(Card), findsNothing);
```

- [ ] **Step 3: Run the two widget test files to capture the expected failures**

Run:

```bash
flutter test test/screens/viewer/viewer_metadata_sidebar_test.dart test/screens/image_detail_screen_test.dart
```

Expected: FAIL because the code still hardcodes `Color(0xFFF3F3F3)`, `Colors.white`, and does not expose the new keys.

### Task 2: Add one shared pane theme helper for surfaces, divider contrast, and text roles

**Files:**
- Create: `flutter_app/lib/widgets/image_metadata_pane_theme.dart`
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart`
- Modify: `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`
- Modify: `flutter_app/lib/screens/image_detail_screen.dart`

- [ ] **Step 1: Create the shared pane theme helper**

Add `flutter_app/lib/widgets/image_metadata_pane_theme.dart` with a focused value object that derives all metadata-pane colors from the active theme.

Use this shape:

```dart
import 'package:flutter/material.dart';

class ImageMetadataPaneTheme {
  final Color paneSurface;
  final Color sectionSurface;
  final Color dividerColor;
  final Color labelColor;
  final Color valueColor;
  final Color helperColor;
  final Color inputFillColor;
  final Color statusFillColor;

  const ImageMetadataPaneTheme({
    required this.paneSurface,
    required this.sectionSurface,
    required this.dividerColor,
    required this.labelColor,
    required this.valueColor,
    required this.helperColor,
    required this.inputFillColor,
    required this.statusFillColor,
  });

  factory ImageMetadataPaneTheme.of(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final isDark = theme.brightness == Brightness.dark;

    return ImageMetadataPaneTheme(
      paneSurface: isDark ? scheme.surfaceContainerLow : scheme.surface,
      sectionSurface: isDark ? scheme.surfaceContainer : scheme.surfaceContainerLow,
      dividerColor: scheme.outlineVariant,
      labelColor: theme.textTheme.bodySmall?.color ?? scheme.onSurfaceVariant,
      valueColor: theme.textTheme.bodyMedium?.color ?? scheme.onSurface,
      helperColor: theme.textTheme.bodySmall?.color ?? scheme.onSurfaceVariant,
      inputFillColor: isDark ? scheme.surfaceContainerHighest : scheme.surfaceContainerLow,
      statusFillColor: isDark ? scheme.surfaceContainerHigh : scheme.surfaceContainer,
    );
  }
}
```

- [ ] **Step 2: Add stable `ValueKey`s for the pane and sections**

Use the same keys across both hosts so tests can target the real UI surface instead of `Container.first`:

```dart
const metadataPaneRootKey = ValueKey('metadata-pane-root');
const metadataSectionBasicKey = ValueKey('metadata-section-basic');
const metadataSectionTagsKey = ValueKey('metadata-section-tags');
const metadataSectionAiKey = ValueKey('metadata-section-ai');
```

Expected outcome: later tests can inspect the actual pane host without relying on brittle widget ordering.

## Chunk 2: Shared Pane Refactor

### Task 3: Convert `ImageMetadataPanel` from white-card stacks into themed pane sections

**Files:**
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart`

- [ ] **Step 1: Remove the forced light pane surface from `build()`**

Delete the current hardcoded block:

```dart
// Force light theme colors for Windows Photos-inspired styling regardless of app theme
const panelSurface = Color(0xFFF3F3F3);
```

Replace it with the shared helper:

```dart
final paneTheme = ImageMetadataPaneTheme.of(context);
```

- [ ] **Step 2: Replace `Card` wrappers with lightweight pane sections**

Replace both `_buildAITagSection` and `_buildTagsSection` `Card` wrappers with a shared, flat section container.

Use one internal helper like this:

```dart
Widget _buildPaneSection({
  required Key key,
  required ImageMetadataPaneTheme paneTheme,
  required Widget child,
}) {
  return Container(
    key: key,
    margin: const EdgeInsets.fromLTRB(12, 8, 12, 0),
    padding: const EdgeInsets.all(16),
    decoration: BoxDecoration(
      color: paneTheme.sectionSurface,
      border: Border.all(color: paneTheme.dividerColor),
      borderRadius: BorderRadius.circular(8),
    ),
    child: child,
  );
}
```

Use it to host:
- the metadata block passed from the host screen
- the AI section
- the tags section

- [ ] **Step 3: Theme every previously hardcoded light control in the AI section**

Update these hardcoded values to theme-aware colors:
- `Colors.white`
- `Color(0xFFEFEFEF)`
- `Color(0xFFFAFAFA)`
- `Color(0xFFE5E5E5)`
- direct black text in status pills

Apply the pane theme and text theme here:

```dart
FilledButton.icon(
  style: FilledButton.styleFrom(
    minimumSize: const Size(0, 32),
    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
  ),
  onPressed: _triggerAITags,
  icon: const Icon(Icons.play_arrow, size: 16),
  label: const Text('生成'),
)
```

For the status pill:

```dart
Container(
  decoration: BoxDecoration(
    color: paneTheme.statusFillColor,
    borderRadius: BorderRadius.circular(999),
  ),
  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
  child: Text(
    _aiStatus!,
    style: Theme.of(context).textTheme.labelSmall?.copyWith(
      color: paneTheme.valueColor,
    ),
  ),
)
```

- [ ] **Step 4: Theme the tag section and pending tag chip controls**

Replace the current light pending-chip shell:

```dart
color: const Color(0xFFF8F8F8),
border: Border.all(color: const Color(0xFFE5E5E5)),
```

with a theme-aware version:

```dart
decoration: BoxDecoration(
  color: paneTheme.sectionSurface,
  border: Border.all(color: paneTheme.dividerColor),
  borderRadius: BorderRadius.circular(6),
),
```

Also theme:
- pending chip text
- divider line inside the chip
- action icons (`check`, `close`, `merge_type`, `edit`)

using `ColorScheme` roles instead of `Colors.black87` / `Colors.blueGrey`.

- [ ] **Step 5: Make the metadata row hierarchy match the approved plan**

Render each metadata field as label above value, not fixed-width left label.

Use this row pattern in the host metadata section:

```dart
Column(
  crossAxisAlignment: CrossAxisAlignment.start,
  children: [
    Text(label, style: Theme.of(context).textTheme.labelMedium?.copyWith(
      color: paneTheme.labelColor,
    )),
    const SizedBox(height: 2),
    Text(value, style: Theme.of(context).textTheme.bodyMedium?.copyWith(
      color: paneTheme.valueColor,
      fontWeight: FontWeight.w600,
    )),
  ],
)
```

- [ ] **Step 6: Run the targeted widget tests again**

Run:

```bash
flutter test test/screens/viewer/viewer_metadata_sidebar_test.dart test/screens/image_detail_screen_test.dart
```

Expected: some failures may remain because the host shells still need to adopt the shared pane surface.

### Task 4: Update the viewer sidebar host to become a real dark info pane shell

**Files:**
- Modify: `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`

- [ ] **Step 1: Remove the hardcoded light background shell**

Delete these assumptions:

```dart
const panelSurface = Color(0xFFF3F3F3);
color: panelSurface,
border: Border(left: BorderSide(color: Color(0xFFE5E5E5))),
```

Replace them with the shared pane theme:

```dart
final paneTheme = ImageMetadataPaneTheme.of(context);

return Container(
  key: const ValueKey('viewer-metadata-sidebar'),
  width: 320,
  decoration: BoxDecoration(
    color: paneTheme.paneSurface,
    border: Border(left: BorderSide(color: paneTheme.dividerColor)),
  ),
  child: Material(
    color: Colors.transparent,
    child: ImageMetadataPanel(...),
  ),
);
```

- [ ] **Step 2: Convert the host metadata section into a flat information block**

Remove the local `Card` from `_buildMetadataSection` and return a section-friendly layout that `ImageMetadataPanel` can wrap.

Expected outcome: the sidebar no longer creates a “card inside pane inside dark app” hierarchy.

## Chunk 3: Details Page Host + Final Verification

### Task 5: Make the dedicated image details screen use the same info-pane shell

**Files:**
- Modify: `flutter_app/lib/screens/image_detail_screen.dart`

- [ ] **Step 1: Convert the metadata column into a pane host**

Use the shared pane theme in both desktop and compact layouts, and expose a stable key:

```dart
final paneTheme = ImageMetadataPaneTheme.of(context);

Container(
  key: const ValueKey('image-detail-metadata-pane'),
  decoration: BoxDecoration(
    color: paneTheme.paneSurface,
    border: Border.all(color: paneTheme.dividerColor),
    borderRadius: BorderRadius.circular(12),
  ),
  child: ImageMetadataPanel(
    imageId: widget.image.id,
    metadataSection: _buildMetadataSection(context, paneTheme),
  ),
)
```

Notes:
- keep the desktop split layout
- reduce the “floating card” feel on the metadata side
- keep the image side more spacious than the metadata side

- [ ] **Step 2: Rebuild the local metadata section using label-above-value rows**

Update `_buildMetadataSection` and `_buildMetadataRow` to accept `ImageMetadataPaneTheme` instead of inferring text color from a forced panel background.

Use a compact section structure like this:

```dart
SelectionArea(
  child: Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text('元数据', style: Theme.of(context).textTheme.titleSmall),
      const SizedBox(height: 12),
      _buildMetadataRow(context, '文件名', widget.image.filename, paneTheme),
      const SizedBox(height: 12),
      _buildMetadataRow(context, '尺寸', widget.image.displaySize, paneTheme),
      ...
    ],
  ),
)
```

- [ ] **Step 3: Remove obsolete surface/foreground helpers after the refactor**

Delete now-redundant helpers when they are no longer used:
- `_foregroundForSurface`
- `_mutedForegroundForSurface`

Expected outcome: metadata coloring is now driven by shared pane theme + text theme only.

### Task 6: Run final verification on the affected files

**Files:**
- Modify: `flutter_app/lib/widgets/image_metadata_pane_theme.dart`
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart`
- Modify: `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`
- Modify: `flutter_app/lib/screens/image_detail_screen.dart`
- Modify: `flutter_app/test/screens/viewer/viewer_metadata_sidebar_test.dart`
- Modify: `flutter_app/test/screens/image_detail_screen_test.dart`

- [ ] **Step 1: Run static analysis on the touched files**

Run:

```bash
flutter analyze lib/widgets/image_metadata_pane_theme.dart lib/widgets/image_metadata_panel.dart lib/screens/viewer/viewer_metadata_sidebar.dart lib/screens/image_detail_screen.dart test/screens/viewer/viewer_metadata_sidebar_test.dart test/screens/image_detail_screen_test.dart
```

Expected: No errors on the changed files.

- [ ] **Step 2: Run the targeted widget tests**

Run:

```bash
flutter test test/screens/viewer/viewer_metadata_sidebar_test.dart test/screens/image_detail_screen_test.dart
```

Expected: PASS.

- [ ] **Step 3: Run one broader smoke pass for nearby theme-sensitive tests**

Run:

```bash
flutter test test/providers/theme_provider_test.dart test/app/material_app_shell_test.dart
```

Expected: PASS, confirming the redesign did not break existing theme plumbing.

- [ ] **Step 4: Manual desktop verification checklist**

Verify manually in the desktop app:
- dark mode shows no white metadata cards
- sidebar and details page feel like the same surface system
- AI section controls remain readable and clickable
- tags and pending-tag chips remain legible
- divider contrast is visible but subtle

- [ ] **Step 5: Stop here unless the user explicitly asks for a commit**

Do not create a git commit automatically.
