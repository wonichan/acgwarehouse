---
status: testing
phase: 12-import-task-auto-scheduling
source: [12-01-SUMMARY.md, 12-02-SUMMARY.md, 12-03-SUMMARY.md, 12-04-SUMMARY.md]
started: 2026-03-29T00:00:00+08:00
updated: 2026-03-29T00:04:00+08:00
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

number: 5
name: 配置热重载验证
expected: |
  运行中的服务修改配置（如 AutoScanIntervalMinutes 从5改为10），调度器会使用新配置重启，不会重复启动或遗漏任务。
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: 停止所有运行的服务。清除临时数据库和缓存。从干净状态启动应用。服务启动无错误，数据库迁移完成（image_tags.source 字段存在），健康检查返回正常状态。
result: pass

### 2. AI 自动调度配置项验证
expected: 查看配置示例文件 deploy/config/config.example.yaml，在 ai 配置块下能看到 AutoAITagOnImport、AutoScanIntervalMinutes、AutoScanBatchSize 三个配置项，并有默认值和说明。
result: pass

### 3. 自动调度启动行为验证
expected: 设置 auto_ai_tag_on_import=true，启动服务。有符合资格的图片（有缩略图且无 AI 标签）时，服务会自动扫描并入队 AI 打标任务到 platform_tasks 表。
result: pass

### 4. 批量入队限制验证
expected: 有超过100张符合资格的图片时，调度器首次扫描最多入队100项任务，第二次扫描继续入队剩余图片（排除已入队的）。
result: pass

### 5. 配置热重载验证
expected: 运行中的服务修改配置（如 AutoScanIntervalMinutes 从5改为10），调度器会使用新配置重启，不会重复启动或遗漏任务。
result: [pending]

### 6. AI 标签来源标记验证
expected: AI 自动打标完成后生成的标签在 image_tags 表中 source 字段值为 'ai'，手动添加的标签 source 值为 'manual'。
result: [pending]

## Summary

total: 6
passed: 4
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

[none yet]