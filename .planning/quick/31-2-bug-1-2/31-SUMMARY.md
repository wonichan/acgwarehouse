# Quick Task 31 Summary: 标签管理页 Bug 修复和删除功能

**状态**: ✅ 已完成
**提交**: 56b93ed
**完成时间**: 2026-03-20

---

## 修复内容

### 1. 后端修复 - 避免级联删除 (Bug 1)

**问题**: `Save` 函数使用 `INSERT OR REPLACE`，更新标签时会先 DELETE 再 INSERT，触发外键级联删除 `image_tags` 记录。

**解决方案**: 
- 在 `tag_repository.go` 中添加 `Update` 方法，使用 `UPDATE` 语句
- 修改 `tag_handler.go` 的 `UpdateTag` 使用新的 `Update` 方法

**验证**: 后端测试通过

---

### 2. 后端添加重名检查 (Bug 2)

**问题**: 创建和更新标签时没有检查重名，可能覆盖其他标签。

**解决方案**:
- `CreateTag`: 添加检查，如果标签名已存在返回 409 Conflict，消息 "标签名已存在"
- `UpdateTag`: 添加检查，如果新名称已被其他标签使用返回 409 Conflict

**验证**: 后端测试通过

---

### 3. 前端添加删除标签按钮

**文件修改**:
- `tag_service.dart`: 添加 `deleteTag` 方法
- `tag_provider.dart`: 添加 `deleteTag` 方法  
- `tag_management_screen.dart`: 
  - 在 PopupMenuButton 添加"删除标签"选项
  - 添加 `_showDeleteTagDialog` 方法，显示确认对话框
  - 删除后刷新列表并显示成功提示

---

### 4. 前端处理重名错误

**文件**: `tag_management_screen.dart`

**修改**: 在 `_showEditTagDialog` 中捕获 409 错误，显示中文提示 "标签名已存在，请使用其他名称"，并重新打开编辑对话框让用户修改。

---

## 测试验证

- 后端测试: ✅ 全部通过
- Flutter 分析: ✅ 无错误（仅有已有的警告）

---

## 提交

```
56b93ed fix: 修复标签管理页编辑bug并添加删除功能
```

**修改文件**:
- internal/handler/tag_handler.go
- flutter_app/lib/services/tag_service.dart
- flutter_app/lib/providers/tag_provider.dart
- flutter_app/lib/screens/tag_management_screen.dart