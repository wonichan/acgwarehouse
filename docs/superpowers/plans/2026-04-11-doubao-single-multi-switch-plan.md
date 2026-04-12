# Doubao Single-Image / Multi-Image Switch Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Doubao-only config switch that explicitly controls worker-side job aggregation plus single-image vs multi-image API path selection, while preserving the existing single-image path and fallback model behavior.

**Architecture:** Keep the policy boundary explicit: config defines the desired mode, app/bootstrap passes the effective mode into the AI tag handler, worker decides whether to claim extra jobs and whether to call single-image or batch generation, and Doubao providers remain responsible for request execution plus fallback retry. `single` preserves the old path, `auto` preserves today’s implicit behavior, and `multi` forces the batch path even for one image.

**Tech Stack:** Go 1.23, existing `internal/config`, `internal/ai`, `internal/worker`, existing app bootstrap wiring, Go unit tests, `go test`

---

## File Map

- **Modify:** `internal/config/config.go`
  - Add `DoubaoBatchMode`, defaulting/normalization, and env override handling.
- **Modify:** `internal/config/config_test.go`
  - Add config parsing/default/override coverage for the new field.
- **Modify:** `deploy/config/config.example.yaml`
  - Document the new Doubao-only mode with example values.
- **Modify:** `internal/ai/doubao_provider.go`
  - Add any provider-local state/helper needed to honor forced batch semantics.
- **Modify:** `internal/ai/doubao_batch_provider.go`
  - Remove the unconditional single-item shortcut so `multi` can force batch for one image.
- **Modify:** `internal/ai/fallback_doubao_provider.go`
  - Ensure fallback clients preserve the same effective batch behavior.
- **Modify:** `internal/ai/provider.go`
  - Inject the normalized Doubao batch mode when constructing providers.
- **Modify:** `internal/ai/ai_test.go`
  - Add provider-construction and forced-single-item-batch regression tests.
- **Modify:** `internal/ai/parse_batch_tags_test.go`
  - Add/extend coverage for single-item numbered batch output if needed.
- **Modify:** `internal/worker/ai_tag_handler.go`
  - Add explicit mode handling for claim/no-claim and single-vs-batch call path decisions.
- **Modify:** `internal/worker/ai_tag_handler_test.go`
  - Add behavior tests for `single`, `auto`, and `multi`.
- **Modify:** `internal/app/bootstrap.go`
  - Pass the effective Doubao mode into the batch AI tag handler at registration time.

## Constraints / Existing Patterns

- `internal/app/bootstrap.go` currently creates the AI provider once, wraps it in `ai.NewRateLimitedClient(...)`, and registers `worker.NewBatchAITagJobHandler(...)` without any mode parameter.
- `internal/worker/ai_tag_handler.go` currently decides path implicitly: one request → `GenerateTags`, multiple requests → `GenerateTagsBatch`.
- `internal/ai/doubao_batch_provider.go` currently hardcodes a single-request shortcut back to `GenerateTags`, which conflicts with the approved `multi` mode.
- `internal/ai/fallback_doubao_provider.go` already retries both `GenerateTags` and `GenerateTagsBatch`; keep that pattern.
- `internal/service/ai_image_source.go` already prefers `ThumbnailLargeUrl`; do not change image source strategy in this plan.
- Do **not** expand scope to Responses API migration, thumbnail generation tuning, or non-Doubao providers.

## Chunk 1: Config and provider construction

### Task 1: Add normalized Doubao batch mode config coverage first

**Files:**
- Modify: `internal/config/config_test.go`
- Modify: `internal/config/config.go` (only after failing tests)

- [ ] **Step 1: Write the failing test**

Add tests that verify all of the following:

```go
func TestLoadConfigParsesDoubaoBatchMode(t *testing.T) {
    configYAML := `ai:
  provider: "doubao"
  api_key: "k"
  model: "doubao-seed-2-0-pro-260215"
  doubao_batch_mode: "multi"
`
    // write temp file, LoadConfig, assert cfg.AI.DoubaoBatchMode == "multi"
}

func TestLoadConfigDefaultsDoubaoBatchModeToAuto(t *testing.T) {
    configYAML := `ai:
  provider: "doubao"
  api_key: "k"
`
    // assert cfg.AI.DoubaoBatchMode == "auto"
}

func TestLoadConfigEnvOverrideDoubaoBatchMode(t *testing.T) {
    t.Setenv("AI_DOUBAO_BATCH_MODE", "single")
    // assert override wins
}
```

If you choose normalization over hard error for invalid values, add a regression test that invalid input normalizes to `auto`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run DoubaoBatchMode -count=1`
Expected: FAIL because the field/default/env override do not exist yet.

- [ ] **Step 3: Write minimal implementation**

In `internal/config/config.go`:

- add `DoubaoBatchMode string `yaml:"doubao_batch_mode"`` to `AIConfig`
- add a small normalization helper such as:

```go
func normalizeDoubaoBatchMode(v string) string {
    switch strings.ToLower(strings.TrimSpace(v)) {
    case "single", "auto", "multi":
        return ...
    default:
        return "auto"
    }
}
```

- in defaults/load path, normalize the field and ensure empty value becomes `auto`
- add `AI_DOUBAO_BATCH_MODE` env override and normalize it

Keep the behavior Doubao-only by policy, but do not block loading when another provider is selected.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run DoubaoBatchMode -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add doubao batch mode config"
```

### Task 2: Document the new config in example YAML

**Files:**
- Modify: `deploy/config/config.example.yaml`

- [ ] **Step 1: Write the failing test**

No automated test required for the example config file. Instead, add an explicit checklist item in the implementation PR/review notes:

- example config shows `doubao_batch_mode`
- comments explain `single`, `auto`, `multi`
- comments say it is Doubao-only

- [ ] **Step 2: Manual verification before edit**

Read `deploy/config/config.example.yaml` and confirm the Doubao section already contains `provider`, `model`, and `fallback_models` but no mode setting.

- [ ] **Step 3: Write minimal implementation**

Add:

```yaml
  # Doubao-only request mode:
  # - single: never batch jobs, always one image per request
  # - auto: keep current behavior (1 image = single, multiple images = batch)
  # - multi: allow batching and force batch request path even for one image
  doubao_batch_mode: "auto"
```

Place it near `model` / `fallback_models`.

- [ ] **Step 4: Manual verification after edit**

Confirm the YAML still reads clearly and the comment does not imply other providers use this field.

- [ ] **Step 5: Commit**

```bash
git add deploy/config/config.example.yaml
git commit -m "docs: document doubao batch mode config"
```

### Task 3: Inject effective mode into constructed Doubao providers

**Files:**
- Modify: `internal/ai/provider.go`
- Modify: `internal/ai/doubao_provider.go`
- Modify: `internal/ai/ai_test.go`

- [ ] **Step 1: Write the failing test**

Add provider tests that assert:

```go
func TestNewProvider_DoubaoDefaultsBatchModeToAuto(t *testing.T) { ... }
func TestNewProvider_DoubaoPropagatesMultiBatchModeToPrimaryAndFallbackClients(t *testing.T) { ... }
```

Use type assertions against `*DoubaoProvider` or `*FallbackDoubaoProvider` and inspect the internal mode field you add.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai -run Doubao.*BatchMode -count=1`
Expected: FAIL because providers do not store mode yet.

- [ ] **Step 3: Write minimal implementation**

Add a small internal mode field to `DoubaoProvider`, for example:

```go
type doubaoBatchMode string

const (
    doubaoBatchModeSingle doubaoBatchMode = "single"
    doubaoBatchModeAuto   doubaoBatchMode = "auto"
    doubaoBatchModeMulti  doubaoBatchMode = "multi"
)
```

Then update `NewProvider` so every constructed `DoubaoProvider` receives the normalized config mode. Keep fallback construction identical aside from passing this extra field through.

Do **not** add a new method to the generic `AIProvider` interface for this.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ai -run Doubao.*BatchMode -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/provider.go internal/ai/doubao_provider.go internal/ai/ai_test.go
git commit -m "feat: pass doubao batch mode into providers"
```

## Chunk 2: Provider-side batch semantics

### Task 4: Make single-item batch honor forced `multi` mode

**Files:**
- Modify: `internal/ai/doubao_batch_provider.go`
- Modify: `internal/ai/ai_test.go`
- Modify: `internal/ai/parse_batch_tags_test.go` (if needed)

- [ ] **Step 1: Write the failing test**

Add tests for both paths:

```go
func TestDoubaoProvider_GenerateTagsBatch_SingleRequestAutoModeFallsBackToSingle(t *testing.T) { ... }

func TestDoubaoProvider_GenerateTagsBatch_SingleRequestMultiModeBuildsBatchRequest(t *testing.T) {
    // start httptest server
    // assert request body uses batch content with numbered prompt
    // return response: "1: tag1,tag2"
    // assert result.Groups has one group with two tags
}
```

If current tests already cover the multi-image body shape, add only the single-request forced-batch regression.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai -run GenerateTagsBatch.*SingleRequest -count=1`
Expected: FAIL because current code always short-circuits single request to `GenerateTags`.

- [ ] **Step 3: Write minimal implementation**

In `internal/ai/doubao_batch_provider.go`:

- keep `len(requests) == 0` behavior unchanged
- change the `len(requests) == 1` shortcut so it only falls back to `GenerateTags` when mode is not `multi`
- when mode is `multi`, build the batch request and parse the numbered response exactly like a larger batch

Keep request/response handling DRY by extracting a helper if needed, e.g. `shouldForceBatchForSingleRequest()` or `generateBatch(ctx, requests)`.

If parsing coverage is missing for one-item numbered output, add or extend `parse_batch_tags_test.go` rather than duplicating parser logic tests inside provider tests.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ai -run GenerateTagsBatch.*SingleRequest -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/doubao_batch_provider.go internal/ai/ai_test.go internal/ai/parse_batch_tags_test.go
git commit -m "feat: honor forced doubao batch mode for single requests"
```

### Task 5: Keep fallback provider behavior consistent across modes

**Files:**
- Modify: `internal/ai/fallback_doubao_provider.go`
- Modify: `internal/ai/ai_test.go`

- [ ] **Step 1: Write the failing test**

Add a focused regression test such as:

```go
func TestFallbackDoubaoProvider_GenerateTagsBatch_PreservesForcedBatchModeAcrossFallbackClients(t *testing.T) {
    // first client in multi mode returns error
    // second client in multi mode returns numbered batch response for one request
    // assert second client path was GenerateTagsBatch, not GenerateTags
}
```

Use small fake clients if easier than spinning up multiple real providers.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai -run FallbackDoubaoProvider.*BatchMode -count=1`
Expected: FAIL if fallback clients do not preserve the same effective mode semantics.

- [ ] **Step 3: Write minimal implementation**

Keep `FallbackDoubaoProvider` API surface unchanged, but ensure all child clients are constructed with the same normalized mode from `NewProvider`. If tests use fakes, update them to model the intended distinction clearly.

Avoid adding policy branches to `FallbackDoubaoProvider` if construction-time propagation already solves the problem.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ai -run FallbackDoubaoProvider.*BatchMode -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/fallback_doubao_provider.go internal/ai/ai_test.go
git commit -m "test: preserve doubao batch mode across fallback models"
```

## Chunk 3: Worker routing and app wiring

### Task 6: Wire effective mode from app bootstrap into the AI tag handler

**Files:**
- Modify: `internal/app/bootstrap.go`
- Modify: `internal/worker/ai_tag_handler.go`
- Modify: `internal/worker/ai_tag_handler_test.go`

- [ ] **Step 1: Write the failing test**

Add/adjust a worker handler construction test that proves the registration path can supply a mode parameter without breaking existing handler registration. Example target:

```go
func TestRegisterBatchAITagHandler_RegistrationWithMode(t *testing.T) { ... }
```

The test should compile only once `RegisterBatchAITagHandler` / `NewBatchAITagJobHandler` accept the mode argument.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/worker -run RegisterBatchAITagHandler -count=1`
Expected: FAIL to compile or fail assertions because the constructor does not accept mode yet.

- [ ] **Step 3: Write minimal implementation**

Update signatures so the mode is injected explicitly during handler construction, for example:

```go
func RegisterBatchAITagHandler(..., batchMode string, ...) { ... }
func NewBatchAITagJobHandler(..., batchMode string, ...) JobFunc { ... }
```

Then update `internal/app/bootstrap.go` to pass `a.config.AI.DoubaoBatchMode` into the handler registration path.

Do **not** make the worker reach back into the global app config.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/worker -run RegisterBatchAITagHandler -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/bootstrap.go internal/worker/ai_tag_handler.go internal/worker/ai_tag_handler_test.go
git commit -m "feat: inject doubao batch mode into ai tag handler"
```

### Task 7: Implement `single|auto|multi` worker routing

**Files:**
- Modify: `internal/worker/ai_tag_handler.go`
- Modify: `internal/worker/ai_tag_handler_test.go`

- [ ] **Step 1: Write the failing test**

Add focused tests for the behavior matrix using a fake repo and fake AI client that records calls:

```go
func TestBatchAITagHandler_SingleModeDoesNotClaimExtraJobsAndUsesGenerateTags(t *testing.T) { ... }
func TestBatchAITagHandler_AutoModeSingleRequestUsesGenerateTags(t *testing.T) { ... }
func TestBatchAITagHandler_AutoModeMultiRequestUsesGenerateTagsBatch(t *testing.T) { ... }
func TestBatchAITagHandler_MultiModeSingleRequestUsesGenerateTagsBatch(t *testing.T) { ... }
```

Assertions should cover:

- whether `FindAndClaimReadyJobs` was called
- whether single-image or batch provider method was invoked
- whether observation / governance flow still completes successfully

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/worker -run BatchAITagHandler_.*Mode -count=1`
Expected: FAIL because current worker logic ignores explicit modes.

- [ ] **Step 3: Write minimal implementation**

In `internal/worker/ai_tag_handler.go`:

- add a small normalized internal mode helper for the worker
- in `single` mode, skip `repo.FindAndClaimReadyJobs(...)`
- in `auto`, keep current behavior
- in `multi`, keep claim behavior and always route to `GenerateTagsBatch`

Keep the rest of the function intact:

- payload parsing
- observation save
- governance merge
- mark complete / failed behavior

Avoid mixing config parsing into the worker; it should consume the injected, already-normalized mode.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/worker -run BatchAITagHandler_.*Mode -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/worker/ai_tag_handler.go internal/worker/ai_tag_handler_test.go
git commit -m "feat: route doubao ai tagging by explicit batch mode"
```

### Task 8: Run regression suites for config + AI + worker together

**Files:**
- Modify: any files changed above only if failures expose real regressions

- [ ] **Step 1: Run targeted package tests**

Run:

```bash
go test ./internal/config ./internal/ai ./internal/worker -count=1
```

Expected: PASS

- [ ] **Step 2: Fix only real regressions introduced by this work**

If failures appear:

- fix changed code only
- do not refactor unrelated behavior
- preserve scope boundaries from the spec

- [ ] **Step 3: Re-run targeted package tests**

Run:

```bash
go test ./internal/config ./internal/ai ./internal/worker -count=1
```

Expected: PASS

- [ ] **Step 4: Manual QA the new config semantics**

Use the smallest practical manual check:

1. Start from a config object or test fixture with `provider: doubao`
2. Verify `single` mode uses one-image path and no extra claim
3. Verify `auto` preserves current behavior
4. Verify `multi` forces `GenerateTagsBatch` even for one image

If no dedicated runtime harness exists, this can be satisfied by a narrow targeted test invocation plus captured logs/assertions in the worker tests. Do not invent a new manual QA tool for this plan.

- [ ] **Step 5: Commit**

```bash
git add internal/config internal/ai internal/worker internal/app deploy/config/config.example.yaml
git commit -m "feat: add doubao single multi switch"
```
