# AI Tag Batch Coordinator Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor AI tag batching to use a single in-memory coordinator that collects up to 4 queued `ai_tag_generation` jobs, waits briefly for more work, and dispatches even when fewer than 4 jobs arrive.

**Architecture:** Remove SQLite-side sibling claiming from the AI tag handler path. Each `ai_tag_generation` handler invocation will submit its own already-running job into a shared worker-layer batch coordinator, which groups jobs in memory and flushes on max-size or timer expiry. Per-job persistence and success/failure handling remain isolated to the invoking worker.

**Tech Stack:** Go, SQLite, ants worker pool, existing AI provider abstraction, Go testing package.

---

## File Structure

- Modify: `internal/worker/ai_tag_handler.go`
  - Remove DB-side extra-claim batching logic.
  - Reuse existing request building / persistence logic against already-collected in-memory batch entries.
- Create: `internal/worker/ai_tag_batch_coordinator.go`
  - Own the short wait window, max batch size, flush logic, and per-job result fan-out.
- Modify: `internal/worker/ai_tag_handler_test.go`
  - Replace claim-oriented tests with queue/coordinator-oriented tests.
- Create: `internal/worker/ai_tag_batch_coordinator_test.go`
  - Focused batching tests for fill-to-4 and partial flush after wait window.
- Modify: `internal/app/bootstrap.go`
  - Construct a single coordinator instance and inject it into `NewBatchAITagJobHandler`.

## Chunk 1: Coordinator tests and design seam

### Task 1: Add failing coordinator tests

**Files:**
- Create: `internal/worker/ai_tag_batch_coordinator_test.go`

- [ ] **Step 1: Write failing test for full batch flush**

```go
func TestAITagBatchCoordinator_FlushesAtFour(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/worker -run TestAITagBatchCoordinator_FlushesAtFour -count=1`
Expected: FAIL because coordinator does not exist yet.

- [ ] **Step 3: Write failing test for partial flush after wait window**

```go
func TestAITagBatchCoordinator_FlushesPartialBatchAfterWaitWindow(t *testing.T) {}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/worker -run TestAITagBatchCoordinator_ -count=1`
Expected: FAIL because batching timer/flush behavior is not implemented.

## Chunk 2: Coordinator implementation

### Task 2: Implement the minimal batch coordinator

**Files:**
- Create: `internal/worker/ai_tag_batch_coordinator.go`
- Test: `internal/worker/ai_tag_batch_coordinator_test.go`

- [ ] **Step 1: Add minimal coordinator types and injectable timer/flush seam**

```go
type aiTagBatchCoordinator struct { /* pending entries, max size, wait window */ }
```

- [ ] **Step 2: Implement submit + flush behavior**

```go
func (c *aiTagBatchCoordinator) Submit(...) error { /* collect and wait for result */ }
```

- [ ] **Step 3: Run coordinator tests**

Run: `go test ./internal/worker -run TestAITagBatchCoordinator_ -count=1`
Expected: PASS.

## Chunk 3: Handler tests first

### Task 3: Add failing handler tests for new batching behavior

**Files:**
- Modify: `internal/worker/ai_tag_handler_test.go`

- [ ] **Step 1: Write failing test for auto mode batching without repo claim**

```go
func TestBatchAITagHandler_AutoModeBatchesConcurrentJobsWithoutClaiming(t *testing.T) {}
```

- [ ] **Step 2: Write failing test for single mode bypass**

```go
func TestBatchAITagHandler_SingleModeBypassesCoordinator(t *testing.T) {}
```

- [ ] **Step 3: Write failing test for per-job result routing**

```go
func TestBatchAITagHandler_PerJobErrorsReturnToTheCorrectCaller(t *testing.T) {}
```

- [ ] **Step 4: Run focused tests to verify RED**

Run: `go test ./internal/worker -run "TestBatchAITagHandler_(AutoModeBatchesConcurrentJobsWithoutClaiming|SingleModeBypassesCoordinator|PerJobErrorsReturnToTheCorrectCaller)" -count=1`
Expected: FAIL for missing/new behavior.

## Chunk 4: Handler refactor

### Task 4: Refactor the AI tag handler to batch in memory

**Files:**
- Modify: `internal/worker/ai_tag_handler.go`
- Modify: `internal/app/bootstrap.go`
- Test: `internal/worker/ai_tag_handler_test.go`

- [ ] **Step 1: Introduce a shared coordinator dependency into `NewBatchAITagJobHandler`**
- [ ] **Step 2: Parse payload before batching and short-circuit skipped jobs before coordinator submission**
- [ ] **Step 3: Extract current request execution / persistence logic into a helper that accepts in-memory batch entries**
- [ ] **Step 4: Remove `FindAndClaimReadyJobs` usage from `handleBatchAITagGeneration`**
- [ ] **Step 5: Remove same-batch sibling isolation/release logic that only existed for DB-claimed siblings**
- [ ] **Step 6: Run focused worker tests**

Run: `go test ./internal/worker -run "AITag|Batch" -count=1`
Expected: PASS.

## Chunk 5: Verification

### Task 5: Verify integration and regressions

**Files:**
- Modify as needed: `internal/worker/ai_tag_handler.go`, `internal/worker/ai_tag_batch_coordinator.go`, related tests only if failures reveal gaps.

- [ ] **Step 1: Run worker package tests**

Run: `go test ./internal/worker -count=1`
Expected: PASS.

- [ ] **Step 2: Run app-level focused tests**

Run: `go test ./internal/app -run "AI|Refill" -count=1`
Expected: PASS or no tests selected.

- [ ] **Step 3: Run diagnostics on changed files**

Use diagnostics on:
- `internal/worker/ai_tag_handler.go`
- `internal/worker/ai_tag_batch_coordinator.go`
- `internal/app/bootstrap.go`

- [ ] **Step 4: Run race test for batch coordinator path**

Run: `go test ./internal/worker -race -run "AITag|Batch" -count=1`
Expected: PASS.

Plan complete and saved to `docs/superpowers/plans/2026-04-12-ai-tag-batch-coordinator.md`. Ready to execute.
