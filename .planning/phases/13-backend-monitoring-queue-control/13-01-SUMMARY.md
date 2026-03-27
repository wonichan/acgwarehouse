---
phase: 13-backend-monitoring-queue-control
plan: 01
subsystem: ui
tags: [admin, html, javascript, css, monitoring, batch-first]
requires:
  - phase: 12
    provides: task_batches and task detail read models plus the auto-scheduling backend context
provides:
  - batch-first admin monitoring shell with batch list on top and task details below
  - stable batch selection with abnormal/running-first task surfacing
  - manual refresh plus 30-second auto-refresh behavior for the admin page
affects: [13-02, 13-03, 13-04, Wave 2 orchestration]
tech-stack:
  added: []
  patterns: [static admin shell, delegated row selection, front-end stable sorting, glassy monitoring panels]
key-files:
  created:
    - .planning/phases/13-backend-monitoring-queue-control/13-01-SUMMARY.md
  modified:
    - web/admin/index.html
    - web/admin/app.js
    - web/admin/styles.css
    - .planning/STATE.md
    - .planning/ROADMAP.md
key-decisions:
  - "Kept the locked batch-first layout: batch list on top, task details below, no jobs-first fallback."
  - "Sorted batches so unfinished items stay ahead, then newest-first; task details surface failed/running items first."
  - "Preserved manual refresh and a 30-second auto-refresh loop without resetting the selected batch."
patterns-established:
  - "Pattern 1: batch rows and recent-error cards use delegated click handling so labels with punctuation stay safe."
  - "Pattern 2: monitoring panes use shared panel/table primitives and CSS variables instead of ad-hoc visual values."
requirements-completed: [PIPE-02, OPS-01]
duration: 20 min
completed: 2026-03-27
---

# Phase 13 Plan 01: Batch-First Admin Monitoring UI Summary

Batch-first admin monitoring landed in the existing `/admin-ui/` shell, keeping the page focused on batches first and task details second.

## Performance

- **Duration:** 20 min
- **Started:** 2026-03-27T13:46:00Z
- **Completed:** 2026-03-27T14:06:35Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Rebuilt `web/admin/index.html` into a batch-first monitoring layout with platform overview, batch table, detail pane, recent errors, and reference data sections.
- Restyled `web/admin/styles.css` to match a single monitoring system with reusable panels, badges, table shells, and responsive behavior.
- Wired `web/admin/app.js` to load `/admin/api/task-batches` and `/admin/api/tasks?batch_id=`, preserve selection across refreshes, and prioritize abnormal/running tasks.

## Task Commits

Each task was committed atomically:

1. **Task 1: 重构后台页面为批次优先监控布局** - `5e730e9` (feat)
2. **Task 2: 接入批次列表、选中批次与异常优先明细加载** - `a83deac` (feat)

**Plan metadata:** `5e730e9` / `a83deac` (task commits in this retry)

## Files Created/Modified

- `.planning/phases/13-backend-monitoring-queue-control/13-01-SUMMARY.md` - execution record for this retry
- `web/admin/index.html` - batch-first shell and section structure
- `web/admin/app.js` - batch loading, sorting, selection, and refresh behavior
- `web/admin/styles.css` - visual system for the batch monitor page
- `.planning/STATE.md` - phase progress bump
- `.planning/ROADMAP.md` - phase 13 plan count bump

## Decisions Made

- Kept the locked batch-first layout and did not reintroduce a jobs-first primary view.
- Used front-end stable sorting to surface unfinished batches first and abnormal tasks first.
- Kept both manual refresh and the 30-second auto-refresh timer.

## Deviations from Plan

None - plan executed exactly as specified for the frontend slice.

## Issues Encountered

- The exact plan `rg` verification command was unavailable in this shell, so I used a `grep` equivalent against the same files.
- `lsp_diagnostics` could not run for `web/admin/app.js` because the TypeScript language server is not installed in this environment; I verified syntax with `node --check` instead.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 13 plan 02 can now build on the batch-first UI shell.
- The admin page no longer depends on the old jobs-first primary view.

## Self-Check

PASSED — summary file exists, task commits exist, and the changed admin files render a coherent batch-first monitoring page.

---
*Phase: 13-backend-monitoring-queue-control*
*Completed: 2026-03-27*
