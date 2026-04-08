---
phase: quick
plan: 40
subsystem: flutter
requirements: [QUICK-40]
tech-stack:
  added: []
  patterns:
    - "Fluent NavigationPane destination extension"
    - "Self-contained CollectionService-driven page state"
    - "Widget tests for empty and loaded collection states"
dependency-graph:
  requires: []
  provides: ["desktop favorites discoverability", "collections browser entry"]
  affects:
    - "flutter_app/lib/providers/navigation_provider.dart"
    - "flutter_app/lib/app/fluent_app_shell.dart"
    - "flutter_app/lib/app/fluent_screens.dart"
    - "flutter_app/lib/widgets/fluent_collections_content.dart"
key-files:
  created:
    - "flutter_app/lib/widgets/fluent_collections_content.dart"
    - "flutter_app/test/widgets/fluent_collections_content_test.dart"
  modified:
    - "flutter_app/lib/providers/navigation_provider.dart"
    - "flutter_app/lib/app/fluent_app_shell.dart"
    - "flutter_app/lib/app/fluent_screens.dart"
    - "flutter_app/test/app/fluent_app_shell_test.dart"
decisions:
  - "Keep existing collection semantics and surface them as 收藏 instead of inventing a new favorites model"
  - "Append the new navigation item at index 7 to avoid shifting existing navigation constants"
metrics:
  duration: "same-session quick task"
  completed_date: "2026-04-08"
  commits: 0
---

# Quick Task 40: Flutter desktop favorites entry - Summary

**One-liner:** Added a discoverable 收藏 destination to the Windows/Fluent desktop shell and implemented a collections browser page so users can view images they previously added via the existing 收藏到合集 flow.

## What Was Built

### Desktop navigation entry
- Added `collectionsIndex = 7` and expanded `NavigationProvider.itemCount` to 8 in `flutter_app/lib/providers/navigation_provider.dart`.
- Added a new `PaneItem` titled `收藏` to `flutter_app/lib/app/fluent_app_shell.dart`.
- Wired the new destination to `FluentCollectionsPage` while preserving all existing navigation indices.

### Collections browser page
- Added `FluentCollectionsPage` in `flutter_app/lib/app/fluent_screens.dart`.
- Added new `flutter_app/lib/widgets/fluent_collections_content.dart`.
- The page:
  - loads collections with `CollectionService.fetchCollections(limit: 200)`
  - auto-selects the first collection when available
  - loads collection images with `CollectionService.fetchCollectionImages(...)`
  - shows dedicated states for loading, empty collections, empty selected collection, and retryable fetch errors
  - reuses `FluentImageCard` for image display and existing image detail navigation on double click

## Tests and Verification

### Widget / shell tests
Command:

```bash
flutter test test/widgets/fluent_collections_content_test.dart test/app/fluent_app_shell_test.dart
```

Result: **PASS**

Coverage added:
- `flutter_app/test/widgets/fluent_collections_content_test.dart`
  - empty state when there are no collections
  - switching between collections and showing image grid vs empty-state message
- `flutter_app/test/app/fluent_app_shell_test.dart`
  - verifies 收藏 destination exists in shell flow
  - verifies page title updates to `收藏`

### Analyzer
Command:

```bash
flutter analyze lib/app/fluent_app_shell.dart lib/app/fluent_screens.dart lib/providers/navigation_provider.dart lib/widgets/fluent_collections_content.dart test/app/fluent_app_shell_test.dart test/widgets/fluent_collections_content_test.dart
```

Result: **PASS** after replacing deprecated `withOpacity` usage.

### Diagnostics
- LSP diagnostics reported no remaining issues in changed Dart source files.

## Notes

- This quick task intentionally did **not** introduce a special backend favorites bucket. The existing collection model already backed the 收藏 UX, so the fix surfaces that model instead of changing storage semantics.
- Material-side navigation was not expanded because the user-reported issue is specific to the Windows/Fluent desktop shell; existing indices were preserved by appending the new destination.
