---
phase: 19-tag-management
plan: 02
subsystem: ui
tags: [flutter, provider, tag-governance, desktop]

requires:
  - phase: 19-01
    provides: backend governance contracts for list/merge/delete-preview/batch cleanup
provides:
  - typed Flutter governance models for rows, delete preview, merge request, and batch failures
  - TagService governance client methods for list/delete-preview/merge/alias CRUD/batch cleanup
  - TagProvider governance orchestration with multi-select, merge source state, and aggregated failure reporting
affects: [19-03, desktop-tag-management, gallery-drilldown]

tech-stack:
  added: []
  patterns: [provider-level batch orchestration, typed governance contracts, partial-failure aggregation]

key-files:
  created: [flutter_app/lib/models/tag_governance.dart]
  modified: [flutter_app/lib/services/tag_service.dart, flutter_app/lib/providers/tag_provider.dart, flutter_app/test/services/tag_service_test.dart, flutter_app/test/providers/tag_provider_test.dart]

key-decisions:
  - "Governance HTTP contracts are parsed into typed models so widgets avoid raw-map access"
  - "Batch governance failures aggregate per-tag tagId/preferredLabel/message in provider-owned state"
  - "Provider exposes governance selection/source state but remains decoupled from navigation and image providers"

patterns-established:
  - "RED→GREEN task slices with targeted flutter test command per layer"
  - "Post-mutation governance refresh happens in provider orchestration, not service transport"

requirements-completed: [DSK-04]

duration: 67 min
completed: 2026-04-05
---

# Phase 19 Plan 02: Tag Governance Flutter Contracts Summary

**Flutter desktop governance now has typed row/preview/batch contracts plus provider-owned multi-select orchestration that reports partial failures per tag.**

## Performance

- **Duration:** 67 min
- **Started:** 2026-04-05T05:52:00Z
- **Completed:** 2026-04-05T06:59:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added `TagGovernanceRow`, `TagDeletePreview`, `TagGovernanceFailure`, `TagGovernanceBatchResult`, and `TagMergeRequest` in `flutter_app/lib/models/tag_governance.dart`.
- Extended `TagService` with Phase 19 governance APIs: list fetch, delete preview, explicit merge request, alias CRUD wrappers, and selected batch cleanup parsing.
- Extended `TagProvider` with dedicated governance state (`governanceRows`, `selectedGovernanceIds`, `activeMergeSource`, `deletePreview`, `lastBatchResult`) and bulk orchestration methods for category/alias/cleanup/merge.
- Added RED/GREEN provider and service coverage for all required Task 1/Task 2 methods and endpoint contracts.

## Task Commits

1. **Task 1 RED:** `5cf9830` — `test(19-02): add failing tag governance service coverage`
2. **Task 1 GREEN:** `3b81279` — `feat(19-02): add typed tag governance service contracts`
3. **Task 2 RED:** `b1b957d` — `test(19-02): add failing tag governance provider coverage`
4. **Task 2 GREEN:** `e0832dc` — `feat(19-02): add tag governance provider state and bulk actions`
5. **Task 2 REFACTOR:** `382a6be` — `refactor(19-02): separate legacy statistics state from governance state`

## Files Created/Modified
- `flutter_app/lib/models/tag_governance.dart` - Typed Phase 19 governance contracts and batch failure model.
- `flutter_app/lib/services/tag_service.dart` - Governance endpoint methods and alias/cleanup wrappers.
- `flutter_app/lib/providers/tag_provider.dart` - Governance list/selection/merge state and bulk operation orchestration.
- `flutter_app/test/services/tag_service_test.dart` - TDD coverage for governance service endpoints and payload contracts.
- `flutter_app/test/providers/tag_provider_test.dart` - TDD coverage for governance provider state slices and bulk behavior.

## Decisions Made
- Kept governance parsing in service and orchestration in provider to preserve transport vs state boundaries.
- Aggregated blocked/failed batch entries into a single typed failure list for deterministic UI rendering.
- Preserved legacy statistics and image-tag flows while adding dedicated governance state paths.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Running targeted Flutter tests updated generated desktop plugin registrant files; those generated changes were intentionally restored after each test run and excluded from commits.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02 provider/service seams are complete and test-backed for list-first governance UI composition.
- Ready for `19-03-PLAN.md` widget-level desktop governance workspace work.

## Verification Results

- `cd flutter_app; flutter test test/services/tag_service_test.dart` ✅ pass
- `cd flutter_app; flutter test test/providers/tag_provider_test.dart` ✅ pass
- `cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart` ✅ pass
