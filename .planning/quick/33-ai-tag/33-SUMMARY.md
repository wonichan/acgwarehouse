# Quick Task 33: 批量选中图片生成AI tag

**状态:** 完成
**日期:** 2026-03-20
**Commit:** 2bcd38e

## 目标

实现批量选中图片触发 AI 标签生成功能，支持并发控制和进度追踪。

## 实现内容

### 1. BatchOperationSheet 新增 AI 生成标签按钮

**文件:** `flutter_app/lib/widgets/batch_operation_sheet.dart`

- 新增 `onGenerateAITags` 回调参数
- 添加 "AI生成标签" 按钮，使用 `Icons.auto_awesome` 图标
- 按钮使用紫色调 (`Color(0xFF5E35B1)`) 突出 AI 属性

### 2. GalleryScreen 集成选择模式

**文件:** `flutter_app/lib/screens/gallery_screen.dart`

- 集成 `SelectionProvider` 管理选择状态
- 长按图片进入选择模式
- 选择模式时 AppBar 显示选中数量和"完成"按钮
- 有选中图片时显示 FloatingActionButton 触发批量操作
- 连接 AI 标签生成 API，成功后显示 SnackBar

### 3. ImageGrid / ImageMasonry 支持选择

**文件:**
- `flutter_app/lib/widgets/image_grid.dart`
- `flutter_app/lib/widgets/image_masonry.dart`

- 接收 `SelectionProvider` 参数
- 长按进入选择模式并选中当前图片
- 选择模式时显示 Checkbox 覆盖层
- 选中图片有半透明遮罩

## 技术要点

### 并发控制

- 后端已有协程池 (`job_manager.go`)，默认 4 个 worker
- 可通过 `config.yaml` 的 `worker_pool.worker_count` 调整并发数
- AI 标签生成任务自动进入任务队列

### 标签状态

- 生成的标签默认为 `pending` 状态
- 用户需手动确认标签

### 进度追踪

- 任务可在管理后台 `/admin` 查看
- 显示任务状态：待处理、运行中、已完成、失败

## 测试覆盖

- `test/widgets/batch_operation_sheet_test.dart` - UI 组件测试
- `test/screens/gallery_screen_test.dart` - 页面测试

## 用户操作流程

1. 在图库页面长按任意图片进入选择模式
2. 点击其他图片进行多选
3. 点击浮动按钮"批量操作"
4. 点击"AI生成标签"按钮
5. 系统触发批量 AI 任务，显示成功提示
6. 任务在后台执行，可在管理后台查看进度
7. 生成完成后，标签显示在图片详情页的待确认区域