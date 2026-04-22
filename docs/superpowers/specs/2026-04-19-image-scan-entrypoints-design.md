# 图片扫描入口、并发与结果契约规范

## 1. Goal / Non-goals

### Goal

定义图片扫描运行时语义，包括：

- `ScannerService.Scan` 的阶段模型
- 入口统一门控与 single-flight
- canonical path 的运行时定义
- `ScanResult` / TaskBatch / CLI / worker 的结果契约

### Non-goals

本规范不定义：

- `path_hit / inserted / rebound / conflict / failed` 的业务裁决语义本身
- 升级迁移、feature flag、upgrade marker、rollout 流程

这些内容分别归：

- 核心身份裁决 → Spec A
- 升级迁移与 rollout → Spec C

## 2. Owned scope / Out of scope

### Owned scope

- 扫描入口：CLI / server / worker / watcher
- 三阶段扫描模型
- cross-process single-flight / lock / coordinator
- canonical path 的运行时语义
- `discoveredPaths` / `cleanupBlocked` / `ScanResult` / TaskBatch 契约
- 最小仓储契约的接口形态、调用时序、结果契约
- `sha256/source_mtime_unix` 采集、缓存刷新、path-hit 补齐历史空 hash 的扫描链路语义

### Out of scope

- 核心身份状态机归属与实体保护哲学
- upgrade command / marker / feature flag 的治理规则

## 3. Shared terms & cross-spec dependencies

### Shared terms owned here

本规范拥有以下运行时语义：

- canonical path equality
- roots/path 规范化规则
- `sameOrChildPath` / path-membership
- Windows 大小写规则
- symlink 是否解析
- `discoveredPaths`
- `cleanupBlocked`

### Dependencies

- Spec A：引用其状态机术语与实体级 cleanup 保护规则
- Spec C：引用其 gate/marker/feature flag 启用前提

## 4. Main rules / contracts

### 4.1 Spec B v0 最低清单

在 Spec A 进入 planning 前，Spec B 至少必须稳定以下术语：

1. roots/path 规范化规则
2. path equality 语义
3. `sameOrChildPath` / path-membership 语义
4. Windows 大小写规则
5. symlink 是否解析
6. `discoveredPaths` 的正式定义
7. `cleanupBlocked` 的正式定义

### 4.2 入口统一门控

需覆盖：

- `cmd/scan`
- server manual scan
- worker scan job
- watcher / 单文件入口

并明确：

- feature flag 与 upgrade marker 的前置检查点
- gate failure 如何对外暴露
- watcher 是否触发 full scan 或被禁用

### 4.3 三阶段扫描模型

至少拆分为：

- Phase A：发现
- Phase B：裁决 / writer
- Phase C：cleanup

### 4.4 结果契约

至少定义：

- `ScanResult`
- CLI 退出语义
- worker / admin / API 的部分失败表现
- TaskBatch 与 partial failure / top-level failure 的关系

### 4.5 最小仓储契约

本规范是最小仓储契约的唯一接口所有者。

需要明确：

- 需要哪些查询 / 更新接口
- 接口返回值与错误语义
- 与 scan-local arbitration / writer phase 的配合方式

## 5. Test requirements

至少覆盖：

- 入口门控
- single-flight
- partial failure 暴露
- TaskBatch 在空 planningItems 与 partial failure 下的状态
- canonical path 运行时语义

## 6. Risks & trade-offs

- 运行时契约过薄会迫使 Spec A 重写扫描语义
- 入口太多会导致 single-flight 策略失控
- 结果契约若不统一，会出现 CLI / worker / API 语义分裂

## 7. Rejected alternatives

- 将 canonical path 运行时语义继续放在 rollout spec
- 将最小仓储契约一半放 A、一半放 B
- 允许 watcher 继续绕过 full scan 直接写入新逻辑
