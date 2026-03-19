# Quick Task 8: Fix Task Loading Race Condition

**Goal:** 修复任务加载竞态条件，确保所有处理器注册完成后再加载数据库中的 ready 任务

**Problem:** 当前代码在 goroutine 中异步加载任务，但处理器在主线程中顺序注册。如果任务加载发生在处理器注册之前，就会出现 `no handler registered` 错误。

**Solution:** 将所有处理器注册完成后，再启动任务加载 goroutine。

---

## Task: Fix Race Condition in main.go

**Files:**
- Modify: `cmd/server/main.go:63-150`

### Current Code Order (有问题的)
```
63: jobManager.Start(ctx)                          ← 启动消费者
64: go func() { load ready tasks... }()             ← 异步加载任务（问题！）
...
148: jobManager.RegisterHandler("manual_scan", ...) ← 处理器在这里注册
```

### Fixed Code Order
```
63: jobManager.Start(ctx)                          ← 启动消费者
...
148: jobManager.RegisterHandler("manual_scan", ...) ← 先注册所有处理器
...
[然后] go func() { load ready tasks... }()         ← 再加载任务
```

---

- [ ] **Step 1: 修改 main.go - 移动任务加载到处理器注册之后**

将第67-91行的任务加载 goroutine 移动到所有 RegisterHandler 调用之后（约第150行之后）。

具体修改：
1. 删除第67-91行的任务加载代码
2. 在 `registerAIWorker` 调用之后（约第157行后）添加任务加载代码
3. 确保此时所有处理器都已注册：
   - thumbnail_generate ✓
   - image_imported ✓
   - manual_scan ✓
   - ai_tag_generation ✓

修改后的代码结构：
```go
// 1. 启动 Job Manager
jobManager.Start(context.Background())
defer jobManager.Stop()

// 2. 注册所有处理器
jobManager.RegisterHandler("thumbnail_generate", ...)
jobManager.RegisterHandler("image_imported", ...)
jobManager.RegisterHandler("manual_scan", ...)
registerAIWorker(...) // ai_tag_generation

// 3. 最后加载任务（确保所有处理器已注册）
go func() {
    jobs, err := jobRepo.FindByStatus("ready")
    // ... 任务加载逻辑
}()
```

- [ ] **Step 2: 验证修改**

检查代码顺序：
1. `jobManager.Start()` 在最前面
2. 所有 `RegisterHandler` 调用在中间
3. 任务加载 goroutine 在最后

- [ ] **Step 3: 提交**

```bash
git add cmd/server/main.go
git commit -m "fix: resolve race condition in task loading

Move task loading goroutine after all handler registrations
to prevent 'no handler registered' errors for pending jobs.

The issue occurred when ready tasks were loaded before
handlers were fully registered, causing job execution to fail."
```

---

## 验证清单

- [ ] `jobManager.Start()` 在处理器注册之前
- [ ] 所有处理器注册在任务加载之前
- [ ] 任务加载 goroutine 在所有处理器注册之后
- [ ] 代码可以正常编译
- [ ] 启动服务器后，任务能正常被处理
