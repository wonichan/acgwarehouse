# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v3.0 — 导入后任务平台化

**Shipped:** 2026-03-29
**Phases:** 4 | **Plans:** 15 | **Sessions:** not tracked centrally

### What Was Built
- Unified post-import batch/task platform on top of the existing `async_jobs` execution layer.
- Automatic AI enqueue flow for eligible imported images with lifecycle wiring through app startup and config reload.
- Batch-first admin monitoring with overview stats, queue controls, retry, cancel, clear, and grouped failure visibility.
- Preview-first backfill / recovery workflow with single-image failure isolation and retry hints.

### What Worked
- Phase-level verification reports made it possible to reconstruct requirement coverage with high confidence.
- Reusing the existing worker engine while adding batch/task semantics avoided a disruptive executor rewrite.
- Keeping admin UX batch-first reduced ambiguity in operator flows and aligned backend contracts with user tasks.

### What Was Inefficient
- `ROADMAP.md` and `REQUIREMENTS.md` drifted behind the implemented state and had to be normalized during archive.
- No standalone milestone audit existed before archive time, so the final audit had to be synthesized from phase artifacts.

### Patterns Established
- Treat `async_jobs` as execution plumbing and batch/task as product semantics.
- Prefer batch-first monitoring and “retry as new batch” semantics for operator-facing recovery flows.
- Use preview-first UX for potentially destructive or high-cost recovery actions.

### Key Lessons
1. Requirement traceability needs to be updated when phase verification passes, not deferred to milestone close.
2. For operations-heavy phases, backend contracts and verification must be the source of truth; human checks should stay limited to visual and true end-to-end flows.

### Cost Observations
- Model mix: not tracked in repo metadata
- Sessions: not tracked centrally
- Notable: phase summaries plus verification docs kept archive reconstruction cheap even though the dedicated milestone audit was missing initially

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | not tracked | 6 | Established the Go + Flutter single-machine image library baseline |
| v2.0 | not tracked | 4 | Added dual-platform UI shells and shared provider / theme architecture |
| v3.0 | not tracked | 4 | Elevated post-import work into a unified task platform with operator controls |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.0 | mixed historical verification | 89% audit coverage | n/a |
| v2.0 | 61+ tests passing | 100% | n/a |
| v3.0 | phase 13 service/handler suites + 27 phase-14-specific tests | 15/15 archived complete | n/a |

### Top Lessons (Verified Across Milestones)

1. Reusing stable foundations and layering product semantics on top scales better than executor rewrites.
2. Milestone archives are easiest to close when roadmap, requirements, and verification stay in sync throughout execution.
