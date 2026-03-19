# Quick Task 8 Summary: Fix Task Loading Race Condition

**Completed:** 2026-03-20
**Commit:** 167e777

## Problem

服务器启动时存在竞态条件：
1. `jobManager.Start()` 启动后台消费者 goroutine
2. 任务加载在独立的 goroutine 中异步执行
3. 处理器在主线程中顺序注册

这导致当 `manual_scan` 等任务在处理器注册前被加载和执行时，出现 `no handler registered` 错误。

## Solution

将任务加载代码从第67-91行移动到所有处理器注册之后（第133-157行）。

### 修改后的执行顺序

```
1. jobManager.Start(context.Background())          ← 启动消费者
2. jobManager.RegisterHandler("thumbnail_generate", ...)  ← 注册缩略图处理器
3. jobManager.RegisterHandler("image_imported", ...)      ← 注册图片导入处理器
4. jobManager.RegisterHandler("manual_scan", ...)         ← 注册手动扫描处理器
5. registerAIWorker(jobManager, ...)              ← 注册AI标签处理器
6. go func() { load ready tasks... }()            ← 加载任务（在所有处理器之后）
```

## 验证

- [x] 代码可以正常编译
- [x] 执行顺序正确：Start → Register Handlers → Load Tasks
- [x] 所有处理器在任务加载前已注册

## 文件变更

- `cmd/server/main.go`: 移动任务加载 goroutine 到处理器注册之后

## 影响

修复后，服务器启动时会确保所有任务处理器都已注册，然后才加载数据库中的 `ready` 状态任务。这彻底解决了 `no handler registered` 错误。
