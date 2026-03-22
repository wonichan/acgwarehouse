---
phase: quick
plan: 39
type: tdd
wave: 1
depends_on: []
files_modified: [
  "flutter_app/test/providers/image_provider_has_tags_test.dart",
  "flutter_app/lib/providers/image_provider.dart",
  "flutter_app/lib/services/api_service.dart",
  "flutter_app/lib/widgets/tag_filter_drawer.dart"
]
autonomous: true
requirements: [BUG-39]
must_haves:
  truths:
    - "User can toggle 'Show untagged images' switch and see only images without tags"
    - "API is called with has_tags=false parameter when untagged filter is enabled"
    - "Tag filter and untagged filter are mutually exclusive"
  artifacts:
    - path: "flutter_app/test/providers/image_provider_has_tags_test.dart"
      provides: "Unit tests for hasTags filter functionality"
    - path: "flutter_app/lib/providers/image_provider.dart"
      provides: "Working setHasTagsFilter method"
    - path: "flutter_app/lib/services/api_service.dart"
      provides: "API calls with correct has_tags parameter"
  key_links:
    - from: "TagFilterDrawer.onHasTagsChanged"
      to: "ImageListProvider.setHasTagsFilter"
      via: "callback invocation"
    - from: "ImageListProvider.setHasTagsFilter"
      to: "ApiService.fetchImages"
      via: "hasTags parameter"
    - from: "ApiService.fetchImages"
      to: "Backend API /api/v1/images"
      via: "has_tags query parameter"
---

<objective>
Fix the bug where the 'Show untagged images' (显示未打标签的图片) feature is not working in the tag filter drawer.

Purpose: The untagged filter switch appears in the UI but does not actually filter images to show only those without tags.

Output: Working untagged filter with passing TDD tests.
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@flutter_app/lib/providers/image_provider.dart
@flutter_app/lib/services/api_service.dart
@flutter_app/lib/widgets/tag_filter_drawer.dart
@internal/handler/image_handler.go

## Current Implementation Status

**Backend (Go):** Working correctly - tests pass for `has_tags=false` filter
- `TestImageHandlerListImagesFiltersByHasTagsFalse` - PASS
- `TestImageHandlerListImagesHasTagsFalseSupportsPagination` - PASS

**Frontend (Flutter):** Bug exists - no proper tests for the flow
- `setHasTagsFilter` exists in ImageListProvider but may have issues
- `ApiService.fetchImages` accepts `hasTags` parameter
- `TagFilterDrawer` has the UI switch but callback may not be wired correctly
- Current test mocks have empty `setHasTagsFilter` implementation

## Key Code Patterns

From `image_provider.dart`:
```dart
Future<void> setHasTagsFilter(bool? hasTags) async {
  debugPrint('setHasTagsFilter 被调用: hasTags=$hasTags');
  _hasTagsFilter = hasTags;
  if (hasTags != null) {
    _selectedTagIds = [];
  }
  _currentOffset = 0;
  _hasMore = true;
  _images = [];
  notifyListeners();
  await loadImages(refresh: true);
}
```

From `api_service.dart`:
```dart
if (hasTags != null) {
  queryParams['has_tags'] = hasTags.toString();
}
```

## Likely Bug Locations

1. **TagFilterDrawer callback:** `onHasTagsChanged` may not be invoking the callback correctly
2. **Parameter serialization:** `hasTags.toString()` produces "false" which the backend may not parse correctly (expects "false" lowercase)
3. **State management:** The filter state may not be properly synchronized between drawer and provider
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write TDD tests for hasTags filter functionality</name>
  <files>flutter_app/test/providers/image_provider_has_tags_test.dart</files>
  <behavior>
    - Test 1: setHasTagsFilter(false) should call API with has_tags=false parameter
    - Test 2: setHasTagsFilter(null) should clear the filter
    - Test 3: setHasTagsFilter(false) should clear selected tag IDs (mutually exclusive)
    - Test 4: setHasTagsFilter should reset pagination (offset=0, hasMore=true)
    - Test 5: TagFilterDrawer switch on should call setHasTagsFilter(false)
    - Test 6: TagFilterDrawer switch off should call setHasTagsFilter(null)
  </behavior>
  <action>
    Create comprehensive TDD tests in `flutter_app/test/providers/image_provider_has_tags_test.dart`:
    
    1. Test that `setHasTagsFilter(false)` calls `ApiService.fetchImages` with `hasTags: false`
    2. Test that `setHasTagsFilter(null)` clears the filter and doesn't pass hasTags to API
    3. Test that setting hasTagsFilter clears any existing tag filter selection
    4. Test pagination reset when filter changes
    5. Mock the ApiService and verify the correct parameters are passed
    6. Test the TagFilterDrawer widget interaction with the switch
    
    Use mocktail for mocking. Follow existing test patterns in `image_provider_test.dart`.
    
    Tests MUST fail initially (RED phase) - this verifies the bug exists.
  </action>
  <verify>
    <automated>cd flutter_app && flutter test test/providers/image_provider_has_tags_test.dart --reporter=compact</automated>
    <expect>Tests should FAIL initially (confirming bug exists)</expect>
  </verify>
  <done>Test file created with 6+ failing tests demonstrating the hasTags filter bug</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Fix hasTags filter in ImageProvider and ApiService</name>
  <files>flutter_app/lib/providers/image_provider.dart, flutter_app/lib/services/api_service.dart</files>
  <behavior>
    - Verify setHasTagsFilter correctly passes hasTags to loadImages
    - Verify loadImages passes hasTags parameter to ApiService.fetchImages
    - Verify ApiService correctly serializes hasTags to query parameter
    - Debug print statements should show correct values
  </behavior>
  <action>
    Fix the hasTags filter bug by investigating and correcting:
    
    1. In `image_provider.dart`:
       - Verify `setHasTagsFilter` calls `loadImages(refresh: true)` after setting state
       - Verify `loadImages` passes `_hasTagsFilter` to `_apiService.fetchImages`
       - Check debug print shows correct hasTags value
    
    2. In `api_service.dart`:
       - Verify `hasTags` parameter is correctly added to queryParams
       - Check that `hasTags.toString()` produces "false" (lowercase) which Go parses correctly
       - Add debug print to verify the API URL includes has_tags parameter
    
    3. Common issues to check:
       - Parameter name mismatch (hasTags vs has_tags)
       - Boolean serialization (False vs false)
       - State not being passed through the call chain
       - notifyListeners() called before async operation completes
    
    Run tests after each fix to see progress (GREEN phase).
  </action>
  <verify>
    <automated>cd flutter_app && flutter test test/providers/image_provider_has_tags_test.dart --reporter=compact</automated>
    <expect>All tests should PASS after fix</expect>
  </verify>
  <done>All TDD tests pass, hasTags filter works correctly from UI to API</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Verify TagFilterDrawer integration and run all tests</name>
  <files>flutter_app/lib/widgets/tag_filter_drawer.dart</files>
  <behavior>
    - TagFilterDrawer switch toggles correctly
    - onHasTagsChanged callback is invoked with correct values
    - Filter drawer state syncs with provider state
  </behavior>
  <action>
    Verify the TagFilterDrawer integration and run full test suite:
    
    1. In `tag_filter_drawer.dart`:
       - Verify SwitchListTile value is `widget.hasTagsFilter == false`
       - Verify onChanged calls `widget.onHasTagsChanged?.call(value ? false : null)`
       - Check that the switch correctly reflects the filter state
    
    2. Add integration-style test if needed to verify end-to-end flow
    
    3. Run full test suite to ensure no regressions:
       - Run all provider tests
       - Run all screen tests
       - Run Go backend tests for has_tags
    
    4. Verify the fix with debug output:
       - Toggle switch in drawer
       - Check debug console shows correct API call with has_tags=false
    
    Refactor if needed while keeping tests passing (REFACTOR phase).
  </action>
  <verify>
    <automated>cd flutter_app && flutter test --reporter=compact 2>&1 | tail -5</automated>
    <expect>All tests pass (9+ tests from image_provider tests + new has_tags tests)</expect>
  </verify>
  <done>All tests pass, integration verified, bug is fixed</done>
</task>

</tasks>

<verification>
1. Run TDD tests: `flutter test test/providers/image_provider_has_tags_test.dart`
2. Run all Flutter tests: `flutter test`
3. Run Go backend tests: `go test ./internal/handler/... -run HasTags`
4. Manual verification:
   - Open gallery screen
   - Open tag filter drawer
   - Toggle "未打标签" switch
   - Verify API call includes `has_tags=false`
   - Verify only untagged images are displayed
</verification>

<success_criteria>
- [ ] TDD test file created with 6+ tests
- [ ] All tests pass after fix implementation
- [ ] TagFilterDrawer correctly invokes callback when switch toggled
- [ ] API calls include correct `has_tags=false` parameter
- [ ] Backend receives and processes the filter correctly
- [ ] No regressions in existing tests
</success_criteria>

<output>
After completion, create `.planning/quick/39-bug/39-SUMMARY.md`
</output>

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1: Write TDD tests | None | Starting point - create failing tests to verify bug |
| Task 2: Fix hasTags filter | Task 1 | Need failing tests first to guide the fix |
| Task 3: Verify integration | Task 2 | Verify the fix works end-to-end |

## Parallel Execution Graph

Wave 1 (Sequential TDD - must be ordered):
├── Task 1: Write TDD tests (no dependencies) 
├── Task 2: Fix hasTags filter (depends: Task 1)
└── Task 3: Verify integration (depends: Task 2)

**Note:** This is a TDD plan, so tasks must execute sequentially in RED→GREEN→REFACTOR order.

## Tasks

### Task 1: Write TDD tests for hasTags filter functionality
**Description**: Create comprehensive test file for hasTags filter with mocktail
**Delegation Recommendation**:
- Category: `deep` - Requires understanding of existing test patterns and proper mocking
- Skills: [`test-driven-development`, `verification-before-completion`] - TDD workflow and test verification
**Skills Evaluation:**
- INCLUDED `test-driven-development`: Required for RED→GREEN→REFACTOR workflow
- INCLUDED `verification-before-completion`: Must verify tests fail initially (confirming bug)
- OMITTED `brainstorming`: Implementation path is clear from existing code patterns
**Depends On**: None
**Acceptance Criteria**: Test file exists with 6+ failing tests, all following mocktail patterns from existing tests

### Task 2: Fix hasTags filter in ImageProvider and ApiService
**Description**: Fix the bug causing hasTags filter to not work
**Delegation Recommendation**:
- Category: `deep` - Requires systematic debugging of the filter flow
- Skills: [`systematic-debugging`, `verification-before-completion`] - Debug the filter chain, verify fix
**Skills Evaluation:**
- INCLUDED `systematic-debugging`: Need to trace through ImageProvider → ApiService → HTTP call
- INCLUDED `verification-before-completion`: Verify each fix with test runs
- OMITTED `test-driven-development`: Already handled in Task 1
**Depends On**: Task 1
**Acceptance Criteria**: All TDD tests pass, debug output shows correct has_tags=false in API calls

### Task 3: Verify TagFilterDrawer integration and run all tests
**Description**: Verify drawer integration and run full test suite
**Delegation Recommendation**:
- Category: `quick` - Integration verification and test running
- Skills: [`verification-before-completion`] - Verify no regressions
**Skills Evaluation:**
- INCLUDED `verification-before-completion`: Must verify all existing tests still pass
- OMITTED `systematic-debugging`: Fix already implemented
- OMITTED `test-driven-development`: Tests already written
**Depends On**: Task 2
**Acceptance Criteria**: All Flutter tests pass (9+ provider tests + existing tests), no regressions

## Commit Strategy

1. **Task 1 commit (RED phase)**:
   ```
   test(39): add TDD tests for hasTags filter functionality
   
   - Add comprehensive tests for setHasTagsFilter
   - Tests currently failing, confirming bug exists
   - Includes tests for parameter passing and state management
   ```

2. **Task 2 commit (GREEN phase)**:
   ```
   fix(39): fix hasTags filter not working in tag filter drawer
   
   - Fix [specific issue found during debugging]
   - All TDD tests now passing
   - API calls correctly include has_tags=false parameter
   ```

3. **Task 3 commit (REFACTOR phase)**:
   ```
   refactor(39): verify integration and clean up
   
   - Verify TagFilterDrawer integration
   - Run full test suite, no regressions
   - Clean up any debug code if needed
   ```

## TODO List (ADD THESE)

> CALLER: Add these TODOs using TodoWrite/TaskCreate and execute by wave.

### Wave 1 (Sequential TDD Execution)

- [ ] **1. Write TDD tests for hasTags filter functionality**
  - What: Create `test/providers/image_provider_has_tags_test.dart` with 6+ tests using mocktail
  - Depends: None
  - Blocks: 2
  - Category: `deep`
  - Skills: [`test-driven-development`, `verification-before-completion`]
  - QA: Run `flutter test test/providers/image_provider_has_tags_test.dart` - expect FAILING tests

- [ ] **2. Fix hasTags filter in ImageProvider and ApiService**
  - What: Debug and fix the hasTags filter bug, ensure API calls include correct parameter
  - Depends: 1
  - Blocks: 3
  - Category: `deep`
  - Skills: [`systematic-debugging`, `verification-before-completion`]
  - QA: Run `flutter test test/providers/image_provider_has_tags_test.dart` - expect ALL PASSING

- [ ] **3. Verify TagFilterDrawer integration and run all tests**
  - What: Verify drawer integration, run full test suite, ensure no regressions
  - Depends: 2
  - Blocks: None
  - Category: `quick`
  - Skills: [`verification-before-completion`]
  - QA: Run `flutter test` - expect ALL tests passing

## Execution Instructions

1. **Task 1**: Create TDD test file (RED phase)
   ```
   category="deep"
   load_skills=["test-driven-development", "verification-before-completion"]
   prompt="Create test/providers/image_provider_has_tags_test.dart with 6+ tests for hasTags filter using mocktail. Follow existing patterns from image_provider_test.dart. Tests should verify: setHasTagsFilter(false) calls API with has_tags=false, setHasTagsFilter clears tag selection, pagination resets, etc. Tests MUST fail initially to confirm bug exists."
   ```

2. **Task 2**: Fix the bug (GREEN phase)
   ```
   category="deep"
   load_skills=["systematic-debugging", "verification-before-completion"]
   prompt="Fix the hasTags filter bug in flutter_app. Debug the flow: TagFilterDrawer → ImageListProvider.setHasTagsFilter → ApiService.fetchImages → HTTP call. Check: parameter passing, boolean serialization, state management. Run tests after each change until all pass."
   ```

3. **Task 3**: Verify and finalize (REFACTOR phase)
   ```
   category="quick"
   load_skills=["verification-before-completion"]
   prompt="Verify TagFilterDrawer integration and run full test suite. Check: drawer switch works, no regressions in existing tests, all 9+ provider tests pass. Clean up if needed."
   ```

4. Final QA: Verify all tasks pass their QA criteria
