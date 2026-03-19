# Quick Task 13: 修复Flutter应用标签筛选无响应问题

**Date:** 2026-03-19
**Status:** 已修复

## 问题分析

在 `tag_filter_drawer.dart` 文件中，`CheckboxListTile` 的 `onChanged` 回调使用了 Consumer builder 参数中的 `provider` 对象：

```dart
onChanged: (checked) {
  provider.toggleTag(tag.id);  // 可能引用旧的 provider 实例
  widget.onFilterChanged?.call(provider.selectedTagIds.toList());
},
```

问题在于当闭包执行时，`provider` 变量可能引用的是 Widget 重建前的旧 provider 实例，导致状态更新失败。

## 修复方案

使用 `context.read<TagProvider>()` 在回调中动态获取当前的 provider 实例：

```dart
onChanged: (checked) {
  final tagProvider = context.read<TagProvider>();
  tagProvider.toggleTag(tag.id);
  widget.onFilterChanged?.call(tagProvider.selectedTagIds.toList());
},
```

## 修改文件

- `flutter_app/lib/widgets/tag_filter_drawer.dart` (第 108-111 行)

## 修复验证

修复确保了：
1. 每次点击标签时都能获取到正确的 provider 实例
2. `toggleTag` 方法能够正确更新状态
3. `onFilterChanged` 回调能够接收到最新的选中标签列表
4. 图片列表会根据选中的标签重新加载
