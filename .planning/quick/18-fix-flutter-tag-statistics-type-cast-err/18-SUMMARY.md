---
phase: quick-18
plan: 01
type: tdd
wave: 1
tags: [flutter, bugfix, tag-statistics, type-cast]
duration: "3 minutes"
completed: "2026-03-20"
files:
  created:
    - flutter_app/test/services/tag_service_test.dart
    - flutter_app/test/services/tag_service_test.mocks.dart
  modified:
    - flutter_app/lib/services/tag_service.dart
commits:
  - hash: a016cbd
    message: "test(quick-18): add failing test for getTagStatistics wrapped response"
  - hash: 247b885
    message: "fix(quick-18): parse wrapped {\"stats\": [...]} response in getTagStatistics"
---

# Quick Task 18: Fix Flutter Tag Statistics Type Cast Error Summary

## One-liner

Fixed type cast error in `getTagStatistics()` by parsing backend's wrapped `{"stats": [...]}` response instead of expecting a direct array.

## Problem

User clicking tag management button caused crash:
```
type '_Map<String, dynamic>' is not a subtype of type 'List<dynamic>' in type cast
```

**Root Cause:** Backend returns `{"stats": [...]}` but Flutter code expected `[...]` directly.

## Solution

### TDD Approach

**RED:** Created test expecting wrapped response - confirmed failure with exact error.

**GREEN:** Fixed parsing to extract `stats` array from wrapped response:

```dart
// Before (BUG):
final json = jsonDecode(response.body) as List;

// After (FIX):
final json = jsonDecode(response.body) as Map<String, dynamic>;
final stats = json['stats'] as List;
```

## Files Changed

| File | Change |
|------|--------|
| `flutter_app/lib/services/tag_service.dart` | Fixed `getTagStatistics()` to parse wrapped response |
| `flutter_app/test/services/tag_service_test.dart` | Added 4 tests for getTagStatistics |

## Test Results

- **New tests:** 4/4 passing
- **Tag-related tests:** All passing (Tag Model: 6, TagProvider: 17, TagService: 4)
- **Pre-existing failure:** `widget_test.dart` (default Flutter counter test, unrelated)

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- [x] Test file exists: `flutter_app/test/services/tag_service_test.dart`
- [x] Commit `a016cbd` exists
- [x] Commit `247b885` exists