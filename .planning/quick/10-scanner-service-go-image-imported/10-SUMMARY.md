# Quick Task 10 Summary

**Task:** 在 scanner_service.go 添加去重检查，避免为已导入的图片重复创建 image_imported 任务  
**Date:** 2026-03-19  
**Status:** ✅ Completed

## 修改内容

### 1. 扩展 JobRepository 接口 (internal/repository/job_repository.go)

在 JobRepository 接口中添加了新方法：

```go
FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error)
```

并在 sqliteJobRepository 中实现了该方法：

```go
func (r *sqliteJobRepository) FindByTypeAndStatus(jobType string, status string) ([]domain.AsyncJob, error) {
	return r.findMany(`
		SELECT id, type, status, payload, progress, error, created_at, started_at, finished_at
		FROM async_jobs WHERE type = ? AND status = ? ORDER BY id DESC
	`, jobType, status)
}
```

### 2. 添加去重逻辑 (internal/service/scanner_service.go)

在 `importFile` 方法中创建 `image_imported` 任务前，添加了去重检查：

```go
// 去重检查：查询是否已存在针对该图片的 image_imported ready 任务
existingJobs, err := s.jobRepo.FindByTypeAndStatus("image_imported", "ready")
if err != nil {
    return fmt.Errorf("检查现有任务失败: %w", err)
}

// 检查是否已有相同 image_id 的待处理任务
for _, job := range existingJobs {
    var payloadData map[string]any
    if err := json.Unmarshal([]byte(job.Payload), &payloadData); err == nil {
        if existingImageID, ok := payloadData["image_id"].(float64); ok {
            if int64(existingImageID) == image.ID {
                // 已存在针对该图片的任务，跳过创建
                return nil
            }
        }
    }
}
```

## 解决的问题

- **问题：** 短时间内多次点击"触发扫描"会产生重复的 `image_imported` 任务，导致同一图片被多次处理缩略图
- **解决方案：** 在创建任务前检查是否已存在针对该图片的待处理任务，如果存在则跳过创建

## 验证

- ✅ `go build ./...` 编译成功
- ✅ 代码逻辑正确：检查 `image_imported` 类型的 `ready` 状态任务
- ✅ 正确解析 payload 比较 image_id
- ✅ 仅在不存在时才创建新任务

## 测试建议

建议添加以下测试场景：
1. 首次导入图片时正常创建 `image_imported` 任务
2. 重复导入相同图片时跳过任务创建
3. 不同图片导入时各自正常创建任务
4. 任务完成后（状态非 ready）再次导入相同图片时正常创建新任务

## 提交

Commit: [待生成]
