# Worker Pool Hot Reload Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 `worker_pool.worker_count` 热更新失效，让运行中的 ants pool 在配置变更后真实调整并发上限。

**Architecture:** 保留现有 `config reload -> bootstrap callback -> jobManager.SetWorkerCount -> pool.Tune` 链路，只移除与 `Tune` 冲突的 `ants.WithPreAlloc(true)`。通过一个新的并发回归测试证明热更新后的真实执行并发能够突破旧上限，并同步修正误导性日志文案。

**Tech Stack:** Go 1.25、ants v2.11.6、SQLite 测试仓储、Go testing

---

## File Map

- Modify: `internal/worker/job_manager.go`
  - 移除 `ants.WithPreAlloc(true)`
  - 调整 `SetWorkerCount` / `GetWorkerCount` 相关日志与注释文案，使其不再冒充“已验证真实生效值”
- Modify: `internal/worker/job_manager_test.go`
  - 保留现有 `TestManager_SetWorkerCount` 作为配置值 smoke test
  - 新增一个真实并发热更新回归测试，验证热更新后并发峰值突破旧上限
- Verify: `docs/superpowers/specs/2026-04-19-worker-pool-hot-reload-design.md`
  - 实现时对照范围，避免扩张到 DB/govips/MinIO

## Chunk 1: 并发回归测试

### Task 1: 为热更新失效写失败测试

**Files:**
- Modify: `internal/worker/job_manager_test.go`
- Reference: `docs/superpowers/specs/2026-04-19-worker-pool-hot-reload-design.md`

- [ ] **Step 1: 在 `job_manager_test.go` 中新增并发热更新测试骨架**

建议测试名：

```go
func TestManager_SetWorkerCountChangesActualConcurrency(t *testing.T) {
    t.Parallel()
}
```

测试结构要求：

- 用临时 SQLite DB + `repository.EnsureScanSchema`
- 创建 `NewManagerWithConfig(jobRepo, 1, 32)`，让初始 worker 很小
- 注册一个阻塞 handler：
  - 进入时原子递增当前并发
  - 更新峰值并发
  - 阻塞在 `releaseCh`
- 启动 manager 后，先调用 `mgr.SetWorkerCount(ctx, 4)`
- 再连续 `AddJob` 4 个以上任务
- 使用条件等待，断言峰值并发最终达到至少 4

- [ ] **Step 2: 运行单测，确认它先失败**

Run:

```powershell
go test ./internal/worker -run TestManager_SetWorkerCountChangesActualConcurrency -count=1
```

Expected:

- 当前代码下 FAIL
- 失败原因应是峰值并发仍停在旧值 1（或未达到期望 4）

- [ ] **Step 3: 如测试因时序不稳失败，先修测试设计，不修生产代码**

测试必须满足：

- 失败原因是“热更新未改变真实并发”
- 不是 sleep 太短、任务未入队、数据库未初始化等无关问题

推荐使用：

- `sync/atomic` 记录 `running` 和 `peak`
- `time.After` + 循环等待条件，而不是单纯固定 `Sleep`
- `defer close(releaseCh)` 或在断言前后明确释放阻塞任务，避免卡死 `mgr.Stop()`

- [ ] **Step 4: 保留现有 `TestManager_SetWorkerCount`，不要删除**

原因：

- 该测试仍可验证“配置层面数值更新”
- 新测试负责验证“真实并发变化”

- [ ] **Step 5: 提交测试阶段改动**

```bash
git add internal/worker/job_manager_test.go
git commit -m "test: cover worker pool hot reload concurrency"
```

## Chunk 2: 最小实现修复

### Task 2: 移除 PreAlloc，恢复 Tune 生效

**Files:**
- Modify: `internal/worker/job_manager.go`
- Test: `internal/worker/job_manager_test.go`

- [ ] **Step 1: 修改 pool 创建参数，删除 `ants.WithPreAlloc(true)`**

当前位置：`internal/worker/job_manager.go` 的 `Start()` 中 `ants.NewPool(...)`。

修改目标：

```go
pool, err := ants.NewPool(
    m.GetWorkerCount(),
    ants.WithPanicHandler(func(i interface{}) {
        logger.Errorf("[ANTS PANIC] task panicked: %v", i)
    }),
    ants.WithExpiryDuration(10*time.Minute),
)
```

不要改动：

- `dispatchLoop`
- `submitJob`
- `Stop()`
- queue/refill 逻辑

- [ ] **Step 2: 修正文案与注释，避免误导**

至少处理以下问题：

- `SetWorkerCount` 中“Worker 数量已调整为 X”这类表述应改得更准确
- `GetWorkerCount` 注释里“返回配置值（更准确，因为 ants.Tune 是异步的）”需要重写，避免把配置值说成真实执行并发

可接受方向：

- 明确这是“当前目标 worker 数/配置中的生效目标值”
- 日志写成“Worker 调优目标已更新为 X”或同等不误导的表达

- [ ] **Step 3: 运行新增单测，确认由红转绿**

Run:

```powershell
go test ./internal/worker -run TestManager_SetWorkerCountChangesActualConcurrency -count=1
```

Expected:

- PASS
- 峰值并发达到新上限附近，并突破旧上限

- [ ] **Step 4: 运行现有 smoke test，确认无回归**

Run:

```powershell
go test ./internal/worker -run ^TestManager_SetWorkerCount$ -count=1
```

Expected:

- PASS

- [ ] **Step 5: 提交最小实现修复**

```bash
git add internal/worker/job_manager.go internal/worker/job_manager_test.go
git commit -m "fix: restore worker pool hot reload"
```

## Chunk 3: 回归验证

### Task 3: 跑相关 worker 测试并记录结果

**Files:**
- Verify: `internal/worker/job_manager_test.go`

- [ ] **Step 1: 运行 worker 包测试**

Run:

```powershell
go test ./internal/worker -count=1
```

Expected:

- 全部 PASS
- 没有新增死锁/卡住/超时

- [ ] **Step 2: 如有并发测试偶发失败，优先修测试稳定性**

只允许修：

- 条件等待
- channel 收敛
- 原子峰值统计

不允许借机扩大实现范围到 DB/govips/MinIO。

- [ ] **Step 3: 人工复核代码范围**

确认最终 diff 只聚焦于：

- `job_manager.go`
- `job_manager_test.go`
- 必要的计划/设计文档（如果执行时同步更新）

- [ ] **Step 4: 如用户要求，再做运行时手工验证**

可选验证路径：

- 启动服务
- 初始 `worker_count=10`
- 热更新到 48
- 观察缩略图/任务日志是否能突破旧上限 10

这一步不是本轮代码修复的硬性 gate，但适合作为后续现实反馈。

- [ ] **Step 5: 仅在验证阶段发现并修复新问题时再提交**

如果 Chunk 3 只是验证且没有新增代码改动，则此步跳过，不创建空提交。

如果验证阶段为了修复测试稳定性或小范围实现问题产生了新改动，再执行：

```bash
git add internal/worker/job_manager.go internal/worker/job_manager_test.go
git commit -m "test: stabilize worker hot reload verification"
```
