---
phase: 14-backfill-recovery-operations
plan: 01
subsystem: api
tags: [go, gin, sqlite, backfill, ai-tags, admin]

requires:
  - phase: 13-backend-monitoring-queue-control
    provides: task platform service, batch model, admin handler baseline
provides:
  - 过滤感知的 AI 回填预览服务（BackfillCandidateFilter → 分类计数）
  - AI 回填执行服务（仅对合格候选者创建 manual_batch）
  - Admin Handler 的 /admin/api/actions/backfill/preview 和 execute 端点
  - BackfillServiceInterface 接口（可 mock，可测试）
affects: [14-03, admin-ui]

tech-stack:
  added: []
  patterns: [filter-aware-backfill, preview-first-ux, structured-skip-reasons]

key-files:
  created:
    - internal/service/ai_backfill_service.go
    - internal/service/ai_backfill_service_test.go
  modified:
    - internal/repository/image_repository.go
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go
    - internal/handler/routes.go

key-decisions:
  - "回填服务作为独立 AIBackfillService 实现，不嵌入现有 handler"
  - "BackfillServiceInterface 接口允许 handler 直接 mock 测试"
  - "parseBackfillFilter 从 JSON body 提取 tag_ids/has_tags/sort_by/sort_dir"
  - "NewAdminHandlerWithBackfill 构造函数保留原有 NewAdminHandler 向后兼容"

patterns-established:
  - "Preview-first backfill: 先预览分类计数，再执行创建"
  - "Skip-reason classification: has_ai_tag vs has_active_task"
  - "Explicit no-op: 零合格候选返回 Success=false + NoOpReason"

requirements-completed: [AIQ-03]

duration: 45min
completed: 2026-03-29
---

# Plan 14-01: 过滤感知回填预览与执行端点

**过滤感知的 AI 标签回填服务：支持预览（命中数/可入队数/跳过原因分类）和执行（仅对合格候选创建 manual_batch），满足 AIQ-03 需求**

## 性能

- **耗时:** ~45 分钟（含 3 次代理超时后的手动修复）
- **任务:** 2/2 完成
- **修改文件:** 5 个

## 成果

- 专用 AIBackfillService：PreviewBackfill + ExecuteBackfill 方法
- BackfillCandidateFilter：支持 tag_ids、has_tags 筛选
- 4 个分类计数查询（命中数、可入队数、跳过-AI标签、跳过-在途任务）
- Admin Handler 暴露 POST /admin/api/actions/backfill/preview 和 execute
- Handler 层 4 个测试全部通过

## 提交记录

1. **Task 1 (Service + Repository)** - `068837a` feat(14-01): add filter-aware backfill preview and enqueue service with real SQL queries
2. **Task 2 (Handler + Tests + Routes)** - `ba6e8e9` feat(14-01): expose admin backfill preview and execute endpoints

## 关键决策

- 回填服务独立于现有 handler，通过 BackfillServiceInterface 接口解耦
- routes.go 中根据 deps 可用性决定使用 NewAdminHandler 或 NewAdminHandlerWithBackfill
- parseBackfillFilter 从 JSON body 统一解析，并在 handler 层做 narrowing 校验

## 偏差

- 代理执行 3 次超时，由编排器直接完成 handler 端点、路由注册和测试
- 14-02 代理为 14-01 的接口方法提供了 stub 实现，14-01 第一次执行时替换为真实 SQL 查询
- routes.go 恢复了被代理破坏的原始结构，仅新增 2 行 backfill 路由

## 下一步准备

- 回填 preview/execute 端点已就绪，14-03 可直接在 admin UI 中接入
- BackfillPreviewResult 和 BackfillExecuteResult 的 JSON 结构已确定

---
*Phase: 14-backfill-recovery-operations*
*Completed: 2026-03-29*
