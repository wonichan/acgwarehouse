# Phase 1: 基础架构、图片扫描与标签基础层 - Context

**Gathered:** 2026-03-14
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付 Go 后端基础骨架、图片扫描与文件夹监控导入链路、标签治理基础 schema，以及为后续 AI 标签处理预留的异步任务状态底座。

本阶段不包含 Flutter 浏览界面、缩略图展示体验设计，也不包含完整 AI 打标与标签归并流程本身。

</domain>

<decisions>
## Implementation Decisions

### 导入后存储路径
- Phase 1 采用保留原路径索引的方式管理图片，不在导入时接管文件到受管库。
- 默认只建立对原文件的引用并提取元数据，不复制也不移动原文件。
- 完整保留来源信息，包括原始路径、来源根目录和原文件名，便于补扫、监控和问题回溯。
- Phase 1 不建立受管目录结构；如后续需要受管库存储，再单独设计迁移策略与目录约定。

### 扫描与监控入口
- 首次导入与手动补扫以 CLI 为主，同时预留 API 触发入口供后续前端或管理端调用。
- 文件夹监控采用“实时监听 + 定时补扫”的组合模式，而不是只监听或只轮询。
- 监控对象采用显式配置的多个根目录，而不是单一图库根目录或任意临时目录。
- 单个文件处理失败时默认跳过并记录，整体扫描/监控任务继续执行。

### 标签基础模型
- AI 或导入链路产生的标签文本先作为原始观测落库，不直接作为长期标准标签契约。
- 标准标签采用宽松大类治理，而不是固定 booru 式细分类或完全无分类。
- Phase 1 先建立 `tag_aliases` 等别名关系能力，但不提前实现复杂自动归并规则。
- 只有已审核或已提升的标签进入主筛选器和自动补全，避免原始观测直接污染检索体验。

### OpenCode's Discretion
- 异步任务底座先覆盖“图片已导入”事件排队与状态记录，复杂优先级、取消与重试策略只做最小可扩展设计。
- 迁移工具与具体数据库访问层可在规划阶段结合 Go 工具链再定，但必须满足 SQLite / PostgreSQL 双模式演进。
- 文件监听的 debounce 时长、批处理间隔、重扫窗口等运行参数由规划阶段细化。

</decisions>

<specifics>
## Specific Ideas

- 当前产品方向更偏向“围绕现有图库做编目”，而不是在 Phase 1 就立即接管所有图片文件。
- 扫描链路应先做成可靠的命令行能力，再给后续 UI 或后台调用提供 API 钩子。
- 标签系统从一开始就保留原始观测与治理层分离，避免后续 AI 标签漂移导致搜索契约失稳。

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.planning/research/ARCHITECTURE.md`: 给出 Go 后端推荐目录结构，包含 `cmd/`、`internal/config/`、`internal/domain/`、`internal/handler/`、`internal/service/`、`internal/repository/`、`internal/worker/`。
- `.planning/research/STACK.md`: 给出 Phase 1 相关推荐栈，包括 Gin、`ncruces/go-sqlite3`、`pgx`、`imagemeta` 等。
- `.planning/research/PITFALLS.md`: 明确约束了文件系统存图、原始观测与标准标签分层、`fsnotify + debounce + 状态持久化` 等关键方向。
- `.planning/PHASE1_PATTERNS.md`: 汇总了外部实现模式，包含 `tag_aliases` 独立表、任务状态枚举/队列管理，以及双 goroutine 扫描架构等可复用参考。

### Established Patterns
- 当前仓库是 greenfield，尚无后端/前端源码；`.planning` 文档就是当前唯一已验证的结构与模式来源。
- 图片文件不进入数据库 BLOB，数据库只保存路径与元数据，这是已锁定约束。
- 标签体系采用“原始观测 -> 标准标签/别名 -> 图片标签关联”的分层治理方向。
- 扫描链路以基础设施优先：CLI 先落地，API 作为后续集成点保留。

### Integration Points
- 扫描 CLI / API 入口需要连接到图片元数据提取、数据库写入和监控根目录配置。
- 文件夹监听需要与增量补扫、来源根目录记录和失败日志/状态记录联动。
- “图片已导入”事件需要接入异步任务状态模型，为后续 AI 标签任务、缩略图任务等后台处理预留入口。
- 标签 schema 需要同时支撑 Phase 1 的观测落库，以及 Phase 3 的归并、别名解析和筛选准入。

</code_context>

<deferred>
## Deferred Ideas

- 受管库存储目录如何设计，以及是否提供“从引用模式迁移到受管库存储”的能力，留待后续单独阶段或后续规划讨论。
- 复杂自动归并、近义词推荐、标签审核工作流细节，留待 Phase 3 细化。
- 异步任务的高级能力（优先级、取消、并发度策略、重试语义）留待后续规划时按任务类型展开。

</deferred>

---

*Phase: 01-foundation-scan-tag-base*
*Context gathered: 2026-03-14*
