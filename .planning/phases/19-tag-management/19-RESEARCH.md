# Phase 19: Tag Management - Research

**Date:** 2026-04-05
**Phase:** 19-tag-management
**Mode:** planning research

## Research Question

What needs to be true for Phase 19 planning to produce executable desktop tag-governance plans that honor the locked decisions, fit the existing Go + Flutter architecture, and keep verification targeted and fast?

## Existing Baseline

- The desktop shell already has a dedicated `标签管理` navigation destination in `flutter_app/lib/app/fluent_app_shell.dart` and `flutter_app/lib/app/fluent_screens.dart`.
- The current desktop tag-management implementation in `flutter_app/lib/screens/tag_management_screen.dart` is Material-oriented, list-based, and supports only lightweight single-row edit/delete plus global unused-tag cleanup.
- `TagProvider` and `TagService` already hold the state and HTTP seams for statistics, update, delete, and cleanup, so extending them is lower risk than adding a parallel governance stack.
- The backend already exposes list, update, alias CRUD, delete, statistics, and cleanup endpoints in `internal/handler/tag_handler.go`, but the current delete path still removes image associations directly, which conflicts with D-04/D-06.
- Existing governance logic in `internal/service/tag_governance_service.go` demonstrates exact-match-first alias resolution and safe merge instincts that should inform admin merge behavior.

## Locked-Decision Implications

### List-first workspace

- D-01/D-02/D-03 require the main desktop workspace to remain list-centric.
- This means the plan should build a Fluent-native workspace with row actions, selection state, and a merge side panel rather than a dedicated modal-first wizard or separate governance dashboard.

### Safe delete semantics

- D-04/D-05/D-06 mean the current `DELETE /api/v1/tags/:id` behavior is incompatible because it deletes image associations before deleting the tag.
- The backend contract must change so delete becomes safe-by-default: preview impact count, block when usage exists, and return structured reasons instead of silently stripping associations.

### Alias and category are first-class governance inputs

- D-07/D-08 mean the governance list cannot stop at counts and preferred labels.
- Frontend and backend contracts need explicit support for alias visibility, alias CRUD in the management workflow, and `primaryCategory` editing in-place or via lightweight dialogs.

### Full batch governance without a new backend silo

- D-10/D-11 do not require a brand-new governance backend subsystem if the existing contracts can support deterministic multi-select orchestration.
- A good fit is: add missing safety-oriented backend primitives (governance list, delete preview, merge endpoint, selected cleanup endpoint), then let `TagProvider` aggregate per-tag success/failure for bulk category/alias/merge actions.

### Cross-page validation loop

- D-12/D-13 fit the existing shell/navigation model well.
- The cheapest valid path is to reuse `NavigationProvider.galleryIndex` plus `ImageListProvider.setTagFilter([tagId])`, optionally clearing other filters first, rather than embedding preview content into the governance page.

## Recommended Architecture

### Backend

1. Add a focused admin-facing service layer for tag governance (`TagAdminService`) rather than placing all orchestration in handlers.
2. Add these contracts:
   - `GET /api/v1/tags/governance` — enriched governance rows with stats, aliases, category, `can_delete`, and `affected_image_count`
   - `GET /api/v1/tags/:id/delete-preview` — explicit safe-delete preview with `affected_image_count`, `can_delete`, and `blocking_reason`
   - `POST /api/v1/tags/:id/merge` — move associations and aliases to a chosen target tag using exact, user-selected targets only
   - `POST /api/v1/tags/batch/cleanup` — delete only selected unused tags and return `deleted`, `blocked`, and `failed` arrays
3. Keep existing `PUT /api/v1/tags/:id`, `GET/POST/DELETE /api/v1/tags/:id/aliases` as the rename/category/alias primitives the frontend composes around.

### Frontend state/data layer

1. Introduce a dedicated governance model file for enriched rows, delete previews, merge requests, and aggregated batch results.
2. Extend `TagService` with typed methods for the new governance endpoints plus alias CRUD wrappers.
3. Extend `TagProvider` to own governance list state, multi-select state, active merge source state, bulk action execution, and failure reporting.

### Desktop UI

1. Replace the current wrapped Material tag-management body with a Fluent-native governance workspace under `FluentTagManagementPage`.
2. Keep the main view list-first with search/filter/sort, summary cards, row actions, and explicit “view affected images” handoff.
3. Use a merge side panel for source-target selection and confirmation.
4. Use a selection toolbar for batch cleanup/category/alias/merge-candidate actions with aggregated result feedback.

## Risks and Mitigations

| Risk | Why it matters | Mitigation |
|------|----------------|------------|
| Safe delete breaks existing assumptions | Current delete endpoint removes image associations directly | Add TDD coverage that forbids deleting used tags and requires `affected_image_count` in responses |
| Merge can duplicate aliases or image-tag rows | Governance merge touches multiple tables | Centralize merge in a service transaction and add handler/service tests for duplicate-safe migration |
| Batch scope grows too large | D-10/D-11 expand beyond the roadmap’s minimal success criteria | Keep backend primitives narrow and let provider aggregate bulk results instead of inventing a giant backend workflow engine |
| UI drifts into a heavy admin console | Context explicitly forbids it | Enforce list-first workspace + merge panel in plan objectives and widget acceptance criteria |
| Flutter full suite is already partially red | STATE.md records unrelated baseline failures | Use targeted widget/service/provider suites plus shell regressions for phase validation |

## Recommended Plan Shape

Use **3 execute plans in 3 waves**:

1. **Wave 1 / Plan 01** — backend governance contracts and safe-delete semantics
2. **Wave 2 / Plan 02** — Flutter governance models/service/provider orchestration
3. **Wave 3 / Plan 03** — Fluent desktop workspace, merge panel, bulk toolbar, and gallery drilldown

This keeps each plan under context budget, matches established phase granularity, and gives the executor an ultrawork-friendly dependency ladder.

## Atomic Commit Guidance

- Prefer RED → GREEN → REFACTOR commits within each task.
- Keep backend contract commits separate from Flutter state-model commits.
- Keep widget layout/interaction commits separate from provider/service contract commits.
- Do not batch backend, provider, and UI edits into one catch-all commit.

## Validation Architecture

### Quick feedback loops

- Backend quick run:
  - `go test ./internal/service ./internal/handler -run 'TestTag|TestTagGovernance' -count=1`
- Flutter quick run:
  - `cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/app/fluent_app_shell_test.dart`
- UI-focused quick run:
  - `cd flutter_app; flutter test test/widgets/tag_management_workspace_test.dart test/widgets/tag_merge_panel_test.dart test/widgets/tag_bulk_action_bar_test.dart`

### Full phase verification

- Backend broader run:
  - `go test ./internal/... -count=1`
- Flutter broader run for phase-owned surfaces:
  - `cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/app/fluent_app_shell_test.dart test/widgets/tag_management_workspace_test.dart test/widgets/tag_merge_panel_test.dart test/widgets/tag_bulk_action_bar_test.dart`

### Known limits

- Do **not** use plain `cd flutter_app; flutter test` as a hard phase gate because `STATE.md` already records unrelated baseline failures in pre-existing suites.
- The plans should still include targeted automated verification for every task, plus one manual smoke-check for desktop navigation handoff if needed.

## Conclusion

Phase 19 should be planned as a focused desktop-governance phase that corrects unsafe backend delete behavior, introduces explicit merge/delete-preview contracts, extends the existing Flutter provider/service stack for governance orchestration, and finishes with a Fluent-native list-first workspace that can hand admins straight back to filtered gallery results.
