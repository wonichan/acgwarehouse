# Hierarchical Tag Governance Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three-level hierarchical tag governance with tree-based filtering, explicit manual level creation, and hierarchy-aware stats while preserving existing flat-tag behavior where possible.

**Architecture:** Extend the existing `tags` entity with `level` and `parent_id`, keep `image_tags` as the only image-association table, and compute descendant-aware filtering/statistics in backend query/service logic. Preserve current direct-usage trigger behavior while adding runtime tree aggregates for governance and filtering.

**Tech Stack:** Go, Gin, SQLite, Flutter, Provider

---

## File Structure / Responsibility Map

### Backend storage and model
- Modify: `internal/domain/tag.go`
- Modify: `internal/repository/schema.go`
- Create: `migrations/005_add_tag_hierarchy.up.sql`
- Create: `migrations/005_add_tag_hierarchy.down.sql`
- Modify: `internal/repository/tag_repository.go`

### Backend business logic
- Modify: `internal/service/tag_governance_service.go`
- Modify: `internal/service/tag_admin_service.go`
- Modify: `internal/repository/image_repository.go`
- Modify: `internal/repository/search_repository.go`

### Backend HTTP layer
- Modify: `internal/handler/tag_handler.go`
- Modify: `internal/handler/image_tag_handler.go`
- Modify: `internal/handler/routes.go`

### Frontend shared data/state
- Modify: `flutter_app/lib/models/tag.dart`
- Modify: `flutter_app/lib/services/tag_service.dart`
- Modify: `flutter_app/lib/providers/tag_provider.dart`

### Frontend UI
- Modify: `flutter_app/lib/screens/tag_management_screen.dart`
- Modify: `flutter_app/lib/widgets/add_tag_dialog.dart`
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
- Add or modify: `flutter_app/lib/widgets/tag_management/*`

### Tests
- Modify/add: `internal/service/tag_governance_service_test.go`
- Modify/add: `internal/repository/image_repository_test.go`
- Modify/add: `internal/repository/search_repository_test.go`
- Modify/add: `internal/handler/*_test.go`
- Modify/add: relevant Flutter widget/provider/service tests

---

## Chunk 1: Schema and Tag Repository

### Task 1: Extend tag domain model

**Files:**
- Modify: `internal/domain/tag.go`

- [ ] **Step 1: Add hierarchy fields to `Tag`**

Add fields for:
- `Level string`
- `ParentID *int64`

- [ ] **Step 2: Add field-level comments/constants if the file already uses that pattern**

- [ ] **Step 3: Run diagnostics**

Run: LSP diagnostics on `internal/domain/tag.go`
Expected: no new errors

### Task 2: Add schema migration

**Files:**
- Create: `migrations/005_add_tag_hierarchy.up.sql`
- Create: `migrations/005_add_tag_hierarchy.down.sql`
- Modify: `internal/repository/schema.go`

- [ ] **Step 1: Write migration adding hierarchy columns**

Migration requirements:
- add `level TEXT NOT NULL DEFAULT 'child'`
- add `parent_id INTEGER NULL`
- backfill existing rows to `child`
- add useful index on `parent_id`

- [ ] **Step 2: Update schema bootstrap in `schema.go`**

Ensure fresh databases include hierarchy columns from first boot.

- [ ] **Step 3: Keep existing trigger behavior untouched for direct usage counts**

- [ ] **Step 4: Run focused repository/schema tests**

Run: relevant Go tests for schema/bootstrap
Expected: pass

### Task 3: Extend tag repository

**Files:**
- Modify: `internal/repository/tag_repository.go`

- [ ] **Step 1: Update scans/inserts/updates to include `level` and `parent_id`**

- [ ] **Step 2: Add hierarchy helper methods**

Add methods for:
- find roots
- find children by parent
- find valid parent candidates
- resolve descendants for one tag / many tags

- [ ] **Step 3: Add repository tests for hierarchy queries**

- [ ] **Step 4: Run targeted tests**

Run: `go test ./internal/repository/...`
Expected: pass

---

## Chunk 2: Hierarchy Rules and Governance Services

### Task 4: Make AI tag creation hierarchy-aware

**Files:**
- Modify: `internal/service/tag_governance_service.go`
- Test: `internal/service/tag_governance_service_test.go`

- [ ] **Step 1: Preserve current exact-match then alias-match flow**

- [ ] **Step 2: Ensure create-new path sets `level=child` and `parent_id=nil`**

- [ ] **Step 3: Add tests proving matched root/parent tags are reused, not recreated**

- [ ] **Step 4: Run focused governance tests**

Run: `go test ./internal/service/... -run TagGovernance`
Expected: pass

### Task 5: Add admin hierarchy operations

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Test: `internal/service/tag_admin_service_test.go` (create if missing)

- [ ] **Step 1: Add validation helpers**

Support checks for:
- legal level transitions
- valid parent assignment
- no cycle
- max depth 3
- child detach-to-orphan behavior (`child.parent_id = NULL`) as an allowed operation

- [ ] **Step 2: Add change-level operation**

- [ ] **Step 3: Add reparent operation**

- [ ] **Step 4: Extend delete preview to block when children exist**

- [ ] **Step 5: Extend delete preview to block when direct image associations exist**

- [ ] **Step 6: Restrict merge to same-level tags**

- [ ] **Step 7: Add service tests for each rule**

- [ ] **Step 8: Run targeted tests**

Run: `go test ./internal/service/... -run TagAdmin`
Expected: pass

---

## Chunk 3: Query, Filter, and Statistics Semantics

### Task 6: Expand image filtering to descendants

**Files:**
- Modify: `internal/repository/image_repository.go`
- Test: `internal/repository/image_repository_test.go`

- [ ] **Step 1: Add descendant expansion before image-tag filtering**

- [ ] **Step 2: Keep AND semantics by evaluating each selected tag as its own expanded clause (`self + descendants`)**

- [ ] **Step 3: Deduplicate images when matching ancestor + descendant tags before pagination**

- [ ] **Step 4: Add tests for root/parent/child filtering including overlapping ancestor+descendant selection**

- [ ] **Step 5: Add descendant-expansion coverage to AI backfill-related tag filters if they use tag selection**

- [ ] **Step 6: Run targeted tests**

Run: `go test ./internal/repository/... -run ImageRepository`
Expected: pass

### Task 7: Expand search filtering to descendants

**Files:**
- Modify: `internal/repository/search_repository.go`
- Test: `internal/repository/search_repository_test.go` or existing search tests

- [ ] **Step 1: Reuse descendant resolution in search tag filters**

- [ ] **Step 2: Deduplicate image matches**

- [ ] **Step 3: Add tests covering ancestor selection**

- [ ] **Step 4: Run targeted tests**

Run: `go test ./internal/repository/... -run Search`
Expected: pass

### Task 8: Add tree statistics

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Modify: `internal/handler/tag_handler.go`
- Test: service/handler tests as needed

- [ ] **Step 1: Keep existing direct count behavior unchanged**

- [ ] **Step 2: Add runtime-computed tree aggregate fields**

- [ ] **Step 3: Keep legacy `usage_count` as direct count and return explicit `direct_*` and `tree_*` fields from governance/stats endpoints**

- [ ] **Step 4: Add tests proving hierarchy changes alter tree counts without altering direct counts**

- [ ] **Step 5: Add tests for duplicate-image dedup in tree counts when an image is linked to both ancestor and descendant tags**

- [ ] **Step 6: Run targeted tests**

Run: `go test ./internal/service/... ./internal/handler/...`
Expected: pass

---

## Chunk 4: HTTP API

### Task 9: Extend tag create/update payloads

**Files:**
- Modify: `internal/handler/tag_handler.go`
- Modify: `internal/handler/routes.go`
- Test: handler tests

- [ ] **Step 1: Extend create-tag request binding to accept `level` and `parent_id`**

- [ ] **Step 2: Validate create rules by level and implement duplicate-match reuse for manual create (preferred label first, alias second)**

- [ ] **Step 3: Add dedicated endpoints for change-level and reparent**

- [ ] **Step 4: Add endpoint for tree data and parent candidates**

- [ ] **Step 5: Extend response JSON with hierarchy fields and tree stats where appropriate**

- [ ] **Step 6: Define response compatibility explicitly: legacy `usage_count` remains direct count in existing payloads**

- [ ] **Step 7: Add handler tests**

- [ ] **Step 8: Run targeted tests**

Run: `go test ./internal/handler/...`
Expected: pass

### Task 10: Update manual image-tag creation flow

**Files:**
- Modify: `internal/handler/image_tag_handler.go`
- Test: relevant handler tests

- [ ] **Step 1: Extend create-new-tag request path to accept chosen level/parent when creating manually**

- [ ] **Step 2: Preserve “reuse existing if matched” behavior for manual creation attempts (exact label, then alias)**

- [ ] **Step 3: Add tests for manual create root/parent/child flows**

- [ ] **Step 4: Add tests for manual create duplicate-match reuse against existing root/parent/child tags**

- [ ] **Step 5: Run targeted tests**

Run: `go test ./internal/handler/... -run ImageTag`
Expected: pass

---

## Chunk 5: Flutter Shared Model and Service Layer

### Task 11: Extend Flutter tag model and service contracts

**Files:**
- Modify: `flutter_app/lib/models/tag.dart`
- Modify: `flutter_app/lib/services/tag_service.dart`
- Modify: `flutter_app/lib/providers/tag_provider.dart`

- [ ] **Step 1: Add hierarchy fields to Flutter `Tag` model**

- [ ] **Step 2: Add request/response handling for tree endpoints, parent candidates, and hierarchy updates**

- [ ] **Step 3: Add provider state for tree data and candidate lists**

- [ ] **Step 4: Add/update tests for JSON parsing and provider behavior**

- [ ] **Step 5: Run Flutter analyzer/tests for changed files**

Expected: pass

---

## Chunk 6: Governance UI and Tree Filter UI

### Task 12: Build hierarchy-aware governance UI

**Files:**
- Modify: `flutter_app/lib/screens/tag_management_screen.dart`
- Modify/add: `flutter_app/lib/widgets/tag_management/*`

- [ ] **Step 1: Replace flat governance list with hierarchy-aware presentation**

- [ ] **Step 2: Add create-tag dialog with explicit level selection**

- [ ] **Step 3: Add change-level and reparent actions**

- [ ] **Step 4: Show direct/tree stats in governance rows**

- [ ] **Step 5: Restrict merge UI to same-level targets**

- [ ] **Step 6: Add/update widget tests**

### Task 13: Replace flat filter pane with full tree control

**Files:**
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
- Modify: related providers if needed

- [ ] **Step 1: Render full three-level tree with expand/collapse**

- [ ] **Step 2: Support multiselect on any level**

- [ ] **Step 3: Display level badges and tree usage counts**

- [ ] **Step 4: Keep untagged/pending toggles working alongside tree selection**

- [ ] **Step 5: Add/update widget tests**

- [ ] **Step 6: Run Flutter analyzer/tests**

Expected: pass

### Task 14: Update image detail tag dialogs

**Files:**
- Modify: `flutter_app/lib/widgets/add_tag_dialog.dart`
- Modify: `flutter_app/lib/widgets/tag_picker_results_panel.dart`
- Modify: `flutter_app/lib/screens/image_detail_screen.dart`
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart`

- [ ] **Step 1: Add explicit level selection when creating a new tag manually**

- [ ] **Step 2: Add parent selection UI when required**

- [ ] **Step 3: Show hierarchy metadata for existing tags**

- [ ] **Step 4: Add/update widget tests**

- [ ] **Step 5: Run Flutter analyzer/tests**

Expected: pass

---

## Chunk 7: End-to-End Verification

### Task 15: Verify backend behavior end-to-end

**Files:**
- No primary code changes; use tests and executable verification

- [ ] **Step 1: Run full backend test suite relevant to changed packages**

Run: `go test ./internal/...`
Expected: pass or only documented pre-existing failures

- [ ] **Step 2: Execute API verification flows**

Use executable checks (curl/http tests) for at least:
- create root
- create parent under root
- create child under parent
- manual create duplicate label reuses existing tag
- AI reuse existing parent/root tag
- filter by root and confirm descendant images appear
- overlapping root+child selection preserves correct AND semantics
- change level and confirm tree stats change immediately
- delete preview blocks tags with direct image links and/or children

- [ ] **Step 3: Record API QA evidence**

### Task 16: Verify Flutter behavior end-to-end

- [ ] **Step 1: Run Flutter analyzer/test suite for affected app areas**

- [ ] **Step 2: Execute browser UI verification flows**

Use executable browser QA where possible for at least:
- governance tree renders correctly
- manual create dialog supports all three levels
- duplicate manual create path reuses existing matched tag
- tree filter UI expands/collapses and filters correctly
- overlapping ancestor/descendant selections behave consistently
- image detail can attach existing parent/root/child tags

- [ ] **Step 3: Record browser QA evidence**

---

## Execution Notes

- Keep direct-usage trigger behavior intact.
- Treat tree stats as runtime-computed in V1.
- Do not introduce cross-level merge in this plan.
- Preserve global unique label semantics.

Plan complete and saved to `docs/superpowers/plans/2026-04-12-hierarchical-tag-governance-plan.md`. Ready to execute?
