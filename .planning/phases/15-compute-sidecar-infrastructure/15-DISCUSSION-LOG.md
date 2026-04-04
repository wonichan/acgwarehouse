# Phase 15: Compute Sidecar Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-03T23:40:43.8145039+08:00
**Phase:** 15-compute-sidecar-infrastructure
**Areas discussed:** 启动编排, 地址发现, 健康监控, 首期契约边界, 未来部署边界

---

## 启动编排

| Option | Description | Selected |
|--------|-------------|----------|
| Go 唯一主控 | Flutter 只依赖 Go；Go 负责拉起、等待、探活、关闭 Python | ✓ |
| Flutter 双拉起 | Flutter 同时治理 Go 与 Python 两个后台进程 | |
| 外层 Launcher | 增加更外层启动器统一拉起 Go，再由 Go 拉起 Python | |

**User's choice:** Go 唯一主控
**Notes:** 用户接受 Go 为当前唯一后台主控，但后续不希望因此把 Go / Python 永久绑定为必须跟 Flutter 同机。

---

## 地址发现

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime Manifest | Go 产出运行时地址清单，Flutter 读取后更新 base URL | ✓ |
| Go 固定端口 | 保留 Go 固定 `127.0.0.1:8080`，只有 Python 随机端口 | |
| 环境变量传递 | 通过环境变量或命令行参数把地址传给 Flutter | |

**User's choice:** Runtime Manifest
**Notes:** 用户确认要摆脱固定端口前提，但同时要求该机制不要把未来 Go / Python 云端拆分路线锁死。

---

## 健康监控

| Option | Description | Selected |
|--------|-------------|----------|
| 分层健康 + 降级可用 | `/health` 与 `/ready` 只表示 Go 基础状态，sidecar 细状态进管理概览 | ✓ |
| 聚合 Ready | 只有 Go 和 Python 都好才算 ready | |
| 仅进程存活 | 只检查进程存在，不暴露诊断细节 | |

**User's choice:** 分层健康 + 降级可用
**Notes:** 用户接受 Python 异常不拖死主图库流程，但希望保留清晰诊断入口。

---

## 首期契约边界

| Option | Description | Selected |
|--------|-------------|----------|
| 纯计算批量契约 | Python 只接收纯计算输入并返回纯计算结果，不接触 DB/业务编排 | ✓ |
| 仅 Runtime 管理 | 只做 `/health`、`/shutdown` 等运行时接口，不预留计算接口 | |
| Python 理解业务对象 | 让 Python 接收 `image_id` 或参与落库 | |

**User's choice:** 纯计算批量契约
**Notes:** 用户接受 Phase 15 先预留批量分析型接口壳，但不把业务主键、落库和推荐保留策略下沉到 Python。

---

## 未来部署边界

| Option | Description | Selected |
|--------|-------------|----------|
| 本地优先但可分离 | 当前按单机落地，但不把 Go/Python 必须同机写死为长期前提 | ✓ |
| 永久同机 | 明确锁死为 Flutter/Go/Python 必须同机部署 | |
| 现在就做云端化 | 直接把本阶段提升为支持云端部署能力 | |

**User's choice:** 本地优先但可分离
**Notes:** 用户明确表达未来可能让 Go 与 Python 与 Flutter 桌面程序分离，并可能部署到云端；本阶段只把它写成 future-facing constraint，不扩成当前交付范围。

## the agent's Discretion

- Runtime manifest 的具体格式与落盘位置
- Sidecar 健康探测的具体阈值与字段命名
- 批量计算契约的细字段组织，只要不泄露业务主键语义

## Deferred Ideas

- Go / Python 云端独立部署能力
- Python 侧车完整故障回退与自动恢复策略
- 重复检测能力与推荐保留逻辑的具体迁移
