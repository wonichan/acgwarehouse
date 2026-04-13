# Hierarchical Tag Governance QA Evidence

Date: 2026-04-13

## Backend verification

Executed:

```powershell
go test ./internal/...
```

Result: PASS

Focused executable API-style coverage run:

```powershell
go test ./internal/handler ./internal/repository ./internal/service -run "TestTagCreateTagCreatesRequestedHierarchyLevel|TestImageTagAddImageTagCreatesRequestedHierarchyTagFromLabel|TestTagGovernanceMergeTagsReusesExistingTagsAcrossLevels|TestImageRepositoryFindByTagIDsExpandsSelectedAncestors|TestImageRepositoryFindByTagIDsKeepsExpandedAndSemantics|TestTagAdminServiceChangeLevelParentToRootDetachesChildren|TestTagAdminServiceGetDeletePreviewBlocksRejectedOnlyAssociation|TestTagDeleteTagBlocksAndKeepsAssociationsWhenUsed" -v
```

Covered flows:

- create root/parent/child capable tags through HTTP and handlers
- manual create with explicit level/parent
- duplicate/alias reuse behavior
- AI reuse across hierarchy levels
- descendant-expanding root filtering
- overlapping ancestor/descendant AND semantics
- parent -> root level change with child detachment
- delete preview blocked by rejected-only direct links
- delete preview blocked by direct image links

Result: PASS

Additional enforcement verification:

```powershell
go test ./internal/handler/... -run "ImageTag|Tag"
```

Covered flows:

- missing hierarchy rejected for unmatched manual create
- cross-level merge rejected
- hierarchy routes `/tags/tree`, `/tags/parent-candidates`, `/tags/:id/change-level`, `/tags/:id/reparent`
- `PUT /tags/:id` rejects hierarchy mutation fields

Result: PASS

## Flutter/UI verification

Targeted analyzer verification:

```powershell
flutter analyze lib/widgets/add_tag_dialog.dart lib/widgets/batch_tag_dialog.dart lib/widgets/edit_tag_dialog.dart lib/widgets/tag_management/tag_edit_dialog.dart lib/widgets/image_metadata_panel.dart lib/services/tag_service.dart test/widgets/batch_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart test/widgets/tag_edit_dialog_test.dart
```

Result: PASS

## Browser-level verification attempt

Executed:

```powershell
flutter run -d web-server --web-port 7357
```

Then navigated with Playwright MCP to `http://127.0.0.1:7357/`.

Observed:

- Page title resolved to `gallery`
- Flutter web app booted without browser console errors
- Network requests completed for Flutter CanvasKit assets

Limitation:

- This build uses CanvasKit, and in this harness Playwright accessibility snapshots were empty, so interactive browser-step assertions on tree widgets were not machine-observable through MCP.
- For user-flow verification in this environment, widget tests were used as the executable substitute for the UI interactions listed above.

Executable UI/widget verification:

```powershell
flutter test test/widgets/batch_tag_dialog_test.dart test/widgets/edit_tag_dialog_test.dart test/widgets/tag_edit_dialog_test.dart test/widgets/image_metadata_panel_test.dart test/widgets/add_tag_dialog_test.dart test/services/tag_service_test.dart
flutter test test/widgets/tag_management_workspace_test.dart test/widgets/fluent_tag_filter_pane_test.dart
```

Covered flows:

- image detail add-tag dialog supports explicit level selection and required parent selection
- batch add dialog supports explicit level selection and required parent selection
- image detail edit dialog only offers same-level replacement targets
- image detail merge dialog excludes cross-level targets
- governance create dialog requires explicit level and supports reparent/change-level
- governance tree search keeps matched descendants with ancestor path
- tree filter uses expand/collapse tree semantics and level badges
- stale flat filter path removed from source

Result: PASS

## Known unrelated baseline issues

Previously observed full `flutter test` app-shell failures were outside the hierarchical tag governance scope and were not required to validate the changed hierarchy paths. The focused hierarchy-related widget and service suites above passed after the final fixes.
