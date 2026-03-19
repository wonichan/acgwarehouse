# Quick Task 11 Summary: Fix Task Progress Display Bug

**Date:** 2026-03-19
**Status:** ✅ Completed

## Problem

后台管理页面的"最近任务"一栏中，任务状态显示为 FINISHED，但进度显示为 1% 而不是预期的 100%。

## Root Cause

- **后端** (`job_manager.go`): 设置 `job.Progress = 1` (意图使用 0-1 范围，其中 1 = 100%)
- **前端** (`web/admin/app.js`): 直接显示 `${job.progress || 0}%` (期望 0-100 范围)
- **结果**: 完成的任务显示 `1%` 而不是 `100%`

## Changes Made

### 1. Fixed Progress Value in job_manager.go
**File:** `internal/worker/job_manager.go` (line 159)
```go
// Before:
job.Progress = 1

// After:
job.Progress = 100
```

### 2. Updated Test File Progress Values
**File:** `internal/handler/ai_tag_handler_test.go`
```go
// Lines 61, 87-88:
// Before:
job.Progress = 0.5
// ...
if resp.Progress != 0.5 {
    t.Fatalf("progress = %f, want 0.5", resp.Progress)
}

// After:
job.Progress = 50
// ...
if resp.Progress != 50 {
    t.Fatalf("progress = %f, want 50", resp.Progress)
}
```

### 3. Added Progress Verification Test
**File:** `internal/worker/job_manager_test.go`
- Added assertions in `TestManagerProcessesJobsSequentially` to verify:
  - Finished jobs have `Progress = 100`

### 4. Fixed Pre-existing Mock Issue
**File:** `internal/worker/ai_tag_handler_test.go`
- Added missing `FindByTypeAndStatus` method to `mockJobRepoForAI`

## Verification

```bash
$ go test ./internal/worker/... ./internal/handler/... -v -count=1
ok      github.com/wonichan/acgwarehouse-backend/internal/worker     0.972s
ok      github.com/wonichan/acgwarehouse-backend/internal/handler    0.611s
```

All tests pass. The fix ensures that:
1. ✅ Finished jobs now show 100% progress in admin dashboard
2. ✅ Progress values use consistent 0-100 range across backend and frontend
3. ✅ No regression in existing functionality

## Commit

```
fix(progress): change progress range from 0-1 to 0-100

- Fix job_manager.go to set Progress = 100 for finished jobs
- Update ai_tag_handler_test.go to use 50 instead of 0.5
- Add test for progress value in job_manager_test.go
- Fix missing FindByTypeAndStatus in mock

Fixes: Task status FINISHED but progress shows 1%
```
