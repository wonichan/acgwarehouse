---
phase: 260329-pnr-scannerservice-scan-queuetask
plan: 01
type: tdd
status: completed
completed_at: "2026-03-29T12:00:00.000Z"
duration: "5 min"
commit: c27f34a
---

# Summary: Thumbnail Task Queueing Fix

## Objective

Fix thumbnail task queueing in ScannerService.Scan(): add QueueTask() calls to convert PlatformTask records into AsyncJob records, enabling workers to process thumbnail generation tasks.

## Problem

ScannerService.Scan() created `thumbnail_generate` PlatformTask records but never called QueueTask() to create AsyncJob records. Tasks remained in "pending" status forever. Workers only process "ready" tasks from async_jobs table.

## Solution

Added QueueTask loop after PlanBatch(), following AITagAutoScheduler pattern:
1. Build imageID→path map from items array
2. For each created task, call QueueTask() with thumbnail_generate payload
3. Payload includes: image_id, path, filename (extracted from path)

## Implementation

### Files Modified

| File | Changes |
|------|---------|
| internal/service/scanner_service.go | Added QueueTask loop (lines 189-213) |
| internal/service/scanner_service_test.go | Added TestScannerServiceScanQueuesThumbnailTasks, updated existing test expectation |

### Code Added

```go
// Build imageID→path map for payload construction
imageIDToPath := make(map[int64]string, len(items))
for _, item := range items {
    imageIDToPath[item.ImageID] = item.SourceDescriptor
}

// Queue thumbnail tasks for worker processing
for i := range planResult.CreatedTasks {
    task := &planResult.CreatedTasks[i]
    path := imageIDToPath[task.ImageID]
    
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

## TDD Workflow

| Phase | Test | Result |
|-------|------|--------|
| RED | TestScannerServiceScanQueuesThumbnailTasks | FAIL: len(jobs) = 0, want 1 |
| GREEN | Added QueueTask loop | PASS: All scanner tests |
| Verify | Full service tests | PASS (except pre-existing failure) |

## Verification

### Must-Haves Verified

- ✅ After Scan() completes, thumbnail_generate tasks have AsyncJob records
- ✅ Tasks are in "queued" status, not stuck in "pending"
- ✅ Workers can pick up queued thumbnail tasks (status="ready")
- ✅ Payload contains image_id, path, filename

### Test Results

```
=== RUN   TestScannerServiceScanQueuesThumbnailTasks
--- PASS (0.32s)

=== RUN   TestScannerCreatesImportBatchAndPlatformTask
--- PASS (0.32s) [Updated expectation from jobs=0 to jobs=1]

=== RUN   TestScannerCreatesNewBatchButSkipsUnchangedImageTasks
--- PASS (0.33s)
```

## Notes

- Pre-existing test failure: TestAdminService_GetTaskPlatformOverview_ReturnsQueueAndPlatformCounts (NULL to string conversion issue) - unrelated to this fix
- Existing test TestScannerCreatesImportBatchAndPlatformTask was documenting the bug behavior (jobs=0), updated to expect correct behavior (jobs=1)

## Commit

```
c27f34a fix(scanner): queue thumbnail tasks after PlanBatch
```