# Phase 16: Duplicate Detection Migration - Context

**Gathered:** 2026-04-04
**Status:** Ready for planning

<domain>
## Phase Boundary

将重复检测计算能力（SHA256 文件哈希、pHash 感知哈希、汉明距离比较、Union-Find 传递性分组、推荐保留选取）从 Go 直接执行完整迁移到 Python 侧车，删除 Go 侧的计算代码。同时升级哈希算法到高精度 pHash（hash_size=16，256-bit），增强推荐保留策略为多维度评分并提供结构化推荐依据。Go 保留落库、API 层和任务编排主控职责。

不包含：前端 UI 重构（Phase 17+）、完整的 Python 自动降级恢复策略（COMP-05 / Phase 22）、远程部署支持。

</domain>

<decisions>
## Implementation Decisions

### 计算迁移边界
- **D-01:** 全计算迁移——Python 承接全部计算步骤：SHA256 文件哈希计算、pHash 感知哈希计算、汉明距离比较、Union-Find 传递性分组、推荐保留选取。
- **D-02:** Go 侧删除现有 `hash_service.go` 中的哈希计算逻辑和 `duplicate_service.go` 中的计算/分组逻辑，仅保留落库、API 层和查询逻辑。
- **D-03:** 哈希算法升级到高精度 pHash（Python `imagehash` 库，hash_size=16，256-bit），替代原有 64-bit pHash。与历史数据不兼容，首次运行时全量重新计算所有图片的新 pHash。
- **D-04:** `domain.Image` 中的 `PHash int64` 字段需要适配新的 256-bit 哈希值存储（字符串或更大的数值表示）。

### Python ↔ Go 调用契约
- **D-05:** 采用异步任务模式——Go 发启动请求，Python 后台处理并返回任务 ID，Go 轮询进度（百分比），处理完成后 Go 获取结果。
- **D-06:** Go 向 Python 传递图片的本地文件系统绝对路径列表 + 每张图片的元数据（分辨率、文件大小、格式等），Python 直接读取本地文件计算哈希。
- **D-07:** Python 返回完整的分组结果，包含每组的推荐保留项、推荐依据（结构化数据）和每个成员的哈希值/距离信息。Go 拿到结果后直接落库。
- **D-08:** 具体的 HTTP 端点设计、JSON schema 和进度轮询频率由工程师在实现时确定，遵循 Phase 15 已建立的纯计算批量契约模式（D-11/D-13）。

### 推荐保留策略增强
- **D-09:** 从单一分辨率排序升级为多维度评分——包含但不限于分辨率、文件大小、格式优先级等维度，按加权综合评分选取推荐项。
- **D-10:** 推荐依据以结构化数据返回，如 `{"reasons": [{"factor": "分辨率", "value": "1920x1080", "weight": 0.5}, ...], "score": 85}`，前端可根据结构化数据自由展示。
- **D-11:** 具体的评分维度、权重值由工程师在实现时确定。

### Python 不可用时的降级行为
- **D-12:** Python 不可用时，重复检测直接返回明确的错误状态（如"计算服务不可用，请检查 Python 侧车状态"），不尝试本地计算回退。符合 ROADMAP 成功标准第 3 条（可诊断的失败状态）。
- **D-13:** 采用前置检查机制——在触发检测前先通过 Phase 15 已有的 sidecar runtime 状态检查确认 Python 就绪，未就绪则立即拒绝并告知原因，不让用户等到超时。
- **D-14:** 图库的主浏览流程不受重复检测服务不可用影响（延续 Phase 15 D-10 的降级可用语义）。

### the agent's Discretion
- 具体的 HTTP 端点路径和 JSON schema 设计
- 评分维度的具体权重分配
- 进度轮询频率和超时配置
- Python 侧新 pHash 全量重算的并发控制策略
- 256-bit pHash 在 SQLite 中的具体存储方案（hex string / blob 等）
- 异步任务的取消和清理机制

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 16 目标、依赖（Phase 15）和 success criteria（3 条）
- `.planning/REQUIREMENTS.md` — `COMP-03`（计算迁移）、`COMP-04`（推荐保留与依据）的正式定义
- `.planning/PROJECT.md` — v4.0 里程碑原则、"Python 仅承担计算层职责"约束

### Prior Phase Decisions
- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md` — Phase 15 的全部实现决定，特别是 D-11（纯计算批量契约）、D-12（Go 保持编排主控）、D-13（预留批量分析接口外壳）、D-08/D-10（分层健康/降级可用语义）

### Historical Context
- `.planning/phases/04-duplicate-detection-search/04-CONTEXT.md` — 原始重复检测 v1.0 的设计决定（组合检测策略、Union-Find 分组、推荐保留按分辨率）

### Research & Architecture
- `.planning/research/SUMMARY.md` — Phase 16 的关键风险与 fallback 方向
- `.planning/research/ARCHITECTURE.md` — Flutter / Go / Python 三层职责边界
- `ACG-Gallery-Go-Python-Flutter-Technical-Plan.md` — Go↔Python HTTP 本地通信技术规划基线

### Existing Implementation (to be migrated)
- `internal/service/duplicate_service.go` — 当前 Go 侧重复检测完整实现（DetectDuplicates、Union-Find、saveGroups、selectRecommended）
- `internal/service/hash_service.go` — 当前 Go 侧 SHA256 + pHash 计算实现
- `internal/domain/duplicate_group.go` — 重复组/关系/展示 domain model
- `internal/handler/duplicate_handler.go` — 重复检测 HTTP API handler
- `internal/repository/duplicate_repository.go` — 重复组持久化层
- `internal/sidecar/runtime.go` — Phase 15 建立的 sidecar runtime（状态机、启动/停止/探活）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/sidecar/runtime.go`：Phase 15 已建立的 sidecar Runtime 状态机（State: not_started → starting → ready → degraded → stopping → stopped），可直接用于前置检查 Python 就绪状态
- `internal/app/bootstrap.go`：应用装配中心，已有 sidecar runtime 初始化入口
- `internal/handler/duplicate_handler.go`：现有重复检测 API handler（DetectDuplicates / ListDuplicates / GetDuplicate / DeleteDuplicate），需要重构调用层但路由结构可保留
- `internal/repository/duplicate_repository.go`：重复组/关系持久化层，Go 侧保留并继续使用
- `internal/domain/duplicate_group.go`：domain model（DuplicateGroup / DuplicateRelation / DuplicateGroupWithImages / DuplicateImage），需要扩展推荐依据字段
- `internal/service/task_platform_service.go`：v3.0 任务平台，可复用编排模式但不强制要求异步检测走任务平台

### Established Patterns
- Go 已形成"纯计算契约"边界意识（Phase 15），Python 只接纯计算输入、返回纯计算结果
- Sidecar 运行时状态通过 `runtime.State()` / `runtime.Status()` 暴露，可在 handler 层做前置检查
- 重复检测 API 已有完整 CRUD 路由（`/api/v1/duplicates/*`），迁移后保持同样的对外契约
- 管理概览已有 sidecar 状态展示入口（`admin_service.go`），Python 不可用时的诊断信息可以在此扩展

### Integration Points
- `duplicate_service.go` → 重构为 Python 调用层：发启动请求、轮询进度、获取结果、落库
- `hash_service.go` → 删除，计算职责完全移交 Python
- `domain.Image.PHash` → 需要从 `int64` 扩展为能存储 256-bit 值的类型
- `duplicate_handler.go` → 增加前置 sidecar 状态检查、异步检测响应模式
- Python sidecar → 新增重复检测计算端点（接受图片路径 + 元数据，返回分组 + 推荐结果）

</code_context>

<specifics>
## Specific Ideas

- 升级到 256-bit pHash 意味着首次运行会全量重算，大图库下这个过程本身就是一次重量级计算，需要在异步任务中有清晰的进度反馈
- 推荐依据的结构化数据设计应该足够灵活，未来可以方便地增加新的评分维度而不改变 API 契约
- Python 侧的 imagehash 库是 PIL/Pillow 生态的成熟选择，hash_size=16 在社区有广泛使用验证

</specifics>

<deferred>
## Deferred Ideas

- Python 不可用时自动降级到 Go 本地后备计算路径 — COMP-05 / Phase 22 范围
- 增量检测优化（只计算新增图片而非每次全量） — 可作为性能优化在 Phase 22 或后续迭代
- 前端重复检测结果 UI 重构 — Phase 17+ 桌面体验重构范围
- 远程/云端 Python 侧车部署支持 — 超出当前本地单机形态

</deferred>

---

*Phase: 16-duplicate-detection-migration*
*Context gathered: 2026-04-04*
