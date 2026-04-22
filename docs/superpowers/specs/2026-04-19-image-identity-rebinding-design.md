# 图片身份重绑定设计（路径变更不丢数据）

## 1. 背景

当前后端以 `images.path` 作为图片导入去重的核心判定键：

- 扫描导入阶段：`SaveImage()` 通过 `INSERT OR IGNORE` + `path UNIQUE` 判定是否为已存在图片。
- 扫描清理阶段：`cleanupStaleImages()` 会删除当前扫描 roots 下、数据库中存在但本轮扫描未再次看到的路径记录。

因此，当图片仅发生重命名或移动时，系统当前行为为：

1. 新路径被识别为新图片并插入。
2. 旧路径由于未在本轮扫描结果中出现，被判定为 stale 并删除。
3. 旧图片关联的标签、收藏夹关联、平台任务等随旧记录级联删除。

该行为不符合“路径变更不丢业务数据”的目标。

## 2. 目标

本设计只解决以下问题：

- 当图片文件仅发生改名或移动时，保留原图片实体。
- 保留原图片关联的标签、收藏夹关系、平台任务关系。
- 将“删除 + 新增”改造为“原记录路径重绑定”。

保证边界：

- 本轮自动重绑定保证，适用于**已具备 `sha256`，或能在本轮扫描中成功计算并刷新出 `sha256` 的记录**。
- 对历史空 `sha256` 且本轮又未成功补齐的记录，只能提供 best-effort 行为，不承诺一定保留原实体关系。

本设计明确**不**覆盖以下目标：

- 不引入完整的重复文件合并系统。
- 不支持同一图片实体同时挂载多个路径。
- 不将系统整体改造为“纯 sha256 唯一身份制”。

## 3. 设计原则

### 3.1 身份分层

- `image_id`：继续作为系统内部主身份。
- `path`：表示当前文件位置，不再独自承担稳定身份语义。
- `sha256`：作为内容级精确指纹，用于识别“路径变化但内容未变”的同一文件。
- `pHash`：继续作为相似图辅助信息，不参与唯一身份裁决。

### 3.2 约束边界

- `sha256` 在本轮设计中用于**重绑定判定**，而不是直接改成表级唯一键。
- 重绑定只在“内容精确一致”条件下自动发生。
- 一旦出现 `sha256` 命中多条记录的歧义场景，不自动裁决为重绑定。

### 3.3 范围边界

- 本轮设计保证的“移动/重命名”范围，限定为：**同一次 `Scan(roots)` 调用覆盖范围内的路径变化**。
- `roots` 可以包含多个 source root；若新旧路径都落在同一次扫描传入的 roots 集合内，则允许自动重绑定并更新 `source_root`。
- 若跨 source root 的移动发生在两次彼此独立的扫描调用之间，本轮不保证自动保留原实体。
- 同一轮 `Scan(roots)` **禁止传入重叠/嵌套 roots**。
- 若存在重叠/嵌套 roots，扫描应直接失败返回，不进入正常发现/裁决/cleanup 流程。

### 3.4 范围判定权威

为避免 `path` 与 `source_root` 双重语义冲突，本设计规定：

- “某旧实体是否属于当前 `Scan(roots)` 覆盖范围”，**以其当前存储的 `image.path` 是否位于任一 root 下为准**。
- `source_root` 视为需同步维护的元数据字段，而不是范围判定权威。
- 重绑定候选筛选、冲突保护筛选、cleanup 保护逻辑，全部使用同一范围判定规则。

### 3.5 路径规范化定义

本设计统一采用以下路径规范化法则：

1. root 与 `image.path` 比较前一律执行 `filepath.Abs(filepath.Clean(...))`。
2. 本轮**不做 symlink 解析**。
3. Windows 下路径比较统一转为小写后再判定。
4. “path 位于 root 下”的语义，等价于 `sameOrChildPath(root, path)`：path 必须等于 root 或为其子路径。
5. 本轮采用“**存储即规范化**”策略：写入 `images.path` 前先执行上述规范化，使数据库中的 `path`、`FindByPath`、`seenPaths` key、rebind 新旧路径比较共享同一语义。
6. Windows 上仅大小写变化的 rename 视为 `path_hit` / no-op，不视为 `rebound`。

canonical form 裁定：

- `images.path`
- `images.source_root`
- `FindByPath` 的输入
- `discoveredPaths`
- `expectedOldPath`
- 规范化迁移逻辑

以上全部必须基于同一个 canonical path 值。

责任边界裁定：

- 仅当 `feature flag && upgrade marker` 同时满足时，repository 层才启用新的 canonicalization 语义。
- 在此之前，legacy `SaveImage/FindByPath` 继续保持旧语义。
- repository 层负责所有持久化与按 path 查询 API 的 canonicalization。
- scanner/service 层负责把 `discoveredPaths`、`roots`、以及 scan-local 仲裁中使用的路径先规范化到同一语义。
- `EnsureScanSchema` 不负责历史 path 规范化；它只负责 fresh DB 所需的 runtime schema 落地。

### 3.6 历史 path 规范化迁移

由于当前数据库中可能已存在未规范化的历史 path，本设计要求在正式启用新规则前执行以下迁移策略：

1. 对 `images.path` 与 `images.source_root` 执行一次性规范化迁移。
2. 若规范化后出现路径唯一键冲突，则迁移必须停止，并输出冲突清单供人工处理；不得静默覆盖。
3. 迁移完成前，不应启用依赖“存储即规范化”的重绑定与 cleanup 新逻辑。

交付形态裁定：

- 本轮统一采用**一个独立 maintenance/upgrade command** 作为交付物，而不是拆成多个命令。
- 该 command 必须按固定顺序执行：先修复本功能依赖的 schema 差异，再执行历史 path 规范化迁移，最后输出结构化冲突清单。
- 该 command 在出现任一 schema 修复失败或 path 规范化冲突时，必须返回失败退出码。
- 新扫描逻辑的启用门槛为：该唯一 maintenance/upgrade command 成功执行。

upgrade command 失败语义与可重入性：

- schema 补差允许持久化，且必须幂等。
- path 规范化迁移步骤必须以事务方式执行。
- 任一冲突或失败发生时，**不得写入 upgrade marker**。
- 该 command 必须可安全重跑，直到完整成功后才视为新逻辑可启用。

运行时启用模型：

- 本轮采用**显式 feature flag** 作为技术门控，默认关闭。
- maintenance/upgrade command 成功执行后，发布流程才允许开启该 feature flag。
- `cmd/scan`、server、watcher 等所有扫描相关入口，只有在 feature flag 开启时才允许进入本轮新扫描逻辑。
- 若 feature flag 关闭，则只能运行旧逻辑或拒绝执行新逻辑；不得隐式落入新逻辑路径。

升级门控真值来源：

- 本轮采用**DB 内持久化 upgrade marker** 作为唯一真值来源。
- maintenance/upgrade command 成功完成 schema 补差与 path 规范化迁移后，必须写入该 marker。
- 所有扫描入口在 feature flag 开启时，进入新逻辑前都必须先做 preflight：检查 marker 是否存在且版本满足要求。
- 若 feature flag 开启但 marker 不满足要求，则扫描入口必须拒绝执行新逻辑并返回失败。

fresh DB / existing DB 分工：

- fresh DB 的目标 schema 必须直接落在 `internal/repository/schema.go`，包括本功能必需的列与索引。
- maintenance/upgrade command 只负责 **existing DB** 的 schema 补差与历史 path 规范化迁移。
- `EnsureScanSchema` 不得承担历史 path 规范化职责。
- `migrations/*.sql` 与相关测试仍作为受支持初始化路径；与 `images` 身份重绑定功能直接相关的 schema 结果，必须与 `schema.go` 保持功能等价。

upgrade marker 落点：

- 本轮固定引入 `scan_upgrades(name TEXT PRIMARY KEY, version INTEGER NOT NULL, applied_at TIMESTAMP NOT NULL)` 作为 upgrade marker 存储对象。
- fresh DB 必须直接由 `internal/repository/schema.go` 创建该表。
- fresh DB 在创建出本功能所需目标 schema 后，必须直接写入 `image_identity_rebind = 1` marker，作为“该库已处于目标模式”的真值来源。
- maintenance/upgrade command 负责在 existing DB 上补齐该表并写入 `image_identity_rebind = 1`。
- 所有扫描入口 preflight 固定检查该 marker。

说明：

- 本轮**不提供**“过渡兼容方案”作为正文内的并行实现路径。
- 若未来需要兼容迁移路线，应另立方案，不属于本轮实现输入。

## 4. 推荐方案

采用“方案 C：内部主身份 + 内容指纹优先重绑定”。

### 4.1 数据模型

在现有 `images` 表中继续使用并补齐以下字段：

- `id`
- `path`
- `sha256`
- `source_mtime_unix`

新增/强化要求：

1. `sha256` 必须在扫描导入时稳定计算。
2. 为 `sha256` 建立普通索引（非唯一索引）。
3. 历史记录允许暂时存在 `sha256` 为空的过渡状态，但在后续扫描中应逐步补齐。

### 4.1.1 扫描链路中的 hash 计算约束

为避免实现阶段出现职责漂移，本设计明确以下约束：

1. `sha256` 与 `source_mtime_unix` 的计算必须纳入扫描导入主链路，而不是依赖事后异步回填。
2. 职责裁定：`MetadataService` 继续只负责轻量 metadata（stat、尺寸、格式等）；`ScannerService` 负责统筹 hash 采集与身份裁决；repository 只负责持久化 `sha256/source_mtime_unix`，不负责文件读取或 hash 计算。
3. 对每个扫描到的图片文件，进入身份判定前必须能拿到：
   - `path`
   - `sha256`
   - `source_mtime_unix`
   - 文件大小、尺寸、格式
4. 当 path 命中已有记录时，若该记录历史上缺失 `sha256` 或其 `source_mtime_unix` 已过期，则本轮扫描必须刷新该记录的 hash 缓存。
5. 历史空 `sha256` 记录的补齐策略为：**随着常规扫描命中逐步补齐**；本轮不要求单独发起一次全库回填任务。
6. 实现阶段可根据 `(source_mtime_unix, file_size)` 判断是否复用已有 hash 缓存，但该优化不能改变身份判定正确性。
7. `source_mtime_unix` 统一存储 `os.Stat(path).ModTime().UnixNano()`。
8. 仅当 `source_mtime_unix` 与当前文件 `mtime` 完全相等，且 `file_size` 相等时，才允许复用已有 `sha256` 缓存。
9. 否则必须重新计算 `sha256` 并刷新缓存。

### 4.1.2 两阶段裁决要求

为避免把“同轮复制出一个内容相同的新文件”误判成“路径变更”，本设计要求身份裁决至少满足以下语义：

1. 系统必须先拿到本轮扫描的完整 `discoveredPaths` 视图，之后再进入身份裁决；不得在 `WalkDir` 过程中边发现边做最终 `rebound` 裁决。
2. 只有当候选旧实体的旧 `path` 已确认**不在**本轮 `discoveredPaths` 中时，才允许执行 `rebound`。
3. 若候选旧实体的旧 `path` 在本轮仍然存在，则该场景不是“路径变更”，不得自动重绑定。
4. cleanup 的安全前提必须覆盖“文件已发现但后续处理失败”的场景；不得因为单文件处理失败而把仍存在于磁盘中的旧实体误删。
5. 只要本轮存在任一 in-scope discovery error，则所有依赖“旧 path 不在 `discoveredPaths` 中”这一负证据的自动 `rebound` 一律禁止；相关项统一降级为 `conflict`。

### 4.1.3 扫描执行模型

本轮冻结为三阶段扫描模型：

并发边界裁定：

- 同一数据库文件上的 `ScannerService.Scan` 必须 **single-flight / 串行化**。
- 本轮要求该约束覆盖**跨进程**入口，而不仅是进程内 mutex。
- 若已有扫描在运行，后续扫描请求必须等待、拒绝或合并为 single-flight；具体交互形式可在实现计划中细化，但语义必须是“同库单飞”。
- watcher、CLI、server manual scan、worker scan job 全部服从这一约束。

1. **Phase A：发现阶段**
   - 遍历 `roots`
   - 收集规范化后的 `discoveredPaths`
   - 记录 walk 错误与不可访问路径
   - 若出现任一 in-scope discovery error，则置 `cleanupBlocked = true`

2. **Phase B：裁决阶段**
   - 基于 `discoveredPaths` 并发执行 metadata/hash/identity adjudication
   - 产出 `inserted / rebound / path_hit / conflict` 等结构化结果
   - 生成 `planningItems`、`protectedImageIDs`、统计信息与冲突审计信息
   - 若出现任一 in-scope processing error，则置 `cleanupBlocked = true`

   在进入任何数据库写入前，Phase B 必须先完成一次 **scan-local 仲裁**：

   - 先对本轮文件完成 metadata/hash 计算
   - 再按 `sha256 -> old candidate imageID -> new paths[]` 分组
   - 若同一旧实体被多个新路径同时命中，则这些新路径全部标记为 `conflict`
   - 只有通过该仲裁后的单一路径候选，才允许进入后续 `rebound` 写入阶段

   Writer phase 裁定：

   - Phase B 完成全轮 arbitration 后，再进入独立 writer phase。
   - 写入阶段采用**串行 writer** 模型，不要求 scan-wide 单事务。
   - 每个 `path_hit / inserted / rebound` 以**单图片短事务**提交。
   - `rebound` 必须使用 `expectedOldPath` 作为 CAS 保护条件。
   - 每次 `rebound` 在 writer 事务内都必须做 revalidation：
     - `image.id = candidateID AND path = expectedOldPath`
     - `newPath` 当前未被占用
   - 若 revalidation 失败，默认推荐降级为 `conflict`。

3. **Phase C：cleanup 阶段**
   - 仅在 `cleanupBlocked = false` 时执行 stale cleanup
   - cleanup 依据 `discoveredPaths + protectedImageIDs` 判定保留范围

禁止事项：

- 禁止沿用“边 `WalkDir` 边立即做最终 rebind 裁决”的流式模型。

### 4.2 核心扫描判定顺序

对每一个扫描到的图片文件，按以下顺序裁定：

#### 步骤 1：按 path 命中

若数据库中已存在相同 `path` 的图片记录，则视为原记录命中：

- 更新必要的元数据（如文件大小、mtime、hash 缓存等）。
- 不创建新图片记录。
- 不变更 `image_id`。

#### 步骤 2：path 未命中，但 sha256 存在 in-scope 候选

若当前 `path` 不存在，则按以下顺序裁决：

1. 先按 `sha256` 查找**全库**候选实体，仅用于审计与误重绑定防护。
2. 再从中筛出属于当前 `Scan(roots)` 覆盖范围内的 **in-scope 候选实体**。
3. 若 in-scope 候选数为 0，则按 `inserted` 处理；roots 外命中仅记录审计信息，不阻止新图落库。
4. 若 in-scope 候选数为 1，且该候选旧 `path` 已确认**不在**本轮 `discoveredPaths` 中，则执行 `rebound`。
5. 若 in-scope 候选数大于 1，进入 `conflict`。
6. 若 in-scope 候选数为 1，但旧 `path` 在本轮仍然存在，则按 `inserted` 处理，因为这属于复制/并存，不属于路径变更。
7. 若同一轮 `discoveredPaths` 中有多个新路径以相同 `sha256` 唯一命中同一个旧实体，则这些新路径整体进入 `conflict`，不得自动选择单一路径执行 `rebound`。
8. 若本轮存在任一 in-scope discovery error，则所有依赖旧 path 缺失判定的候选一律不得 `rebound`，应进入 `conflict`。
9. 若发现“旧实体疑似存在但历史记录缺少 `sha256`，因而无法安全判定是否应重绑定”，则该场景一律进入 `conflict`，不得自动插入新实体，也不得自动删除旧实体。

说明：

- 第 7 条只适用于该旧实体本来已经满足 `rebound` 前提的场景，尤其是旧 `path` 不在 `discoveredPaths` 中。
- 若旧 `path` 仍在 `discoveredPaths` 中，则这些新路径属于复制/并存，应按 `inserted` 处理，而不是 `conflict`。

当满足 `rebound` 条件时：

- 将该已有记录的 `path` 更新为新路径。
- 更新 `filename`、`source_root`、`file_size`、尺寸、格式、mtime 等元数据。
- 保留该记录原有 `id` 及其全部业务关联。
- 将本次命中的旧记录标记为“已在本轮扫描中重绑定”。

#### 步骤 3：path 未命中，sha256 也未命中

视为真正新图片：

- 新建图片记录。
- 分配新的 `image_id`。

#### 步骤 4：path 未命中，且 in-scope 候选冲突

进入歧义分支：

- `conflict` 是本轮所有“不可安全自动裁决”场景的总类，至少包括：
  - `multi_old_candidates`
  - `multi_new_paths_for_one_old`
  - `rebind_blocked_by_discovery_error`
- 不自动执行重绑定。
- 不插入新图片记录。
- 不删除任何 sha256 命中的旧实体。
- 当前扫描文件在该轮次被记为“冲突待处理”，不进入自动任务规划。
- 该分支必须产生日志，便于后续审计。

该策略的核心原则是：**遇到歧义时宁可不自动接管，也不得错误串改实体归属。**

### 4.3 并发与原子性要求

当前扫描实现为多 worker 并发导入，因此重绑定判定必须满足以下硬约束：

1. “按 sha256 查询候选 → 判断是否唯一命中 → 更新原记录 path/metadata” 必须在单个 repository 事务或等效原子临界区内完成。
2. 任一图片记录在同一轮扫描中最多只能被成功重绑定一次。
3. 扫描过程中产生的辅助状态（例如 seen paths、冲突集合）必须是线程安全的；若可以完全由数据库最终状态推导，则优先采用数据库状态，减少进程内共享状态。
4. 若发生路径交换或并发竞争，系统必须保证：
   - 不产生违反唯一约束的中间状态；
   - 不出现同一新路径被多个旧实体同时绑定；
   - 不出现同一旧实体被多个新路径覆盖。

### 4.4 扫描结果契约

为避免与当前 `SaveImage() -> isNew bool` 语义冲突，本设计要求实现阶段把扫描导入结果升级为结构化状态。推荐至少区分以下五类：

1. `path_hit`
   - 含义：按 path 命中原记录
   - 统计：计入 skipped/unchanged
   - 任务规划：默认跳过

2. `rebound`
   - 含义：path 未命中，且 sha256 在当前 roots 范围内恰好命中 1 个旧实体，同时该旧 path 本轮未出现，因而完成路径重绑定
   - 统计：单独计入 `rebound_count`，不得混入 inserted 或 ordinary skipped
   - 任务规划：默认跳过；本轮目标只解决实体与关联保留，不强制因路径变化重新生成任务
   - `SourceDescriptor`：使用重绑定后的**新路径**

3. `inserted`
   - 含义：真正新图片
   - 统计：计入 imported
   - 任务规划：进入自动任务规划

4. `conflict`
    - 含义：所有不可安全自动裁决的场景总类，包括多旧候选、多新路径争抢单一旧实体、以及 discovery error 阻断 rebind
    - 统计：单独计入 `conflict_count`
    - 任务规划：跳过
    - 日志：必须输出冲突信息

5. `failed`
   - 含义：metadata/hash/DB write 等处理阶段失败
   - 统计：计入 `Failed`
   - 错误：写入 `Errors`
   - 任务规划：跳过
   - cleanup：可触发 `cleanupBlocked = true`

补充：

- “多个新路径争抢同一旧实体”的场景也归入 `conflict`。
- `failed` 不属于 `conflict`；两者必须在状态机与统计口径上严格分离。

状态边界裁定：

- discovery error：若导致依赖“旧 path 缺失”负证据的 `rebound` 无法安全成立，则相关候选降级为 `conflict`。
- processing error（metadata/hash/DB write）：相关文件记为 `failed`，**不**记为 `conflict`；但可置 `cleanupBlocked=true`。

最小对外结果契约：

- `ScanResult` 必须新增 `Rebound`、`Conflict`、`IgnoredNonImageCount`、`ConflictEntries`、`CleanupBlocked`。
- `cmd/scan` 必须输出上述新增扫描统计。
- worker 触发型入口可不返回完整 `ConflictEntries` 明细，但至少保留 `Conflict` 计数，且日志必须结构化。

### 4.4.1 服务返回值与 CLI 退出契约

为避免实现阶段在 `Scan(ctx, roots)` 的返回值、`ScanResult` 字段与 CLI 退出码之间分叉，本设计固定如下：

| 失败类别 | `Scan(ctx, roots)` 返回 | `ScanResult` 表现 | CLI 退出码 |
|---|---|---|---|
| invalid roots（含重叠 roots） | `nil, err` | 不返回部分结果 | 非零 |
| discovery error | `result, nil` | 写入 `Errors`，`CleanupBlocked=true`，不计入 `Failed` | 非零 |
| metadata/hash failure | `result, nil` | 计入 `Failed`，写入 `Errors`，`CleanupBlocked=true` | 非零 |
| DB write failure | `result, nil` | 计入 `Failed`，写入 `Errors`，`CleanupBlocked=true` | 非零 |
| cleanup failure | `result, err` | 返回已有统计 + 错误 | 非零 |
| task planning failure | 默认推荐 `result, err` | 保留已完成扫描统计 | 非零 |

补充约束：

- 单文件级 discovery/processing error 不应中断整轮扫描；应尽量继续收集结果，并通过 `ScanResult` 暴露。
- 只有 invalid roots、cleanup failure 等顶层不可继续场景，才强制要求 `Scan(ctx, roots)` 返回 non-nil error。
- invalid roots 必须在 discovery 前失败：不创建 batch、不返回部分统计、不执行 cleanup、不进入 task planning。

非 CLI 入口契约：

- worker / manual scan job / admin 触发扫描，只要 `ScanResult.Errors` 非空，就必须传播“本次扫描失败”语义。
- worker/job 可继续保留部分扫描统计，但最终任务状态不得标记为完全成功。
- admin/API 是否返回非 2xx 可在实现计划中细化，但至少必须把部分失败显式暴露给调用方，不能伪装成纯成功。

### 4.5 扫描聚合状态契约

由于 cleanup 不再能只依赖任务规划 items，本设计要求扫描流程在单次 `Scan(roots)` 调用内维护一份聚合状态。推荐最少包含：

- `seenPaths`
- `discoveredPaths`
- `protectedImageIDs`
- `planningItems`
- `reboundCount`
- `conflictCount`
- `conflictEntries`
- `cleanupBlocked`

语义要求：

1. `discoveredPaths`：记录本轮在 roots 下发现的全部图片文件路径，不论后续处理成功或失败。
2. `seenPaths`：记录本轮成功完成身份裁决并纳入最终状态判断的路径集合。该集合仅用于调试/观测，不用于判断“旧 path 是否仍然存在”。
3. `protectedImageIDs`：记录因 sha256 多命中冲突而需要在 cleanup 中保留的候选旧实体 ID。
3. `planningItems`：仅收集允许进入平台任务规划的图片项。
4. `reboundCount`：统计成功路径重绑定的次数。
5. `conflictCount`：统计冲突次数。
6. `conflictEntries`：记录每次冲突的结构化审计信息。
7. `cleanupBlocked`：只要 Phase A 或 Phase B 出现任一 in-scope discovery/processing error，则置为 true。

cleanup 必须消费该聚合状态，而不是仅从任务规划 items 反推保护信息。

术语封印：

- 判断“旧 path 在本轮是否仍存在”时，只允许使用 `discoveredPaths`。
- cleanup 的保留判定，只允许使用 `discoveredPaths + protectedImageIDs + 全轮失败保护规则`。
- 全轮失败保护规则统一由 `cleanupBlocked` 表达。
- `seenPaths` 不参与存在性裁决，避免与 `discoveredPaths` 语义混淆。

### 4.6 与当前扫描统计的映射

为避免实现阶段统计口径分叉，本设计要求状态与 `ScanResult` 的映射固定如下：

| 状态 | ScanResult 统计 | 是否生成 planning item | 任务规划 |
|---|---|---|---|
| `path_hit` | `Skipped++` | 否 | 跳过 |
| `rebound` | 新增 `Rebound++` | 否 | 跳过 |
| `inserted` | `Imported++` | 是 | 进入 |
| `conflict` | 新增 `Conflict++` | 否 | 跳过 |

补充约束：

- `rebound` 不得混入普通 `Skipped`，避免遮蔽路径重绑定行为。
- `conflict` 不得混入 `Failed`，除非实现阶段明确定义冲突为错误；本设计默认将其视为可审计但非失败状态。
- 若现有 `ScanResult` 结构无法表达 `Rebound/Conflict`，则实现阶段必须扩展结构，而不是把新状态挤进旧字段语义。

发现阶段计数边界：

- 当前代码中 discovery-level 的“非图片文件被忽略”与 identity-level 的 `path_hit` 不能继续共用同一个 `Skipped` 语义。
- 本轮裁定：`Skipped` 专用于 **identity adjudication 中的 `path_hit` 数量**。
- discovery-level 的非图片/不可处理文件，必须迁移到新字段，例如 `IgnoredNonImageCount`（命名可在实现计划中细化，但语义必须独立）。

### 4.6.1 与 TaskBatch 的边界

本轮裁定：`PlanBatch` **只接收 `inserted` 状态的图片项**。

本轮同时裁定：**每次 `Scan(roots)` 仍创建一个 TaskBatch，即使 `planningItems` 为空。**

因此：

- `path_hit / rebound / conflict` 只体现在 `ScanResult` 级统计中。
- 它们**不进入** `planningItems`。
- `BatchNewImages / BatchSkippedImages / BatchSkippedUnchanged / BatchSkippedDuplicateTasks` 不再承担表达扫描阶段全部结果的职责。
- 若后续需要让 TaskBatch 也表达这些状态，应单独设计新的批任务统计模型，而不是在本轮强行复用旧字段。

现有批处理字段的新语义：

| 字段 | 新语义 |
|---|---|
| `TotalImagesInBatch` | 仅表示进入 `PlanBatch` 的 `inserted` 图片总数 |
| `CreatedTasks` | 基于 `inserted` 图片实际创建的平台任务数 |
| `SkippedTasks` | 仅表示任务规划阶段的跳过数，不再覆盖扫描阶段的 `path_hit/rebound/conflict` |
| `BatchNewImages` | 等同于 `inserted` 图片数 |
| `BatchSkippedImages` | deprecated，不再表达扫描阶段全部跳过项 |
| `BatchSkippedUnchanged` | deprecated，若保留仅用于任务规划内部统计 |
| `BatchSkippedDuplicateTasks` | 仅表示任务规划阶段的重复任务跳过数 |

若前端或 API 仍需展示扫描阶段全貌，应直接读取 `ScanResult` 中的扫描统计，而不是复用上述 batch 字段。

即使 `planningItems` 为空：

- 也应创建空 TaskBatch
- 其 `TotalImagesInBatch = 0`
- `CreatedTasks = 0`
- 相关 batch 统计字段按零值返回

部分失败时的 batch 边界：

- 只要扫描已进入 Phase B 并产出最终统计，就应创建 TaskBatch，即使存在 `discovery error` 或 `failed` 项。
- 此时 batch 状态应显式区分为 `partial_failed` 或等价失败态，具体命名可在实现计划中细化。
- 只有 invalid roots、cleanup failure、task planning failure 这类顶层失败场景，才允许不创建 TaskBatch。

### 4.7 path 权威下的仓储契约

既然本设计已将“当前扫描范围判定”权威固定为 `image.path`，则所有相关仓储查询必须遵循同一规则。

硬约束如下：

1. 凡是判定“某旧实体是否属于当前 `Scan(roots)` 覆盖范围”的逻辑，必须基于**规范化后的 `image.path` 是否落在任一 root 下**。
2. cleanup 的候选集枚举，不得继续仅凭 `source_root` 选取；必须改为按 path-membership 枚举当前 roots 范围内的图片。
3. sha256 候选查找在获得命中结果后，也必须按同一 path-membership 规则过滤出“当前扫描范围内候选实体”。
4. `source_root` 仅在实体最终被保留/重绑定后同步更新，不作为扫描范围判定依据。

说明：

- 实现阶段允许通过新增 repository/service API，而不是继续复用单一 `isNew bool` 返回值。
- 若保持旧接口不变，则必须在 service 层额外封装足够的结果状态，禁止把 `rebound` 伪装成“普通未变化”。

### 4.8 watcher 与单文件增量入口边界

当前代码库存在 watcher/单文件增量导入入口。本轮裁定如下：

- watcher/单文件增量导入**不纳入本轮自动 rebind 机制**。
- 它不得自行执行依赖 `discoveredPaths` 完整视图的 `rebound` 判定。
- 推荐做法是：watcher 仅触发所属 root 的一次 `Scan(roots)`，由完整三阶段扫描模型完成裁决。
- 在 watcher 尚未改造成触发 root scan 之前，其行为只能保持 path-only 导入语义。
- watcher 若触发 `Scan(roots)`，也必须服从上面的 single-flight / 串行化约束。
- 凡是会写 `images` 的扫描相关入口（含 watcher / 单文件 `importFile`）都必须服从同一个 scan coordinator / lock。
- 在 full scan 运行期间，单文件 `importFile` 必须被阻塞、拒绝，或纳入同一串行化门控；不得绕过 full scan 的仲裁与 writer phase。

启用策略裁定：

- 当本轮 feature flag 开启时，watcher/任何单文件导入入口不得继续以旧的 path-only 语义写入 `images`。
- 在此模式下，它们只能：
  1. 触发所属 root 的完整 `Scan(roots)`；或
  2. 被显式拒绝/不注册。

### 4.9 失败结果契约

为与当前 `ScanResult.Failed` / `Errors` 对齐，本设计明确：

- metadata 失败
- hash 计算失败
- repository / DB write 失败

以上都归类为 `failed`，而不是 `conflict`。

其行为要求：

1. 计入 `ScanResult.Failed`
2. 追加到 `ScanResult.Errors`
3. 不产生 `planningItem`
4. 可触发 `cleanupBlocked = true`

### 4.9.1 历史空 sha256 记录的 fallback 裁定

当出现以下场景时：

- `path miss`
- 当前扫描文件可见
- 存在疑似旧实体，但旧实体历史上缺少 `sha256`

本轮统一裁定：

1. 不自动执行 `rebound`
2. 不自动插入新实体
3. 进入 `conflict`
4. 将疑似旧实体加入 `protectedImageIDs`
5. 禁止 cleanup 删除该旧实体

理由：

- 本轮目标首先是“避免数据与关联被误删”，而不是在证据不足时追求自动接管。

### 4.10 discovery error 结果契约

为与当前 `cmd/scan` 与 `ScanResult` 行为对齐，本设计明确：

- walk error
- 不可访问目录
- 权限错误
- 其他 discovery 阶段错误

统一归类为 `discovery error`，其行为要求：

1. 写入 `ScanResult.Errors`
2. **不计入** `ScanResult.Failed`（`Failed` 仅用于 metadata/hash/DB write 失败）
3. 置 `cleanupBlocked = true`
4. 触发依赖“旧 path 缺失”负证据的 `rebound` 降级为 `conflict`
5. `cmd/scan` 输出当前扫描统计后，以**非零退出码**结束
6. `ScanResult` 应对外暴露 `CleanupBlocked`，避免 `DeletedStale=0` 与“安全跳过 cleanup”混淆

## 5. stale cleanup 新规则

当前 stale cleanup 的错误根源在于：它只看“旧 path 是否再次出现在本轮扫描结果中”，而不理解“旧实体是否已被新 path 重绑定”。

### 5.1 现状问题

现有逻辑等价于：

- 若数据库中的 `image.Path` 不在本轮 `seen paths` 集合中，则删除该图片记录。

这会把“路径变化”错误当成“文件消失”。

### 5.2 新逻辑

stale cleanup 的主判定逻辑以**导入阶段提交后的数据库最终 path 状态**为准。

在推荐实现中，重绑定会先完成数据库更新，因此 cleanup 只需要维护：

- `discoveredPaths`：本轮扫描发现的全部图片路径。
- `protectedImageIDs`：因 sha256 多命中冲突而被保护的旧实体 ID 集合。
- `cleanupBlocked`：本轮是否因 discovery/processing error 而整体禁止 cleanup。

对数据库中的图片逐条判断时，应改为：

1. 若 `cleanupBlocked = true`，则**整轮跳过 stale cleanup**。
2. 若 `image.Path` 在 `discoveredPaths` 中，保留。
3. 若 `image.ID` 在 `protectedImageIDs` 中，保留。
4. 仅当以上条件都不满足时，才判定为 stale 并删除。

说明：

- 对于成功完成重绑定的记录，其 `path` 已经更新为新路径，因此命中 `discoveredPaths` 即可保留，不再要求额外维护 `reboundImageIDs` 作为通用必要条件。
- `protectedImageIDs` 是**单次扫描调用内的线程安全内存态集合**，不是新增数据库字段。
- 仅在冲突分支中，才允许引入附加保护状态，避免“未能自动重绑定但实体被 cleanup 误删”。
- 若存在 in-scope 文件已加入 `discoveredPaths`，但后续在 metadata/hash/identity 判定阶段失败，则本轮直接跳过 stale cleanup，优先保证不误删旧实体。
- `seenPaths` 若继续保留，也仅用于调试与观测；cleanup 绝不读取它。
- Phase A 的 walk error、不可访问目录、权限错误等 discovery error，同样属于 `cleanupBlocked` 触发条件。

### 5.3 冲突保护

若某扫描文件进入“sha256 多命中冲突分支”，则 cleanup 必须满足：

- `protectedImageIDs` 的定义是：**所有必须在 cleanup 中保留的旧实体 ID 集合**。
- 对于 `multi_old_candidates`，将与该 sha256 冲突相关、且属于当前 `Scan(roots)` 覆盖范围内的候选旧实体 ID 全部加入 `protectedImageIDs`；
- 对于 `multi_new_paths_for_one_old`，必须将被多个新路径争抢的那个旧实体 ID 加入 `protectedImageIDs`；
- 对于 `rebind_blocked_by_discovery_error` 或 processing error，依赖 `cleanupBlocked = true` 直接阻断 cleanup，而不是靠逐项 `protectedImageIDs` 保护；
- cleanup 不删除这些被保护的候选旧实体；
- 不将当前扫描文件自动落库为新实体；
- 通过日志或统计结果暴露冲突，等待后续人工或专用治理流程处理。

### 5.4 冲突审计载荷

为确保人工后续能够定位并处理冲突，本设计要求每条冲突记录至少输出以下结构化信息：

- `new_path`
- `sha256`
- `candidate_image_ids`
- `candidate_paths`

若实现阶段将冲突信息写入日志，则必须采用可解析的结构化日志格式；若写入扫描结果，则 `conflictEntries` 至少应包含上述字段。

### 5.5 预期效果

当文件从 `oldPath` 改到 `newPath`：

- 扫描阶段通过 `sha256` 命中原图片。
- 原图片记录更新为 `newPath`。
- stale cleanup 不再删除该记录。

## 6. 业务结果

该设计落地后，图片仅改名/移动时应满足：

- 图片详情 ID 不变。
- 标签不丢。
- 收藏夹关系不丢。
- 平台任务关系不丢。
- 缩略图关联继续挂在原图片实体上。
- 图库中表现为“同一张图路径更新”，而不是“旧图消失、新图出现”。

### 6.1 关于未完成任务 payload 的边界

当前代码中部分异步 job payload 会内嵌 `path` 与 `filename`。因此本轮设计对“平台任务关系保留”的定义，限定为：

- `platform_tasks` 继续通过 `image_id` 关联同一图片实体；
- `async_jobs` 继续通过 `platform_task_id` 关联原平台任务；
- 已存在任务记录不会因图片重绑定而级联消失。

本轮**不保证**：

- 已经处于 queued/running 状态、且 payload 内嵌旧路径的 job 会被自动改写；
- 在途 job 一定不会因旧路径失效而失败。

对该类任务，推荐后续单独立项治理，策略可包括：

- 在任务执行前按 `image_id` 重新解析最新 path；或
- 在重绑定后同步刷新未完成任务 payload。

该治理不纳入本轮目标。

## 7. 错误处理与边界条件

### 7.1 sha256 缺失

若历史记录尚未补齐 `sha256`，则：

- 该记录在未补齐期间无法参与自动重绑定。
- 系统可退回 path-only 逻辑。
- 后续扫描应优先补齐 hash 缓存。
- 在补齐前，该记录仅享受 best-effort 保留，不属于本轮“路径变更不丢业务数据”的强保证范围。

### 7.2 内容发生微小变化

若文件发生重压缩、元数据变化、重新保存，导致 `sha256` 改变，则本设计不会将其视为同一实体。

这是有意为之：

- 本轮只做精确内容重绑定。
- 不做近似内容归并。

### 7.3 重复内容文件

若库中已有多条记录拥有相同 `sha256`，说明系统历史上已存在重复内容实体。此时：

- 只有当**当前 `Scan(roots)` 范围内的 in-scope 候选旧实体数大于 1** 时，才进入冲突分支。
- 若重复实体全部位于 roots 外，则它们只用于审计，不阻断当前新文件按 `inserted` 处理。
- 不能自动判断当前扫描文件应重绑定到哪一条时，必须进入冲突分支。

### 7.4 路径交换

若两个文件在同一轮扫描中发生复杂路径互换，自动重绑定逻辑必须保证：

- 不产生中间唯一键冲突。
- 更新路径时使用安全顺序或事务内临时占位策略。

该细节在实现计划中展开。

### 7.5 冲突文件的可见性

当扫描文件因 sha256 多命中而进入冲突分支时：

- 本轮不为它创建新的图片实体；
- 本轮不自动替换任何旧实体；
- 本轮不自动规划关联任务；
- 必须通过日志或扫描结果统计暴露该冲突，并满足第 5.4 节规定的最小审计载荷。

## 8. 测试要求

至少覆盖以下测试：

1. **重命名重绑定成功**
   - 初始扫描导入 `before.png`
   - 文件改名为 `after.png`
   - 再次扫描后：数据库仍只有 1 条记录，且 `image_id` 不变，`path` 更新为 `after.png`

2. **移动目录重绑定成功**
   - 同一 source root 下移动路径
   - 记录保留、路径更新

3. **旧路径不会被 stale cleanup 误删**
   - 验证重绑定后 cleanup 不删除原实体

4. **sha256 未命中时仍新增**
   - 新图片正常插入

5. **sha256 多命中进入冲突分支**
   - 不自动重绑定错误实体
   - 不创建新实体
   - 相关旧实体不会被 cleanup 误删

6. **多个新路径争抢同一旧实体时进入冲突分支**
   - 同轮扫描中多个新路径以相同 sha256 唯一命中同一个旧实体
   - 不自动选择任一路径执行重绑定
   - 全部进入 conflict

7. **外部真实删除仍能被清理**
   - 文件确实消失且未发生重绑定时，记录仍会被 stale cleanup 删除

8. **关联保留**
   - 标签、收藏夹、平台任务在路径变更后仍指向同一 `image_id`

9. **path 命中时可补齐历史空 hash**
   - 原记录历史上没有 sha256
   - 本轮扫描命中 path 后补齐 sha256/source_mtime_unix

10. **跨多 root 的同轮扫描可更新 source_root**
   - 同一次 `Scan(roots)` 调用中，文件从 rootA 移到 rootB
   - 原实体保留，`source_root` 更新

11. **roots 范围外的唯一 sha256 命中不会被错误重绑定**
   - 当前扫描只覆盖 rootA
   - 库中 rootB 存在唯一相同 sha256 记录
   - 本轮不自动把 rootA 新文件重绑定到 rootB 旧实体

12. **同轮复制不会被误判成重绑定**
   - 旧 path 与新 path 在同一轮扫描中同时存在
   - sha256 唯一命中旧实体
   - 本轮不得执行 `rebound`

13. **Phase A discovery error 会阻断 cleanup**
   - roots 下某子目录不可访问或 walk 失败
   - 本轮不得执行 stale cleanup

14. **重叠/嵌套 roots 会直接失败**
   - 传入重叠或嵌套 roots
   - 扫描应直接返回错误，不进入正常裁决流程

## 9. 风险与代价

### 9.1 扫描成本上升

为每个文件计算 `sha256` 会增加扫描耗时，尤其在大图库下更明显。

### 9.2 历史数据补齐成本

旧数据若缺少 `sha256`，需要逐步补齐，才能享受完整重绑定能力。

### 9.3 歧义冲突处理

一旦历史上已经存在重复内容实体，自动重绑定必须偏保守，宁可放弃自动合并，也不能串改实体归属。

### 9.4 在途任务失败风险仍存在

由于本轮不接管已排队/运行任务 payload 的路径刷新，因此少量在途任务可能因旧路径失效而失败。该风险被接受，并通过后续任务执行层治理单独处理。

### 9.5 schema 双来源维护风险

当前项目的运行时扫描 schema 与正式 migrations 存在分离现象。任何涉及 `sha256` 索引、扫描状态字段或图片表结构的变更，都必须显式审查并同步对应 schema 入口，避免测试环境、初始化逻辑与迁移逻辑继续漂移。

边界澄清：

- 本 spec 在 schema 层的强制目标，仅限**图片身份重绑定功能直接依赖的对象**。
- 本轮不宣称会把整个 runtime schema 与 migrations 的所有历史漂移一次性消除。
- 就当前代码库上下文而言，本轮 schema 的强制落点以 `internal/repository/schema.go` 的 runtime schema 为准。

### 9.6 仓储接口扩张风险

当前仓储抽象主要围绕 `path` 与 `source_root` 构建。本设计落地时，需要新增 path-membership 枚举、sha256 候选查询、原子重绑定等接口，实施阶段必须控制接口膨胀并避免 service/repository 责任漂移。

## 10. 不采纳的方案

### 10.0 schema 变更落点不明

不采纳原因：

- 任何数据模型调整若不明确落到当前代码库的 schema 入口，就会导致规划与实现脱节。

实施约束：

- 本设计涉及的 schema 变更，至少必须同步更新 `internal/repository/schema.go`。
- fresh DB 必须仅通过 `internal/repository/schema.go` 即可创建出本功能所需的目标 schema（含必需列/索引）。
- 在当前代码库上下文中，必须提供一个独立 maintenance/upgrade command，用于修复历史数据库中的 `sha256`、`source_mtime_unix` 与 `idx_images_sha256` 等图片身份重绑定所依赖的 schema 差异。
- 若未来项目重新建立正式 migration 链，则再将上述维护逻辑迁入正式 migration；这不作为本轮前置要求。
- maintenance/upgrade command 不得替代 fresh DB runtime schema 的创建职责；它只处理 existing DB 的补差与历史 path 规范化。
- 若其他非本功能依赖对象（例如 task platform 相关表）也存在 runtime/migration 漂移，不在本 spec 的强制补齐范围内，应另立治理项处理。

说明：

- `phash_hex` 不再列为本轮 upgrade command 的 MUST 对象。
- 若实现阶段继续复用现有 duplicate-hash cache 契约而需要同步维护 `phash_hex`，可作为兼容性增强项处理，但不属于本轮图片身份重绑定的上线前置条件。

## 10.0.1 分阶段交付裁定

为避免实现规划继续被过宽范围拖垮，本轮强制拆为两层：

### Phase 1 / MUST

- rebind adjudication
- cleanup 新规则
- `ScanResult` / CLI / worker 的新契约
- canonical path 规则
- upgrade marker / feature flag / preflight
- watcher 不得绕过 full scan

### Phase 2 / SHOULD

- rollout hardening
- 更完备的跨进程锁优化
- 非核心兼容性增强项（例如与旧 duplicate-hash 契约更深层对齐）

裁定：

- 进入 implementation planning 时，Phase 1 必须视为上线阻塞项。
- Phase 2 允许作为后续独立计划，不阻塞 Phase 1 核心实现。

### 10.1 path 继续作为唯一身份

不采纳原因：

- 无法从根本上解决路径变化导致的实体丢失问题。

### 10.2 sha256 直接升级为唯一键

不采纳原因：

- 会把“路径变更修复”问题升级成“重复文件身份制度重构”问题。
- 会改变系统对“相同内容但不同来源是否视为同一图”的业务边界。

### 10.3 pHash 作为唯一身份

不采纳原因：

- pHash 是近似相似，不是精确身份。
- 会引入错误合并风险。

## 10.4 最小仓储契约

为使 implementation planning 不再停留在抽象口号层，本设计明确要求最少具备以下仓储能力：

1. `FindBySHA256(sha256)` 或等价候选查询接口，用于获取同内容实体候选。
2. “按 normalized path-membership 枚举当前 roots 范围图片”的接口，用于替代仅按 `source_root` 的 cleanup 候选枚举。
3. 原子 `RebindImagePath(imageID, expectedOldPath, newPath, metadata...)` 能力，且必须满足：
   - 仅当 `expectedOldPath` 仍匹配时成功；
   - 若 `newPath` 已存在则失败或返回冲突；
   - 同事务内更新 path 与相关元数据。
4. cleanup 所需的 `protectedImageIDs` 可由 service 以内存态维护，但其候选筛选必须复用与仓储枚举相同的 normalized path-membership 规则。

## 11. 最终结论

推荐在当前系统中采用：

- `image_id` 统御内部实体
- `sha256` 负责路径重绑定识别
- `path` 记录当前位置
- stale cleanup 以更新后的 path 状态为主，并对冲突实体提供保护，避免误删

该方案能以最小制度扩张代价，解决“图片路径变更导致旧记录删除、新记录新增、业务关联丢失”的问题。
