# Phase 15: Compute Sidecar Infrastructure - Context

**Gathered:** 2026-04-03
**Status:** Ready for planning

<domain>
## Phase Boundary

建立 Go ↔ Python 计算侧车的进程生命周期、通信和就绪基础设施，让桌面应用在当前本地单机形态下能够完成 `Flutter → Go → Python` 启动协同，并让 Flutter 在不依赖固定端口的前提下连接到 Go。重复检测等具体计算迁移、推荐保留策略和完整故障回退路径不属于本阶段。

</domain>

<decisions>
## Implementation Decisions

### 启动编排
- **D-01:** Phase 15 采用 `Go 唯一主控` 模式：Flutter 只依赖 Go 服务入口，不直接治理 Python 侧车生命周期。
- **D-02:** Go 负责 Python 侧车的拉起、等待、探活、关闭与错误回传；Python 继续保持“仅计算层”职责，不扩展为业务主控。
- **D-03:** 当前阶段按本地单机启动链路落地，但不把“Flutter、Go、Python 必须同机且必须是父子进程”写死为长期架构前提。

### 地址发现
- **D-04:** Phase 15 采用 `Runtime Manifest` 作为桌面形态下的地址发现机制：由 Go 产出运行时地址信息，Flutter 读取后更新 Go base URL。
- **D-05:** Flutter 不能再把 `localhost:8080` 视为固定真实地址；固定端口只允许作为开发期兜底，不作为阶段目标。
- **D-06:** Python 地址只作为 Go 的内部依赖暴露给编排层消费，不直接暴露给 Flutter 作为产品层契约。
- **D-07:** Runtime manifest 只是当前桌面分发形态下的发现手段，不代表未来 Go / Python 只能本地部署。

### 健康监控
- **D-08:** Phase 15 采用 `分层健康 + 降级可用` 语义：`/health` 只表示 Go 进程存活，`/ready` 只表示 Go 自身可接流量。
- **D-09:** Python sidecar 的运行状态、最近一次探测结果、错误摘要等细粒度诊断信息进入管理概览，而不是并入基础 `/health` 语义。
- **D-10:** Python 不可用时系统在本阶段定义为“降级可用”，不因为计算侧车异常而阻断图库主流程；完整恢复与回退策略留给后续阶段扩展。

### 首期契约边界
- **D-11:** Phase 15 锁定 `纯计算批量契约`：Python 只接收 Go 预处理后的纯计算输入并返回纯计算结果，不接触数据库、批次语义和业务对象持久化。
- **D-12:** Go 继续保留任务编排、批次管理、落库、审计和后续推荐保留策略的主控职责；Phase 15 不把业务 ID 语义下沉到 Python。
- **D-13:** Phase 15 可以预留批量分析型接口外壳，为 Phase 16 的重复检测迁移铺路，但不在本阶段完成具体重复检测能力迁移。

### 部署演进约束
- **D-14:** 当前阶段以本地桌面单机为默认交付形态，但要求接口与发现机制保持“Flutter 依赖 Go 服务契约，而不是依赖 Go / Python 必须为本机子进程”的可分离性。
- **D-15:** 未来如果 Go 与 Python 独立部署到云端，应优先复用 Go 对外服务契约和 Python 纯计算边界，而不是推翻当前职责拆分。

### the agent's Discretion
- Runtime manifest 的具体落盘位置、命名和清理策略
- Go 对 Python 探活的具体频率、超时阈值与错误字段组织
- 批量计算契约的字段命名，只要不泄露数据库与业务主键语义

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 15 的目标、依赖与 success criteria；同时约束 Phase 16/20/21 的后续承接关系
- `.planning/REQUIREMENTS.md` — `COMP-01`、`COMP-02`、`COMP-06` 的正式定义，以及“禁止硬编码固定端口”“Python 不直接持久化数据库”等边界
- `.planning/PROJECT.md` — v4.0 里程碑原则、`Python 仅承担计算层职责`、Windows 单机可打包/可诊断/可回退约束
- `.planning/STATE.md` — 当前阶段位置、v4.0 相位顺序与 Phase 15 风险提示

### Research & Architecture
- `.planning/research/SUMMARY.md` — Phase 15 的关键风险、随机端口、HTTP localhost、WAL、进程治理与诊断方向
- `.planning/research/STACK.md` — Python sidecar、Go 进程管理与 Windows 打包相关技术栈研究
- `.planning/research/ARCHITECTURE.md` — Flutter / Go / Python 三层职责边界与推荐结构
- `.planning/research/PITFALLS.md` — 僵尸进程、端口冲突、侧车故障诊断与生命周期失控等高风险陷阱

### Historical & Technical Plan
- `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md` — Go↔Python HTTP 本地通信、Go 拉起 Python、随机端口与打包链路的技术规划基线

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/bootstrap.go`：当前 Go 应用初始化装配中心，适合作为侧车 runtime / process manager 接入点
- `internal/app/app.go`：已有统一 shutdown 语义，适合作为 Go 主控关闭链路的宿主
- `internal/service/admin_service.go`：已有管理概览聚合点，适合扩展 sidecar 诊断与状态摘要
- `internal/service/task_platform_service.go`：v3.0 已完成 batch/task 平台建模，后续计算迁移可复用编排层而不是重做任务语义
- `internal/worker/job_manager.go`：已有 Worker/队列治理能力，可作为后续计算任务接入时的编排背景设施

### Established Patterns
- Go 已经是业务初始化、生命周期与恢复逻辑中心，说明 Phase 15 继续收口在 Go 最符合现有仓库演进方向
- Flutter 现有 `api_config.dart` 仍带有 `localhost:8080` 的固定入口思路，说明地址发现必须在本阶段显式校正
- 仓库已形成“管理总览与操作入口分层”的后台模式，说明 sidecar 诊断应优先进入 overview，而不是污染基础健康接口
- v3.0 已锁定批次优先、失败隔离和 overview 契约，说明计算侧车迁移应复用既有编排而不是新增一套并行平台

### Integration Points
- Flutter 启动层需要在应用真正连接 API 前读取 Go 提供的运行时地址信息
- Go 初始化链需要新增 sidecar runtime / process manager，并把 readiness 与 shutdown 接入现有 App 生命周期
- Admin overview 需要增加 sidecar 状态、最近错误摘要与探测结果，为 Phase 20 继续扩展预留入口
- Duplicate detection 相关 Go service 后续会是最先消费纯计算批量契约的业务入口，但能力迁移发生在 Phase 16

</code_context>

<specifics>
## Specific Ideas

- 当前阶段按本地桌面单机交付，但应避免把“Go/Python 必须始终跟 Flutter 同机”固化成长期前提
- Flutter 长期应依赖 Go 的服务契约与地址发现机制，而不是依赖本机固定端口或本机子进程假设
- Runtime manifest 仅是桌面形态下的发现桥梁，后续如果 Go / Python 云端部署，优先保留契约一致性而不是保留本地实现细节

</specifics>

<deferred>
## Deferred Ideas

- Go / Python 云端独立部署能力 — 属于未来部署形态演进，不在 Phase 15 当前交付范围内
- Python 侧车完整故障回退与自动恢复策略 — 需求映射到后续 `COMP-05` / 相关运营阶段继续细化
- 重复检测具体迁移、推荐保留项与推荐依据 — 属于 Phase 16 范围

</deferred>

---

*Phase: 15-compute-sidecar-infrastructure*
*Context gathered: 2026-04-03*
