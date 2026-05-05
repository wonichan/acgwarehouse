# Backend Refactoring Plan

本文档记录后端 Go 代码的重构与优化计划，重点处理 Brooks 分析中发现的高收益技术债：标签治理巨型服务、AI 回填查询规则重复、组合根装配分散、缩略图双实现重复，以及少量 context/接口边界问题。

## 目标

- 降低 `internal/service/tag_admin_service.go` 的变更风险，让标签治理、层级、合并、统计分别有清晰归属。
- 收敛重复 SQL 规则，避免 AI 回填、标签筛选、任务状态判断在多个查询里漂移。
- 保持现有 HTTP API、数据库 schema 和 Flutter/admin 调用契约不变。
- 每一步都能用现有测试保护，避免一次性大改。

## 非目标

- 不在本轮更换 SQLite、Gin、worker 框架或整体目录结构。
- 不重写业务流程，不改变标签层级语义、AI 标签审核语义、缩略图生成策略。
- 不做大规模性能重写；性能优化只在重构后以基准测试验证。

## 优先级总览

| 优先级 | 主题 | 主要文件 | 价值 | 风险 |
| --- | --- | --- | --- | --- |
| P0 | 标签治理拆分 | `internal/service/tag_admin_service.go` | 降低最大热点文件复杂度 | 中 |
| P1 | AI 回填查询规则收敛 | `internal/repository/image_repository.go` | 防止 list/count 规则漂移 | 低 |
| P1 | 组合根收敛 | `internal/app/*`, `internal/handler/routes.go` | 减少重复 new 服务 | 中 |
| P2 | 缩略图策略去重 | `internal/service/thumbnail_service*.go` | 双 build tag 实现保持一致 | 中 |
| P2 | 小接口与 context 清理 | repository/service/handler | 提升可测性和取消语义 | 低 |

## Phase 0: 安全基线

先建立可比较的行为基线，不改业务代码。

1. 记录当前测试命令：
   - `go test ./internal/... ./cmd/...`
   - `go test ./test/e2e/...`
   - `go test ./test/perf/... -run ^$ -bench . -benchmem -count=1`
2. 若本机 Go telemetry/build cache 权限仍报错，临时使用工作区缓存：
   - PowerShell: `$env:GOCACHE="$PWD/.codex-gocache"; $env:GOMODCACHE="$PWD/.codex-gomodcache"`
3. 给标签治理核心路径补齐 characterization tests，只覆盖现有行为：
   - merge 同级标签、禁止跨级 merge、source 有 children 时拒绝。
   - delete tag 时 children detach、image_tags 删除、FTS 同步。
   - governance list 的 usage/source/orphan/filter 组合。

验收标准：测试全绿；新增测试只描述现有行为，不引入新规则。

## Phase 1: 拆分标签治理读模型

目标是先拆只读逻辑，风险最低。

1. 新增 `internal/service/tag_governance_read_service.go`：
   - 移入 `ListGovernanceTags`
   - 移入 `ListGovernanceTagsFiltered`
   - 移入 candidate/filter/page 构建逻辑
2. 新增 `internal/service/tag_stats_query.go`：
   - 移入 `batchResolveDescendants`
   - 移入 `batchComputeHierarchyStats`
   - 移入 direct/tree stats 批量查询
3. 保留 `TagAdminService` 原有 public API，内部委托给新的 read/stat 组件，避免 handler 一起改。
4. 把 `applyMemoryFilters` 改为接收 `ctx context.Context`，禁止内部使用 `context.Background()`。

验收标准：

- `tag_admin_service_test.go` 全绿。
- `ListGovernanceTags*` 的返回 JSON 字段和排序不变。
- `tag_admin_service.go` 行数明显下降，读模型和命令模型不再混在一个文件里。

## Phase 2: 拆分标签命令服务

目标是把合并、删除、层级变更从巨型 service 中分离。

1. 新增 `internal/service/tag_hierarchy_service.go`：
   - 移入 `GetParentCandidates`
   - 移入 `GetTagTree`
   - 移入 `ChangeLevel`
   - 移入 `ReparentTag`
   - 移入 `validateHierarchyAssignment` 和 descendant 检查
2. 新增 `internal/service/tag_merge_service.go`：
   - 移入 `MergeTags`
   - 移入 `mergeImageAssociationsTx`
   - 移入 `mergeAliasesTx`
3. 新增 `internal/service/tag_delete_service.go`：
   - 移入 `GetDeletePreview`
   - 移入 `DeleteTag`
   - 移入 `CleanupUnusedTags`
4. 将 `handler/tag_hierarchy_helper.go` 中的手动标签创建和层级校验迁到 service：
   - 新建 `TagCreationService`
   - handler 只负责解析 HTTP request 和 status code 映射

验收标准：

- `TagHandler` 不再直接实现标签创建业务规则。
- `TagAdminService` 可退化为 facade，或者逐步删除 facade 后让 routes 注入更小接口。
- 标签相关 handler/service 测试全绿。

## Phase 3: 收敛事务与 SQL 边界

目标是让 service 表达业务步骤，repository 负责 SQL。

1. 新增轻量事务接口：

```go
type TxRunner interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}
```

2. 新增标签治理 repository 或 store：
   - `LoadTagForUpdate`
   - `ListChildren`
   - `MoveImageAssociations`
   - `MoveAliases`
   - `DeleteTag`
   - `SyncImageFTS`
3. 替换 `TagAdminService` 内部直接 `*sql.DB` / `*sql.Tx` 使用。
4. `handler.Dependencies` 中移除或缩小 `DB *sql.DB` 的用途；FTS rebuild 可通过 `SearchMaintenanceService` 暴露。

验收标准：

- handler 层不再需要 raw DB 来创建业务服务。
- service 测试可用 fake store 覆盖关键命令路径。
- repository 集成测试继续覆盖 SQL 正确性。

## Phase 4: AI 回填查询规则收敛

目标是消除 `FindBackfillCandidates`、多个 count 方法中的重复规则。

1. 新增 `internal/repository/backfill_query.go`。
2. 抽出共享 SQL fragment builder：
   - base image filter
   - already has accepted AI tag
   - has active AI tag task
   - eligible for backfill
3. 使用 `domain.PlatformTaskTypeAITagGeneration`、`domain.PlatformTaskStatusPending/Queued/Running`、`domain.ImageTagSourceAI` 常量生成查询参数，减少字符串散落。
4. 为 list/count/skipped 三类查询增加表驱动测试，确保规则一致。

验收标准：

- 所有 backfill list/count 只通过同一个 eligibility builder 表达规则。
- 新增 active status 时只需改一个地方。
- `ai_backfill_service_test.go` 和 `image_repository_test.go` 全绿。

## Phase 5: 组合根和路由装配收敛

目标是让 `internal/app` 成为唯一对象装配位置。

1. 在 `App` 中一次性创建：
   - `TaskPlatformService`
   - `AIBackfillService`
   - `TagAdmin` facade 或拆分后的标签服务
   - `SearchMaintenanceService`
2. `handler.SetupRoutes` 只接收已经构造好的服务和 repository，不在 routes 内部 `NewTaskPlatformService` 或 `NewTagAdminService`。
3. 缩小 `handler.Dependencies`：
   - 优先传接口，不传 `*sql.DB`
   - 保留 repository 只给纯 CRUD handler

验收标准：

- `routes.go` 只做路由绑定和 handler 构造，不承担业务服务装配。
- 新增服务依赖时只修改 `app` 组合根。
- `routes_test.go` 和 admin/AI handler 测试全绿。

## Phase 6: 缩略图策略去重

目标是保持 libvips 与 nolibvips 两套底层实现，但共享策略。

1. 新增 `internal/service/thumbnail_policy.go`：
   - small/large 参数
   - dynamic size 决策
   - pre-scale width 决策
   - large 必须大于 small 的压缩循环策略
2. `thumbnail_service.go` 和 `thumbnail_service_nolibvips.go` 只保留解码、resize、encode adapter。
3. 使用现有 thumbnail tests 验证尺寸、质量、small/large 关系。

验收标准：

- 调整尺寸或质量策略只改一个文件。
- libvips/nolibvips 行为差异只存在于图像库 adapter。

## Phase 7: 接口分割与命名清理

低风险收尾，适合穿插在前几个阶段之后。

1. 拆分 `ImageRepository` 大接口：
   - `ImageReader`
   - `GalleryImageQuery`
   - `BackfillImageQuery`
   - `ImageMutationStore`
2. service 构造函数按实际需要接收小接口。
3. 统一标签状态、任务状态、来源字符串到 domain 常量。
4. 保留兼容 facade，避免一次性改动所有调用点。

验收标准：

- 主要 service 的 mock 只需要实现实际使用的方法。
- raw string 状态判断明显减少。

## 推荐执行顺序

1. Phase 0，先建测试基线。
2. Phase 1，拆标签治理读模型。
3. Phase 4，收敛 AI backfill 查询，低风险高收益。
4. Phase 2，拆标签命令模型。
5. Phase 5，收敛组合根。
6. Phase 6 和 Phase 7 作为后续清理。

不要把 Phase 1 到 Phase 5 合成一个 PR。建议每个 phase 单独 PR；Phase 2 还可以按 hierarchy、merge、delete 再拆三个 PR。

## 每个 PR 的检查清单

- API response 字段不变。
- 数据库 schema 不变，除非 PR 明确声明 migration。
- 先移动代码，再改边界；避免移动和行为优化混在同一个提交。
- 为移动前后的 public 方法保留 characterization tests。
- 跑对应测试：
  - 标签治理：`go test ./internal/service ./internal/handler`
  - repository 查询：`go test ./internal/repository`
  - 路由装配：`go test ./internal/app ./internal/handler`
  - 缩略图：`go test ./internal/service -run Thumbnail`

## 风险控制

- 保留 facade 直到调用点全部迁移完成。
- SQL builder 抽取后先做等价测试，不急着优化查询计划。
- 对 FTS 同步、事务提交后副作用、worker job loading 这三类路径特别谨慎；它们容易出现“测试绿但运行时顺序变了”的回归。
- 如果某阶段 PR 超过约 500 行净变更，应继续拆小。
