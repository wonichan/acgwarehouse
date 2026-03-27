## Phase 13-04 Summary

Implemented failed-task retry as create-a-new-batch semantics across the backend admin service, HTTP handlers, and the existing batch-first admin page.

### What changed
- Added retry-first admin service methods that create a fresh tracked batch from failed platform tasks instead of resetting old work.
- Locked retry behavior to `failed` tasks only; non-failed tasks are rejected.
- Added batch-level retry and single-task retry admin endpoints with explicit retry count and new batch ID responses.
- Extended the batch monitor UI with batch retry and task retry actions, plus success toasts that point users to the newly created batch.
- Preserved the existing batch-first page layout and kept destructive controls separate from retry controls.

### Checkpoint
- Human-verify checkpoint was treated as **approved** per explicit user auto-advance choice.

### Verification
- `go test ./internal/service/... -run "RetryFailed|RetryBatch|RetryTask" -count=1`
  - Result: passed
- `go test ./internal/handler/... -run "Retry" -count=1`
  - Result: passed
- `go test ./internal/service/... ./internal/handler/... -run "Retry|Cancel|Clear|PauseResume" -count=1`
  - Result: passed
- `go test ./internal/service/... ./internal/handler/... -count=1`
  - Result: passed
- `node --check web/admin/app.js`
  - Result: passed
- `lsp_diagnostics` on modified Go files: passed with only non-blocking hints
- `lsp_diagnostics` on `web/admin/app.js`: unavailable (`typescript-language-server` not installed)

### Notes
- Retry creates a new batch and queues fresh jobs; it does not revive old failed jobs in place.
- Unrelated workspace dirty files (`config.yaml`, untracked planning docs, `nul`, `server.exe`) were left untouched.
