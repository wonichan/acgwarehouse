# 代码设计最佳实践与日志输出规范

## 1. 函数与接口设计
- **单一职责**：一个函数只做一件事。严禁将复杂的业务逻辑与底层的 I/O、DB 操作混在同一个函数体内。
- **参数精简**：函数参数**不能超过 5 个**。如果参数过多，必须封装为结构体对象传入。
- **控制长度与复杂度**：理想函数体在 30 行以内，**硬性上限为 50 行**。函数的**圈复杂度必须 ≤ 10**。
- **接口声明**：提倡小而专注的单方法接口。遵循 Go 隐式接口理念，接口命名通常使用 `-er` 后缀（如 `LineageReader`）。优先依赖接口抽象而非具体结构体。

## 2. 方法接收者 (Receiver) 选择判定
在为结构体编写方法时，需遵循以下矩阵进行判定：
- **指针接收者 `(s *Struct)`**：
  1. 方法内部需要修改结构体内部状态或属性。
  2. 结构体体积较大，使用指针接收以避免大对象值复制带来的内存开销。
- **值接收者 `(s Struct)`**：
  1. 方法内无需修改状态，属于不可变（Immutable）操作，且结构体为小对象。
  2. 需要确保该操作并发安全、避免竞态条件时。

## 3. 函数/方法必须有中文注释
所有导出和未导出的函数、方法，必须以中文注释说明其用途。注释应简明扼要，描述"做什么"而非"怎么做"。

   ```go
   // ListReconcileProviders 获取所有已注册的权限对账提供者，按资源类型名字典序排列后返回。
   func ListReconcileProviders() []ReconcileProvider { ... }

   // normalizePath 去除路径首尾空白和尾部斜杠，返回标准化后的路径。
   func normalizePath(path string) string { ... }
   ```

复杂逻辑必须有中文注释：当一段逻辑包含多步推导、状态转换、diff 计算、并发控制等非直观流程时，必须在关键步骤处添加中文注释，说明"为什么这样做"而非重复代码本身。

   ```go
   // 按 key 排序，保证输出确定性。Go map 迭代顺序随机，
   // 不排序会导致相同输入产生不同输出顺序，影响测试稳定性和日志可读性。
    sort.Strings(resourceTypes)
    ```

## 4. 场景：外部同步 Upsert 不得覆盖业务生命周期字段

### 1. Scope / Trigger
- 触发：COS、消息队列、第三方 API 等外部源同步到本地表，并使用唯一键 upsert。
- 原因：外部源只是真相的一部分；软删除、隐藏、审核状态属于本地业务生命周期，不能被重复同步恢复。

### 2. Signatures
- Repository：`UpsertBy<ExternalKey>(ctx context.Context, value do.X) (do.X, error)`。
- DB：唯一键列（如 `cos_key`）用于幂等；生命周期列（如 `status`、`deleted_at`）只由业务命令更新。

### 3. Contracts
- 同步可更新：外部元数据字段，如 `filename`、`size`、`last_modified`、`width`、`height`、`category`。
- 同步不可更新：`status`、`deleted_at`、审核状态、owner 管理字段、用户行为计数字段。
- 业务恢复必须走显式命令或接口，不能借同步任务隐式恢复。

### 4. Validation & Error Matrix
- 外部凭证缺失或占位符 -> 同步命令返回带堆栈错误，不落库。
- 同步 upsert 命中已软删除记录 -> 只刷新允许的外部元数据，仍不出现在公开查询。
- 派生索引更新失败 -> SQLite 已提交时仅 warn，后续用全量 reindex 修复。

### 5. Good/Base/Bad Cases
- Good：同一 `cos_key` 重复同步后只有一行记录，宽高/大小更新，软删除状态保持。
- Base：新外部对象首次同步时写入默认 active 状态。
- Bad：`OnConflict` 更新 `status` 或 `deleted_at`，导致用户已删除内容在同步后重新公开。

### 6. Tests Required
- Repository 单测：重复 upsert 不新增记录，元数据更新。
- Repository 回归：先写入 deleted 记录，再同步同 key，断言 `ListActive` 不返回它。
- Command smoke：占位凭证失败路径；`--reindex` 可在空库上成功。

### 7. Wrong vs Correct

#### Wrong
```go
clause.AssignmentColumns([]string{"filename", "size", "status", "deleted_at"})
```

#### Correct
```go
clause.AssignmentColumns([]string{"filename", "size", "last_modified", "width", "height", "category"})
```

## 5. 场景：本地维护型 CLI 只加载所需配置

### 1. Scope / Trigger
- 触发：新增 `cmd/<tool>` 本地维护命令，用于直接维护 SQLite、索引、缓存等基础设施数据。
- 原因：维护命令通常不启动 HTTP 服务，不应被 JWT、COS、CORS 等无关服务配置阻塞。

### 2. Signatures
- 命令入口：`go run ./cmd/<tool> <flags>`。
- DB-only 配置入口：`conf.LoadDatabase() conf.DatabaseConfig`。
- SQLite 初始化：`db.NewSQLite(conf.LoadDatabase())`。

### 3. Contracts
- 环境变量：
  - `SQLITE_PATH`：可选，默认 `data/acgwarehouse.db`。
  - `SQLITE_BUSY_TIMEOUT_MS`：可选，默认 `5000`。
  - `SQLITE_READ_MAX_OPEN_CONNS`：可选，默认 `CPU * 4`。
- 维护型 CLI 不得调用 `conf.Load()`，除非确实需要完整服务配置。
- 维护型 CLI 不得要求 `JWT_SECRET`、COS 密钥或 CORS 配置。

### 4. Validation & Error Matrix
- 未提供操作 flag -> 返回非零错误，不落库。
- 同时提供多个操作 flag -> 返回非零错误，不落库。
- 提供操作 flag 但值为空白 -> 返回非零错误，不落库。
- 数据库打开失败 -> 返回带上下文错误。
- 资源关闭失败且主流程成功 -> 返回关闭错误。

### 5. Good/Base/Bad Cases
- Good：`SQLITE_PATH=/tmp/tool.db go run ./cmd/tagctl -a "调月莉音"` 只打开 SQLite 并写入 tag。
- Base：未设置 `SQLITE_PATH` 时使用 `data/acgwarehouse.db`。
- Bad：CLI 调用 `conf.Load()`，导致未配置 `JWT_SECRET` 时无法执行本地数据库维护。

### 6. Tests Required
- Config 单测：`JWT_SECRET` 为空时 `LoadDatabase()` 仍返回 `SQLITE_PATH` 配置。
- CLI 单测：零操作、多操作、空白操作值都返回错误。
- Repository 回归：删除数据时验证关联表清理契约。
- Smoke：用临时 `SQLITE_PATH` 执行 add/update/delete。

### 7. Wrong vs Correct

#### Wrong
```go
cfg, err := conf.Load()
if err != nil {
	return errors.WithMessage(err, "load config")
}
sqliteDB, err := db.NewSQLite(cfg.Database)
```

#### Correct
```go
sqliteDB, err := db.NewSQLite(conf.LoadDatabase())
if err != nil {
	return errors.WithMessage(err, "init sqlite")
}
```

