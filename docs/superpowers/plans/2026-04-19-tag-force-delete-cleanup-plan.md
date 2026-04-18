# Tag Force Delete Cleanup Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow tags to be deleted from the governance UI even when they are in use or have direct children, while automatically removing direct image-tag associations and detaching direct children to top-level.

**Architecture:** Keep the existing delete-preview flow, but change it from a blocking gate into an informational preview. Implement the actual delete path as a single backend transaction that detaches direct children, removes direct `image_tags` associations, deletes aliases, deletes the tag, and returns a cleanup summary that the Flutter UI can present and refresh from.

**Tech Stack:** Go, Gin, SQLite, Flutter, Provider

---

## File Structure / Responsibility Map

### Backend service and cleanup transaction
- Modify: `internal/service/tag_admin_service.go`
- Modify: `internal/repository/image_tag_repository.go` (if a bulk delete helper is needed)
- Modify: `internal/repository/tag_repository.go` (if a child-detach helper is needed)

### Backend HTTP contract
- Modify: `internal/handler/tag_handler.go`

### Frontend model / service / state
- Modify: `flutter_app/lib/models/tag_governance.dart`
- Modify: `flutter_app/lib/services/tag_service.dart`
- Modify: `flutter_app/lib/providers/tag_provider.dart`

### Frontend UI
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`
- Review if needed: `flutter_app/lib/widgets/tag_management/tag_management_list.dart`

### Tests
- Modify: `internal/service/tag_admin_service_test.go`
- Modify: `internal/handler/tag_handler_test.go`
- Modify: `internal/repository/image_tag_repository_test.go` (only if repository helper behavior changes)
- Modify: `flutter_app/test/services/tag_service_test.dart`
- Modify: `flutter_app/test/providers/tag_provider_test.dart`
- Modify: `flutter_app/test/widgets/tag_management_workspace_test.dart`

---

## Chunk 1: Backend Delete Semantics and Transaction

### Task 1: Turn delete preview into a non-blocking cleanup preview

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Test: `internal/service/tag_admin_service_test.go`

- [ ] **Step 1: Write the failing service test for previewing deletions of used tags and tags with children**

Cover these expectations:
- preview returns `affected_image_count` for direct image associations
- preview returns direct child count (and child details only if that is already idiomatic in this file)
- preview no longer blocks deletion because the tag is used or has children

- [ ] **Step 2: Run the focused service test to verify it fails for the expected reason**

Run: `go test ./internal/service/... -run DeletePreview`
Expected: FAIL because preview still marks used/parent tags as non-deletable or lacks the new fields

- [ ] **Step 3: Implement the minimal preview changes in `tag_admin_service.go`**

Requirements:
- keep tag existence validation
- keep direct association counting
- keep direct child lookup
- return preview as informational data rather than a blocking gate
- preserve compatibility only where it does not keep the old block behavior alive

- [ ] **Step 4: Re-run the focused service test**

Run: `go test ./internal/service/... -run DeletePreview`
Expected: PASS

### Task 2: Add transactional cleanup deletion in the admin service

**Files:**
- Modify: `internal/service/tag_admin_service.go`
- Modify if needed: `internal/repository/image_tag_repository.go`
- Modify if needed: `internal/repository/tag_repository.go`
- Test: `internal/service/tag_admin_service_test.go`
- Test if needed: `internal/repository/image_tag_repository_test.go`

- [ ] **Step 1: Write the failing service tests for actual delete execution**

Add tests for:
- deleting a tag that still has direct image associations removes those associations
- deleting a tag that still has direct children detaches those children by setting `parent_id = NULL`
- deleting a tag removes aliases and the tag row itself
- cleanup result returns `affected_image_count` and `detached_child_count`

- [ ] **Step 2: Run the focused delete-execution tests to verify they fail correctly**

Run: `go test ./internal/service/... -run DeleteTag`
Expected: FAIL because current delete logic blocks or does not clean up associations/children

- [ ] **Step 3: Add the minimal transaction-oriented delete service implementation**

Implementation requirements:
- validate tag existence first
- load the current preview / counts needed for response data
- detach direct children inside the transaction
- remove direct `image_tags` rows for the deleted tag
- perform required FTS sync for affected images/search state
- delete aliases for the tag
- delete the tag row
- rollback on any error

- [ ] **Step 4: Add or extend repository helpers only if the service cannot express cleanup safely with existing methods**

Prefer:
- reusing existing repository delete/update APIs
- adding narrowly scoped helpers such as “delete associations by tag id” only if current methods would require error-prone row-by-row orchestration

- [ ] **Step 5: Re-run the focused backend tests**

Run: `go test ./internal/service/... -run DeleteTag`
Expected: PASS

- [ ] **Step 6: Run repository tests if helper behavior changed**

Run: `go test ./internal/repository/... -run ImageTag`
Expected: PASS

---

## Chunk 2: HTTP Contract and Flutter Data Flow

### Task 3: Update the delete endpoint contract

**Files:**
- Modify: `internal/handler/tag_handler.go`
- Test: `internal/handler/tag_handler_test.go`

- [ ] **Step 1: Write the failing handler tests for deleting used/parent tags**

Cover these expectations:
- `DELETE /tags/{id}` returns `200` for a tag with direct image usage
- `DELETE /tags/{id}` returns `200` for a tag with direct children
- success response includes `deleted_tag_id`, `affected_image_count`, and `detached_child_count`
- old `409 conflict` behavior is gone for these cases

- [ ] **Step 2: Run the focused handler tests to verify they fail for the expected reason**

Run: `go test ./internal/handler/... -run DeleteTag`
Expected: FAIL because handler still returns conflict or old payload

- [ ] **Step 3: Implement the minimal handler changes**

Requirements:
- call the new cleanup-delete service path
- keep `404` behavior for missing tags
- return the cleanup summary payload on success

- [ ] **Step 4: Re-run the focused handler tests**

Run: `go test ./internal/handler/... -run DeleteTag`
Expected: PASS

### Task 4: Update Flutter models, service parsing, and provider flow

**Files:**
- Modify: `flutter_app/lib/models/tag_governance.dart`
- Modify: `flutter_app/lib/services/tag_service.dart`
- Modify: `flutter_app/lib/providers/tag_provider.dart`
- Test: `flutter_app/test/services/tag_service_test.dart`
- Test: `flutter_app/test/providers/tag_provider_test.dart`

- [ ] **Step 1: Write the failing Flutter service/model tests first**

Cover these expectations:
- delete preview parses the new non-blocking fields (for example child count and any new summary fields)
- delete API parsing tolerates and optionally exposes cleanup summary fields

- [ ] **Step 2: Run the focused Flutter service tests to verify they fail correctly**

Run: `flutter test test/services/tag_service_test.dart --plain-name "delete"`
Expected: FAIL because models/parsers do not match the new response shape

- [ ] **Step 3: Implement the minimal model/service changes**

Requirements:
- align `TagDeletePreview` with the approved contract
- keep service methods narrow; do not add unrelated API surface
- preserve existing call sites where possible

- [ ] **Step 4: Write or extend the failing provider tests**

Cover these expectations:
- loading delete preview stores the new informational fields
- deleting a tag still triggers governance list refresh and tag tree refresh

- [ ] **Step 5: Run the focused provider tests to verify they fail correctly**

Run: `flutter test test/providers/tag_provider_test.dart --plain-name "delete"`
Expected: FAIL because provider assumptions still reflect blocking delete semantics

- [ ] **Step 6: Implement the minimal provider updates**

Requirements:
- remove assumptions that delete is only available when `canDelete` is true
- keep refresh behavior after successful delete
- keep error propagation explicit

- [ ] **Step 7: Re-run focused Flutter service/provider tests**

Run: `flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart`
Expected: PASS

---

## Chunk 3: Tag Management UI and End-to-End Verification

### Task 5: Convert the delete dialog from blocking UI to impact-summary UI

**Files:**
- Modify: `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`
- Review if needed: `flutter_app/lib/widgets/tag_management/tag_management_list.dart`
- Test: `flutter_app/test/widgets/tag_management_workspace_test.dart`

- [ ] **Step 1: Write the failing widget tests for the new delete confirmation behavior**

Cover these expectations:
- the delete action remains visible from the governance list
- the dialog shows affected image count
- the dialog shows detached child count
- the final “删除” button is shown even when there are affected images or children
- the dialog no longer renders blocking-reason-only behavior

- [ ] **Step 2: Run the focused widget tests to verify they fail for the expected reason**

Run: `flutter test test/widgets/tag_management_workspace_test.dart --plain-name "删除"`
Expected: FAIL because the dialog still hides the final delete button behind `canDelete`

- [ ] **Step 3: Implement the minimal widget changes**

Requirements:
- keep the existing row-level delete entry point
- change the dialog copy to informational wording
- always render the final delete action once preview is loaded successfully
- include a clear irreversible-operation warning

- [ ] **Step 4: Re-run the focused widget tests**

Run: `flutter test test/widgets/tag_management_workspace_test.dart --plain-name "删除"`
Expected: PASS

### Task 6: Run final verification for the whole change set

**Files:**
- Verify modified backend and Flutter files from this plan

- [ ] **Step 1: Run Go diagnostics/tests for touched backend packages**

Run: `go test ./internal/service/... ./internal/handler/... ./internal/repository/...`
Expected: PASS

- [ ] **Step 2: Run Flutter tests for touched layers**

Run: `flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/widgets/tag_management_workspace_test.dart`
Expected: PASS

- [ ] **Step 3: Run LSP diagnostics on all changed files**

Expected: no new errors

- [ ] **Step 4: Perform a manual QA pass on the governance delete flow**

Manual checklist:
- open the tag management page
- delete a tag that has image usage
- confirm the dialog reports affected image count
- delete a tag that has direct children
- confirm those direct children appear as top-level nodes after refresh
- verify the deleted tag no longer appears in governance results or tag tree

- [ ] **Step 5: If manual QA reveals contract drift, fix the smallest responsible layer and re-run the focused tests before re-running this full verification step**

---

## Notes for the Implementer

- Do not reintroduce any “used tags cannot be deleted” or “tags with children cannot be deleted” guard in UI or handler code.
- Do not silently migrate image associations to another tag.
- Do not change child tag `level`; only clear `parent_id`.
- Keep the cleanup result payload small and explicit.
- Prefer transaction safety over clever incremental updates.

## Execution Order Recommendation

1. Backend preview semantics
2. Backend transactional delete
3. Handler contract
4. Flutter model/service/provider updates
5. Widget dialog update
6. Full verification

Plan complete and saved to `docs/superpowers/plans/2026-04-19-tag-force-delete-cleanup-plan.md`. Ready to execute?
