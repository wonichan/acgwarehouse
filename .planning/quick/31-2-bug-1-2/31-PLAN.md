# Quick Task 31: 标签管理页 Bug 修复和删除功能

**状态**: 待执行
**创建时间**: 2026-03-20
**目标**: 修复标签管理页的编辑 bug 并添加删除功能

---

## 任务列表

### Task 1: 修复后端 UpdateTag - 避免级联删除问题

**文件**: 
- `internal/repository/tag_repository.go`

**问题**: 
`Save` 函数使用 `INSERT OR REPLACE`，在更新标签时会触发 DELETE 操作，导致 `image_tags` 表的级联删除。

**修改方案**:
1. 在 `TagRepository` 接口中添加 `Update` 方法（只更新字段，不使用 INSERT OR REPLACE）
2. 修改 `tag_handler.go` 中的 `UpdateTag`，改用 `Update` 方法

**验证**:
- 修改标签名后，数据库中 image_tags 表的关联记录应保持不变
- 运行后端测试: `go test ./internal/repository/... ./internal/handler/...`

**完成条件**:
- 修改标签名后，图片与标签的关联保持不变
- 现有测试通过

---

### Task 2: 后端添加标签重名检查

**文件**: 
- `internal/handler/tag_handler.go`

**问题**: 
`CreateTag` 和 `UpdateTag` 没有检查标签名是否已存在，导致可能覆盖其他标签。

**修改方案**:
1. 在 `CreateTag` 中添加检查：如果 `preferred_label` 已存在，返回 409 Conflict 错误
2. 在 `UpdateTag` 中添加检查：如果新名称已被其他标签使用，返回 409 Conflict 错误
3. 返回中文错误消息: "标签名已存在"

**验证**:
- 创建同名标签时返回 409 错误
- 更新为已存在的名称时返回 409 错误
- 运行后端测试

**完成条件**:
- 重名时返回明确的错误提示
- 现有测试通过

---

### Task 3: 前端添加删除标签按钮

**文件**: 
- `flutter_app/lib/screens/tag_management_screen.dart`
- `flutter_app/lib/services/tag_service.dart`
- `flutter_app/lib/providers/tag_provider.dart`

**问题**: 
标签管理页只有编辑功能，没有删除功能。

**修改方案**:
1. 在 `tag_service.dart` 中添加 `deleteTag` 方法
2. 在 `tag_provider.dart` 中添加 `deleteTag` 方法
3. 在 `tag_management_screen.dart` 的 PopupMenuButton 中添加"删除标签"选项
4. 删除前显示确认对话框
5. 删除成功后刷新列表并显示提示

**验证**:
- 点击删除按钮显示确认对话框
- 确认后删除标签及关联
- 刷新列表后标签消失

**完成条件**:
- 删除按钮功能正常
- 删除后关联的图片标签也被清除
- 用户看到成功提示

---

### Task 4: 前端处理重名错误

**文件**: 
- `flutter_app/lib/screens/tag_management_screen.dart`

**问题**: 
编辑标签时如果重名，前端没有显示友好的错误提示。

**修改方案**:
1. 在 `_showEditTagDialog` 中捕获 409 错误
2. 显示中文错误提示: "标签名已存在，请使用其他名称"

**验证**:
- 尝试改为已存在的名称时显示错误提示
- 对话框保持打开，用户可以修改后重试

**完成条件**:
- 重名时显示明确的中文错误提示
- 用户可以修改后重试

---

## 技术要点

1. **SQLite INSERT OR REPLACE 陷阱**: 该语句会先 DELETE 再 INSERT，触发外键级联删除
2. **UNIQUE 约束**: 数据库已有 `preferred_label` 的 UNIQUE 约束，但需要在应用层提供友好错误
3. **级联删除**: `image_tags` 表的外键设置了 `ON DELETE CASCADE`，删除标签时会自动删除关联