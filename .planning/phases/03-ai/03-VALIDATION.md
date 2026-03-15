---
phase: 03
slug: ai
status: revised-gap-closure
nyquist_compliant: true
wave_0_complete: false
created: 2026-03-15
updated: 2026-03-15
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution, revised for the Phase 03 gap-closure plan set (`03-05`..`03-07`).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Go framework** | `go test` |
| **Flutter framework** | `flutter test` |
| **Quick run command** | `go test ./internal/service/... ./internal/worker/... ./internal/repository/... ./internal/handler/...` |
| **Frontend quick run command** | `flutter test test/services/tag_service_test.dart test/providers/tag_provider_test.dart` |
| **Full backend suite** | `go test ./...` |
| **Full frontend suite** | `flutter test` |
| **Estimated runtime** | backend ~45s, frontend ~60s |

---

## Sampling Rate

- **After every task commit:** run the task's `<automated>` command.
- **After every backend gap-closure plan:** run `go test ./internal/service/... ./internal/worker/... ./internal/repository/... ./internal/handler/...`.
- **After the frontend gap-closure plan:** run `flutter test` and `flutter analyze`.
- **Before `/gsd-verify-work`:** run `go test ./...` and `flutter test`.
- **Max feedback latency:** 60 seconds.

---

## Revised Gap-Closure Plan Inventory

| Plan | Wave | Depends On | Scope | Requirements |
|------|------|------------|-------|--------------|
| `03-05-PLAN.md` | 4 | `03-01`, `03-02`, `03-03` | AI worker -> governance merge wiring, alias-aware merge | `AIRE-03`, `AIRE-05` |
| `03-06-PLAN.md` | 5 | `03-03`, `03-05` | governed image filtering + merge/stats APIs | `AIRE-05`, `TAGS-03`, `TAGS-05` |
| `03-07-PLAN.md` | 6 | `03-04`, `03-05`, `03-06` | Flutter filter wiring, AI status/merge UI, governance screen | `AIRE-05`, `TAGS-03`, `TAGS-05` |

---

## Per-task Verification Map

| task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-05-01 | 05 | 4 | AIRE-03 | unit | `go test ./internal/service/... -run TestTagGovernance` | ✅ | ⬜ pending |
| 03-05-02 | 05 | 4 | AIRE-03, AIRE-05 | unit | `go test ./internal/worker/... -run TestAITagHandler` | ✅ | ⬜ pending |
| 03-06-01 | 06 | 5 | TAGS-03 | integration | `go test ./internal/repository/... ./internal/handler/... -run "Test(ImageRepository|ImageHandler)"` | ❌ W0 | ⬜ pending |
| 03-06-02 | 06 | 5 | AIRE-05, TAGS-05 | integration | `go test ./internal/repository/... ./internal/handler/... -run "Test(ImageTagRepository|ImageTagHandler|TagHandler)"` | ✅ | ⬜ pending |
| 03-07-01 | 07 | 6 | TAGS-03 | unit/widget | `flutter test test/services/api_service_test.dart test/providers/image_provider_test.dart` | ❌ W0 | ⬜ pending |
| 03-07-02 | 07 | 6 | AIRE-05 | widget | `flutter test test/services/tag_service_test.dart test/screens/image_detail_screen_test.dart` | ❌ W0 | ⬜ pending |
| 03-07-03 | 07 | 6 | TAGS-05 | widget | `flutter test test/providers/tag_provider_test.dart test/screens/tag_management_screen_test.dart` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

These test scaffolds do not exist yet, so execution must create them before claiming the corresponding task is complete.

### Go Backend

- [ ] `internal/repository/image_repository_test.go` — governed image filtering repository coverage for `03-06-01`
- [ ] `internal/handler/image_handler_test.go` — image list/detail handler coverage for `03-06-01`

### Flutter Frontend

- [ ] `flutter_app/test/services/api_service_test.dart` — image query serialization coverage for `03-07-01`
- [ ] `flutter_app/test/providers/image_provider_test.dart` — provider filter-state coverage for `03-07-01`
- [ ] `flutter_app/test/screens/image_detail_screen_test.dart` — AI polling / merge UI coverage for `03-07-02`
- [ ] `flutter_app/test/screens/tag_management_screen_test.dart` — governance screen coverage for `03-07-03`

### Already Present and Reused

- [x] `internal/worker/ai_tag_handler_test.go`
- [x] `internal/service/tag_governance_service_test.go`
- [x] `internal/repository/image_tag_repository_test.go`
- [x] `internal/handler/image_tag_handler_test.go`
- [x] `internal/handler/tag_handler_test.go`
- [x] `flutter_app/test/services/tag_service_test.dart`
- [x] `flutter_app/test/providers/tag_provider_test.dart`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| AI job creates reviewable pending tags after completion | `AIRE-05` | Requires live provider credentials and async processing | 1. Configure AI provider keys. 2. Run server. 3. Trigger `POST /api/v1/images/:id/ai-tags`. 4. Wait for completion. 5. Verify the image detail UI shows new pending tags without reopening the page. |
| Gallery filter returns the expected AND-intersection result set | `TAGS-03` | Requires integrated dataset + UI behavior confirmation | 1. Seed images with overlapping governed tags. 2. Select two tags in the drawer. 3. Confirm only images containing both tags remain visible. |
| Governance screen metrics match backend state | `TAGS-05` | Visual verification against realistic data | 1. Open the governance screen. 2. Compare usage/pending/source counts against API responses from `/api/v1/tags/stats`. |

---

## Validation Sign-Off

- [x] All revised gap-closure tasks have `<automated>` verify commands or explicit Wave 0 file scaffolds.
- [x] Sampling continuity is maintained across the revised plan set.
- [x] Wave 0 covers every missing revised test file.
- [x] No watch-mode commands are used.
- [x] The revised plan set stays within the verification gaps only.
- [ ] `wave_0_complete: true` can only be set after the missing scaffolds exist.

**Approval:** pending execution
