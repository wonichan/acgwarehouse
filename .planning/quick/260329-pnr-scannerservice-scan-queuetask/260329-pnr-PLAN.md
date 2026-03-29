---
phase: 260329-pnr-scannerservice-scan-queuetask
plan: 01
type: tdd
wave: 1
depends_on: []
files_modified:
  - internal/service/scanner_service.go
  - internal/service/scanner_service_test.go
autonomous: true
requirements:
  - THUMBNAIL-QUEUE-01
user_setup: []

must_haves:
  truths:
    - "After Scan() completes, thumbnail_generate tasks have AsyncJob records"
    - "Tasks are in 'ready' status, not stuck in 'pending'"
    - "Worker can pick up queued thumbnail tasks"
  artifacts:
    - path: "internal/service/scanner_service.go"
      contains: "QueueTask"
      min_lines: 1
    - path: "internal/service/scanner_service_test.go"
      contains: "TestScannerServiceScanQueuesThumbnailTasks"
      exports: ["TestScannerServiceScanQueuesThumbnailTasks"]
  key_links:
    - from: "internal/service/scanner_service.go::Scan()"
      to: "s.taskSvc.QueueTask()"
      via: "loop over planResult.CreatedTasks"
      pattern: "QueueTask.*thumbnail_generate"
---

<objective>
Fix thumbnail task queueing in ScannerService.Scan(): add QueueTask() calls to convert PlatformTask records into AsyncJob records, enabling workers to process thumbnail generation tasks.

**Purpose:** Thumbnail tasks created during import scan never get queued, remaining in 'pending' status forever. Workers only process 'ready' tasks from async_jobs table.

**Output:** Working thumbnail task queueing with test coverage.
</objective>

<execution_context>
@D:/production/goworkspace/ACGWarehouse/.opencode/get-shit-done/workflows/execute-plan.md
@D:/production/goworkspace/ACGWarehouse/.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# Reference pattern from AITagAutoScheduler (lines 153-169):
```go
for i := range plan.CreatedTasks {
    image, ok := imagesByID[plan.CreatedTasks[i].ImageID]
    if !ok {
        continue
    }
    payload, err := json.Marshal(map[string]any{
        "image_id": image.ID,
        "path":     image.Path,
    })
    if err != nil {
        return queued, fmt.Errorf("marshal AI tag payload: %w", err)
    }
    if _, err := s.taskPlatform.QueueTask(ctx, &plan.CreatedTasks[i], domain.PlatformTaskTypeAITagGeneration, string(payload)); err != nil {
        return queued, fmt.Errorf("queue AI tag task for image %d: %w", image.ID, err)
    }
    queued++
}
```

# Current ScannerService.Scan() structure (lines 165-189):
- Calls PlanBatch() ✓
- Gets planResult.CreatedTasks ✓
- Collects CreatedPlatformTaskIDs ✓
- **MISSING: QueueTask() calls for each created task**
</context>

<interfaces>
<!-- Key types and contracts the executor needs -->

From internal/domain/platform_task.go:
```go
type PlatformTask struct {
    ID       int64  `json:"id" db:"id"`
    ImageID  int64  `json:"image_id" db:"image_id"`
    TaskType string `json:"task_type" db:"task_type"`
    Status   string `json:"status" db:"status"`
}
const PlatformTaskTypeThumbnailGenerate = "thumbnail_generate"
```

From internal/service/task_platform_service.go:
```go
func (s *TaskPlatformService) QueueTask(ctx context.Context, task *domain.PlatformTask, jobType, payload string) (*domain.AsyncJob, error)
```

From internal/service/scanner_service.go (TaskPlatformPlanItem):
```go
type TaskPlatformPlanItem struct {
    ImageID          int64
    SourceDescriptor string  // <-- This is the path we need for payload
}
```

Payload structure (from importFile line 222-226):
```json
{
    "image_id": int64,
    "path": string,
    "filename": string  // extracted from path, without extension
}
```
</interfaces>

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1: Write failing test | None | TDD RED phase - test defines expected behavior |
| Task 2: Implement QueueTask calls | Task 1 | TDD GREEN phase - make test pass |
| Task 3: Verify integration | Task 2 | Final verification after implementation |

## Parallel Execution Graph

**Wave 1 (Sequential - TDD workflow):**
```
Task 1 (RED) → Task 2 (GREEN) → Task 3 (Verify)
```

**Critical Path:** Task 1 → Task 2 → Task 3
**TDD workflow is inherently sequential - cannot parallelize RED→GREEN→REFACTOR**

---

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write failing test for thumbnail task queueing</name>
  <files>internal/service/scanner_service_test.go</files>
  <behavior>
    - Test: After Scan() with new images, planResult.CreatedTasks should have corresponding AsyncJob records
    - Test: Each AsyncJob should have PlatformTaskID linked to the created task
    - Test: AsyncJob.Type should be "thumbnail_generate"
    - Test: AsyncJob.Status should be "ready" (not pending)
    - Test: Payload should contain image_id, path, filename
  </behavior>
  <action>
Create test `TestScannerServiceScanQueuesThumbnailTasks` in scanner_service_test.go:

1. Setup test environment with:
   - Mock imageRepo that returns isNew=true for test images
   - Real TaskPlatformService with in-memory repositories
   - Test images with known paths (e.g., "/test/image.png")

2. Call Scan() with test root directory

3. Verify expectations:
   ```go
   // After Scan, check that AsyncJobs were created
   jobs, err := jobRepo.FindByStatus(ctx, "ready", 100)
   assert.NoError(t, err)
   assert.Len(t, jobs, len(result.CreatedPlatformTaskIDs))
   
   for _, job := range jobs {
       assert.Equal(t, "thumbnail_generate", job.Type)
       assert.Equal(t, "ready", job.Status)
       assert.NotNil(t, job.PlatformTaskID)
       
       // Verify payload structure
       var payload map[string]any
       json.Unmarshal([]byte(job.Payload), &payload)
       assert.Contains(t, payload, "image_id")
       assert.Contains(t, payload, "path")
       assert.Contains(t, payload, "filename")
   }
   ```

4. Run test - EXPECT FAILURE (current code doesn't call QueueTask)
  </action>
  <verify>
    <automated>go test ./internal/service/... -run TestScannerServiceScanQueuesThumbnailTasks -v</automated>
    Expected: Test FAILS with "expected X jobs, got 0" or similar
  </verify>
  <done>Test exists and FAILS, demonstrating the bug exists</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement QueueTask calls in ScannerService.Scan()</name>
  <files>internal/service/scanner_service.go</files>
  <behavior>
    - After PlanBatch(), iterate through planResult.CreatedTasks
    - Build imageID→path map from items array
    - For each task, call QueueTask() with thumbnail_generate payload
    - Return error if any QueueTask fails
  </behavior>
  <action>
Modify ScannerService.Scan() after line 189 (after collecting CreatedPlatformTaskIDs):

1. Build imageID→itemInfo map from items array:
   ```go
   // Build map for payload construction
   imageIDToPath := make(map[int64]string, len(items))
   for _, item := range items {
       imageIDToPath[item.ImageID] = item.SourceDescriptor
   }
   ```

2. Queue each created task (follow AITagAutoScheduler pattern):
   ```go
   // Queue thumbnail tasks for worker processing
   for i := range planResult.CreatedTasks {
       task := &planResult.CreatedTasks[i]
       path := imageIDToPath[task.ImageID]
       
       // Extract filename without extension
       filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
       
       payload, err := json.Marshal(map[string]any{
           "image_id": task.ImageID,
           "path":     path,
           "filename": filename,
       })
       if err != nil {
           return nil, fmt.Errorf("marshal thumbnail payload for task %d: %w", task.ID, err)
       }
       
       if _, err := s.taskSvc.QueueTask(ctx, task, domain.PlatformTaskTypeThumbnailGenerate, string(payload)); err != nil {
           return nil, fmt.Errorf("queue thumbnail task %d: %w", task.ID, err)
       }
   }
   ```

3. Add required imports if missing: `encoding/json`, `strings`, `filepath`

4. Run test - EXPECT PASS
  </action>
  <verify>
    <automated>go test ./internal/service/... -run TestScannerServiceScanQueuesThumbnailTasks -v</automated>
    Expected: Test PASSES
  </verify>
  <done>Test passes, QueueTask calls added to Scan()</done>
</task>

<task type="auto">
  <name>Task 3: Verify full service test suite and edge cases</name>
  <files>internal/service/scanner_service_test.go</files>
  <action>
1. Run full scanner_service test suite to ensure no regression:
   ```bash
   go test ./internal/service/... -run "Scanner" -v
   ```

2. Add edge case test: Scan with taskSvc=nil (should not queue, should use legacy job path)
   - Verify existing behavior preserved for non-platform mode

3. Run all service tests to verify no side effects:
   ```bash
   go test ./internal/service/... -v
   ```
  </action>
  <verify>
    <automated>go test ./internal/service/... -v</automated>
    Expected: All tests PASS
  </verify>
  <done>All scanner_service tests pass, no regressions introduced</done>
</task>

</tasks>

<verification>
1. Test TestScannerServiceScanQueuesThumbnailTasks passes
2. Test verifies AsyncJob records created with "ready" status
3. Test verifies payload contains image_id, path, filename
4. No regression in existing scanner_service tests
5. Edge case: taskSvc=nil preserves legacy behavior
</verification>

<success_criteria>
- [ ] Test file exists with TestScannerServiceScanQueuesThumbnailTasks
- [ ] Test passes (AsyncJob records created after Scan)
- [ ] ScannerService.Scan() contains QueueTask() loop
- [ ] Payload structure matches: {image_id, path, filename}
- [ ] All existing scanner_service tests still pass
- [ ] No regression in task_platform_service tests
</success_criteria>

<output>
After completion, create `.planning/quick/260329-pnr-scannerservice-scan-queuetask/260329-pnr-SUMMARY.md`
</output>

---

## Commit Strategy

**Atomic commits following TDD workflow:**

1. **RED commit** (after Task 1):
   ```bash
   git add internal/service/scanner_service_test.go
   git commit -m "test(scanner): add failing test for thumbnail task queueing"
   ```

2. **GREEN commit** (after Task 2):
   ```bash
   git add internal/service/scanner_service.go
   git commit -m "fix(scanner): queue thumbnail tasks after PlanBatch"
   ```

3. **Final commit** (after Task 3, if test changes):
   ```bash
   git add internal/service/scanner_service_test.go
   git commit -m "test(scanner): add edge case for taskSvc=nil"
   ```

---

## Category + Skills Recommendations

### Task 1: Write failing test for thumbnail task queueing

**Delegation Recommendation:**
- Category: `quick` - Single test file creation, straightforward TDD setup
- Skills: [`test-driven-development`] - TDD workflow for writing failing test first

**Skills Evaluation:**
- ✅ INCLUDED `test-driven-development`: Core TDD workflow - RED phase
- ❌ OMITTED `systematic-debugging`: Not debugging, creating new test
- ❌ OMITTED `brainstorming`: Clear requirement, no design exploration needed

### Task 2: Implement QueueTask calls in ScannerService.Scan()

**Delegation Recommendation:**
- Category: `quick` - Small code addition following established pattern
- Skills: [`test-driven-development`] - TDD workflow for making test pass

**Skills Evaluation:**
- ✅ INCLUDED `test-driven-development`: TDD GREEN phase
- ❌ OMITTED `systematic-debugging`: Not debugging, implementing solution
- ❌ OMITTED `git-master`: Simple atomic commit, standard git add/commit

### Task 3: Verify full service test suite and edge cases

**Delegation Recommendation:**
- Category: `quick` - Running tests and adding edge case
- Skills: [`verification-before-completion`] - Ensure all tests pass before claiming done

**Skills Evaluation:**
- ✅ INCLUDED `verification-before-completion`: Must verify all tests pass
- ❌ OMITTED `systematic-debugging`: Not debugging unless tests fail
- ❌ OMITTED `requesting-code-review`: Simple fix, self-verification sufficient

---

## TODO List (ADD THESE)

> CALLER: Add these TODOs using TodoWrite/TaskCreate and execute sequentially (TDD workflow).

### Wave 1 (Sequential - TDD RED→GREEN)

- [ ] **1. Write failing test for thumbnail task queueing**
  - What: Create TestScannerServiceScanQueuesThumbnailTasks that expects AsyncJob records after Scan()
  - Depends: None
  - Blocks: 2
  - Category: `quick`
  - Skills: [`test-driven-development`]
  - QA: `go test ./internal/service/... -run TestScannerServiceScanQueuesThumbnailTasks -v` FAILS

- [ ] **2. Implement QueueTask calls in ScannerService.Scan()**
  - What: Add QueueTask loop after PlanBatch, build imageID→path map, marshal payload
  - Depends: 1
  - Blocks: 3
  - Category: `quick`
  - Skills: [`test-driven-development`]
  - QA: `go test ./internal/service/... -run TestScannerServiceScanQueuesThumbnailTasks -v` PASSES

- [ ] **3. Verify full service test suite and edge cases**
  - What: Run all scanner tests, add taskSvc=nil edge case, verify no regression
  - Depends: 2
  - Blocks: None
  - Category: `quick`
  - Skills: [`verification-before-completion`]
  - QA: `go test ./internal/service/... -v` ALL PASS

### Execution Instructions

TDD workflow is sequential - execute one task at a time:

```
# RED phase
task(category="quick", load_skills=["test-driven-development"], run_in_background=false, prompt="Task 1: Write failing test...")

# GREEN phase (after test fails)
task(category="quick", load_skills=["test-driven-development"], run_in_background=false, prompt="Task 2: Implement QueueTask...")

# Verify phase (after test passes)
task(category="quick", load_skills=["verification-before-completion"], run_in_background=false, prompt="Task 3: Verify full test suite...")
```