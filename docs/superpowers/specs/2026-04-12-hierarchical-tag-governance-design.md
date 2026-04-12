# Hierarchical Tag Governance Design

**Date:** 2026-04-12

## Goal

Add hierarchical tag governance to the existing flat tag system with at most three levels: `root`, `parent`, and `child`. Existing tags must migrate to `child` by default. AI and manual tagging must reuse existing tags when matched and only create new tags when no match exists.

## Confirmed Product Rules

1. Tag hierarchy is a strict single-parent tree with at most 3 levels.
2. Levels are:
   - `root` = 祖级
   - `parent` = 父级
   - `child` = 子级
3. Existing tags migrate to `child` with no parent.
4. AI-generated tags:
   - first try to match an existing tag by label, then alias
   - if matched, reuse that exact tag regardless of level
   - if not matched, create a new `child` tag
5. Manual tag creation must let the user explicitly choose `root`, `parent`, or `child`.
6. Images may be associated directly with any level (`root`, `parent`, or `child`).
7. Search/filter semantics are hierarchical:
   - selecting a tag matches images linked directly to that tag
   - and images linked to any descendant tags
8. Filter UI must be a full tree control, not a flat list.
9. Tag governance statistics must update correctly when hierarchy changes.

## Current-State Constraints From Code

- `tags.preferred_label` is globally unique.
- `tags.slug` is globally unique.
- aliases resolve directly to a single `tag_id`.
- current AI governance flow is label-first, alias-second, create-last.
- current SQLite triggers only maintain direct per-tag usage counts from `image_tags`.
- current image filtering logic assumes flat tag IDs.

These constraints should remain unless explicitly changed. In particular, global unique labels are important because the approved behavior is “match existing tag, do not create another one.”

## Data Model Changes

Extend `tags` with:

- `level TEXT NOT NULL` with values `root | parent | child`
- `parent_id INTEGER NULL`

V1 should **not** add `root_id`; correctness is more important than denormalized acceleration in the first release.

### Invariants

- `root` => `parent_id IS NULL`
- `parent` => `parent_id` references a `root`
- `child` => `parent_id IS NULL` or references a `parent`
- no cycles
- maximum depth is 3

### Migration

- backfill all existing tags to `level = 'child'`
- backfill all existing tags to `parent_id = NULL`

## Persistence Semantics

`image_tags` remains the single source of truth for image-to-tag association.

No extra association rows are created for ancestors. If an image is linked to a `child`, the `parent` and `root` match only through hierarchical query expansion, not duplicate rows.

## Tag Creation and Matching Rules

### AI flows

AI generation continues to route through `TagGovernanceService.MergeTags`.

For each generated label:

1. exact-match existing tag by preferred label
2. if not found, match alias
3. if found, reuse that tag regardless of level
4. if not found, create a new tag with:
   - `level = child`
   - `parent_id = NULL`
   - existing pending-review behavior preserved

### Manual tagging flows

When attaching an existing tag to an image, reuse the selected tag regardless of level.

When manually creating a new tag:

- user must choose `root`, `parent`, or `child`
- `root`: no parent allowed
- `parent`: must choose a `root`
- `child`: may be orphaned or may choose a `parent`

### Manual create duplicate handling

Manual create must follow the same dedupe-first rule as AI create:

1. exact-match existing tag by preferred label
2. if not found, match alias
3. if found, reuse that existing tag instead of creating a duplicate
4. if not found, create the requested level subject to hierarchy validation

The UI may still present this as a “create” flow, but the backend behavior is deterministic: matched tag => reuse, unmatched tag => create.

## Governance Operations

### Create

Support creating `root`, `parent`, and `child` from governance UI and image-detail tag creation flows.

### Change level

Allowed with validation:

- `child -> parent`
- `child -> root`
- `parent -> root`
- `parent -> child` only when it has no children
- `root -> parent/child` only when it has no descendants

### Reparent

- `parent` may only be attached to `root`
- `child` may only be attached to `parent`
- `root` cannot be reparented
- reject cycles and depth overflow
- `child` may also be detached into an orphan `child` by setting `parent_id = NULL`

### Delete

Delete preview and delete enforcement must consider:

1. direct image associations
2. existence of child tags

A tag with children cannot be deleted until the children are moved or removed.
A tag with direct image associations cannot be deleted until those direct image-tag links are removed or reassigned.

### Merge

V1 should allow merge only between tags of the same level.

Reason: cross-level merge makes image semantics ambiguous and complicates tree integrity.

## Search and Filter Semantics

When one or more tag IDs are selected for filtering, the backend must expand each selected tag into:

- the tag itself
- all descendants

Then query `image_tags` against the expanded set and deduplicate image IDs before pagination/response assembly.

### Multi-select semantics

Existing multi-tag filtering keeps current **AND semantics**.

For each selected tag:

1. expand that tag to `self + descendants`
2. treat the expanded set as one logical clause
3. an image satisfies that clause if it is associated with any tag in that expanded set

If the user selects both an ancestor and one of its descendants, the descendant selection is logically redundant, but still valid. The query result must remain correct and must not duplicate or exclude images incorrectly.

This same expansion rule should be used by:

- image list filtering
- search endpoints with tag filters
- AI backfill filters that depend on tag selection

## Statistics Model

Current trigger-maintained `tags.usage_count` should be treated as **direct usage count** only.

Backward compatibility rule:

- keep returning `usage_count` in existing payloads
- define `usage_count` as the direct count in V1
- add explicit `direct_*` and `tree_*` fields in governance/statistics/tree-oriented endpoints

V1 statistics should expose both direct and tree aggregates:

- `direct_usage_count`
- `tree_usage_count`
- `direct_pending_count`
- `tree_pending_count`
- `direct_confirmed_count`
- `tree_confirmed_count`
- `direct_ai_count`
- `tree_ai_count`
- `direct_manual_count`
- `tree_manual_count`

### Direct counts

Maintained by existing SQLite trigger strategy from `image_tags` changes.

### Tree counts

Computed at runtime in V1 by expanding descendants and deduplicating matched images. Do not add a cache table in the first release.

### Hierarchy-change behavior

When a tag is upgraded, downgraded, or reparented:

- direct counts do not change unless direct image associations changed
- tree counts must reflect the new structure immediately

## Backend Changes

### Domain / repository

- extend `internal/domain/tag.go`
- update `internal/repository/schema.go`
- add migration(s) for new columns and backfill
- extend `internal/repository/tag_repository.go`
- add repository helpers for:
  - list children by parent
  - list roots
  - list valid parent candidates by target level
  - resolve descendants for one or many tags

### Services

- update `internal/service/tag_governance_service.go`
- extend `internal/service/tag_admin_service.go`
- update search/image filtering services to expand descendants

### HTTP API

Existing tag payloads should return hierarchy fields:

- `level`
- `parent_id`

Add/extend endpoints for:

- create tag with explicit level
- update tag hierarchy metadata
- fetch tree data for filter/governance UI
- fetch parent candidates
- change level
- reparent

## Frontend Changes

### Shared model/service/provider

- extend `flutter_app/lib/models/tag.dart`
- extend `flutter_app/lib/services/tag_service.dart`
- extend `flutter_app/lib/providers/tag_provider.dart`

### Governance UI

The management experience should become tree-aware:

- tree display
- create root/parent/child actions
- change level action
- reparent action
- delete preview with child/dependency blockers
- same-level merge only

### Image detail / add-tag flows

- selecting existing tags remains allowed at any level
- manual create flow must require choosing level
- parent selection UI must appear when required by chosen level

### Filter UI

Replace flat tag filter UI with a full tree control:

- expand/collapse nodes
- show level badges
- show `tree_usage_count`
- allow multiselect
- preserve existing filter semantics, but with hierarchical expansion handled by backend

## Risks

1. Duplicate image matches if ancestor/descendant links are not deduplicated.
2. Broken tree invariants during level change or reparent.
3. Existing merge/delete behavior missing child-awareness.
4. Flat assumptions in search and AI backfill filters.
5. UI complexity in tree selection and hierarchy editing.

## V1 Scope Boundaries

Included:

- three-level hierarchy
- direct association to any level
- AI reuse existing tags across all levels
- manual create with explicit level selection
- full tree filter UI
- runtime tree stats
- same-level merge only

Excluded:

- cached aggregate stats table
- cross-level merge rules
- more than three levels
- multi-parent graph semantics
