# Phase 12: 导入后任务接入与自动调度 - Research

**Researched:** 2026-03-26
**Model:** inherit
**Status:** Ready for planning

---

## Research Question

**What do I need to know to PLAN this phase well?**

如何将 AI 标签任务接入统一任务平台，实现导入完成后的自动入队与条件过滤，同时保持与现有任务平台架构的一致性。

---

## 1. Existing Architecture Analysis

### 1.1 Task Platform Foundation (Phase 11 Complete)

Phase 11 已建立完整的任务平台基础：

**核心组件：**
- `TaskPlatformService`：统一任务规划与入队入口
- `TaskBatch`：批次模型，支持 `import_scan`、`manual_single`、`manual_batch` 来源
- `PlatformTask`：平台任务，支持去重（`image_version_key + task_type`）、生命周期管理
- `async_jobs`：执行层，通过 `platform_task_id` 关联平台语义

**关键方法：**
```go
// TaskPlatformService
PlanBatch(ctx, TaskPlatformPlanRequest) (*TaskPlatformPlanResult, error)
QueueTask(ctx, *PlatformTask, jobType, payload string) (*AsyncJob, error)
MarkJobRunning/Completed/Failed(ctx, jobID) error
```

**现有入口：**
- 导入扫描：创建 `import_scan` 批次，规划缩略图任务
- 手动 AI 触发：创建 `manual_single` / `manual_batch` 批次
- `image_imported`：内部调度触发缩略图生成

### 1.2 Image Tag Model Analysis

**当前 `image_tags` 表结构：**
```sql
CREATE TABLE image_tags (
    image_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    source_observation_id INTEGER,  -- 关联 AI 原始观测，非空 = AI 生成
    confidence REAL,
    review_state TEXT DEFAULT 'pending',
    PRIMARY KEY (image_id, tag_id)
);
```

**现有判定方式：**
- `source_observation_id IS NOT NULL` 表示 AI 生成标签
- 手动添加的标签 `source_observation_id` 为 NULL

**Context.md 决策：**
> 需要为 `image_tag` 表添加 `source` 字段（或 `ai_generated` 布尔字段）

**分析结论：** 现有 `source_observation_id` 已能区分 AI/手动标签，但查询语义不够直观。可保留现有字段作为"AI 来源观测关联"，新增 `source` 字段作为"标签来源类型"枚举，便于后续统计和过滤。

### 1.3 AI Tag Handler Analysis

**现有入队流程 (`ai_tag_handler.go`)：**
```go
func (h *AITagHandler) enqueueAITagBatch(ctx, sourceType, images, prompt) {
    // 1. 构建 TaskPlatformPlanItem 列表
    // 2. 调用 TaskPlatformService.PlanBatch()
    // 3. 为每个创建的任务调用 QueueTask()
    // 4. 通过 jobManager.LoadExistingJob() 加载到 worker
}
```

**可复用点：**
- `enqueueAITagBatch` 的入队逻辑可直接被定时扫描服务复用
- `TaskPlatformService` 已实现去重、批次创建、状态同步

---

## 2. Implementation Strategy

### 2.1 定时扫描服务设计

**触发方式决策：**

| 方案 | 优点 | 缺点 |
|------|------|------|
| goroutine + ticker | 简单，无需外部依赖 | 进程重启后重新计时 |
| 外部 cron | 进程无关，运维可控 | 增加部署复杂度 |
| 数据库调度 | 持久化状态 | SQLite 不适合 |

**推荐方案：** goroutine + ticker，符合 Context.md 决策。进程重启后下次扫描自然补偿。

**实现位置：**
- 新建 `internal/service/ai_tag_auto_scheduler.go`
- 在 `App.Run()` 中启动 goroutine

### 2.2 扫描条件查询

**条件：** 缩略图已完成 + 无 AI 标签关联

**SQL 查询设计：**
```sql
SELECT i.id, i.path, i.source_root, i.file_size, i.phash
FROM images i
WHERE i.thumbnail_small_url IS NOT NULL 
  AND i.thumbnail_large_url IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM image_tags it 
    JOIN tag_observations obs ON it.source_observation_id = obs.id
    WHERE it.image_id = i.id 
      AND obs.evidence_type = 'ai_generated'
  )
ORDER BY i.id
LIMIT ?
```

**优化建议：**
- 为 `images.thumbnail_small_url` 添加索引（已有 IS NOT NULL 过滤）
- 可考虑为 `image_tags.source_observation_id` 添加部分索引

**新增仓库方法：**
```go
// ImageRepository 或新建 AITagAutoSchedulerRepository
FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error)
```

### 2.3 image_tag.source 字段设计

**推荐方案：** 添加 `source` TEXT 字段作为枚举

```go
const (
    ImageTagSourceAI     = "ai"      // AI 生成
    ImageTagSourceManual = "manual"  // 手动添加
)
```

**迁移脚本：**
```sql
-- migrations/003_add_image_tag_source.up.sql
ALTER TABLE image_tags ADD COLUMN source TEXT DEFAULT 'manual';

-- 回填现有数据
UPDATE image_tags SET source = 'ai' WHERE source_observation_id IS NOT NULL;
```

**影响范围：**
- `domain/image_tag.go`：添加 `Source` 字段
- `repository/image_tag_repository.go`：Save/FindByImageID 需处理新字段
- `worker/ai_tag_handler.go`：创建标签时设置 `source = 'ai'`

### 2.4 配置扩展

**新增配置项：**
```yaml
ai:
  auto_ai_tag_on_import: true  # 默认开启
  auto_scan_interval_minutes: 5  # 扫描间隔（分钟）
  auto_scan_batch_size: 100  # 每批最大入队数
```

**配置结构修改：**
```go
type AIConfig struct {
    // ... existing fields ...
    AutoAITagOnImport      bool `yaml:"auto_ai_tag_on_import"`       // 默认 true
    AutoScanIntervalMinutes int `yaml:"auto_scan_interval_minutes"` // 默认 5
    AutoScanBatchSize      int `yaml:"auto_scan_batch_size"`        // 默认 100
}
```

---

## 3. Dependency Analysis

### 3.1 Inbound Dependencies (What this phase needs)

| 依赖 | 来源 | 状态 |
|------|------|------|
| TaskPlatformService | Phase 11 | ✓ 已实现 |
| PlatformTaskRepository | Phase 11 | ✓ 已实现 |
| TaskBatchRepository | Phase 11 | ✓ 已实现 |
| async_jobs.platform_task_id | Phase 11 | ✓ 已实现 |
| TaskBatchSourceImportScan | Phase 11 | ✓ 已定义 |
| AITagConcurrencyLimiter | v2.0 | ✓ 已实现 |
| jobManager | v1.0 | ✓ 已实现 |
| AI handler registration | Phase 11 | ✓ 已实现 |

### 3.2 Outbound Dependencies (What this phase provides)

| 提供 | 消费者 | Phase |
|------|--------|-------|
| 自动入队的 AI 标签任务 | Phase 13 后台监控 | Phase 13 |
| image_tag.source 字段 | 统计、过滤 | Phase 13, 14 |

---

## 4. Risk Assessment

### 4.1 Technical Risks

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 大量图片同时扫描导致队列积压 | 中 | 批量限制 100 张/次，依赖现有 RPM/并发限制 |
| 配置关闭时仍有代码路径触发入队 | 低 | 在调度服务入口检查配置开关 |
| 迁移脚本执行失败 | 低 | 使用 IF NOT EXISTS 语法，幂等迁移 |

### 4.2 Common Pitfalls (from similar implementations)

1. **忘记检查配置开关**：在定时器回调开始时立即检查，而非在入队时
2. **重复入队**：依赖 `FindActiveByDedupeKey` 去重，不自行实现
3. **panic 导致 goroutine 退出**：使用 recover + log.Printf 防止崩溃
4. **context 传播不当**：使用独立的 context.Background() 而非继承请求 context

---

## 5. Task Breakdown Preview

基于以上分析，Phase 12 可分解为以下任务：

### Plan 01: 数据模型与配置扩展
- 为 `image_tags` 表添加 `source` 字段
- 扩展 AI 配置结构
- 实现数据库迁移

### Plan 02: 扫描查询与调度服务
- 新增 `FindImagesWithoutAITags` 查询
- 创建 `AITagAutoScheduler` 服务
- 集成到应用启动流程

### Plan 03: 集成验证
- 端到端验证自动入队流程
- 验证配置开关生效
- 验证去重逻辑

### Plan 04: 真实批次验证
- 使用真实导入批次验证自动调度行为
- 性能监控

---

## 6. Validation Architecture

### 6.1 Test Scenarios

| 场景 | 预期行为 |
|------|----------|
| 导入新图片 + 无 AI 标签 | 下次扫描自动入队 |
| 导入新图片 + 已有 AI 标签 | 不入队 |
| 配置关闭 | 不扫描、不入队 |
| 已有 pending/queued 任务 | 去重跳过 |
| 扫描超过 100 张 | 分批处理，每批最多 100 |

### 6.2 Verification Commands

```bash
# 单元测试
go test ./internal/service/... -run "AITagAutoScheduler" -count=1
go test ./internal/repository/... -run "FindImagesWithoutAITags" -count=1

# 集成测试
go test ./... -run "Phase12" -count=1

# 手动验证
curl -X POST http://localhost:8080/api/v1/scan
# 等待 5 分钟后检查 platform_tasks 表
sqlite3 data/acgwarehouse.db "SELECT COUNT(*) FROM platform_tasks WHERE task_type='ai_tag_generation' AND source_type='import_scan'"
```

### 6.3 Success Criteria Mapping

| 成功标准 | 验证方法 |
|----------|----------|
| 用户导入图片后，符合条件的图片会自动加入 AI 打标签队列 | 扫描后检查 platform_tasks 表有新记录 |
| 默认只有没有 AI 标签的图片会自动进入 AI 打标签队列 | 查询验证 source='ai' 的 image_tags 不在入队列表 |
| 导入后的处理不再依赖逐图人工触发 | 扫描后无需手动触发，任务自动入队 |

---

## 7. Codebase Patterns to Follow

### 7.1 Service Layer Pattern
```go
// 参考 scanner_service.go, task_platform_service.go
type AITagAutoScheduler struct {
    imageRepo      ImageRepository
    taskPlatform   *TaskPlatformService
    jobManager     *worker.Manager
    config         *config.Config
    stopCh         chan struct{}
}
```

### 7.2 Repository Method Pattern
```go
// 参考 image_repository.go 的查询风格
func (r *sqliteImageRepository) FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error) {
    query := `SELECT ... FROM images i WHERE ...`
    // 使用 QueryRowContext + Scan 模式
}
```

### 7.3 Configuration Pattern
```go
// 参考 config.go 的 applyDefaults 模式
func applyDefaults(cfg *Config) {
    if !cfg.AI.AutoAITagOnImport {
        // 默认值在 yaml.Unmarshal 后设置
    }
}
```

---

## 8. Recommendations

### 8.1 Implementation Order

1. **先完成数据模型变更**：确保 `image_tag.source` 字段迁移完成
2. **再实现查询**：`FindImagesWithoutAITags` 是调度服务的基础
3. **最后集成调度服务**：依赖前两者完成后才能完整测试

### 8.2 Testing Strategy

- 使用 TDD 方式实现 `FindImagesWithoutAITags` 和 `AITagAutoScheduler`
- 在集成测试中使用短间隔（1 秒）验证定时触发
- 使用配置开关控制测试环境行为

### 8.3 Don't Hand-Roll

- 任务去重：使用 `TaskPlatformService.PlanBatch` 内置去重
- 任务入队：使用 `TaskPlatformService.QueueTask`
- 状态同步：使用现有 `registerPlatformTaskHandler` 包装器

---

## Research Complete

**Phase 12 研究完成。核心发现：**

1. Phase 11 已建立完整任务平台基础，可直接复用
2. 现有 `source_observation_id` 可区分 AI/手动标签，但新增 `source` 字段语义更清晰
3. 定时扫描使用 goroutine + ticker 方式最简单可靠
4. 需新增 `FindImagesWithoutAITags` 查询方法
5. 需扩展 AI 配置结构支持 `auto_ai_tag_on_import` 等选项

**下一步：** 进入规划阶段，创建 PLAN.md 文件。

---

*Phase: 12-import-task-auto-scheduling*
*Research completed: 2026-03-26*