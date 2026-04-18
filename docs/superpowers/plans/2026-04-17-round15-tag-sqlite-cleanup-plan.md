# Round 15 Tag SQLite Cleanup Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and validate the fifteenth deterministic tag-cleanup batch in the isolated worktree, then replay the validated actions to production.

**Architecture:** Reuse the existing action-table/apply-validation workflow already used for rounds 1-14. Extend only the current deterministic script/data path for the exact round-15 reparent set, create a fresh working backup from the current production database, validate on the backup first, and replay to production only after the same validation succeeds.

**Tech Stack:** Python, SQLite, existing repository cleanup scripts, Markdown/CSV/SQL artifacts

---

### Task 1: Read prior workflow inputs

**Files:**
- Read: `artifacts/tag-cleanup/action-table-v1.csv`
- Read: `artifacts/tag-cleanup/action-table-v2.csv`
- Read: `artifacts/tag-cleanup/action-table-v3.csv`
- Read: `artifacts/tag-cleanup/action-table-v4.csv`
- Read: `artifacts/tag-cleanup/action-table-v5.csv`
- Read: `artifacts/tag-cleanup/action-table-v6.csv`
- Read: `artifacts/tag-cleanup/action-table-v7.csv`
- Read: `artifacts/tag-cleanup/action-table-v8.csv`
- Read: `artifacts/tag-cleanup/action-table-v9.csv`
- Read: `artifacts/tag-cleanup/action-table-v10.csv`
- Read: `artifacts/tag-cleanup/action-table-v11.csv`
- Read: `artifacts/tag-cleanup/action-table-v12.csv`
- Read: `artifacts/tag-cleanup/action-table-v13.csv`
- Read: `artifacts/tag-cleanup/action-table-v14.csv`
- Read: `artifacts/tag-cleanup/apply-action-table-v1.py`
- Read: `artifacts/tag-cleanup/validation/production-round14-validation.json`

- [ ] Confirm prior artifact formats and deterministic workflow entry points.
- [ ] Confirm round-15 scope is only the exact 13 `reparent_to` rows and no other action types.

### Task 2: Implement round-15 workflow extension

**Files:**
- Modify: `artifacts/tag-cleanup/apply-action-table-v1.py`
- Create: `artifacts/tag-cleanup/action-table-v15.csv`
- Create: `artifacts/tag-cleanup/action-table-v15.md`
- Create: `artifacts/tag-cleanup/action-table-v15.sql`

- [ ] Extend the existing deterministic workflow rather than creating a parallel implementation.
- [ ] Encode strict preconditions and one-transaction execution for the round-15 batch.
- [ ] Generate CSV/Markdown/SQL artifacts for the exact provided rows only.

### Task 3: Validate on fresh working backup

**Files:**
- Create: `artifacts/tag-cleanup/backups/acgwarehouse.round15-working.db`
- Create: `artifacts/tag-cleanup/validation/round15-working-validation.json`
- Create: `artifacts/tag-cleanup/validation/round15-working-validation.md`

- [ ] Copy the current production database into a fresh round-15 working backup.
- [ ] Apply round-15 actions to the working backup.
- [ ] Validate outcomes and verify artifacts exist and are non-empty.

### Task 4: Replay to production and re-check labels

**Files:**
- Modify: `data/acgwarehouse.db`
- Create: `artifacts/tag-cleanup/validation/production-round15-validation.json`
- Create: `artifacts/tag-cleanup/validation/production-round15-validation.md`

- [ ] Replay the validated round-15 actions to production only after working validation passes.
- [ ] Re-run production validation and verify artifacts exist and are non-empty.
- [ ] Re-check the required labels after replay: 上衣, 裤子, 套装, 丝袜, 手持物, 露背, 吊带睡裙, 挑染, 红色礼服, 露肩裙, 风衣, 黑色吊带, 黑色短裙, 武器（太刀）, 围裙, 披风, 黑裙, 光脚.
