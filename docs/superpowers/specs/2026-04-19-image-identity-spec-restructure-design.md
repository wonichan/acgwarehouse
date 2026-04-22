# 图片身份重绑定规范重构设计

## 1. 背景

当前仓库已有一份单体规格文档：

- `docs/superpowers/specs/2026-04-19-image-identity-rebinding-design.md`

该文档经过多轮审查后暴露出同一个结构性问题：

- 核心身份裁决规则
- 扫描入口/并发/结果契约
- 升级迁移/rollout/feature gate

这三类问题被写进同一份法典，导致每修补一个边角，就会重新牵动另外两层制度。

结果是：

- 规格越来越长
- 审查意见不断跨层打架
- implementation planning 无法聚焦在单一责任面

因此，本次工作的目标不再是继续给单体 spec 打补丁，而是将其重构为更小、更稳、可独立进入实现规划的多份规格。

## 2. 重构目标

本次规范重构的目标如下：

1. 将“核心身份判定”从“上线治理”和“入口控制”中剥离。
2. 为每份 spec 建立单一责任边界，避免跨层耦合。
3. 使每份 spec 都能够单独接受审查，并可按依赖顺序分别进入 implementation planning。
4. 降低后续继续修改单个子系统时对整份法典的冲击。

## 3. 不重构的内容

本次工作是**规范重构**，不是功能实现。因此：

- 不修改 Go 代码
- 不修改数据库 schema
- 不修改测试
- 不直接废弃旧 spec

本次只定义：

- 新 spec 应拆成几份
- 每份 spec 的责任边界
- 从旧单体 spec 向新 spec 的迁移方式

## 4. 推荐拆分方案

推荐将当前单体 spec 拆成 **3 份**。

### 4.1 Spec A：核心身份裁决规范

建议文件名：

- `2026-04-19-image-identity-core-design.md`

只负责回答以下问题：

- 图片身份如何判定
- `path_hit / inserted / rebound / conflict / failed` 的状态机如何定义
- `sha256` 在身份判定中的角色是什么
- 历史空 `sha256`、多新路径争抢旧实体、多旧候选等场景如何裁决
- stale cleanup 在实体层面如何避免误删

明确不放入本 spec 的内容：

- feature flag
- upgrade marker
- maintenance command
- watcher/CLI/server 的入口治理
- TaskBatch / worker / CLI 返回契约

### 4.2 Spec B：扫描入口、并发与结果契约规范

建议文件名：

- `2026-04-19-image-scan-entrypoints-design.md`

说明：

- 文件名中的 `entrypoints` 仅是简称。
- Spec B 的权威范围以本节职责条目为准，不局限于入口本身。

只负责回答以下问题：

- `ScannerService.Scan` 的三阶段模型
- single-flight / lock / coordinator 的作用域
- canonical path 在扫描运行时语义中的定义（用于 roots、`discoveredPaths`、path-membership、cleanup 判定）
- 最小仓储契约：扫描所需 repository 查询/更新接口形态、调用时序与结果契约
- `sha256/source_mtime_unix` 的采集、缓存刷新、path-hit 补齐历史空 hash 的扫描链路语义
- watcher / manual scan / CLI / worker 如何进入扫描逻辑
- `ScanResult` 的结构化返回字段
- CLI、API、worker、job 对 partial failure 的行为约定
- TaskBatch 在空 planningItems、partial failure、top-level failure 下如何表达

补充说明：

- Spec B 所说的“cleanup 判定”仅指 cleanup 的输入路径语义、path-membership 与 in-scope 判定。
- Spec B **不**定义 stale entity 是否可删、哪些冲突/失败阻断 cleanup，这些实体级保护规则归 Spec A。

明确不放入本 spec 的内容：

- `sha256` 的业务身份哲学
- 历史迁移步骤
- schema upgrade

### 4.3 Spec C：升级迁移与 rollout 规范

建议文件名：

- `2026-04-19-image-identity-rollout-design.md`

只负责回答以下问题：

- fresh DB 与 existing DB 的职责分工
- `EnsureScanSchema`、runtime schema、migrations 的边界
- maintenance/upgrade command 做什么、不做什么
- upgrade marker 存储对象与 feature flag 门控
- rollout 的前置条件、upgrade command / marker / gate 检查本身的失败条件、重试与可重入性

补充说明：

- Spec C 不再拥有 canonical path 的运行时语义定义。
- Spec C 只拥有 canonical path **何时启用、如何迁移、如何门控** 的治理规则。

明确不放入本 spec 的内容：

- 具体的 `inserted/rebound/conflict` 业务判定
- TaskBatch / worker 语义

## 5. 为什么不是 2 份或 4 份

### 5.1 不采用 2 份的原因

若拆成 2 份：

- A：核心身份裁决
- B：其它全部

则“入口/并发/结果契约”仍会和“升级迁移/rollout”继续纠缠在一起。

这意味着：

- 一次 CLI 契约改动可能重新牵动 feature flag 与 upgrade marker
- 一次 watcher 策略改动可能重新牵动 path migration 条款

因此 2 份仍然过粗。

### 5.2 不采用 4 份的原因

若拆成 4 份：

- 身份裁决
- cleanup 与冲突保护
- 入口/批次/结果契约
- 升级迁移与 rollout

则会过度切片，增加维护成本。

当前项目规模下，cleanup 与冲突保护仍然是“核心身份裁决”的组成部分，还没必要独立成城。

因此 4 份过细。

## 6. 文档边界法典

为防止拆分后再次回流成单体文档，本次重构要求每份 spec 遵守以下法典：

### 6.1 单一责任原则

每份 spec 只能有一个主问题：

- 核心身份裁决
- 扫描入口/并发/结果契约
- 升级迁移/rollout

超出主问题的内容，只能引用，不得展开。

### 6.2 横向引用，不纵向吞并

允许在文档中写：

- “本条依赖 Spec B 中的 `ScanResult` 契约”
- “本条以前置条件方式依赖 Spec C 的 upgrade marker”

禁止：

- 在 Spec A 中重新完整复写 Spec C 的升级流程
- 在 Spec B 中重新完整复写 Spec A 的 conflict 状态机

### 6.3 旧 spec 降级为总索引/归档

原单体文档不应再继续作为实现输入主文档。

本节描述的是**最终状态要求**，不是立即执行顺序。

最终裁定：

- 先将旧单体 spec 强制归档到：`docs/superpowers/specs/archive/2026-04-19-image-identity-rebinding-design.full.md`
- 保留原文件路径不变
- 将原单体 spec 改造成**索引页**，明确声明“已拆分，请以新 spec 为准”
- 旧全文 archive 副本不是可选项，而是强制步骤

切换门槛：

- 仅当 Spec A/B/C 均完成迁移、去重，并各自审查通过后，才允许把旧 spec 改成索引页。
- 在迁移过程中，旧单体 spec 仍是源文档，但不得继续承接新增规范条款。

## 6.4 共享概念所有权矩阵

为避免拆分后再次互相吞并，以下共享概念的所有权必须固定：

| 概念 | 主所有者 | 其他 spec 的使用方式 |
|---|---|---|
| `path_hit / inserted / rebound / conflict / failed` 状态机 | Spec A | B/C 仅引用，不重写 |
| 实体级 cleanup 保护规则 | Spec A | B 只负责“何时运行 cleanup” |
| `discoveredPaths`、`cleanupBlocked`、三阶段扫描 | Spec B | A/C 只引用 |
| `ScanResult`、TaskBatch、CLI/API/worker 返回契约 | Spec B | A/C 只引用 |
| canonical path 的运行时语义定义 | Spec B | A/C 只引用 |
| canonical path 的启用/迁移/门控 | Spec C | A/B 只把它作为前置条件引用 |
| feature flag / upgrade marker / rollout 前提 | Spec C | A/B 不重写实现流程 |
| preflight gate failure 的对外暴露契约 | Spec B | C 只定义失败触发条件 |

补充裁定：

- Spec A 负责“实体应该如何判定与保护”
- Spec B 负责“扫描如何执行、结果如何暴露”
- Spec C 负责“这些新规则何时允许启用、如何迁移到位”

## 6.5 旧单体 spec 章节迁移映射表

为避免拆分时遗漏“测试/风险/仓储契约/不采纳方案”等尾部内容，旧 spec 主题必须按下表迁移：

| 旧单体 spec 主题 | 新 spec 归属 | 处理方式 |
|---|---|---|
| 身份状态机与 `path_hit/inserted/rebound/conflict/failed` | Spec A | 主体迁移 |
| stale cleanup 的实体保护规则 | Spec A | 主体迁移 |
| Scan phases / single-flight / writer phase | Spec B | 主体迁移 |
| `ScanResult` / CLI / worker / TaskBatch 契约 | Spec B | 主体迁移 |
| canonical path 的运行时语义定义 | Spec B | 主体迁移 |
| upgrade marker / feature flag / upgrade command / canonical path migration | Spec C | 主体迁移 |
| fresh DB / existing DB / schema.go / migrations 边界 | Spec C | 主体迁移 |
| watcher / manual / single-file incremental entry 边界 | Spec B | 主体迁移 |
| 业务结果 / 任务 payload / 对外结果可见性 | Spec B | 主体迁移 |
| `sha256` 缺失、重复内容、路径交换、冲突文件可见性等业务裁决边界 | Spec A | 主体迁移 |
| `sha256/source_mtime_unix` 采集、缓存刷新、path-hit 补齐历史空 hash | Spec B | A 只引用“有/无 hash 对裁决的影响” |
| 测试要求 | 按 A/B/C 分拆 | 不得集中遗留在旧文档 |
| 风险与代价 | 按 A/B/C 分拆 | 各自保留本层风险 |
| 不采纳方案 | 按 A/B/C 分拆 | 仅保留与该 spec 直接相关者 |
| 最小仓储契约 | Spec B | A 仅引用判定需求，不定义接口 |
| 任务 payload 边界 | Spec B | 主体迁移 |
| preflight gate failure contract | Spec B | 失败触发条件引用 Spec C |
| Phase 1 / MUST 与 Phase 2 / SHOULD 分阶段交付裁定 | Spec C | 主体迁移 |

禁止事项：

- 禁止把“测试要求 / 风险 / rejected alternatives” 留在旧文档不搬运
- 禁止把“最小仓储契约”同时在 A/B 两份新 spec 中完整复写
- 禁止在 C 中定义 CLI/API/worker 如何对外暴露 gate failure 结果

最小仓储契约裁定：

- **Spec A** 只定义仓储能力必须满足的语义前提与正确性约束。
- **Spec B** 是最小仓储契约的唯一接口所有者，负责定义查询/更新接口形态、调用时序与结果契约。
- Spec A 不得在正文中重写具体 repository API 形态。

## 7. 从旧 spec 迁移到新 spec 的建议方式

### Phase 1：建立新骨架

先创建 3 份新 spec 文件骨架，每份只有：

- 目标
- 边界
- 与其他 spec 的依赖
- 初始条款目录

补充裁定：

- “Spec B 的最小共享术语骨架”就是 **Spec B 文件本身的 v0 初稿**。
- 不额外创建第四份临时文档。
- 该 **Spec B v0 最低清单** 必须包含：
  - roots/path 的规范化规则
  - path equality 语义
  - `sameOrChildPath` / path-membership 语义
  - Windows 大小写规则
  - symlink 是否解析
  - `discoveredPaths` 的正式术语定义
  - `cleanupBlocked` 的正式术语定义

执行门槛裁定：

- Spec A 进入 planning 前，Spec B v0 必须满足上述**完整最低清单**。
- 不存在“只满足其中 3 项就可先开 A planning”的弱化门槛。

### Phase 2：按主题迁移内容

将旧单体 spec 中的条文按责任面搬运：

- 身份状态机与 cleanup 保护 → Spec A
- scan phases / single-flight / watcher / ScanResult / TaskBatch → Spec B
- canonical path 的运行时语义定义 → Spec B
- canonical path 的启用 / 迁移 / 门控、upgrade command / marker / feature flag / migrations → Spec C

### Phase 3：删除重复条款

迁移完成后，对 3 份新 spec 执行一次去重：

- 重复定义只保留一处
- 其余位置改成“依赖引用”

### Phase 4：旧 spec 降级

执行顺序固定如下：

1. 创建 archive 全文副本：`docs/superpowers/specs/archive/2026-04-19-image-identity-rebinding-design.full.md`
2. 校验 A/B/C 均已完成内容迁移与去重
3. 校验 A/B/C 均已审查通过
4. 将旧文档原路径改造成“已拆分，请以新 spec 为准”的索引页

## 7.1 planning 前置条件

三份新 spec 是**独立审查单元**，但不是完全无依赖的独立规划单元。

planning 顺序裁定如下：

1. **Spec A 先进入 planning**
   - 前置条件：A 自身审查通过，且 Spec B v0 已按 Phase 1 的完整最低清单起草完成
2. **Spec B 后进入 planning**
   - 前置条件：A 已冻结核心状态机术语
3. **Spec C 最后进入 planning**
   - 前置条件：A/B 已冻结运行时语义与启用前提引用点

因此：

- A/B/C 可以分别审查
- 但 implementation planning 的推进顺序按 **A → B → C** 执行
- 不得误解为三份 spec 可以在无依赖条件下并行规划

补充裁定：

- Spec A 可以先规划主体身份规则，但不得自行重写 canonical path 的运行时语义。
- 这些语义必须以引用方式依赖 Spec B 的共享术语骨架。
- “先审 A”并不意味着 B 完全缺席；B 的最小术语骨架必须先存在，但完整深审仍在 A 之后。

## 8. 审查策略

重构后不应再对旧单体 spec 继续做深审。

正确做法是：

0. 先起草 **Spec B v0**（内容必须满足上文“Spec B v0 最低清单”，不做深审）
1. 先审 Spec A
2. 再审并冻结完整 Spec B
3. 最后审 Spec C

原因：

- Spec B 依赖 Spec A 的状态机术语
- Spec C 依赖前两者决定“上线前提究竟是什么”

补充说明：

- Spec C 只定义 gate failure 的触发条件与 rollout 约束。
- gate failure 如何经 CLI/API/worker/TaskBatch 对外暴露，统一归 Spec B。

## 9. 成功标准

此次重构完成的标志，不是“旧文档更长了”，而是：

1. 新建 3 份 spec，且边界明确
2. 每份 spec 都能独立解释“自己解决什么，不解决什么”
3. 审查意见不再跨三个层次乱跳
4. implementation planning 可以针对单份 spec 开始，而不是必须吞整本法典

## 9.1 新 spec 最低结构模板

为避免拆分产物风格不一、尾部条款遗漏，A/B/C 每份新 spec 至少必须包含以下章节：

1. Goal / Non-goals
2. Owned scope / Out of scope
3. Shared terms & cross-spec dependencies
4. Main rules / contracts
5. Test requirements
6. Risks & trade-offs
7. Rejected alternatives

裁定：

- 拆分执行者不仅要知道“内容搬到哪”，还必须知道“新 spec 至少长成什么结构”。

## 10. 最终裁定

对当前图片身份重绑定规范，最稳的重构方案是：

- **Spec A：核心身份裁决规范**
- **Spec B：扫描入口、并发与结果契约规范**
- **Spec C：升级迁移与 rollout 规范**

该拆分能够以最小制度成本，结束单体 spec 的无限增殖，并把后续实现规划改造成可逐城攻破的硅基施工序列。
