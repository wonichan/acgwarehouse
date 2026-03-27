---
phase: "13"
plan: "01"
subsystem: "admin-dashboard"
tags: [frontend, monitoring, batch-first, ui-refactor]
requires: [API endpoints for /admin/api/task-batches, /admin/api/tasks]
provides: [Batch-first monitoring UI, Task detail view, Error priority display]
affects: [web/admin/index.html, web/admin/styles.css, web/admin/app.js]
tech-stack:
  added: [Vanilla JS batch filtering, Status badge system, Collapsible sections]
  patterns: [Batch list above task details, Failed-first sorting, Click-to-select interaction]
key-files:
  created: []
  modified:
    - web/admin/index.html
    - web/admin/styles.css
    - web/admin/app.js
decisions:
  - Batch-first layout with filters at top
  - Status and source type filters as dropdowns
  - Failed/running batches sorted to top
  - Failed tasks sorted to top within each batch
  - Collapsible library/config section at bottom
metrics:
  duration: "~15 minutes"
  completed_date: "2026-03-27"
  tasks_completed: 2
  files_modified: 3
  lines_added: 556
  lines_removed: 186
---

# Phase 13 Plan 01: 批次优先监控台布局重构 Summary

## One-liner
重构后台管理页面为批次优先监控布局，实现批次列表筛选、点击查看任务明细、异常优先显示。

## Objective Met
✅ 批次列表在页面主位置，任务明细在下
✅ 状态与来源筛选入口
✅ 点击批次展示任务明细
✅ 失败任务/批次优先显示
✅ 保留30秒自动刷新

## Implementation Details

### Task 1: HTML/CSS 重构 (commit: 71bc12c)

**index.html changes:**
- 标题改为"批次监控台"
- 新增 `batch-monitor-section` 作为主监控区
- 新增 `task-detail-section` 默认隐藏，点击批次后显示
- 新增状态筛选下拉框 (`batchStatusFilter`)
- 新增来源筛选下拉框 (`sourceTypeFilter`)
- 平台概览新增：待处理/运行中/失败/已完成批次卡片
- 图库与配置区改为可折叠 (`collapsible` class)

**styles.css additions:**
- `.filter-bar` 筛选栏样式
- `.batch-table-wrapper` / `.task-table-wrapper` 表格包装器
- `.task-counts` / `.type-counts` 分布统计徽标
- `.status-badge` 状态徽标（pending/running/completed/failed等）
- `.task-detail-section` 任务明细区高亮边框
- `.collapsible` 可折叠区块

### Task 2: JavaScript 逻辑实现 (commit: de2baa6)

**新增函数:**
- `loadBatches()`: 调用 `/admin/api/task-batches` 获取批次列表
- `loadTasks(batchId)`: 调用 `/admin/api/tasks?batch_id=X` 获取任务明细
- `selectBatch()`: 选中批次，显示任务明细区
- `closeTaskDetail()`: 关闭任务明细区
- `renderBatches()`: 渲染批次表格，异常优先排序
- `renderTasks()`: 渲染任务表格，异常优先排序
- `handleFilterChange()`: 处理筛选变化

**排序逻辑:**
- 批次：failed > partial_failed > running > pending > completed > cancelled
- 任务：failed > running > pending > completed > skipped

**DOM 元素更新:**
- 移除旧的 `taskTotal/taskReady/taskRunning/taskFinished/taskFailed`
- 新增 `pendingBatches/runningBatches/failedBatches/completedBatches`
- 新增 `queueState/queueSize` 显示队列状态

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written.

## Verification

### Files Created/Modified
```bash
$ git diff --stat web/admin/
 web/admin/index.html |  118 +++++++++---
 web/admin/styles.css |  222 ++++++++++++++++++++++
 web/admin/app.js     |  317 +++++++++++++++++++++++++++++
 3 files changed, 556 insertions(+), 186 deletions(-)
```

### Key Markers Present
- ✅ `batch-monitor-section` class in HTML
- ✅ `task-detail-section` class in HTML
- ✅ `batchStatusFilter` and `sourceTypeFilter` dropdowns
- ✅ `loadBatches()` function in JS
- ✅ `loadTasks()` function in JS
- ✅ Status badge styles for all states

## Known Stubs
None - all data wired to backend APIs.

## Next Steps
- Plan 13-02: 后端 API 实现批次状态统计
- Plan 13-03: 添加批次详情页（跳转或模态框）
- Plan 13-04: 批次操作（取消、重试）按钮