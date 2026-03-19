---
phase: quick-18
plan: 01
type: tdd
wave: 1
depends_on: []
files_modified:
  - flutter_app/lib/services/tag_service.dart
  - flutter_app/test/services/tag_service_test.dart
autonomous: true
requirements: []
must_haves:
  truths:
    - "User can view tag statistics without type cast error"
    - "Tag statistics display shows correct data from API"
  artifacts:
    - path: "flutter_app/lib/services/tag_service.dart"
      provides: "Tag statistics API parsing"
      fix: "Parse wrapped JSON response {'stats': [...]}"
  key_links:
    - from: "tag_service.dart"
      to: "/tags/stats API"
      via: "getTagStatistics()"
      pattern: "json['stats'] as List"
---

<objective>
Fix Flutter tag statistics type cast error where API returns `{"stats": [...]}` but code expects direct `[...]`.

Purpose: Enable tag management button to display statistics without crashing.
Output: Working tag statistics with test coverage.
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
## Bug Analysis

**Error:** `type '_Map<String, dynamic>' is not a subtype of type 'List<dynamic>' in type cast`

**Root Cause:** 
- Backend (`internal/handler/tag_handler.go:321`) returns: `gin.H{"stats": stats}` → `{"stats": [...]}`
- Flutter (`tag_service.dart:199`) expects: `as List` → direct array

**Fix Required:**
```dart
// Before (BUG):
final json = jsonDecode(response.body) as List;

// After (FIX):
final json = jsonDecode(response.body) as Map<String, dynamic>;
final stats = json['stats'] as List;
```

## Files

- `flutter_app/lib/services/tag_service.dart:192-203` - Contains buggy `getTagStatistics()`
- `flutter_app/lib/models/tag.dart:94-125` - `TagStatistics` model with `fromJson`
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write failing test for getTagStatistics</name>
  <files>flutter_app/test/services/tag_service_test.dart</files>
  <behavior>
    - Test: getTagStatistics parses wrapped {"stats": [...]} response
    - Given: Mock HTTP response with {"stats": [{"tag_id": 1, "preferred_label": "test", ...}]}
    - Expect: Returns List<TagStatistics> with correct data
  </behavior>
  <action>
Create test file `flutter_app/test/services/tag_service_test.dart`:

1. Create a mock HTTP client that returns a wrapped JSON response:
   ```dart
   {
     "stats": [
       {"tag_id": 1, "preferred_label": "anime", "usage_count": 100, "pending_count": 5}
     ]
   }
   ```

2. Test that `getTagStatistics()` correctly extracts the `stats` array and parses TagStatistics objects.

3. Use mocktail or mockito for HTTP mocking (check project's existing test patterns in tag_provider_test.dart).

Run test to confirm it FAILS with current code (type cast error).
  </action>
  <verify>
    <automated>cd flutter_app && flutter test test/services/tag_service_test.dart 2>&1 | grep -E "(FAILED|type.*Map.*is not a subtype)"</automated>
  </verify>
  <done>Test file exists and fails with type cast error, proving the bug exists.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Fix getTagStatistics to parse wrapped response</name>
  <files>flutter_app/lib/services/tag_service.dart</files>
  <behavior>
    - Test from Task 1 now passes
    - API response {"stats": [...]} correctly parsed
  </behavior>
  <action>
Fix `flutter_app/lib/services/tag_service.dart` lines 199-202:

**Before:**
```dart
final json = jsonDecode(response.body) as List;
return json
    .map((e) => TagStatistics.fromJson(e as Map<String, dynamic>))
    .toList();
```

**After:**
```dart
final json = jsonDecode(response.body) as Map<String, dynamic>;
final stats = json['stats'] as List;
return stats
    .map((e) => TagStatistics.fromJson(e as Map<String, dynamic>))
    .toList();
```

This change extracts the `stats` array from the wrapped response before mapping to TagStatistics.
  </action>
  <verify>
    <automated>cd flutter_app && flutter test test/services/tag_service_test.dart</automated>
  </verify>
  <done>Test passes. getTagStatistics correctly parses {"stats": [...]} response.</done>
</task>

<task type="auto">
  <name>Task 3: Verify existing tests still pass</name>
  <files>flutter_app/test/</files>
  <action>
Run all existing Flutter tests to ensure the fix doesn't break anything:
```bash
cd flutter_app && flutter test
```

This validates that:
1. tag_provider_test.dart still works with the fixed service
2. No regressions in other tag-related functionality
  </action>
  <verify>
    <automated>cd flutter_app && flutter test 2>&1 | tail -5</automated>
  </verify>
  <done>All tests pass (no failures).</done>
</task>

</tasks>

<verification>
1. Unit test for getTagStatistics passes
2. All existing Flutter tests pass
3. Manual verification (optional): Run Flutter app, click tag management, see statistics load without error
</verification>

<success_criteria>
- [ ] Test file created with failing test proving the bug
- [ ] tag_service.dart fixed to parse wrapped response
- [ ] Test passes after fix
- [ ] All existing tests pass
- [ ] Commit with descriptive message
</success_criteria>

<output>
After completion:
1. Commit changes with message: `fix(flutter): parse wrapped {"stats": [...]} response in getTagStatistics`
2. Create `.planning/quick/18-fix-flutter-tag-statistics-type-cast-err/18-SUMMARY.md`
</output>

<commit_strategy>
## Atomic Commit Strategy

**Single commit after all tasks complete:**
```
fix(flutter): parse wrapped {"stats": [...]} response in getTagStatistics

- Add test for getTagStatistics with mocked HTTP response
- Fix JSON parsing to extract 'stats' array from wrapped response
- Backend returns {"stats": [...]}, not direct [...]

Fixes: type '_Map<String, dynamic>' is not a subtype of type 'List<dynamic>'
```

**Files to commit:**
- flutter_app/lib/services/tag_service.dart
- flutter_app/test/services/tag_service_test.dart (new file)
</commit_strategy>