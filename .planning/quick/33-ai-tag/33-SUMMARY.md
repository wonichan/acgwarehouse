# Quick Task 33: 批量选中图片生成AI tag

**状态:** 完成
**日期:** 2026-03-20
**Commit:** 2bcd38e (前端), 1e001cf (后端并发控制)

## 目标

实现批量选中图片触发 AI 标签生成功能，支持并发控制和进度追踪。

## 实现内容

### 1. 前端：BatchOperationSheet 新增 AI 生成标签按钮

**文件:** `flutter_app/lib/widgets/batch_operation_sheet.dart`

- 新增 `onGenerateAITags` 回调参数
- 添加 "AI生成标签" 按钮，使用 `Icons.auto_awesome` 图标
- 按钮使用紫色调 (`Color(0xFF5E35B1)`) 突出 AI 属性

### 2. 前端：GalleryScreen 集成选择模式

**文件:** `flutter_app/lib/screens/gallery_screen.dart`

- 集成 `SelectionProvider` 管理选择状态
- 长按图片进入选择模式
- 选择模式时 AppBar 显示选中数量和"完成"按钮
- 有选中图片时显示 FloatingActionButton 触发批量操作
- 连接 AI 标签生成 API，成功后显示 SnackBar

### 3. 前端：ImageGrid / ImageMasonry 支持选择

**文件:**
- `flutter_app/lib/widgets/image_grid.dart`
- `flutter_app/lib/widgets/image_masonry.dart`

- 接收 `SelectionProvider` 参数
- 长按进入选择模式并选中当前图片
- 选择模式时显示 Checkbox 覆盖层
- 选中图片有半透明遮罩

### 4. 后端：AI 标签生成并发控制

**文件:**
- `internal/config/config.go` - 添加 `AI.MaxConcurrency` 配置项
- `internal/ai/concurrency_limiter.go` - 创建并发控制器（信号量实现）
- `internal/worker/ai_tag_handler.go` - 集成并发控制
- `internal/app/bootstrap.go` - 初始化并发控制器

**配置项:**
```yaml
ai:
  max_concurrency: 3  # 同时执行的最大AI请求数，默认 3
```

**环境变量:** `AI_MAX_CONCURRENCY`

## 技术要点

### 并发控制（独立于 worker_pool）

- `worker_pool.worker_count` 控制缩略图等通用任务的并发数
- `ai.max_concurrency` **专门控制 AI 标签生成的并发数**
- 使用信号量实现，支持阻塞等待和上下文取消
- 与现有的 `requests_per_minute` 速率限制配合使用

### 标签状态

- 生成的标签默认为 `pending` 状态
- 用户需手动确认标签

### 进度追踪

- 任务可在管理后台 `/admin` 查看
- 显示任务状态：待处理、运行中、已完成、失败

## 测试覆盖

- `flutter_app/test/widgets/batch_operation_sheet_test.dart` - UI 组件测试
- `flutter_app/test/screens/gallery_screen_test.dart` - 页面测试
- `internal/ai/concurrency_limiter_test.go` - 并发控制器单元测试

## 用户操作流程

1. 在图库页面长按任意图片进入选择模式
2. 点击其他图片进行多选
3. 点击浮动按钮"批量操作"
4. 点击"AI生成标签"按钮
5. 系统触发批量 AI 任务，显示成功提示
6. 任务在后台执行，最多 `max_concurrency` 个并发
7. 可在管理后台查看进度
8. 生成完成后，标签显示在图片详情页的待确认区域