# Phase 14 Research — Backfill Recovery Operations

**Phase:** 14  
**Date:** 2026-03-29  
**Status:** Complete

## Research Goal

Answer: what must be true to plan Phase 14 well, given the locked decisions in `14-CONTEXT.md` and the existing Phase 11-13 platform semantics.

## Locked Decisions Carried Into Research

- Use the **current filtered result** as the backfill scope, never an implicit whole-library action (`D-01`, `D-02`, `D-03`).
- Only act on images that **currently have no AI tags**, including images whose AI tags were deleted later (`D-06`, `D-07`, `D-08`).
- Skip images that already have active `pending` / `queued` / `running` AI work, and report that skip reason explicitly (`D-09`, `D-12`).
- Show a **preflight confirmation** with at least hit count, created-task count, and skip count before enqueueing (`D-10`).
- Reuse the Phase 13 retry UX baseline: toast feedback plus navigation to the new batch (`D-11`).
- Keep failure recovery semantics split: **retry failed tasks** remains the primary recovery action for failed batches; “backfill missing AI tags” is a separate path (`D-18`, `D-19`, `D-20`).
- Show failure summaries **at batch-list level**, grouped by reason with a retryability hint (`D-14`–`D-17`).

## Existing Code Findings

### 1. The queueing foundation already exists

The current AI enqueue flow in `internal/handler/ai_tag_handler.go` already routes through `TaskPlatformService.PlanBatch()` and `QueueTask()`. This means Phase 14 does **not** need a new queueing subsystem. It needs a new orchestration layer that:

- resolves the current filtered candidate set,
- classifies skipped images,
- creates a manual batch with explicit feedback,
- preserves the existing “one action = one new batch” invariant.

### 2. Candidate eligibility is partially implemented but too narrow for Phase 14

`ImageRepository.FindImagesWithoutAITags(ctx, limit)` already enforces two critical rules:

- current image has **no AI tag** association;
- image has **no active AI task**.

That matches `D-06` through `D-09`, but it is limited to a blind scan (`limit` only). Phase 14 needs a **filtered-result-aware** query contract that can:

- evaluate the full filtered result set, not just one page;
- compute counts by skip reason;
- support preview and then enqueue from the same filter snapshot.

### 3. The current image-list filter contract is usable

`internal/handler/image_handler.go` and the Flutter `ImageListProvider` already expose a filter model based on:

- `tag_ids`
- `has_tags`
- `sort_by`
- `sort_dir`
- pagination (`limit`, `offset`)

This is the best existing contract to anchor “current filtered result.” The admin recovery flow should reuse the same semantics rather than inventing a second incompatible filter language.

### 4. Failure isolation is already mostly in place, but it needs explicit regression coverage

The worker manager in `internal/worker/job_manager.go` processes jobs independently. The task platform marks failure per job via `TaskPlatformService.MarkJobFailed()`, and batch status is already able to become `partial_failed` through repository aggregation. That means SAFE-01 is **architecturally supported already**, but Phase 14 still needs:

- regression tests proving one failed image does not stall sibling tasks in the same batch;
- operational summaries that explain the failure pattern, not only the latest error string.

### 5. Current failure visibility is too shallow for Phase 14

The batch read model currently exposes only:

- `FailureSummary` ← mapped from `latest_error_summary`

That is insufficient for the locked UX. Phase 14 needs grouped summaries such as:

- reason/category,
- affected task count,
- retry recommendation.

The right place to derive this is the existing batch read path:

- `internal/repository/task_batch_read_repository.go`
- `internal/service/task_read_service.go`

This keeps the admin page batch-first and avoids making the browser recompute operational meaning from raw task rows.

## Recommended Implementation Shape

### Recommendation 1 — Add a dedicated backfill orchestration service

Create a focused service (for example `internal/service/ai_backfill_service.go`) instead of pushing preview/count/skip logic into handlers.

This service should:

- accept a filter contract compatible with the image list query shape;
- return a **preview** model with:
  - total matched images,
  - enqueueable images,
  - skipped-with-ai-tag count,
  - skipped-with-active-task count,
  - optional candidate IDs for execution;
- create a **manual batch** with `TaskBatchSourceManualBatch` and queue only the enqueueable candidates.

Why this is the right seam:

- it makes TDD practical;
- it avoids duplicating repository logic in handlers;
- it gives the admin handler and any future UI the same consistent backfill contract.

### Recommendation 2 — Split preview and execute into separate endpoints

Do not let the UI post a blind “run backfill now” request.

Use two steps:

1. **Preview endpoint** — validates scope and returns counts.
2. **Execute endpoint** — repeats or consumes the same filter contract and creates the batch.

This directly supports `D-10` and `D-13`, and it gives the UI enough structured data to explain “nothing to backfill” without pretending that `0 created` is success.

### Recommendation 3 — Derive grouped failure summaries in the read model, not in the browser

Add a structured failure summary array to the batch read contract, for example:

- `failure_groups[]`
  - `reason_key`
  - `reason_label`
  - `count`
  - `retry_recommended`
  - `retry_hint`

Keep the existing top-level `failure_summary` as a short text fallback for backward compatibility, but make the new grouped data the source of truth for the admin UI.

### Recommendation 4 — Treat retryability as a deterministic rule set

Do not make the browser infer “safe to retry.”

Define backend rules such as:

- transient/network/API timeout → retry recommended
- auth/configuration/model setup error → retry not recommended
- empty payload / malformed record / missing image path → retry not recommended

The exact labels are discretionary, but the classifier should live in backend code so that:

- summaries stay stable,
- tests can lock the behavior,
- the admin page stays thin.

### Recommendation 5 — Keep failure recovery and missing-tag backfill explicitly separate

Do not turn backfill into the main action for failed batches.

The admin page should present:

- **Retry failed tasks** as the primary action on `failed` / `partial_failed` batches;
- **Backfill missing AI tags** as a separate global/filter-scoped action.

That preserves the distinct user mental models required by `D-18` through `D-20`.

## TDD Implications

Phase 14 is a strong TDD candidate because the core behaviors are contract-driven:

- filtered preview output,
- skip-reason accounting,
- zero-create rejection messaging,
- per-image failure isolation,
- grouped failure summaries with retry hints.

The best plan shape is:

1. backend contract tests first,
2. read-model aggregation tests first,
3. admin UI wiring after the backend payloads are stable.

## Risks and Mitigations

### Risk: preview and execute drift apart

Mitigation: both paths should use the same service-level candidate classifier.

### Risk: full filtered result requires a large in-memory ID list

Mitigation: plan for repository methods that can produce aggregate counts and then page candidate IDs for execution in chunks, rather than loading all image rows eagerly.

### Risk: admin UI becomes a second image browser

Mitigation: keep the Phase 14 UI narrow:

- filter scope,
- preview dialog,
- execute action,
- batch jump,
- grouped failure summaries.

Do not add gallery browsing or full image-management features.

## Recommended Plan Split

### Plan 14-01
- Create the filtered backfill preview + execute backend contract.
- Lock skip-reason accounting and empty-scope behavior.

### Plan 14-02
- Add explicit isolation regression coverage and grouped failure summaries with retry hints.
- Keep all logic in backend/read-model seams.

### Plan 14-03
- Wire the admin page UX for backfill preview/execute and grouped failure visibility.
- Reuse the existing retry toast + batch navigation pattern.

## Validation Architecture

### Fast loop
- Service/repository/handler tests for backfill preview and failure summaries.
- JavaScript syntax validation for the admin UI.

### Phase-level loop
- Run focused Go test subsets after each task.
- Run all touched Go package tests and `node --check web/admin/app.js` at the end of each wave.

### Manual checks still needed
- Verify the backfill confirmation clearly explains hit/create/skip counts.
- Verify failed batches show grouped reasons and retry guidance directly in the batch list.

## Final Planning Guidance

- Reuse existing queue and batch primitives; do not invent a new platform layer.
- Reuse the current image filter semantics; do not invent a second filter language.
- Put preview/execute/failure classification logic in backend services that are easy to test first.
- Keep the admin page batch-first and operationally dense.
- Make the plan explicitly TDD-oriented, with atomic red/green/refactor commit slices.
