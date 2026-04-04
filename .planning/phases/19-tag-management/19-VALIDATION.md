---
phase: 19
slug: tag-management
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-05
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` + `flutter test` |
| **Config file** | `go.mod`, `flutter_app/pubspec.yaml` |
| **Quick run command** | `go test ./internal/service ./internal/handler -run 'TestTag|TestTagGovernance' -count=1` + `cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/app/fluent_app_shell_test.dart` |
| **Full suite command** | `go test ./internal/... -count=1` + `cd flutter_app; flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart test/app/fluent_app_shell_test.dart test/widgets/tag_management_workspace_test.dart test/widgets/tag_merge_panel_test.dart test/widgets/tag_bulk_action_bar_test.dart` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run the task’s targeted `go test` or `flutter test` command.
- **After every plan wave:** Run the corresponding broader backend or Flutter phase suite.
- **Before `/gsd-verify-work`:** All Phase 19 targeted suites must be green.
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 1 | DSK-04 | backend unit/handler | `go test ./internal/service ./internal/handler -run 'TestTagAdminService|TestTagGovernance' -count=1` | ✅ | ⬜ pending |
| 19-01-02 | 01 | 1 | DSK-04 | backend handler | `go test ./internal/handler -run 'TestTag(Delete|Cleanup)' -count=1` | ✅ | ⬜ pending |
| 19-02-01 | 02 | 2 | DSK-04 | flutter service | `cd flutter_app; flutter test test/services/tag_service_test.dart` | ✅ | ⬜ pending |
| 19-02-02 | 02 | 2 | DSK-04 | flutter provider | `cd flutter_app; flutter test test/providers/tag_provider_test.dart` | ✅ | ⬜ pending |
| 19-03-01 | 03 | 3 | DSK-04 | widget/app | `cd flutter_app; flutter test test/widgets/tag_management_workspace_test.dart test/app/fluent_app_shell_test.dart` | ❌ W0 | ⬜ pending |
| 19-03-02 | 03 | 3 | DSK-04 | widget | `cd flutter_app; flutter test test/widgets/tag_merge_panel_test.dart test/widgets/tag_bulk_action_bar_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `flutter_app/test/widgets/tag_management_workspace_test.dart` — workspace and gallery-handoff coverage scaffold
- [ ] `flutter_app/test/widgets/tag_merge_panel_test.dart` — merge-panel workflow scaffold
- [ ] `flutter_app/test/widgets/tag_bulk_action_bar_test.dart` — bulk-governance toolbar scaffold

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Desktop drilldown opens gallery with the chosen tag filter applied | DSK-04 | Final navigation feel is easier to confirm interactively after automated widget checks | Launch desktop shell, open tag management, click “View affected images” on a tag row, confirm shell switches to gallery and filtered results match the tag |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
