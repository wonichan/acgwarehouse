---
phase: 17
slug: desktop-shell-foundation
status: completed
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-04
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `flutter test` + `go test` |
| **Config file** | `flutter_app/pubspec.yaml`; Go uses repo-standard `go test` |
| **Quick run command** | `cd flutter_app && flutter test test/app/fluent_app_shell_test.dart test/widgets/fluent_gallery_content_test.dart` |
| **Full suite command** | `cd flutter_app && flutter test && cd .. && go test ./internal/handler ./internal/service -count=1` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd flutter_app && flutter test <task-targets>` or the corresponding `go test` target
- **After every plan wave:** Run `cd flutter_app && flutter test && cd .. && go test ./internal/handler ./internal/service -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | DSK-01 | widget | `cd flutter_app && flutter test test/app/fluent_app_shell_test.dart test/app/desktop_shell_top_bar_test.dart` | ✅ / ⚠️ new test file | ✅ green |
| 17-02-01 | 02 | 1 | DSK-02 | widget | `cd flutter_app && flutter test test/widgets/fluent_gallery_content_test.dart test/widgets/gallery_filter_panel_test.dart` | ✅ / ⚠️ new test file | ✅ green |
| 17-03-01 | 03 | 2 | DSK-01 | go handler | `go test ./internal/handler ./internal/service -run 'Test(ImageHandler_TriggerImport|TestAdminService_TriggerScan)' -count=1` | ✅ / ⚠️ new test file | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `flutter_app/test/app/desktop_shell_top_bar_test.dart` — top-bar shell tests for search/import/settings
- [x] `flutter_app/test/widgets/gallery_filter_panel_test.dart` — right-side filter panel coverage
- [x] `internal/handler/image_handler_test.go` — thin import endpoint contract coverage

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Top-bar overflow remains usable at narrow widths | DSK-01 | Visual layout and real window constraints | Run desktop app, shrink window, confirm search remains reachable and buttons remain usable |
| Grid tiles stay visually square during window resize | DSK-02 | Widget tests assert structure, not perception | Resize desktop window and confirm tile ratio still reads as square/stable |
| Filter panel focus order feels accessible | DSK-03 | Keyboard traversal quality is interactive | Use Tab/Shift+Tab through search, import, settings, panel controls, and confirm predictable order |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved after Phase 17 verification (`17-VERIFICATION.md`)
