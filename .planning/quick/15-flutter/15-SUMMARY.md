# 快速任务 15 总结: 修复 Flutter 排序按钮

**任务：** 修复 Flutter 桌面程序顶部排序按钮不生效的问题
**日期：** 2026-03-19
**状态：** 已完成

## 修复内容

### 问题诊断

在 `gallery_screen.dart` 的 `_buildSortButton` 方法中，排序按钮点击后使用以下代码匹配枚举值：

```dart
provider.setSort(
  SortField.values.firstWhere((f) => f.name == field),
  asc,
);
```

**问题：** `firstWhere` 在没有找到匹配项时会抛出 `StateError` 异常，导致排序操作失败。

### 修复方案

将枚举匹配改为显式的 if-else 语句：

```dart
SortField? sortField;
if (field == 'createdAt') {
  sortField = SortField.createdAt;
} else if (field == 'filename') {
  sortField = SortField.filename;
} else if (field == 'fileSize') {
  sortField = SortField.fileSize;
}

if (sortField != null) {
  provider.setSort(sortField, asc);
}
```

### 修改文件

- `flutter_app/lib/screens/gallery_screen.dart` (第 175-196 行)
  - 重构 `_buildSortButton` 方法中的排序字段匹配逻辑
  - 添加安全的枚举匹配，避免异常

## 结果

排序按钮现在可以正常工作，用户可以通过顶部工具栏的排序按钮选择不同的排序方式（最新导入、最早导入、文件名 A-Z、文件名 Z-A、文件最大、文件最小）。
