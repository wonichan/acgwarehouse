# Quick Task 36: 批量操作弹框添加/移除标签按钮无响应

**Status:** Ready for execution
**Created:** 2026-03-19

## Problem

`gallery_screen.dart` 中调用 `BatchOperationSheet.show` 时，`onAddTags` 和 `onRemoveTags` 回调参数没有传递，导致按钮点击无响应。

## Root Cause

```dart
// gallery_screen.dart:297-304
Future<void> _showBatchOperations(BuildContext context) {
  return BatchOperationSheet.show(
    context,
    selectionProvider: selectionProvider,
    onGenerateAITags: () => _generateAITags(context),
    // ❌ Missing: onAddTags, onRemoveTags
  );
}
```

## Tasks

### Task 1: 实现批量添加标签功能

**Files:**
- `flutter_app/lib/screens/gallery_screen.dart`

**Action:**
1. 创建 `_batchAddTags()` 方法，显示对话框让用户选择/输入标签
2. 调用 `TagService.addImageTag` 为所有选中图片添加标签
3. 在 `_showBatchOperations` 中传递 `onAddTags: () => _batchAddTags(context)`

**Verify:**
- 长按图片 → 批量操作 → 添加标签 → 弹出标签选择对话框
- 选择标签后，所有选中图片都添加了该标签

**Done:**
- 按钮有响应，标签成功添加

---

### Task 2: 实现批量移除标签功能

**Files:**
- `flutter_app/lib/screens/gallery_screen.dart`

**Action:**
1. 创建 `_batchRemoveTags()` 方法，显示对话框让用户选择要移除的标签
2. 调用 `TagService.removeImageTag` 为所有选中图片移除标签
3. 在 `_showBatchOperations` 中传递 `onRemoveTags: () => _batchRemoveTags(context)`

**Verify:**
- 长按图片 → 批量操作 → 移除标签 → 弹出标签选择对话框
- 选择标签后，所有选中图片都移除了该标签

**Done:**
- 按钮有响应，标签成功移除

---

## must_haves

- `flutter_app/lib/screens/gallery_screen.dart` - 添加 `_batchAddTags` 和 `_batchRemoveTags` 方法
- 按钮点击有响应（不是 null）
- 后端收到 API 请求