---
phase: 30-filter-untagged-images
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/repository/image_repository.go
  - internal/handler/image_handler.go
  - flutter_app/lib/services/api_service.dart
  - flutter_app/lib/providers/image_provider.dart
  - flutter_app/lib/widgets/tag_filter_drawer.dart
autonomous: true
requirements: [FILTER-UNTAGGED-01]
must_haves:
  truths:
    - "User can filter to see only untagged images"
    - "Untagged filter is mutually exclusive with tag selection"
    - "Untagged filter respects pagination and sorting"
  artifacts:
    - path: "internal/repository/image_repository.go"
      provides: "FindUntagged and CountUntagged methods"
      contains: "FindUntagged"
    - path: "internal/handler/image_handler.go"
      provides: "has_tags parameter handling"
      contains: "has_tags"
    - path: "flutter_app/lib/widgets/tag_filter_drawer.dart"
      provides: "Untagged filter toggle UI"
      min_lines: 140
  key_links:
    - from: "flutter_app/lib/widgets/tag_filter_drawer.dart"
      to: "ImageListProvider.setHasTagsFilter"
      via: "onChanged callback"
    - from: "image_handler.go"
      to: "image_repository.FindUntagged"
      via: "has_tags=false parameter"
---

<objective>
Add untagged image filter functionality allowing users to view only images without any tags.

Purpose: Help users identify images that need tagging attention - a common workflow in image library management.
Output: Working filter accessible from the tag filter drawer.
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

## Key Interfaces

### Backend Repository (internal/repository/image_repository.go)
```go
type ImageRepository interface {
    FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
    CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error)
    // ADD: FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
    // ADD: CountUntagged(ctx context.Context) (int64, error)
}
```

### Backend Handler (internal/handler/image_handler.go)
```go
// Current: tag_ids parameter for filtering images WITH specific tags
// ADD: has_tags parameter (true/false) for filtering untagged images
// MUTUAL EXCLUSIVITY: has_tags=false is incompatible with tag_ids
```

### Frontend API (flutter_app/lib/services/api_service.dart)
```dart
Future<PaginationResponse<ImageModel>> fetchImages({
  int offset = 0,
  int limit = 20,
  String sortBy = 'created_at',
  String sortDir = 'desc',
  List<int>? tagIds,
  // ADD: bool? hasTags,
});
```

### Frontend Provider (flutter_app/lib/providers/image_provider.dart)
```dart
class ImageListProvider extends ChangeNotifier {
  List<int> _selectedTagIds = [];
  // ADD: bool? _hasTagsFilter;
  // ADD: Future<void> setHasTagsFilter(bool? hasTags);
}
```

### Frontend UI (flutter_app/lib/widgets/tag_filter_drawer.dart)
```dart
// ADD: Toggle widget for "未打标签" option
// When ON: calls onFilterChanged with hasTagsFilter=true
// Mutually exclusive with tag selection
```

## Database Schema (for reference)
```sql
-- Images table
CREATE TABLE images (id INTEGER PRIMARY KEY, ...);

-- Image-Tag relationship
CREATE TABLE image_tags (
  id INTEGER PRIMARY KEY,
  image_id INTEGER REFERENCES images(id) ON DELETE CASCADE,
  tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
  review_state TEXT DEFAULT 'pending'
);
```
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Backend Repository - Add Untagged Query Methods</name>
  <files>
    internal/repository/image_repository.go,
    internal/repository/image_repository_test.go
  </files>
  <behavior>
    - Test 1: FindUntagged returns only images without any image_tags entries
    - Test 2: FindUntagged respects limit/offset pagination
    - Test 3: FindUntagged respects sortBy and sortDir parameters
    - Test 4: CountUntagged returns correct count of untagged images
    - Test 5: FindUntagged returns empty slice when all images have tags
  </behavior>
  <action>
    1. Add `FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)` to ImageRepository interface
    2. Add `CountUntagged(ctx context.Context) (int64, error)` to ImageRepository interface
    3. Implement in sqliteImageRepository using LEFT JOIN pattern:
       ```sql
       SELECT i.* FROM images i
       LEFT JOIN image_tags it ON it.image_id = i.id
       WHERE it.id IS NULL
       ORDER BY [sortBy] [sortDir]
       LIMIT ? OFFSET ?
       ```
    4. Write tests first (TDD), then implement
    5. Reuse the scanRow helper pattern from existing FindAll method
  </action>
  <verify>
    <automated>go test ./internal/repository/... -run TestFindUntagged -v</automated>
  </verify>
  <done>
    - FindUntagged method returns only images without tags
    - CountUntagged returns accurate count
    - All tests pass
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Backend Handler - Add has_tags Parameter</name>
  <files>
    internal/handler/image_handler.go,
    internal/handler/image_handler_test.go
  </files>
  <behavior>
    - Test 1: GET /api/v1/images?has_tags=false returns untagged images
    - Test 2: GET /api/v1/images?has_tags=true returns all images (default behavior)
    - Test 3: GET /api/v1/images?has_tags=false&tag_ids=1 returns 400 Bad Request
    - Test 4: Response includes correct total count for untagged images
    - Test 5: Pagination works correctly with has_tags=false
  </behavior>
  <action>
    1. In ListImages handler, parse `has_tags` query parameter:
       ```go
       hasTagsStr := c.Query("has_tags")
       var hasTags *bool
       if hasTagsStr != "" {
           val := hasTagsStr == "true"
           hasTags = &val
       }
       ```
    2. Add validation: if has_tags=false AND tag_ids provided, return 400:
       ```json
       {"error": "has_tags=false is incompatible with tag_ids parameter"}
       ```
    3. When has_tags=false, call FindUntagged and CountUntagged
    4. When has_tags=true or omitted, maintain existing behavior
    5. Write handler tests first (TDD)
  </action>
  <verify>
    <automated>go test ./internal/handler/... -run TestListImages_has_tags -v</automated>
  </verify>
  <done>
    - Handler accepts has_tags parameter
    - has_tags=false returns untagged images
    - Mutual exclusivity validation works
    - All tests pass
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Frontend API & Provider - Add hasTags Filter State</name>
  <files>
    flutter_app/lib/services/api_service.dart,
    flutter_app/lib/providers/image_provider.dart,
    flutter_app/test/services/api_service_test.dart,
    flutter_app/test/providers/image_provider_test.dart
  </files>
  <behavior>
    - Test 1: fetchImages(hasTags: false) sends has_tags=false query param
    - Test 2: fetchImages(hasTags: true) sends has_tags=true query param
    - Test 3: fetchImages(hasTags: null) does not send has_tags param
    - Test 4: setHasTagsFilter(false) clears selectedTagIds and reloads
    - Test 5: setTagFilter([1,2]) clears hasTagsFilter
  </behavior>
  <action>
    1. Update ApiService.fetchImages signature:
       ```dart
       Future<PaginationResponse<ImageModel>> fetchImages({
         int offset = 0,
         int limit = 20,
         String sortBy = 'created_at',
         String sortDir = 'desc',
         List<int>? tagIds,
         bool? hasTags,  // NEW
       })
       ```
    2. Add has_tags to queryParams when hasTags is not null
    3. Update ImageListProvider:
       ```dart
       bool? _hasTagsFilter;
       bool? get hasTagsFilter => _hasTagsFilter;
       
       Future<void> setHasTagsFilter(bool? hasTags) async {
         _hasTagsFilter = hasTags;
         if (hasTags != null) {
           _selectedTagIds = [];  // Clear tag selection
         }
         _currentOffset = 0;
         _hasMore = true;
         _images = [];
         notifyListeners();
         await loadImages(refresh: true);
       }
       ```
    4. Modify setTagFilter to clear _hasTagsFilter when setting tags
    5. Update loadImages to pass hasTags to API
    6. Write tests first (TDD)
  </action>
  <verify>
    <automated>cd flutter_app && flutter test test/services/api_service_test.dart test/providers/image_provider_test.dart</automated>
  </verify>
  <done>
    - API service sends has_tags parameter
    - Provider manages hasTagsFilter state
    - Mutual exclusivity enforced in provider
    - All tests pass
  </done>
</task>

<task type="auto">
  <name>Task 4: Frontend UI - Add Untagged Filter Option</name>
  <files>
    flutter_app/lib/widgets/tag_filter_drawer.dart
  </files>
  <action>
    1. Add callback parameter to TagFilterDrawer for hasTags filter:
       ```dart
       class TagFilterDrawer extends StatefulWidget {
         final Function(List<int> tagIds)? onFilterChanged;
         final Function(bool hasTags)? onHasTagsChanged;  // NEW
         final bool? hasTagsFilter;  // NEW - for initial state
         ...
       }
       ```
    2. Add toggle widget at top of drawer (after header, before search):
       ```dart
       // Untagged filter toggle
       SwitchListTile(
         title: const Text('未打标签'),
         subtitle: const Text('显示没有标签的图片'),
         value: widget.hasTagsFilter == false,
         onChanged: (value) {
           widget.onHasTagsChanged?.call(!value);
         },
       ),
       ```
    3. Update gallery_screen.dart to pass callbacks:
       ```dart
       TagFilterDrawer(
         hasTagsFilter: provider.hasTagsFilter,
         onFilterChanged: (tagIds) {
           context.read<ImageListProvider>().setTagFilter(tagIds);
         },
         onHasTagsChanged: (hasTags) {
           context.read<ImageListProvider>().setHasTagsFilter(hasTags ? null : false);
         },
       )
       ```
    4. Update the "clear selection" button to also clear hasTagsFilter
    5. Add visual indicator in drawer header when untagged filter is active
  </action>
  <verify>
    <automated>cd flutter_app && flutter analyze lib/widgets/tag_filter_drawer.dart lib/screens/gallery_screen.dart</automated>
  </verify>
  <done>
    - Toggle appears in tag filter drawer
    - Toggling ON shows untagged images
    - Toggling OFF returns to normal mode
    - Selecting a tag clears untagged filter
    - No analyzer errors
  </done>
</task>

</tasks>

<verification>
## Manual Verification Steps

1. Start backend: `go run cmd/server/main.go`
2. Start frontend: `cd flutter_app && flutter run -d chrome`
3. Open tag filter drawer
4. Toggle "未打标签" ON
5. Verify only untagged images appear
6. Toggle OFF, verify all images return
7. Select a specific tag, verify untagged filter clears
</verification>

<success_criteria>
1. Backend `GET /api/v1/images?has_tags=false` returns untagged images with correct pagination
2. Backend returns 400 when both `has_tags=false` and `tag_ids` are provided
3. Frontend toggle "未打标签" filters to untagged images
4. Mutual exclusivity: selecting tags clears untagged filter, toggling untagged clears tag selection
5. All tests pass (Go and Flutter)
</success_criteria>

<output>
After completion, create `.planning/quick/30-筛选未打标签图片/30-SUMMARY.md`
</output>