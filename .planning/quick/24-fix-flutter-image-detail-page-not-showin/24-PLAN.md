# Quick Task 24: 修复Flutter图片详情页AI生成标签不立即显示问题

## 任务描述
通过AI生成标签后，生成的标签不会立即展示在Flutter程序的图片详情页中，需要重新进入页面才能看到生成的标签。

## 问题分析

经过代码审查，发现问题可能出在以下几个方面：

1. **延迟时间可能不足**：当前代码使用500ms延迟，但后端标签写入可能需要更长时间
2. **缺乏重试机制**：单次加载失败后没有重试逻辑
3. **竞态条件**：后端事务提交和前端查询之间的时间窗口

## 后端流程
1. AI任务完成 → 保存observation记录 → 调用MergeTags
2. MergeTags: 查找/创建标签 → 保存image_tags（状态为pending）
3. 这些操作是同步顺序执行的，但数据库写入仍可能有延迟

## 当前前端实现
```dart
if (statusStr == 'completed' || statusStr == 'failed') {
  timer.cancel();
  if (statusStr == 'completed') {
    // 当前只有500ms延迟
    await Future.delayed(const Duration(milliseconds: 500));
    await _loadImageTags();
  }
}
```

## 修复方案

### 方案：增加延迟时间 + 添加重试机制

1. **增加延迟时间**：将500ms增加到1500ms，给后端更充足的写入时间
2. **添加重试机制**：如果第一次加载没有新标签，等待后再次尝试

### 修改文件
- `flutter_app/lib/screens/image_detail_screen.dart`

### 具体修改
1. 修改`_startPolling`方法中的延迟时间
2. 添加智能重试逻辑：检查是否有pending标签增加，如果没有则重试

## 验证步骤
1. 进入图片详情页
2. 点击"生成"按钮触发AI标签分析
3. 等待状态变为"已完成"
4. 验证标签列表是否自动显示新分析出的标签
5. 不需要重新进入页面即可看到新标签
