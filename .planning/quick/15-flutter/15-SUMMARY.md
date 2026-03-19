# 快速任务 15 总结: 修复 Flutter 排序按钮

**任务：** 修复 Flutter 桌面程序顶部排序按钮不生效的问题
**日期：** 2026-03-19
**状态：** 已完成

## 修复内容

### 问题诊断（前端）

在 `gallery_screen.dart` 的 `_buildSortButton` 方法中，排序按钮点击后存在两个问题：

1. **枚举匹配问题：** 使用 `SortField.values.firstWhere((f) => f.name == field)` 在没有找到匹配项时会抛出 `StateError` 异常
2. **Context 获取问题：** 使用 `context.read<ImageListProvider>()` 在 `PopupMenuButton` 的 `onSelected` 回调中可能无法正确获取 provider

### 修复方案（前端）

**修复 1：安全的枚举匹配**

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

**修复 2：使用 Consumer 获取 Provider**

将 `_buildSortButton` 重构为使用 `Consumer<ImageListProvider>` 包裹，确保在正确的上下文中访问 provider：

```dart
Widget _buildSortButton(BuildContext context) {
  return Consumer<ImageListProvider>(
    builder: (context, provider, _) {
      return PopupMenuButton<String>(
        // ... 使用 provider 参数直接调用 setSort
      );
    },
  );
}
```

**修复 3：添加调试日志**

在关键位置添加调试日志以便排查问题：
- `gallery_screen.dart`: 打印排序菜单选择和解析结果
- `image_provider.dart`: 打印 `setSort` 调用参数和 `loadImages` 的 sort 参数

### 问题诊断（后端）

前端代码工作正常，调试日志显示排序参数已正确传递到后端：
```
排序菜单被选中: filename_desc
解析结果: field=filename, asc=false, sortField=SortField.filename
setSort 被调用: field=filename, asc=false
加载图片: offset=0, tagIds=[], sortBy=filename, sortDir=desc
```

但后端返回的图片没有按指定字段排序，因为 **后端 API 根本没有处理排序参数**！

### 根本原因（后端）

1. **ImageHandler.ListImages 没有解析排序参数** - 没有读取 `sort_by` 和 `sort_dir` query 参数
2. **ImageRepository 硬编码排序** - `FindAll` 和 `FindByTagIDs` 方法写死了 `ORDER BY id`
3. **Repository 接口不支持排序** - 方法签名中没有排序参数

### 修复方案（后端）

**修复 1：更新 ImageRepository 接口**

```go
type ImageRepository interface {
    // ... 其他方法
    FindAll(limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
    FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error)
    // ...
}
```

**修复 2：实现动态排序**

在 `FindAll` 和 `FindByTagIDs` 中：
- 验证排序字段（支持：created_at, filename, file_size, id）
- 验证排序方向（asc/desc）
- 使用 fmt.Sprintf 构建带动态排序的 SQL 查询

**修复 3：更新 ImageHandler**

在 `ListImages` 中：
- 解析 `sort_by` 和 `sort_dir` query 参数
- 验证参数有效性
- 将排序参数传递给 Repository

**修复 4：更新所有调用处**

更新以下文件中对新接口的调用：
- `internal/service/search_service.go`
- `internal/service/duplicate_service.go`
- `test/perf/gallery_benchmark_test.go`
- `internal/repository/image_repository_test.go`

### 修改文件

**前端修改：**
1. `flutter_app/lib/screens/gallery_screen.dart`
   - 重构 `_buildSortButton` 方法，使用 `Consumer` 包裹
   - 添加安全的枚举匹配
   - 添加调试日志

2. `flutter_app/lib/providers/image_provider.dart`
   - 在 `setSort` 方法中添加调试日志
   - 在 `loadImages` 方法中添加 sort 参数到调试日志

**后端修改：**
3. `internal/repository/image_repository.go`
   - 更新接口定义，添加排序参数
   - 修改 `FindAll` 实现，支持动态排序
   - 修改 `FindByTagIDs` 实现，支持动态排序

4. `internal/handler/image_handler.go`
   - 在 `ListImages` 中解析和验证排序参数
   - 将排序参数传递给 Repository

5. `internal/service/search_service.go`
   - 更新 `tagSearch` 和 `allImages` 方法，传递排序参数

6. `internal/service/duplicate_service.go`
   - 更新 `FindAll` 调用，添加默认排序参数

7. `test/perf/gallery_benchmark_test.go`
   - 更新所有 `FindAll` 和 `FindByTagIDs` 调用

8. `internal/repository/image_repository_test.go`
   - 更新所有 `FindByTagIDs` 调用

## 结果

排序按钮现在可以正常工作：
- 前端正确传递排序参数
- 后端正确解析并应用排序
- 返回的图片列表按指定字段排序

支持的排序字段：
- `created_at` - 创建时间
- `filename` - 文件名
- `file_size` - 文件大小
- `id` - 图片ID（默认）

支持的排序方向：
- `asc` - 升序
- `desc` - 降序（默认）

## 调试信息

现在可以通过浏览器控制台查看以下调试信息：
- `排序菜单被选中: xxx` - 显示用户选择的排序选项
- `解析结果: field=xxx, asc=xxx, sortField=xxx` - 显示解析后的参数
- `setSort 被调用: field=xxx, asc=xxx` - 显示 setSort 被调用的参数
- `加载图片: offset=xxx, tagIds=xxx, sortBy=xxx, sortDir=xxx` - 显示 API 调用的参数
