---
phase: quick
plan: 11
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/worker/job_manager.go
  - internal/handler/ai_tag_handler_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - "When a job finishes, progress displays as 100% in the admin dashboard"
    - "When a job is 50% complete, progress displays as 50% in the admin dashboard"
  artifacts:
    - path: "internal/worker/job_manager.go"
      provides: "Job completion with correct progress value"
    - path: "internal/handler/ai_tag_handler_test.go"
      provides: "Test with correct progress expectation"
  key_links:
    - from: "internal/worker/job_manager.go"
      to: "web/admin/app.js"
      via: "API JSON response"
      pattern: "Progress: 100 for finished jobs"
---

<objective>
Fix task status FINISHED showing progress as 1% instead of 100% in admin dashboard.

Purpose: Ensure consistent progress reporting between backend and frontend.
Output: Jobs show correct progress percentage (0-100 range).
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

## Bug Analysis

**Root Cause:**
- Backend sets `job.Progress = 1` when job finishes (intending 1 = 100% in 0-1 range)
- Frontend displays `${job.progress || 0}%` directly (expecting 0-100 range)
- Result: Finished jobs show `1%` instead of `100%`

**Decision:**
Change backend to use 0-100 range to match frontend expectation. This is cleaner than changing frontend to multiply by 100, as it makes progress values human-readable in the database and API responses.

## Current Code

From `internal/worker/job_manager.go`:
```go
} else {
    job.Status = "finished"
    job.Progress = 1  // BUG: Should be 100
    duration := time.Since(started)
    log.Printf("任务 %s #%d 执行完成，耗时: %.2f秒", job.Type, job.ID, duration.Seconds())
}
```

From `web/admin/app.js`:
```javascript
<td>${job.progress || 0}%</td>  // Expects 0-100 range
```

From `internal/handler/ai_tag_handler_test.go`:
```go
job.Progress = 0.5  // BUG: Should be 50 to match 0-100 range
```
</context>

<tasks>

<task type="auto" tdd="true">
  <name>task 1: Write failing test for progress value</name>
  <files>internal/worker/job_manager_test.go</files>
  <behavior>
    - Test: When job finishes successfully, Progress should be 100.0
    - Test: When job starts, Progress should be 0
  </behavior>
  <action>
    Add test cases to `internal/worker/job_manager_test.go` to verify:
    1. Job progress is 0 when created
    2. Job progress is 100.0 when finished successfully

    The test should fail with current code (progress = 1) and pass after fix.
  </action>
  <verify>
    <automated>go test ./internal/worker/... -run TestProgress -v</automated>
  </verify>
  <done>Test exists and fails with current implementation</done>
</task>

<task type="auto" tdd="true">
  <name>task 2: Fix progress value in job_manager.go</name>
  <files>internal/worker/job_manager.go</files>
  <behavior>
    - When job finishes successfully: Progress = 100.0
    - When job fails: Progress remains unchanged (no explicit change needed)
  </behavior>
  <action>
    In `internal/worker/job_manager.go`, line ~159, change:
    ```go
    job.Progress = 1
    ```
    to:
    ```go
    job.Progress = 100
    ```

    This ensures progress is in 0-100 range, matching the frontend expectation.
  </action>
  <verify>
    <automated>go test ./internal/worker/... -run TestProgress -v</automated>
  </verify>
  <done>Progress test passes, finished jobs have Progress = 100</done>
</task>

<task type="auto">
  <name>task 3: Fix test file using old progress convention</name>
  <files>internal/handler/ai_tag_handler_test.go</files>
  <action>
    In `internal/handler/ai_tag_handler_test.go`, find the line:
    ```go
    job.Progress = 0.5
    ```
    and change it to:
    ```go
    job.Progress = 50
    ```

    This aligns the test with the new 0-100 progress range convention.
  </action>
  <verify>
    <automated>go test ./internal/handler/... -run TestGetAITagStatus -v</automated>
  </verify>
  <done>Test file uses 0-100 progress range</done>
</task>

</tasks>

<verification>
1. Run all tests: `go test ./...`
2. Start server and trigger a scan job via admin dashboard
3. Verify finished job shows 100% progress in admin dashboard
</verification>

<success_criteria>
- All tests pass
- Job progress shows 100% when status is "finished"
- No regression in existing functionality
</success_criteria>

<output>
After completion, create `.planning/quick/11-fix-task-status-finished-but-progress-sh/11-SUMMARY.md`
</output>