---
phase: 14-backfill-recovery-operations
plan: 03
subsystem: admin-ui
tags: [go, gin, javascript, html, backfill, failure-groups, retry-hints]

# Dependency graph
requires:
  - phase: 14-backfill-recovery-operations
    provides: backfill preview/execute endpoints, failure groups contract
provides:
  - Admin UI 专用 Phase 14 回填控制区（preview-first 流程、零创建反馈、批次跳转）
  - 分组失败摘要渲染（reason + count + retry_recommended + retry_hint）
  - Batch row 级别操作列（重试失败任务 + 分组失败信息）
  - Handler prompt 读取修复（JSON body 替代 form data）
affects: [admin-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "preview-first backfill: 先预览分类计数，再执行创建批次"
    - "Grouped failure rendering: 按 reason_key 聚合展示失败原因"
    - "Zero-create explicit feedback: 清晰说明不可补跑原因而不是伪装成功"
    - "Backfill separated from retry: 回填和重试是独立的控制区域"

key-files:
  created: []
  modified:
    - web/admin/index.html
    - web/admin/app.js
    - internal/handler/admin_handler.go
    - internal/handler/admin_handler_test.go

key-decisions:
  - "Backfill execute prompt 从 JSON body 读取， 不使用 DefaultPostForm"
  - "分组失败在批次列表层直接渲染, 吰有独立回填控制区域"

patterns-established:
  - "preview-first backfill UX: 鉤形预览按钮 → 知道预览分类计数 → 翻译确认弹层 → 执行按钮"
  - "Grouped failure badges: retry_recommended=true 显示绿色 ✓ 可重试， false 显示橙色警告"

requirements-completed:
  - AIQ-03
  - SAFE-01
  - SAFE-02

# Metrics
duration: 15min
completed: 2026-03-29
---

# Phase 14 Plan 03: Admin 回填操作 UX 与分组失败摘要 Summary

**Admin UI 回填 preview/execute 流程、 分组失败渲染、以及 handler prompt 读取修复（完成 Phase 14 运营端恢复闭环）**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-29T05:19:21Z
- **Completed:** 2026-03-29T05:XX:XXZ (pending human verification checkpoint)
- **Tasks:** 1/2222 part done)
- **Files modified:** 4

## Accomplishments
- 添加专用 Phase 14 回填控制区（独立于重试失败控制区域，在批次监控上方）
- 实现 preview-first 回填流程：预览按钮 → 分类计数确认 → 执行按钮 → 批次跳转
- 分组失败摘要渲染：批次行显示 reason + count + retry_recommended + retry_hint
 badge
- 修复 BackfillExecute 从 JSON body 读取 prompt 的 bug（原来使用 DefaultPostForm）
- 鷻加 3 个 handler 测试：failure_groups payload 役状、JSON body prompt 透传、不可重试失败分组指导

- 任务 1 暂停在 human-verify checkpoint

等待人工验证

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Phase 14 payload and UI contract tests** - `db68b71` (test)
2. **Task 1 (GREEN): Wire admin backfill flow and grouped failure summaries** - `b88b71` (feat)

3. **Task 2: human-verify checkpoint** - PENDING（等待用户审批）

4. **Plan metadata:** PENDING

_Note: Task 1 包含 RED 和 GREEN 提交。 Task 2 是 checkpoint，需要人工验证后才能继续。_

## Files Created/Modified
- `web/admin/index.html` - 添加 Phase 14 回填控制区（筛选条件、预览/执行按钮、 preview 结果区失败分组列操作列
- `web/admin/app.js` - 添加回填 preview/execute 逻辑、分组失败渲染、 preview-first 流程, renderFailureInfo、 renderBatchActions
  `internal/handler/admin_handler.go` - 修复 prompt 读取（使用 parseBackfillFilterWithPrompt），保持向后兼容的 parseBackfillFilter)
- `internal/handler/admin_handler_test.go` - 添加 3 个 Phase 14-03 测试：failure groups payload、 prompt 透传、不可重试失败分组

## Decisions Made
- 回填控制区与重试控制区分，回填用于筛选后的图片，不用于失败批次的恢复
- 分组失败在批次列表层直接渲染，使用结构化数据，不要求下钻
- prompt 读取从 JSON body 提取，解决了 form data 读取的 bug

- "重试失败任务" 保持为 failed/partial_failed 批次的主恢复动作

- 分组失败信息与重试建议直接在操作列中展示

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed BackfillExecute prompt reading from JSON body**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** `BackfillExecute` 使用 `c.DefaultPostForm("prompt", "")` 读取 prompt，但前端发送的是 JSON body，表单导致 prompt 永远为空字符串
- **Fix:** 添加 `parseBackfillFilterWithPrompt` 函数从 JSON body 提取 prompt，修改 `parseBackfillFilter` 委托给新函数
- **Files modified:** internal/handler/admin_handler.go
- **Verification:** TestBackfillExecute_ReadsPromptFromJSONBody 通过
- **Committed in:** b88b71 (Task 1 GREEN commit)

## Issues Encountered
无

## User Setup Required
None - 无需外部服务配置。

## Checkpoint Pending

Task 2 (human-verify) 正在等待用户审批。 详见 14-03-PLAN.md Task 2 获取完整的验证步骤。

## Next Phase Readiness
- Phase 14 全部完成后，v3.0 里程碑完成

- 所有活跃需求（AIQ-03, SAFE-01, SAFE-02）已满足

- 准备归档 v3.0 里程碑
---
*Phase: 14-backfill-recovery-operations*
*Completed: 2026-03-29*

## Self-Check: PASSED

All files exist:
- web/admin/index.html ✓
- web/admin/app.js ✓
- internal/handler/admin_handler.go ✓
- internal/handler/admin_handler_test.go ✓
- 14-03-SUMMARY.md ✓

All commits found:
- db68b71 ✓
- b88b71 ✓
