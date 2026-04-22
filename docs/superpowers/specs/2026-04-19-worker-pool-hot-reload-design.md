## 背景

当前后台任务池使用 `github.com/panjf2000/ants/v2`。系统支持在配置热更新时调整 `worker_pool.worker_count`，调用链路为：

- `internal/config` 监听配置文件变化
- `internal/app/bootstrap.go` 在配置变化回调中调用 `jobManager.SetWorkerCount(...)`
- `internal/worker/job_manager.go` 在 `SetWorkerCount` 中执行 `pool.Tune(newCount)`

运行日志显示配置热更新后出现了“Worker 数量已调整为 48”，但缩略图任务的实际并发仍长期停留在 10 左右。

根因已确认：`job_manager.go` 在创建 ants pool 时启用了 `ants.WithPreAlloc(true)`，而 ants 的 `Pool.Tune` 在 pre-alloc 模式下不会生效。因此系统当前只更新了“意图值/日志”，没有真正改变底层 pool 的容量。

## 目标

本次修复目标限定为：

1. 让 `worker_pool.worker_count` 的热更新真实作用到底层 ants pool
2. 保持现有任务队列、dispatch、refill 行为不变
3. 补充测试，防止未来再次出现“日志显示已生效、实际未生效”的回归
4. 改善日志/观测文案，减少“配置值冒充真实生效值”的误导

非目标：

- 不在本次修复中处理缩略图链路的 CPU / govips / SQLite / MinIO 吞吐瓶颈
- 不重构任务管理器整体架构
- 不实现 queue 容量的运行时热扩容

## 备选方案

### 方案 A：移除 `WithPreAlloc(true)`，保留 `Tune`

做法：

- 在 `internal/worker/job_manager.go` 中删除 `ants.WithPreAlloc(true)`
- 保留现有 `SetWorkerCount -> pool.Tune(newCount)` 热更新路径
- 保留现有 `workerCount` 原子字段作为当前目标 worker 数的来源

优点：

- 改动最小，最聚焦当前 bug
- 不引入 pool 重建、在途任务迁移、切换窗口等额外复杂度
- 与现有架构兼容，风险最小

缺点：

- 失去 pre-allocation 的内存预分配优化
- `GetWorkerCount()` 仍主要反映“当前设定值”，不是 ants 内部的绝对真实指标

### 方案 B：worker 数变化时重建 pool

做法：

- 当 `worker_count` 变化时释放旧 pool，创建新 pool
- 需要设计在途任务如何收敛、dispatch 如何切换

优点：

- 语义最明确，不依赖 `Tune`
- 即使未来再次启用 pre-allocation，也不会失效

缺点：

- 改动面大，风险高
- 需要处理切换时序、任务提交失败、竞态等问题
- 超出本次“最小修复 bug”的范围

### 方案 C：保留 `PreAlloc`，禁用热更新

做法：

- 维持当前 pool 创建方式
- 检测到运行时调整 worker 数时拒绝执行，并提示重启生效

优点：

- 实现简单，行为稳定

缺点：

- 不满足用户目标
- 等于承认现有热更新能力是伪能力

## 推荐方案

推荐采用 **方案 A**：移除 `ants.WithPreAlloc(true)`，保留 `Tune`。

原因：

1. 根因直接且明确：`PreAlloc` 与 `Tune` 冲突
2. 修复目标是“恢复热更新”，不是重建整个任务调度体系
3. 保留现有调用链路，最大限度降低回归风险

## 设计细节

### 1. Pool 创建策略

文件：`internal/worker/job_manager.go`

当前实现：

- 创建 pool 时指定 `ants.WithPreAlloc(true)`
- 同时设置 panic handler 与 expiry duration

调整后：

- 删除 `ants.WithPreAlloc(true)`
- 保留 panic handler 与 expiry duration

这样可以恢复 `pool.Tune(newCount)` 的有效性。

### 2. 热更新路径

不改变以下链路：

- `config.NewReloader(...)`
- `bootstrap.go` 中的配置更新回调
- `jobManager.SetWorkerCount(...)`

`SetWorkerCount` 仍承担两件事：

1. 对运行中的 pool 调用 `Tune(newCount)`
2. 更新内部保存的 worker 目标值

这保证现有外部接口和调用点不需要改变。

### 3. 观测与日志

需要减少“日志表述先于真实生效”的误导。

建议：

- 将 `Worker 数量已调整为 %d` 这类日志文案改为更准确的“已请求调整 / 已调优目标值”为主的表达，或至少明确这是 tune 操作结果而不是 pre-verified 的真实活跃数
- 启动日志与热更新日志保持语义一致：一个表示初始配置值，一个表示运行时调优目标值

本次不强制新增复杂指标，但至少要避免再次制造“已生效”的错觉。

### 4. 测试设计

必须使用 TDD，先写失败测试。

核心回归测试目标：

1. 启动时 worker 数为较小值（例如 1 或 2）
2. 在 manager 启动后调用 `SetWorkerCount(更大值)`
3. 提交足够多的阻塞任务
4. 断言可同时进入执行区的任务数能突破旧上限，证明热更新真的作用到 pool

推荐测试形态：

- 在 `internal/worker/job_manager_test.go` 中新增一个并发测试
- 用 channel 阻塞任务体，统计同时进入 handler 的任务数
- 在热更新前后比较峰值，或直接验证热更新后峰值不再被旧值卡死

注意：

- 测试应尽量减少时间敏感性，优先使用 channel、原子计数、条件等待
- 避免仅断言 `GetWorkerCount()` 返回新值，因为这正是当前 bug 中的“假成功”来源

## 风险与边界

### 风险

1. 去掉 pre-allocation 后，pool 的内存行为可能变化
2. 并发测试如果写得过于依赖 sleep，容易产生偶发失败

### 缓解

1. 本次只移除一项 ants 选项，不改队列和任务处理逻辑
2. 使用阻塞任务 + 条件等待构造确定性的并发测试
3. 保留现有测试并回归执行，确保行为稳定

## 验收标准

满足以下条件即可视为本次设计完成：

1. 修改 `worker_pool.worker_count` 并触发热更新后，底层 ants pool 的可执行并发上限真实变化
2. 新增测试在修复前失败、修复后通过
3. 现有任务管理器测试不回归
4. 运行日志不再把“配置值更新”误表述成“真实并发已验证生效”

## 后续工作（不在本次实现范围内）

如果修复后缩略图吞吐仍明显低于新的 worker 数，应继续单独审判：

- `govips` / `libvips` 的内部并发与 CPU 饱和度
- SQLite 连接池上限与任务状态写入频率
- MinIO 上传链路延迟
- 管理面是否需要增加更真实的 pool 活跃度/容量指标
