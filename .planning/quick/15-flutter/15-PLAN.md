# 快速任务 15: 修复 Flutter 排序按钮

## 任务

修复 Flutter 桌面程序顶部排序按钮不生效的问题

## 变更文件

- `flutter_app/lib/screens/gallery_screen.dart`

## 修复内容

**问题原因：**
在 `_buildSortButton` 方法中，使用 `SortField.values.firstWhere((f) => f.name == field)` 来匹配枚举值时，如果找不到匹配项会抛出异常，导致排序功能失效。

**修复方案：**
使用显式的 if-else 语句安全地匹配 `SortField` 枚举值：
- `'createdAt'` → `SortField.createdAt`
- `'filename'` → `SortField.filename`
- `'fileSize'` → `SortField.fileSize`

同时添加空值检查，确保只在找到有效排序字段时才调用 `provider.setSort()`。

## 验证

- [x] 代码语法正确
- [x] 排序按钮点击后应该能正确切换排序方式
