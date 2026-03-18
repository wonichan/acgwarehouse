# Phase 6: 优化与部署 - Context

**Gathered:** 2026-03-18
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段按本次讨论收敛为：以 SQLite 为主的性能优化、单机 Docker 自部署、基础 Web 管理后台、部署文档与性能报告。

原路线图中的 PostgreSQL 迁移支持，本次讨论决定正式移出 Phase 6；后续研究与规划必须按此范围调整，并显式记录与 `ROADMAP.md` 的差异。

本阶段不新增多用户、云同步、Booru 集成等能力，也不把基础 Web 管理后台扩展为完整业务前台替代物。

</domain>

<decisions>
## Implementation Decisions

### 阶段范围调整
- PostgreSQL 正式移出 Phase 6，researcher / planner 不再以 PostgreSQL 交付为本阶段前提。
- 本阶段数据库主路径保持 SQLite，部署与性能验收都以 SQLite 模式为准。
- 后续文档需要显式说明当前讨论结果与 `.planning/ROADMAP.md` 现有 Phase 6 描述存在范围偏差。

### Docker 部署形态
- 目标环境是单机自部署，面向个人电脑、家用主机、NAS 或单台 Linux 主机，不按多机编排设计。
- 部署体验以一次 `docker compose up -d` 启动为目标。
- 图片库数据和 SQLite 数据库都落宿主机可见目录，优先保证备份、排查和迁移方便。
- 配置方式以直接修改 YAML 为主，不把 `.env` 作为主路径。

### 性能验收重点
- 10k+ 图片场景下，优先保证图库滚动浏览体感顺畅。
- 验收口径是“日常使用顺畅”，而不是仅仅可运行或不崩溃。
- AI 与其他后台任务允许比前台慢，但必须稳定、可观察、不中断主要浏览体验。
- 最终交付既包含实际优化，也包含可复现的性能报告和测试方法。

### 基础 Web 管理后台边界
- 管理后台定位为本机/内网使用的基础运维仪表盘，不替代 Flutter 主客户端。
- 页面形态采用单页仪表盘。
- 默认以只读为主，允许少量安全操作：手动触发扫描、重试失败任务、暂停后台任务、刷新系统状态。
- 首页优先展示服务与健康状态、任务队列状态、图库规模信息、存储与路径配置。
- 异常反馈采用明显状态卡片加最近错误列表。
- 访问保护只需简单保护，按本机或内网使用场景处理，不引入完整认证系统。

### OpenCode's Discretion
- Compose 内服务拆分方式，以及 Flutter Web 产物与 Go 服务的具体装配方式。
- 宿主机目录结构命名、默认挂载路径和示例配置组织方式。
- 性能基准工具、测试数据生成方式、报告展示格式。
- 仪表盘的具体布局、视觉表达和安全操作的交互细节。

</decisions>

<specifics>
## Specific Ideas

- 部署体验应尽量接近“改完 YAML 就能一条 `docker compose up -d` 拉起来”。
- 性能优化优先服务日常浏览体验，而不是把 Phase 6 做成纯压测工程。
- 基础 Web 管理后台不要长成第二个 Flutter 客户端，重点是状态查看和轻量运维。

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/config.go`: 已有 YAML 配置加载和环境变量覆盖，可直接支撑容器化配置注入。
- `cmd/server/main.go`: 已有服务启动装配、Gin release mode 切换、依赖初始化和 JobManager 启动逻辑。
- `cmd/scan/main.go`: 已有独立扫描 CLI，可作为后台安全操作或容器内维护动作的接入点。
- `internal/handler/health_handler.go` 与 `internal/handler/routes.go`: 已有 `/health` 和 `/ready` 探针端点，可直接用于部署探活。
- `internal/worker/job_manager.go`: 已有任务队列和状态模型，可为后台仪表盘提供任务状态数据来源。
- `Makefile`: 已有 build/run/test/migrate-up/migrate-down 基础命令，可扩展 Docker 与性能验证入口。

### Established Patterns
- 当前已验证主路径是 Go 后端 + Flutter 前端，保持 Repository / Service / Handler 分层。
- 当前数据库运行时 schema 由 `internal/repository/schema.go` 统一确保，并且明确依赖 SQLite FTS5 与触发器。
- `config.yaml` 当前默认数据库类型是 `sqlite`，PostgreSQL 仅保留配置入口，`cmd/server/main.go` 和 `cmd/scan/main.go` 尚未实现 PostgreSQL 启动。
- 搜索、任务队列、重复检测和收藏夹等核心能力都已经落在现有 SQLite schema 上，性能优化需要围绕这条主路径做增量改进。

### Integration Points
- Docker 部署需要围绕 `cmd/server/main.go` 启动主服务，并挂载 `config.yaml`、SQLite 数据文件路径和图片扫描目录。
- 性能优化需要重点关注驱动 Flutter 图库浏览的查询、缩略图访问和滚动相关接口链路。
- 管理后台可复用健康检查、任务状态和图库统计数据，再补最小量安全操作端点。
- 手动触发扫描、重试失败任务、暂停后台任务等能力应尽量复用现有 scanner / worker / job manager，而不是新增平行系统。

</code_context>

<deferred>
## Deferred Ideas

- PostgreSQL 迁移支持：用户要求正式移出 Phase 6，待后续单独阶段或路线图更新时重新规划。
- 面向公网或多用户的后台认证、细粒度权限控制、反向代理强化配置。
- Kubernetes、多机编排、云上高可用部署。
- 将 Web 管理后台扩展为完整图片/标签/收藏夹业务管理端。

</deferred>

---

*Phase: 06-optimization-deployment*
*Context gathered: 2026-03-18*
