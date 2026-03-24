# Phase 11: 任务平台基础与批次模型 - Research

**Created:** 2026-03-24
**Status:** Ready for planning

---

## Executive Summary

Phase 11 的目标不是直接把 AI 自动入队做完，而是先把“导入后任务”从零散的 `image_imported` / `thumbnail_generate` / `ai_tag_generation` 触发点，收敛为统一的**批次（batch）+ 平台任务（task）+ 状态流转（lifecycle）**模型。

结合现有代码，最佳方案是：

1. **保留 `async_jobs` 作为执行队列底座**，不要推翻 `worker.Manager`。
2. **新增批次与平台任务语义层**，让批次/平台任务成为面向产品和后台的主模型。
3. **把现有 `async_jobs` 降为执行记录**，通过外键或关联字段与平台任务关联，而不是继续让后台直接看裸 `async_jobs`。
4. **在 `ScannerService` 建立导入批次入口**，统一记录新导入、跳过、重复保护摘要。
5. **在 `bootstrap` 的 `image_imported -> thumbnail_generate` 骨架之上收敛为统一分发入口**，Phase 11 只做骨架，不做 Phase 12 的 AI 自动规则。

这条路线完全遵循当前 Go + SQLite + 单机 worker 约束，不引入新依赖，且与已存在的 Repository / Service / Handler 分层一致。

---

## 1. Existing Code Reality

### 1.1 已有可复用底座

| 位置 | 当前能力 | 对 Phase 11 的意义 |
|------|----------|-------------------|
| `internal/domain/async_job.go` | 统一异步任务实体 | 可继续作为底层执行记录 |
| `internal/repository/job_repository.go` | 任务持久化、按状态查询、失败重置 | 可复用为 worker 队列持久层 |
| `internal/worker/job_manager.go` | 多 worker、Pause/Resume、恢复 ready 任务、处理器注册 | 作为平台执行引擎，不必重写 |
| `internal/service/scanner_service.go` | 扫描导入后只对新图片创建 `image_imported` 任务 | 是建立批次入口和去重汇总的最佳切入点 |
| `internal/app/bootstrap.go` | `image_imported -> thumbnail_generate` 调度骨架、AI handler 注册 | 可演进为统一调度入口 |
| `internal/service/admin_service.go` / `internal/handler/admin_handler.go` | 后台 summary/jobs 查询与控制接口 | 可扩展为批次/任务读模型入口 |

### 1.2 当前缺口

当前系统只有“执行任务”，没有“产品语义上的平台任务”：

- 没有批次实体，无法表达“一次导入 / 一次手动触发 / 一次批量补跑”
- 没有平台任务与执行记录的分层，后台只能看到裸 `async_jobs`
- `image_imported` 更像内部事件，却被当成普通任务记录
- 手动单图 AI 与导入后缩略图仍是平行入口，未收敛到统一模型
- “未变更图片不重复入队”只在导入插入层有弱语义，未升级为平台级规则

---

## 2. Recommended Architecture

### 2.1 模型分层

建议采用三层语义：

1. **Batch（批次）**
   - 表示一次触发动作：导入、单图手动触发、批量补跑
   - 负责来源信息、摘要、跳过统计、总状态

2. **Platform Task（平台任务）**
   - 表示“某批次内、某图片、某任务类型”的业务任务
   - 是后台与后续调度关注的主对象
   - 需要支持类型、状态、去重键、失败原因摘要、与批次关联

3. **Async Job（执行记录）**
   - 表示真正进入 `worker.Manager` 队列执行的一次 job
   - 服务于运行时调度、恢复与并发控制
   - 与 Platform Task 建立关联，避免后台继续直接读裸 job

### 2.2 为什么不要直接扩展 `async_jobs` 充当全部语义

虽然可以给 `async_jobs` 直接加 `batch_id`、`image_id`、`dedupe_key` 等字段，但这会把：

- 用户可见任务语义
- 内部事件语义
- worker 执行语义

全部混在一张表中，后续 Phase 13/14 做可视化、控制动作、失败恢复时会持续产生歧义。

因此更稳妥的设计是：

- `task_batches`：批次主表
- `task_batch_sources`：批次来源根目录/来源明细
- `platform_tasks`：业务任务主表
- `async_jobs`：底层执行表（保留，增加 `platform_task_id` 即可）

---

## 3. Data Modeling Recommendations

### 3.1 批次模型

建议批次字段至少包括：

```text
task_batches
- id
- source_type            # import_scan / manual_single / manual_batch
- trigger_key            # 可选，短窗口合批或追踪来源用
- summary_label          # 系统自动摘要
- status                 # pending / running / completed / partial_failed / failed / cancelled
- total_images
- new_images
- skipped_images
- skipped_unchanged
- skipped_duplicate_tasks
- created_at
- started_at
- finished_at
```

若一次扫描覆盖多个 scan root，建议单独建：

```text
task_batch_sources
- id
- batch_id
- source_root
- source_label
```

这样符合用户锁定决策：**一次扫描即使覆盖多个 root，仍属于同一批次，但要能记录多个来源根目录。**

### 3.2 平台任务模型

建议字段：

```text
platform_tasks
- id
- batch_id
- image_id
- task_type              # thumbnail_generate / ai_tag_generation / ...
- source_type            # import_scan / manual_single / manual_batch
- status                 # pending / queued / running / completed / failed / cancelled / skipped
- dedupe_key             # image_id + task_type + version/hash 派生
- image_version_key      # 基于内容/版本，不基于路径
- latest_async_job_id    # 可空，关联最近一次执行
- skip_reason            # unchanged / already_completed / already_running / missing_prerequisite
- error_summary
- created_at
- queued_at
- started_at
- finished_at
```

其中 `skipped` 是否成为真实任务终态，研究结论是：

- **应允许 `platform_tasks.status = skipped`**，但只为“需要保留任务级痕迹的跳过”使用
- 对大批量“未变更而未建任务”的情况，不要逐条落大量 skipped 记录；改为批次摘要统计

这能兼容 CONTEXT.md 中“可由 OpenCode 决定 skipped 落点”的裁量要求，并控制 SQLite 写放大。

### 3.3 重复保护键

重复保护必须以**图片内容/版本**为主，而不是路径。推荐：

- 优先复用 `images` 中稳定内容标识（若已有文件大小 + path 不够，需要 Phase 11 预留版本键位）
- `dedupe_key = {image_id}:{task_type}:{image_version_key}`
- 对同一 `dedupe_key`：
  - 若已有 `pending/queued/running` → 新批次记录跳过摘要，不重复创建执行任务
  - 若已有 `completed` 且版本未变 → 跳过
  - 若任务类型不同 → 只补缺失任务类型

这直接满足 SAFE-03，并符合“缩略图完成但 AI 未完成时只补缺失类型”的锁定决策。

---

## 4. Lifecycle Design

### 4.1 平台任务状态机

建议统一口径：

```text
pending   -> queued -> running -> completed
pending   -> skipped
queued    -> cancelled
running   -> failed
running   -> completed
failed    -> queued      # Phase 13/14 重试时使用
```

说明：

- `pending`：业务任务已建立，但尚未映射到底层执行队列
- `queued`：已创建/关联 `async_job`，等待 worker 执行
- `running`：worker 正在处理
- `completed` / `failed` / `cancelled` / `skipped`：终态

### 4.2 批次状态聚合规则

批次状态不应手写维护为“导入结束即完成”，应由任务聚合得出：

- 存在 `running/queued/pending` → 批次 `running`
- 全部终态且至少一个 `failed` → `partial_failed` 或 `failed`
- 全部终态且同时存在 `completed/skipped`、无失败 → `completed`
- 全部 `cancelled` → `cancelled`

**推荐保留 `partial_failed`**，因为这是 CONTEXT 中明确锁定的批次语义。

---

## 5. Scheduling and Integration

### 5.1 Phase 11 应做的调度骨架

Phase 11 不实现 AI 自动规则，但应把分发路径统一成：

```text
触发动作
  -> 创建/定位 batch
  -> 计算本次应有的平台任务集合
  -> 应建则创建 platform_tasks
  -> 需要立即执行的 task 进入 async_jobs
  -> worker.Manager 执行
  -> 执行结果回写 platform_tasks / task_batches
```

### 5.2 对现有入口的收敛建议

| 当前入口 | Phase 11 调整方向 |
|----------|------------------|
| `ScannerService` 新图导入后写入 `image_imported` | 改为创建导入批次 + 生成平台任务；`image_imported` 降为内部调度事件或直接取消对外可见性 |
| `createImageImportedHandler()` 里直接创建 `thumbnail_generate` | 改为走统一 task dispatcher，由 dispatcher 决定创建/跳过 thumbnail task |
| `POST /api/v1/images/:id/ai-tags` | 不再直接裸 `AddJob(ai_tag_generation)`，改为创建 `manual_single` 小批次 + 平台任务 |
| `POST /api/v1/images/batch-ai-tags` | 改为创建 `manual_batch` 批次，但完整补跑逻辑留给 Phase 14 |

---

## 6. Read Model Recommendations

Phase 11 已明确要交付后台查询读模型，因此建议在本阶段就提供：

### 6.1 批次列表读模型

字段建议：

- batch id
- source type
- summary label
- source roots / source summary
- total images / new images / skipped images
- by-task-type counters
- by-status counters
- created_at / finished_at
- latest error summary

### 6.2 任务列表读模型

字段建议：

- platform task id
- batch id
- image id
- image path/filename summary
- task type
- status
- skip reason / error summary
- latest async job id
- created_at / finished_at

### 6.3 查询接口边界

Phase 11 只需要为后续后台接入提供读模型 API，不需要先把完整 UI 做出来。接口可遵循现有 `/admin/api/*` 风格，例如：

- `GET /admin/api/task-batches`
- `GET /admin/api/task-batches/:id`
- `GET /admin/api/tasks?batch_id=...`

---

## 7. Recommended File-Level Implementation Shape

建议遵循当前 repository/service/handler 模式，新增如下文件族：

### Domain

- `internal/domain/task_batch.go`
- `internal/domain/platform_task.go`

### Repository

- `internal/repository/task_batch_repository.go`
- `internal/repository/platform_task_repository.go`
- `internal/repository/task_batch_read_repository.go`（若读模型查询较复杂，可拆开）

### Service

- `internal/service/task_platform_service.go` — 批次创建、任务规划、重复保护
- `internal/service/task_read_service.go` — 后台查询聚合

### Integration / Wiring

- 修改 `internal/service/scanner_service.go`
- 修改 `internal/app/bootstrap.go`
- 修改 `internal/handler/ai_tag_handler.go`
- 修改 `internal/service/admin_service.go`
- 修改 `internal/handler/admin_handler.go`
- 修改 `internal/handler/routes.go`

---

## 8. Common Pitfalls

| 风险 | 原因 | 规避策略 |
|------|------|----------|
| 继续把 `async_jobs` 当产品主模型 | 后续批次/控制动作会越来越绕 | 明确分层：平台任务对外，async job 对内 |
| 在批次完成时机上仍沿用“导入结束即完成” | 与后处理实际完成脱节 | 批次状态必须由任务终态聚合 |
| 跳过逻辑逐条落库成海量 skipped task | 大批导入会放大 SQLite 写入 | 大量未变更项仅进批次摘要；必要时才建 skipped task |
| 单图手动 AI 与导入后任务继续双轨 | Phase 12/13 会重复做模型整合 | Phase 11 就收敛到小批次模型 |
| 用路径做重复保护 | 改名/移动会误判 | 使用 image version/content key |
| 调度入口仍散落在 handler / bootstrap 各处 | 未来新增任务类型会继续复制粘贴 | 在 service 层建立统一 dispatcher / planner |

---

## 9. Validation Architecture

### 9.1 测试基础设施

- 测试框架：`go test`
- 现有模式：repository/service/handler 均已有针对 SQLite 的测试
- Phase 11 不需要新增测试框架，只需扩展现有测试族

### 9.2 必测行为

1. **导入批次创建**
   - 扫描导入后创建 1 个批次
   - 多 source roots 仍归属同一批次，但来源明细完整

2. **统一任务建模**
   - 新导入图片会生成对应平台任务
   - 单图手动触发创建 `manual_single` 小批次

3. **状态流转**
   - 平台任务从 pending/queued/running 到终态
   - 批次状态能聚合出 completed / partial_failed

4. **重复保护**
   - 同一图片、同一任务类型、版本未变时不重复入队
   - 若缩略图已完成但 AI 未创建，仅补 AI

5. **读模型查询**
   - 后台能按批次查询汇总
   - 后台能查看某批次下任务列表与状态统计

### 9.3 适合的自动化命令

```bash
go test ./internal/repository/... -run "Task|Batch|Job" -count=1
go test ./internal/service/... -run "Task|Batch|Scanner|Admin" -count=1
go test ./internal/handler/... -run "Admin|AITag" -count=1
```

### 9.4 Nyquist 结论

Phase 11 具备自动化验证基础，可以生成 VALIDATION.md；无需单独 Wave 0 安装测试框架。

---

## 10. Planning Guidance

Phase 11 最适合拆成 4 个顺序计划：

1. **定义批次 / 平台任务契约与 schema**
2. **实现 repository + service 状态流转与重复保护**
3. **接入导入与手动触发的统一调度骨架**
4. **提供后台查询所需读模型接口**

理由：

- 这些工作共享核心文件，天然是串行波次
- 若把读模型和核心状态机混在同一个计划里，会超出上下文预算
- 先固化契约，再接入调度，最后暴露读接口，最适合 `/gsd-execute-phase`

---

*研究完成时间：2026-03-24*
*状态：Ready for planning*
