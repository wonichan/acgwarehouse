---
phase: 14
slug: backfill-recovery-operations
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-29
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + node --check |
| **Config file** | none — existing Go package tests |
| **Quick run command** | `go test ./internal/service/... ./internal/handler/... ./internal/repository/... -run "Backfill|AITag|TaskPlatform|Failure|Retry" -count=1 && node --check web/admin/app.js` |
| **Full suite command** | `go test ./internal/... -count=1 && node --check web/admin/app.js` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/service/... ./internal/handler/... ./internal/repository/... -run "Backfill|AITag|TaskPlatform|Failure|Retry" -count=1 && node --check web/admin/app.js`
- **After every plan wave:** Run `go test ./internal/... -count=1 && node --check web/admin/app.js`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | AIQ-03 | service/repository | `go test ./internal/service/... ./internal/repository/... -run "BackfillPreview|BackfillCandidate|FindImagesWithoutAITags" -count=1` | ✅ | ⬜ pending |
| 14-01-02 | 01 | 1 | AIQ-03 | handler | `go test ./internal/handler/... -run "Backfill|AITag" -count=1` | ✅ | ⬜ pending |
| 14-02-01 | 02 | 1 | SAFE-01 | service/worker | `go test ./internal/service/... -run "MarkJobFailed|PartialFailed|Isolation" -count=1` | ✅ | ⬜ pending |
| 14-02-02 | 02 | 1 | SAFE-02 | repository/service | `go test ./internal/repository/... ./internal/service/... -run "FailureSummary|RetryHint|TaskRead" -count=1` | ✅ | ⬜ pending |
| 14-03-01 | 03 | 2 | AIQ-03, SAFE-02 | handler/ui | `go test ./internal/handler/... -run "Backfill|TaskPlatform" -count=1 && node --check web/admin/app.js` | ✅ | ⬜ pending |
| 14-03-02 | 03 | 2 | SAFE-01, SAFE-02 | integration/manual-check scaffold | `go test ./internal/... -run "Backfill|Retry|Failure|TaskPlatform" -count=1 && node --check web/admin/app.js` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Backfill confirmation clearly explains hit/create/skip totals | AIQ-03 | Final wording and UX comprehension | Open `/admin-ui/`, trigger the backfill action from a narrowed filter scope, confirm the dialog shows hit count, created-task count, and skip buckets before submit |
| Batch list displays grouped failure reasons with retry guidance | SAFE-02 | Visual information density and readability | Load batches with `failed` / `partial_failed`, confirm the batch row exposes grouped reasons and a retry recommendation without opening task details |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
