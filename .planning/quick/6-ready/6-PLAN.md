# Quick Task 6: 添加任务加载逻辑

**任务编号:** 6  
**任务描述:** 执行方案三：添加任务加载逻辑，服务器启动时自动处理数据库中ready任务  
**创建日期:** 2026-03-19  
**状态:** Completed

---

## 任务清单

### 任务 1: 分析现有代码 ✅
**文件:**
- `internal/worker/job_manager.go`
- `cmd/server/main.go`
- `internal/repository/job_repository.go`

**动作:**
1. ✅ 已确认 Manager 结构体的 queue 是私有的，无法直接访问
2. ✅ 已确认需要添加一个方法将已有任务加载到队列
3. ✅ 已确认 `FindByStatus` 方法已存在，可以查询 ready 状态的任务

**完成标志:**
完成代码分析，确定需要添加 `LoadExistingJob` 方法

---

### 任务 2: 修改代码 ✅
**文件:**
- `internal/worker/job_manager.go` (已修改)
- `cmd/server/main.go` (已修改)

**动作:**

1. **修改 `internal/worker/job_manager.go`：**
   - 在 `AddJob` 方法后添加 `LoadExistingJob` 方法
   - 该方法直接将已有任务发送到队列，不创建新记录

2. **修改 `cmd/server/main.go`：**
   - 在 `jobManager.Start` 后启动 goroutine
   - 从数据库查询所有 `ready` 状态的任务
   - 使用 `LoadExistingJob` 方法将任务加载到队列
   - 添加日志输出便于调试

**验证:**
- [x] 代码可以编译通过
- [x] `LoadExistingJob` 方法正确实现
- [x] 任务加载逻辑完整

**完成标志:**
修改完成，代码编译通过

---

## 修改详情

### 修改的文件 1: `internal/worker/job_manager.go`

在 `AddJob` 方法后添加：

```go
// LoadExistingJob 将已有的任务加载到队列中（不创建新记录）
func (m *Manager) LoadExistingJob(job *domain.AsyncJob) bool {
	select {
	case m.queue <- job:
		return true
	default:
		return false
	}
}
```

### 修改的文件 2: `cmd/server/main.go`

在 `jobManager.Start` 后添加：

```go
// 加载数据库中所有 ready 状态的任务到队列
go func() {
	jobs, err := jobRepo.FindByStatus("ready")
	if err != nil {
		log.Printf("加载待处理任务失败: %v", err)
		return
	}
	if len(jobs) > 0 {
		log.Printf("发现 %d 个待处理任务，正在加载到队列...", len(jobs))
		loadedCount := 0
		skippedCount := 0
		for i := range jobs {
			job := &jobs[i]
			// 使用 LoadExistingJob 方法直接加载已有任务到队列
			if jobManager.LoadExistingJob(job) {
				loadedCount++
				log.Printf("已加载任务: %s #%d", job.Type, job.ID)
			} else {
				skippedCount++
				log.Printf("任务队列已满，跳过任务 #%d", job.ID)
			}
		}
		log.Printf("任务加载完成，已加载 %d 个，跳过 %d 个", loadedCount, skippedCount)
	}
}()
```

---

## 工作流程

修改后，服务器启动时：

1. **启动服务器**
   ```powershell
   go run cmd/server/main.go
   ```

2. **自动加载任务**
   - 查询数据库中所有 `ready` 状态的任务
   - 将任务加载到队列中
   - 输出日志："发现 X 个待处理任务，正在加载到队列..."

3. **自动处理任务**
   - JobManager 从队列中取出任务
   - 调用对应的处理器
   - 任务被处理完成

---

## 注意事项

1. **不创建重复记录**：`LoadExistingJob` 方法不创建新的数据库记录，直接复用已有任务
2. **队列满处理**：如果队列已满，会跳过任务并记录日志
3. **异步加载**：任务加载在单独的 goroutine 中执行，不阻塞服务器启动
4. **中文日志**：日志使用中文，便于调试

---

## 计划元数据

```yaml
must_haves:
  truths:
    - LoadExistingJob 方法正确实现
    - 任务加载逻辑完整
    - 不创建重复的任务记录
    - 代码可以编译通过
  artifacts:
    - internal/worker/job_manager.go (已修改)
    - cmd/server/main.go (已修改)
  key_links:
    - internal/worker/job_manager.go
    - cmd/server/main.go
    - internal/repository/job_repository.go
```

---

*计划创建时间: 2026-03-19*
