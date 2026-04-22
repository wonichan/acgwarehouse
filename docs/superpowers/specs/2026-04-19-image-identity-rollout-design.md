# 图片身份升级迁移与 Rollout 规范

## 1. Goal / Non-goals

### Goal

定义新身份规则的启用与迁移方式，包括：

- fresh DB / existing DB 分工
- `EnsureScanSchema`、runtime schema、migrations 边界
- maintenance/upgrade command
- upgrade marker / feature flag / preflight
- rollout 前提、失败语义、可重入性

### Non-goals

本规范不定义：

- 核心身份状态机
- `ScanResult` / TaskBatch / CLI/worker 结果暴露
- canonical path 的运行时语义细节

## 2. Owned scope / Out of scope

### Owned scope

- canonical path 的启用/迁移/门控
- fresh DB / existing DB 职责划分
- schema 结果与 upgrade marker
- maintenance/upgrade command
- feature flag / preflight / rollout gate

### Out of scope

- 运行时路径相等性细节
- 业务裁决状态机
- 扫描结果对外契约

## 3. Shared terms & cross-spec dependencies

### References Spec B

本规范不定义 canonical path 的运行时语义，只引用 Spec B 的定义，并规定何时启用它。

### References Spec A

本规范只把 A 的规则视为“启用后生效的业务法典”，不重写其业务裁决。

## 4. Main rules / contracts

### 4.1 fresh DB / existing DB

需明确：

- fresh DB 由什么创建目标模式
- existing DB 由什么升级到目标模式
- `EnsureScanSchema` 做什么、不做什么

### 4.2 maintenance/upgrade command

需定义：

- schema 补差
- canonical path migration
- upgrade marker 写入
- 失败语义与可重入性

### 4.3 Feature flag / marker / preflight

需定义：

- feature flag 配置位置
- upgrade marker 存储对象
- preflight 何时执行
- gate failure 的触发条件

注意：

- gate failure 如何经 CLI/API/worker 暴露，归 Spec B

### 4.4 分阶段交付

需承接旧单体 spec 中的 Phase 1 / MUST 与 Phase 2 / SHOULD 裁定。

## 5. Test requirements

至少覆盖：

- fresh DB 初始化
- existing DB 升级
- marker / feature flag / preflight
- maintenance command 幂等与失败重跑

## 6. Risks & trade-offs

- rollout 规则过宽会吞掉 B 的对外契约
- rollout 规则过窄会让上线门控失效

## 7. Rejected alternatives

- 不做 upgrade marker
- 让运行时直接猜测是否已升级
- 让 watcher/CLI/server 各自实现独立 gate 逻辑
