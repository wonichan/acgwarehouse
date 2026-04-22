# 图片身份核心裁决规范

## 1. Goal / Non-goals

### Goal

定义图片身份裁决核心规则，包括：

- `path_hit / inserted / rebound / conflict / failed`
- `sha256` 在身份判定中的地位
- 历史空 `sha256`、多新路径争抢旧实体、多旧候选等场景
- 实体级 cleanup 保护规则

### Non-goals

本规范不定义：

- 扫描入口、single-flight、CLI/worker 返回契约
- upgrade command、feature flag、upgrade marker、rollout

## 2. Owned scope / Out of scope

### Owned scope

- 身份状态机
- 业务裁决优先级
- 实体级 cleanup 保护规则
- conflict / failed 的业务语义区分

### Out of scope

- canonical path 的运行时定义
- ScanResult / TaskBatch / CLI 契约
- rollout / migration

## 3. Shared terms & cross-spec dependencies

### Depends on Spec B v0

本规范引用但不重写：

- canonical path equality
- `discoveredPaths`
- `cleanupBlocked`

### Depends on Spec C

本规范只把 feature flag / marker 当启用前提引用，不定义其实现。

## 4. Main rules / contracts

### 4.1 状态机

需定义：

- `path_hit`
- `inserted`
- `rebound`
- `conflict`
- `failed`

### 4.2 核心裁决场景

至少覆盖：

- `path miss + sha hit`
- `path miss + sha miss`
- 多旧候选
- 多新路径争抢单一旧实体
- discovery error 阻断 rebind
- 历史空 `sha256`

### 4.3 cleanup 保护规则

本规范定义：

- 哪些旧实体必须被保护
- 哪些冲突/失败会阻断 cleanup 的业务原因

但不定义：

- cleanup 何时执行
- cleanup 如何通过 `discoveredPaths` / `cleanupBlocked` 落地

这些归 Spec B。

### 4.4 仓储能力语义前提

本规范只定义仓储能力必须满足的**语义前提**，不定义具体接口形态。

## 5. Test requirements

至少覆盖：

- rename / move rebind
- conflict 场景
- 历史空 `sha256`
- cleanup 保护不误删

## 6. Risks & trade-offs

- 规则过宽会把扫描运行时细节重新吞回 A
- 规则过窄会导致 B 无法正确承接执行

## 7. Rejected alternatives

- 继续以 path 作为唯一身份
- 直接把 `sha256` 升级为唯一键
- 用 pHash 做唯一身份
