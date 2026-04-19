# Tag Governance Filter Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add AND-based composite filtering to the tag governance page, supporting level, orphan-only, usage-range, and AI/manual source filters via the existing `GET /tags/governance` endpoint.

**Architecture:** Extend the existing governance query pipeline end-to-end: handler parses new query params → service applies all filters before pagination → frontend uses draft/applied filter state to batch edits before triggering reload. No new endpoints, no schema changes.

**Tech Stack:** Go, Gin, SQLite, Flutter, Provider, fluent_ui

**Spec:** `docs/superpowers/specs/2026-04-19-tag-filter-design.md`

---

## File Structure / Responsibility Map

### Backend domain
- Create: `internal/domain/tag_filter.go` — `GovernanceTagFilter` struct + validation

### Backend handler
- Modify: `internal/handler/tag_handler.go:420-438` — `GetGovernanceTags` param parsing, `GetTagStats` call site migration

### Backend service
- Modify: `internal/service/tag_admin_service.go:704-765` — `resolveTagSlice` → filter-aware `resolveFilteredTagSlice`, `ListGovernanceTags` signature change

### Frontend model
- Create: `flutter_app/lib/models/tag_governance_filter.dart` — `TagGovernanceFilterState` with draft/applied semantics

### Frontend service
- Modify: `flutter_app/lib/services/tag_service.dart:297-327` — `fetchGovernanceTags` accepts filter params

### Frontend provider
- Modify: `flutter_app/lib/providers/tag_provider.dart:28-42,489-543,736-773` — draft/applied filter state, selection clearing on filter apply

### Frontend UI
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart` — filter panel insertion
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_list.dart` — remove any result-set-altering local filtering

### Tests
- Create: `internal/domain/tag_filter_test.go`
- Modify: `internal/handler/tag_handler_test.go`
- Modify: `internal/service/tag_admin_service_test.go`
- Create: `flutter_app/test/models/tag_governance_filter_test.dart`
- Modify: `flutter_app/test/services/tag_service_test.dart`
- Modify: `flutter_app/test/providers/tag_provider_test.dart`
- Create: `flutter_app/test/widgets/tag_management_list_test.dart`
- Modify: `flutter_app/test/widgets/tag_management_workspace_test.dart`

---

## Chunk 1: Backend Domain — Filter Struct and Validation

### Task 1: Create GovernanceTagFilter with validation

**Files:**
- Create: `internal/domain/tag_filter.go`
- Test: `internal/domain/tag_filter_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/domain/tag_filter_test.go`:

```go
package domain

import "testing"

func TestGovernanceTagFilterDefaults(t *testing.T) {
	f := GovernanceTagFilter{}
	if f.HasFilters() {
		t.Error("empty filter should report HasFilters=false")
	}
}

func TestGovernanceTagFilterValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*GovernanceTagFilter)
		wantErr bool
	}{
		{
			name:    "valid empty filter",
			modify:  func(f *GovernanceTagFilter) {},
			wantErr: false,
		},
		{
			name:    "valid levels",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"root", "parent"} },
			wantErr: false,
		},
		{
			name:    "invalid level value",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"grandpa"} },
			wantErr: true,
		},
		{
			name:    "orphan only is valid",
			modify:  func(f *GovernanceTagFilter) { f.OrphanOnly = true },
			wantErr: false,
		},
		{
			name:    "negative min usage is invalid",
			modify:  func(f *GovernanceTagFilter) { n := int(-1); f.MinUsageCount = &n },
			wantErr: true,
		},
		{
			name:    "negative max usage is invalid",
			modify:  func(f *GovernanceTagFilter) { n := int(-5); f.MaxUsageCount = &n },
			wantErr: true,
		},
		{
			name:    "min greater than max is invalid",
			modify:  func(f *GovernanceTagFilter) { lo, hi := 100, 10; f.MinUsageCount = &lo; f.MaxUsageCount = &hi },
			wantErr: true,
		},
		{
			name:    "valid usage range",
			modify:  func(f *GovernanceTagFilter) { lo, hi := 10, 100; f.MinUsageCount = &lo; f.MaxUsageCount = &hi },
			wantErr: false,
		},
		{
			name:    "source flags valid",
			modify:  func(f *GovernanceTagFilter) { f.SourceAI = true; f.SourceManual = true },
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := GovernanceTagFilter{}
			tt.modify(&f)
			err := f.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/... -run TestGovernanceTagFilter -v`
Expected: FAIL — `GovernanceTagFilter` type does not exist

- [ ] **Step 3: Write the implementation**

Create `internal/domain/tag_filter.go`:

```go
package domain

import (
	"errors"
	"fmt"
)

var validLevels = map[string]bool{"root": true, "parent": true, "child": true}

// GovernanceTagFilter carries all filter parameters for the governance tag list query.
type GovernanceTagFilter struct {
	Search         string
	Levels         []string
	OrphanOnly     bool
	MinUsageCount  *int
	MaxUsageCount  *int
	SourceAI       bool
	SourceManual   bool
	Limit          int
	Offset         int
}

// HasFilters returns true if any filter beyond search/pagination is active.
func (f *GovernanceTagFilter) HasFilters() bool {
	return len(f.Levels) > 0 ||
		f.OrphanOnly ||
		f.MinUsageCount != nil ||
		f.MaxUsageCount != nil ||
		f.SourceAI ||
		f.SourceManual
}

// Validate checks all filter fields for semantic correctness.
func (f *GovernanceTagFilter) Validate() error {
	for _, l := range f.Levels {
		if !validLevels[l] {
			return fmt.Errorf("invalid level %q: must be root, parent, or child", l)
		}
	}
	if f.MinUsageCount != nil && *f.MinUsageCount < 0 {
		return errors.New("min_usage_count must be non-negative")
	}
	if f.MaxUsageCount != nil && *f.MaxUsageCount < 0 {
		return errors.New("max_usage_count must be non-negative")
	}
	if f.MinUsageCount != nil && f.MaxUsageCount != nil && *f.MinUsageCount > *f.MaxUsageCount {
		return errors.New("min_usage_count must not exceed max_usage_count")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/... -run TestGovernanceTagFilter -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/tag_filter.go internal/domain/tag_filter_test.go
git commit -m "feat(tag-filter): add GovernanceTagFilter domain struct with validation"
```

---

## Chunk 2: Backend Handler — Parameter Parsing

### Task 2: Extend GetGovernanceTags handler to parse filter params

**Files:**
- Modify: `internal/handler/tag_handler.go:420-438`
- Test: `internal/handler/tag_handler_test.go`

- [ ] **Step 1: Write the failing handler test**

Add to `internal/handler/tag_handler_test.go`:

```go
func TestGetGovernanceTags_WithFilters(t *testing.T) {
	// Setup: create handler with test repos and admin service
	// Seed tags with known levels, parents, and usage counts

	tests := []struct {
		name       string
		query      string
		wantCode   int
		wantTotal  int
	}{
		{
			name:      "filter by levels root,parent",
			query:     "levels=root,parent&limit=50&offset=0",
			wantCode:  200,
		},
		{
			name:      "filter orphan_only excludes root",
			query:     "orphan_only=true&limit=50&offset=0",
			wantCode:  200,
		},
		{
			name:      "filter by source_ai",
			query:     "source_ai=true&limit=50&offset=0",
			wantCode:  200,
		},
		{
			name:      "invalid level returns 400",
			query:     "levels=grandpa&limit=50&offset=0",
			wantCode:  400,
		},
		{
			name:      "min > max returns 400",
			query:     "min_usage_count=100&max_usage_count=10&limit=50&offset=0",
			wantCode:  400,
		},
		{
			name:      "negative usage count returns 400",
			query:     "min_usage_count=-1&limit=50&offset=0",
			wantCode:  400,
		},
		{
			name:      "invalid boolean returns 400",
			query:     "source_ai=yes&limit=50&offset=0",
			wantCode:  400,
		},
		{
			name:      "search plus filters returns 200",
			query:     "search=test&levels=child&source_ai=true&limit=50&offset=0",
			wantCode:  200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// req := httptest.NewRequest("GET", "/api/v1/tags/governance?"+tt.query, nil)
			// w := httptest.NewRecorder()
			// handler.GetGovernanceTags(w, req)
			// assert response code matches tt.wantCode
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/... -run TestGetGovernanceTags_WithFilters -v`
Expected: FAIL — new query params ignored or type not recognized

- [ ] **Step 3: Implement handler parameter parsing**

Modify `GetGovernanceTags` in `internal/handler/tag_handler.go`:

```go
func (h *TagHandler) GetGovernanceTags(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)
	search := strings.TrimSpace(c.Query("search"))

	filter := domain.GovernanceTagFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	// Parse levels (comma-delimited)
	if raw := strings.TrimSpace(c.Query("levels")); raw != "" {
		parts := strings.Split(raw, ",")
		seen := make(map[string]bool, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" && !seen[p] {
				seen[p] = true
				filter.Levels = append(filter.Levels, p)
			}
		}
	}

	// Parse booleans — reject any non-empty value that isn't "true" or "false"
	if raw := c.Query("orphan_only"); raw != "" {
		if raw != "true" && raw != "false" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "orphan_only must be true or false"})
			return
		}
		filter.OrphanOnly = raw == "true"
	}
	if raw := c.Query("source_ai"); raw != "" {
		if raw != "true" && raw != "false" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_ai must be true or false"})
			return
		}
		filter.SourceAI = raw == "true"
	}
	if raw := c.Query("source_manual"); raw != "" {
		if raw != "true" && raw != "false" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_manual must be true or false"})
			return
		}
		filter.SourceManual = raw == "true"
	}

	// Parse usage range
	if raw := c.Query("min_usage_count"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_usage_count must be an integer"})
			return
		}
		filter.MinUsageCount = &v
	}
	if raw := c.Query("max_usage_count"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "max_usage_count must be an integer"})
			return
		}
		filter.MaxUsageCount = &v
	}

	if err := filter.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rows, total, err := h.adminSvc.ListGovernanceTagsFiltered(c.Request.Context(), filter)
	if err != nil {
		logger.Errorf("GetGovernanceTags failed: filter=%+v err=%v", filter, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rows": rows, "total": total})
}
```

- [ ] **Step 4: Run handler test to verify it passes**

Run: `go test ./internal/handler/... -run TestGetGovernanceTags_WithFilters -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handler/tag_handler.go internal/handler/tag_handler_test.go
git commit -m "feat(tag-filter): extend GetGovernanceTags handler with filter param parsing and 400 validation"
```

---

## Chunk 3: Backend Service — Filter-Aware Query

### Task 3: Add ListGovernanceTagsFiltered to admin service

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Modify: `internal/handler/tag_handler.go:347-355` — migrate `GetTagStats` call site
- Test: `internal/service/tag_admin_service_test.go`

- [ ] **Step 1: Write the failing service test**

Add to `internal/service/tag_admin_service_test.go`:

```go
func TestListGovernanceTagsFiltered_Levels(t *testing.T) {
	// Seed tags with levels root, parent, child
	// Filter for levels=root only
	// Assert all returned rows have level="root"
	// Assert total matches count of root tags
}

func TestListGovernanceTagsFiltered_OrphanOnly(t *testing.T) {
	// Seed: one root (parent_id=NULL, level=root), one orphan child (parent_id=NULL, level=child)
	// Filter orphan_only=true
	// Assert only the orphan child is returned, not the root
}

func TestListGovernanceTagsFiltered_SourceBoth(t *testing.T) {
	// Seed tags with known ai_count and manual_count
	// Filter source_ai=true, source_manual=true
	// Assert only tags with both ai_count > 0 AND manual_count > 0 are returned
}

func TestListGovernanceTagsFiltered_UsageRange(t *testing.T) {
	// Seed tags with known usage_count
	// Filter min=10, max=50
	// Assert all returned rows have usage_count in [10,50]
}

func TestListGovernanceTagsFiltered_CombinedAND(t *testing.T) {
	// Seed diverse tags
	// Filter: levels=parent, min_usage=5, source_ai=true
	// Assert result matches the AND intersection
}

func TestListGovernanceTagsFiltered_PaginationAfterFilter(t *testing.T) {
	// Seed 30 matching tags
	// Filter with limit=10, offset=0 → total=30, len(rows)=10
	// Filter with limit=10, offset=10 → total=30, len(rows)=10
	// Assert no overlap between pages
}

func TestListGovernanceTagsFiltered_SearchPlusFilters(t *testing.T) {
	// Seed tags with known labels, levels, and source stats
	// Search for label that also matches via alias
	// Apply levels + source filter
	// Assert total and results reflect the AND of search + filters
}

func TestListGovernanceTagsFiltered_InvalidReturns0(t *testing.T) {
	// Filter: orphan_only=true AND levels=root
	// This is contradictory: orphan excludes root, levels=root only includes root
	// Assert: total=0, empty rows
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/... -run TestListGovernanceTagsFiltered -v`
Expected: FAIL — function does not exist

- [ ] **Step 3: Implement ListGovernanceTagsFiltered**

Add to `internal/service/tag_admin_service.go`:

The key change is replacing the two-phase "resolve tag slice → compute stats" with a unified flow:

1. Resolve full candidate set (search + levels + orphan filter)
2. Compute direct stats for ALL candidates
3. Apply in-memory filters (usage range, source)
4. Set `total` from filtered count
5. Apply `limit/offset` pagination
6. Compute tree stats only for the paginated page

```go
func (s *TagAdminService) ListGovernanceTagsFiltered(ctx context.Context, filter domain.GovernanceTagFilter) ([]*TagGovernanceRow, int, error) {
	// 1. Resolve full candidate set
	candidates, err := s.resolveFilteredCandidates(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if len(candidates) == 0 {
		return []*TagGovernanceRow{}, 0, nil
	}

	// 2. Compute direct stats for all candidates
	tagIDs := make([]int64, len(candidates))
	for i, t := range candidates {
		tagIDs[i] = t.ID
	}
	directStats, err := s.batchDirectStats(ctx, tagIDs)
	if err != nil {
		return nil, 0, err
	}

	// 3. Apply in-memory filters (usage range, source)
	filtered := s.applyMemoryFilters(candidates, directStats, filter)

	// 4. Total from filtered set
	total := len(filtered)

	// 5. Paginate
	offset := filter.Offset
	if offset >= total {
		return []*TagGovernanceRow{}, total, nil
	}
	end := total
	if offset+filter.Limit < end {
		end = offset + filter.Limit
	}
	page := filtered[offset:end]

	// 6. Build governance rows with stats
	// (compute tree stats for page only, same as current buildGovernanceRows)
	rows, err := s.buildGovernanceRows(ctx, page, directStats)
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
```

Implement helper methods:
- `resolveFilteredCandidates` — uses existing search logic (label + alias) + applies `WHERE level IN (...)` and `WHERE parent_id IS NULL AND level != 'root'` as in-memory or repository-level conditions; this must NOT paginate before filtering
- `batchDirectStats` — reuses existing `computeHierarchyStats` for direct counts
- `applyMemoryFilters` — applies usage range and source boolean filters in-memory on the full candidate set
- `buildGovernanceRows` — reuses existing row construction + tree stats for the paginated page only

Also update `GetTagStats` in `tag_handler.go` to call `ListGovernanceTagsFiltered` with an empty filter instead of `ListGovernanceTags`:

```go
// In GetTagStats:
rows, _, err := h.adminSvc.ListGovernanceTagsFiltered(ctx, domain.GovernanceTagFilter{
    Search: "",
    Limit:  total,
    Offset: 0,
})
```

- [ ] **Step 4: Run service test to verify it passes**

Run: `go test ./internal/service/... -run TestListGovernanceTagsFiltered -v`
Expected: PASS

- [ ] **Step 5: Run all existing service tests to verify no regression**

Run: `go test ./internal/service/... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/tag_admin_service.go internal/service/tag_admin_service_test.go internal/handler/tag_handler.go
git commit -m "feat(tag-filter): add ListGovernanceTagsFiltered with pre-pagination filtering"
```

---

## Chunk 4: Frontend Model — Filter State

### Task 4: Create TagGovernanceFilterState model

**Files:**
- Create: `flutter_app/lib/models/tag_governance_filter.dart`
- Test: `flutter_app/test/models/tag_governance_filter_test.dart`

- [ ] **Step 1: Write the failing test**

Create `flutter_app/test/models/tag_governance_filter_test.dart`:

```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:acg_warehouse/models/tag_governance_filter.dart';

void main() {
  test('empty filter has isEmpty=true', () {
    final filter = TagGovernanceFilterState();
    expect(filter.isEmpty, isTrue);
    expect(filter.isNotEmpty, isFalse);
  });

  test('toQueryParameters omits defaults', () {
    final filter = TagGovernanceFilterState();
    final params = filter.toQueryParameters();
    expect(params.containsKey('levels'), isFalse);
    expect(params.containsKey('orphan_only'), isFalse);
    expect(params.containsKey('min_usage_count'), isFalse);
    expect(params.containsKey('source_ai'), isFalse);
  });

  test('toQueryParameters includes active filters', () {
    final filter = TagGovernanceFilterState(
      levels: {'root', 'parent'},
      orphanOnly: true,
      minUsageCount: 10,
      sourceAI: true,
      search: '发色',
    );
    final params = filter.toQueryParameters();
    expect(params['levels'], 'root,parent');
    expect(params['orphan_only'], 'true');
    expect(params['min_usage_count'], '10');
    expect(params['source_ai'], 'true');
    expect(params['search'], '发色');
  });

  test('summaryChips returns correct descriptions', () {
    final filter = TagGovernanceFilterState(
      levels: {'root'},
      orphanOnly: true,
      sourceAI: true,
      sourceManual: true,
    );
    final chips = filter.summaryChips;
    expect(chips, containsAll(['层级: root', '无父级', '来源: AI+手动']));
  });

  test('copyWith preserves unmodified fields', () {
    final filter = TagGovernanceFilterState(levels: {'root'}, sourceAI: true);
    final modified = filter.copyWith(orphanOnly: true);
    expect(modified.levels, {'root'});
    expect(modified.sourceAI, isTrue);
    expect(modified.orphanOnly, isTrue);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd flutter_app && flutter test test/models/tag_governance_filter_test.dart`
Expected: FAIL — file does not exist

- [ ] **Step 3: Write the implementation**

Create `flutter_app/lib/models/tag_governance_filter.dart`:

```dart
class TagGovernanceFilterState {
  final Set<String> levels;
  final bool orphanOnly;
  final int? minUsageCount;
  final int? maxUsageCount;
  final bool sourceAI;
  final bool sourceManual;
  final String search;

  const TagGovernanceFilterState({
    this.levels = const {},
    this.orphanOnly = false,
    this.minUsageCount,
    this.maxUsageCount,
    this.sourceAI = false,
    this.sourceManual = false,
    this.search = '',
  });

  bool get isEmpty =>
      levels.isEmpty &&
      !orphanOnly &&
      minUsageCount == null &&
      maxUsageCount == null &&
      !sourceAI &&
      !sourceManual &&
      search.isEmpty;

  bool get isNotEmpty => !isEmpty;

  TagGovernanceFilterState copyWith({
    Set<String>? levels,
    bool? orphanOnly,
    int? minUsageCount,
    bool clearMinUsage = false,
    int? maxUsageCount,
    bool clearMaxUsage = false,
    bool? sourceAI,
    bool? sourceManual,
    String? search,
  }) {
    return TagGovernanceFilterState(
      levels: levels ?? this.levels,
      orphanOnly: orphanOnly ?? this.orphanOnly,
      minUsageCount: clearMinUsage ? null : (minUsageCount ?? this.minUsageCount),
      maxUsageCount: clearMaxUsage ? null : (maxUsageCount ?? this.maxUsageCount),
      sourceAI: sourceAI ?? this.sourceAI,
      sourceManual: sourceManual ?? this.sourceManual,
      search: search ?? this.search,
    );
  }

  Map<String, String> toQueryParameters() {
    final params = <String, String>{};
    if (levels.isNotEmpty) {
      params['levels'] = levels.toList()..sort();
      params['levels'] = (params['levels'] as List).join(',');
    }
    if (orphanOnly) params['orphan_only'] = 'true';
    if (minUsageCount != null) params['min_usage_count'] = minUsageCount.toString();
    if (maxUsageCount != null) params['max_usage_count'] = maxUsageCount.toString();
    if (sourceAI) params['source_ai'] = 'true';
    if (sourceManual) params['source_manual'] = 'true';
    if (search.isNotEmpty) params['search'] = search;
    return params;
  }

  List<String> get summaryChips {
    final chips = <String>[];
    if (levels.isNotEmpty) {
      chips.add('层级: ${levels.toList()..sort().join(',')}');
    }
    if (orphanOnly) chips.add('无父级');
    if (minUsageCount != null || maxUsageCount != null) {
      final lo = minUsageCount ?? 0;
      final hi = maxUsageCount != null ? '~$maxUsageCount' : '+';
      chips.add('使用量: $lo$hi');
    }
    if (sourceAI && sourceManual) {
      chips.add('来源: AI+手动');
    } else if (sourceAI) {
      chips.add('来源: AI');
    } else if (sourceManual) {
      chips.add('来源: 手动');
    }
    if (search.isNotEmpty) chips.add('关键词: $search');
    return chips;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd flutter_app && flutter test test/models/tag_governance_filter_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/models/tag_governance_filter.dart flutter_app/test/models/tag_governance_filter_test.dart
git commit -m "feat(tag-filter): add TagGovernanceFilterState model with query serialization"
```

---

## Chunk 5: Frontend Service — Filter Params in HTTP Call

### Task 5: Extend fetchGovernanceTags to accept filter state

**Files:**
- Modify: `flutter_app/lib/services/tag_service.dart:297-327`
- Test: `flutter_app/test/services/tag_service_test.dart`

- [ ] **Step 1: Write the failing test**

Add to `flutter_app/test/services/tag_service_test.dart`:

```dart
test('fetchGovernanceTags with filter state sends correct query params', () async {
  final filter = TagGovernanceFilterState(
    levels: {'root', 'parent'},
    orphanOnly: true,
    minUsageCount: 10,
    sourceAI: true,
  );
  // Setup mock client to capture request URI
  // Call fetchGovernanceTags(filter: filter)
  // Assert request URI contains levels=root,parent&orphan_only=true&min_usage_count=10&source_ai=true
});

test('fetchGovernanceTags with empty filter sends only limit/offset', () async {
  final filter = TagGovernanceFilterState();
  // Call fetchGovernanceTags(filter: filter)
  // Assert no filter params in URI
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd flutter_app && flutter test test/services/tag_service_test.dart`
Expected: FAIL — `filter` parameter not accepted

- [ ] **Step 3: Modify fetchGovernanceTags**

```dart
Future<GovernanceTagsPage> fetchGovernanceTags({
  String? search,
  int limit = 50,
  int offset = 0,
  TagGovernanceFilterState? filter,
}) async {
  final queryParams = <String, String>{
    'limit': limit.toString(),
    'offset': offset.toString(),
  };

  if (filter != null && filter.isNotEmpty) {
    queryParams.addAll(filter.toQueryParameters());
  } else if (search != null && search.isNotEmpty) {
    queryParams['search'] = search;
  }

  final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/governance')
      .replace(queryParameters: queryParams);

  final response = await _client.get(uri);
  if (response.statusCode != 200) {
    throw Exception('Failed to fetch governance tags: ${response.statusCode}');
  }

  final json = jsonDecode(response.body) as Map<String, dynamic>;
  final rows = (json['rows'] as List? ?? [])
      .map((entry) => TagGovernanceRow.fromJson(entry as Map<String, dynamic>))
      .toList();
  final total = json['total'] as int? ?? 0;
  return GovernanceTagsPage(rows: rows, total: total);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd flutter_app && flutter test test/services/tag_service_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/services/tag_service.dart flutter_app/test/services/tag_service_test.dart
git commit -m "feat(tag-filter): extend fetchGovernanceTags to accept filter state params"
```

---

## Chunk 6: Frontend Provider — Draft/Applied Filter State

### Task 6: Add dual-state filter management to TagProvider

**Files:**
- Modify: `flutter_app/lib/providers/tag_provider.dart`
- Test: `flutter_app/test/providers/tag_provider_test.dart`

- [ ] **Step 1: Write the failing test**

Add to `flutter_app/test/providers/tag_provider_test.dart`:

```dart
test('applyGovernanceFilter copies draft to applied and reloads', () async {
  // Setup mock service
  // Set draft filter with levels={root}
  // Call applyGovernanceFilter()
  // Assert appliedFilter matches draft
  // Assert loadGovernanceTags was called with filter
  // Assert selectedGovernanceIds is cleared
});

test('resetGovernanceFilter clears both states and reloads', () async {
  // Set both draft and applied
  // Call resetGovernanceFilter()
  // Assert both are empty
  // Assert loadGovernanceTags was called without filter
});

test('updateGovernanceDraft does not trigger service call', () async {
  // Call updateGovernanceDraft() multiple times
  // Assert no service calls made
  // Assert draft state updated
});

test('loadMoreGovernanceTags uses applied filter', () async {
  // Set applied filter
  // Call loadMoreGovernanceTags()
  // Assert service called with applied filter, not draft
});

test('governance action refresh preserves applied filter', () async {
  // Set applied filter
  // Simulate a governance action completion that triggers refresh
  // Assert refresh uses applied filter
});

test('search updates draft but does not call service', () async {
  // Call updateGovernanceDraft with search text
  // Assert draft.search is updated
  // Assert no fetchGovernanceTags calls
});

test('apply sends search plus other filters together', () async {
  // Set draft with search='test' and levels={root}
  // Call applyGovernanceFilter()
  // Assert service called with filter containing both search and levels
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd flutter_app && flutter test test/providers/tag_provider_test.dart`
Expected: FAIL — new methods do not exist

- [ ] **Step 3: Implement dual-state filter in TagProvider**

Add to `flutter_app/lib/providers/tag_provider.dart` after line 42:

```dart
  // Governance filter state
  TagGovernanceFilterState _governanceDraftFilter = const TagGovernanceFilterState();
  TagGovernanceFilterState _governanceAppliedFilter = const TagGovernanceFilterState();
```

Add getters:

```dart
  TagGovernanceFilterState get governanceDraftFilter => _governanceDraftFilter;
  TagGovernanceFilterState get governanceAppliedFilter => _governanceAppliedFilter;
```

Add methods:

```dart
  /// Update draft filter without triggering a reload.
  void updateGovernanceDraft(TagGovernanceFilterState draft) {
    _governanceDraftFilter = draft;
    notifyListeners();
  }

  /// Apply the draft filter: copy to applied, clear selection, reload first page.
  Future<void> applyGovernanceFilter() async {
    _governanceAppliedFilter = _governanceDraftFilter;
    _selectedGovernanceIds.clear();
    _activeMergeSource = null;
    _governanceOffset = 0;
    _governanceRows = [];
    _hasMoreGovernance = true;
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        filter: _governanceAppliedFilter.isNotEmpty ? _governanceAppliedFilter : null,
        limit: pageSize,
        offset: 0,
      );
      _governanceRows = page.rows;
      _governanceTotal = page.total;
      _governanceOffset = page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }

  /// Reset both draft and applied filters, clear selection, reload default list.
  Future<void> resetGovernanceFilter() async {
    _governanceDraftFilter = const TagGovernanceFilterState();
    _governanceAppliedFilter = const TagGovernanceFilterState();
    _selectedGovernanceIds.clear();
    _activeMergeSource = null;
    await loadGovernanceTags();
  }
```

Modify `loadGovernanceTags` to use applied filter:

```dart
  Future<void> loadGovernanceTags({String? search}) async {
    _governanceSearch = search;
    _governanceOffset = 0;
    _governanceRows = [];
    _hasMoreGovernance = true;
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        search: search,
        filter: _governanceAppliedFilter.isNotEmpty ? _governanceAppliedFilter : null,
        limit: pageSize,
        offset: 0,
      );
      _governanceRows = page.rows;
      _governanceTotal = page.total;
      _governanceOffset = page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }
```

Modify `loadMoreGovernanceTags` to use applied filter:

```dart
  Future<void> loadMoreGovernanceTags() async {
    if (_isLoadingMoreGovernance || !_hasMoreGovernance) return;
    _isLoadingMoreGovernance = true;
    notifyListeners();

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        filter: _governanceAppliedFilter.isNotEmpty ? _governanceAppliedFilter : null,
        limit: pageSize,
        offset: _governanceOffset,
      );
      _governanceRows = [..._governanceRows, ...page.rows];
      _governanceTotal = page.total;
      _governanceOffset += page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
    } finally {
      _isLoadingMoreGovernance = false;
      notifyListeners();
    }
  }
```

Modify `_refreshGovernanceRows` to preserve applied filter:

```dart
  Future<void> _refreshGovernanceRows({
    String? search,
    bool asPrimaryAction = false,
  }) async {
    _governanceSearch = search ?? _governanceSearch;
    _governanceOffset = 0;
    _hasMoreGovernance = true;

    if (asPrimaryAction) {
      _isRunningGovernanceAction = true;
      _governanceError = null;
      notifyListeners();
    }

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        search: _governanceSearch,
        filter: _governanceAppliedFilter.isNotEmpty ? _governanceAppliedFilter : null,
        limit: pageSize,
        offset: 0,
      );
      _governanceRows = page.rows;
      _governanceTotal = page.total;
      _governanceOffset = page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
    } finally {
      if (asPrimaryAction) {
        _isRunningGovernanceAction = false;
      }
      notifyListeners();
    }
  }
```

Also ensure `_runGovernanceAction` and similar methods clear selection on completion.

- [ ] **Step 4: Run provider test to verify it passes**

Run: `cd flutter_app && flutter test test/providers/tag_provider_test.dart`
Expected: PASS

- [ ] **Step 5: Run all existing provider tests to verify no regression**

Run: `cd flutter_app && flutter test test/providers/`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add flutter_app/lib/providers/tag_provider.dart flutter_app/test/providers/tag_provider_test.dart
git commit -m "feat(tag-filter): add draft/applied filter state to TagProvider"
```

---

## Chunk 7: Frontend UI — Filter Panel and Summary

### Task 7: Add filter panel widget to governance workspace

**Files:**
- Create: `flutter_app/lib/widgets/tag_management/tag_governance_filter_panel.dart`
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_list.dart`
- Test: `flutter_app/test/widgets/tag_management_workspace_test.dart`

- [ ] **Step 1: Write the failing widget test**

Add to `flutter_app/test/widgets/tag_management_workspace_test.dart`:

```dart
testWidgets('filter panel renders level chips and controls', (tester) async {
  // Pump TagManagementWorkspace with mocked provider
  // Assert level chip buttons for root, parent, child are visible
  // Assert orphan-only toggle is visible
  // Assert usage count inputs are visible
  // Assert AI/manual toggles are visible
  // Assert "应用筛选" and "重置" buttons are visible
});

testWidgets('clicking apply calls applyGovernanceFilter', (tester) async {
  // Pump workspace
  // Tap a level chip to select it
  // Tap "应用筛选"
  // Verify provider.applyGovernanceFilter was called
});

testWidgets('clicking reset calls resetGovernanceFilter', (tester) async {
  // Pump workspace
  // Tap "重置"
  // Verify provider.resetGovernanceFilter was called
});

testWidgets('summary chips show applied filter state', (tester) async {
  // Set appliedFilter in mock provider with known state
  // Pump workspace
  // Assert summary chip text is visible
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd flutter_app && flutter test test/widgets/tag_management_workspace_test.dart`
Expected: FAIL — filter panel does not exist

- [ ] **Step 3: Create the filter panel widget**

Create `flutter_app/lib/widgets/tag_management/tag_governance_filter_panel.dart`:

A stateful widget that:
- Receives `draftFilter`, `appliedFilter`, `onDraftChanged`, `onApply`, `onReset` callbacks
- Renders level selection chips (multi-select toggle buttons)
- Renders orphan-only toggle switch
- Renders min/max usage count text inputs
- Renders AI/manual toggle switches
- Renders "应用筛选" and "重置" buttons
- When `appliedFilter.isNotEmpty`, shows summary chips below the controls

The widget does **not** import Provider directly — it receives all state via constructor parameters, making it testable.

- [ ] **Step 4: Integrate filter panel into TagManagementWorkspace**

Modify `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`:

- Insert `TagGovernanceFilterPanel` between stats cards and governance list
- Wire `onDraftChanged` to `provider.updateGovernanceDraft()`
- Wire `onApply` to `provider.applyGovernanceFilter()`
- Wire `onReset` to `provider.resetGovernanceFilter()`
- Pass `provider.governanceDraftFilter` and `provider.governanceAppliedFilter`

- [ ] **Step 5: Remove result-set-altering local filtering from tag_management_list.dart**

Review `flutter_app/lib/widgets/tag_management/tag_management_list.dart`:
- If any local search/filter logic exists that would change the visible result set beyond what the backend returns, remove or neutralize it
- Keep display-only enhancements (e.g., highlighting) if they exist

- [ ] **Step 6: Run workspace widget test to verify it passes**

Run: `cd flutter_app && flutter test test/widgets/tag_management_workspace_test.dart`
Expected: PASS

- [ ] **Step 7: Run all existing tag management widget tests**

Run: `cd flutter_app && flutter test test/widgets/tag_management_workspace_test.dart test/widgets/tag_management_list_test.dart test/widgets/tag_edit_dialog_test.dart test/widgets/tag_bulk_action_bar_test.dart test/widgets/tag_merge_panel_test.dart`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add flutter_app/lib/widgets/tag_management/tag_governance_filter_panel.dart flutter_app/lib/widgets/tag_management/tag_management_workspace.dart flutter_app/lib/widgets/tag_management/tag_management_list.dart flutter_app/test/widgets/tag_management_workspace_test.dart
git commit -m "feat(tag-filter): add governance filter panel with draft/apply/reset UI"
```

---

## Chunk 7.5: Search Integration and List Test

### Task 7.5: Merge search into draft/applied and create list test

**Files:**
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart` — remove immediate search-on-change, bind search to draft filter
- Create: `flutter_app/test/widgets/tag_management_list_test.dart`
- Test: `flutter_app/test/widgets/tag_management_workspace_test.dart`

- [ ] **Step 1: Write the failing test for search-in-draft behavior**

Add to `flutter_app/test/widgets/tag_management_workspace_test.dart`:

```dart
testWidgets('search input updates draft without triggering request', (tester) async {
  // Pump workspace with mock provider
  // Find search input
  // Type text into search
  // Verify provider.updateGovernanceDraft was called with search text
  // Verify provider.applyGovernanceFilter was NOT called
  // Verify provider.loadGovernanceTags was NOT called
});

testWidgets('search plus filter applied together on apply click', (tester) async {
  // Type search text
  // Toggle a level chip
  // Click "应用筛选"
  // Verify applyGovernanceFilter was called once
  // Verify draft includes both search and level
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd flutter_app && flutter test test/widgets/tag_management_workspace_test.dart`
Expected: FAIL — search still triggers immediate load

- [ ] **Step 3: Modify workspace search to use draft filter**

In `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`:

1. Locate the search input `onChanged` callback that calls `provider.loadGovernanceTags(search: value)`
2. Replace it with `provider.updateGovernanceDraft(provider.governanceDraftFilter.copyWith(search: value))`
3. The search box now only updates the draft; the "应用筛选" button triggers the actual request

- [ ] **Step 4: Create list test to verify no local result-set filtering**

Create `flutter_app/test/widgets/tag_management_list_test.dart`:

```dart
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('governance list displays all rows from provider without local filtering', (tester) async {
    // Setup mock provider with 10 governance rows
    // Pump TagManagementList
    // Verify all 10 rows are rendered
    // No local search/filter should hide any row
  });

  testWidgets('governance list does not re-filter by label or level', (tester) async {
    // Setup mock provider with mixed level rows (root, parent, child)
    // Pump TagManagementList
    // Verify all rows appear regardless of level or label
  });
}
```

- [ ] **Step 5: Run all workspace and list tests**

Run: `cd flutter_app && flutter test test/widgets/tag_management_workspace_test.dart test/widgets/tag_management_list_test.dart`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add flutter_app/lib/widgets/tag_management/tag_management_workspace.dart flutter_app/test/widgets/tag_management_workspace_test.dart flutter_app/test/widgets/tag_management_list_test.dart
git commit -m "feat(tag-filter): integrate search into draft/apply cycle and add list test"
```

---

## Chunk 8: Integration Verification

### Task 8: End-to-end smoke test and regression check

**Files:** No new files — verification only

- [ ] **Step 1: Run all backend tests**

Run: `go test ./internal/... -v`
Expected: All PASS

- [ ] **Step 2: Run all frontend tests**

Run: `cd flutter_app && flutter test`
Expected: All PASS

- [ ] **Step 3: Manual smoke test checklist**

Start the backend and frontend:
- [ ] Open tag governance page
- [ ] Verify no filter is active on load, list loads normally
- [ ] Select "祖级" chip, click "应用筛选" → only root tags appear
- [ ] Select "父级" chip too → root + parent tags appear
- [ ] Toggle "无父级" → only orphan tags appear (no root)
- [ ] Set usage range 10–100 → only tags with usage in range appear
- [ ] Toggle "AI 生成" → only tags with AI associations appear
- [ ] Toggle "手动生成" too → only tags with both AI and manual appear
- [ ] Click "重置" → all filters cleared, full list reloads
- [ ] Scroll to bottom → load more works with active filter
- [ ] Apply filter, select some tags, apply different filter → selection clears

- [ ] **Step 4: Final commit (if any test fixes needed)**

```bash
git add -A
git commit -m "fix(tag-filter): address integration test findings"
```
