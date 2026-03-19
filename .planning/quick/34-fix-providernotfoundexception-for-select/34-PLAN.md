---
task_type: quick
task_id: 34
created: 2026-03-20
status: complete
---

# Quick Task 34: 修复批量操作页 ProviderNotFoundException

## 目标

修复长按图片点击批量操作时的 `ProviderNotFoundException` 错误。

## 问题分析

`showModalBottomSheet` 创建了一个新的 widget 树，这个树不在 `GalleryScreen` 的 `MultiProvider` 范围内，导致 `BatchOperationSheet` 中的 `Consumer<SelectionProvider>` 找不到 provider。

## 任务列表

### Task 1: 修改 BatchOperationSheet 组件

**Files:**
- `flutter_app/lib/widgets/batch_operation_sheet.dart`

**Action:**
- 将 `Consumer<SelectionProvider>` 改为接收 `SelectionProvider` 作为必需参数
- 移除 `provider` 包的导入（不再需要 `Consumer`）
- 更新 `show` 静态方法添加 `selectionProvider` 参数

**Verify:** LSP 无错误

**Done:** ✅ 完成

---

### Task 2: 更新 GalleryScreen 调用

**Files:**
- `flutter_app/lib/screens/gallery_screen.dart`

**Action:**
- 在 `_showBatchOperations` 方法中使用 `context.read<SelectionProvider>()` 获取 provider
- 将 provider 传递给 `BatchOperationSheet.show()`

**Verify:** LSP 无错误

**Done:** ✅ 完成

---

### Task 3: 更新测试文件

**Files:**
- `flutter_app/test/widgets/batch_operation_sheet_test.dart`

**Action:**
- 更新测试以传递 `selectionProvider` 参数
- 移除不必要的 `ChangeNotifierProvider` 包装

**Verify:** 测试通过

**Done:** ✅ 完成