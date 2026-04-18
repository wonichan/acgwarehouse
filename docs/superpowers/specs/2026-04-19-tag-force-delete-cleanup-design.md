# 标签强制删除与关联清理设计

- 日期：2026-04-19
- 主题：允许在标签治理页删除标签，并在删除时自动清理图片关联与层级关系

## 目标

解决当前“标签已被使用或存在子标签时无法删除”的限制，让用户可以在标签治理页直接清理冗余标签。

删除标签时，系统需要：

1. 明确提示该标签会影响多少张图片。
2. 从所有持有该标签的图片上移除该标签。
3. 将该标签的直接子标签提升为顶级标签，即 `parent_id = NULL`。
4. 删除标签自身及其别名记录。

## 已确认产品规则

1. 允许删除仍被图片直接使用的标签。
2. 允许删除仍有直接子标签的标签。
3. 删除时只处理“当前 tag_id 本身”的图片关联，不联动删除其他 tag_id。
4. 删除时不删除子标签；只将这些子标签的 `parent_id` 置空。
5. 删除前需要告知用户：
   - 受影响图片数量
   - 将被提升为顶级标签的直接子标签数量
6. 删除确认后，后端必须一次性完成全部清理动作，避免出现只删了一半的中间状态。

## 当前实现与问题

当前代码中，标签删除被设计为“预检 + 阻止”模式：

- `internal/service/tag_admin_service.go` 中的 `GetDeletePreview` 会在标签有直接图片关联或有子标签时返回 `CanDelete = false`。
- `internal/handler/tag_handler.go` 中的 `DeleteTag` 会在 `CanDelete = false` 时直接返回 `409 Conflict`。
- `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart` 中的删除确认弹窗，只有在 `canDelete` 为真时才展示最终的“删除”按钮。

这导致两个实际问题：

1. 用户无法高效清理历史冗余标签。
2. 标签树和图片关联需要手工逐个清理，治理成本过高。

## 推荐方案

采用“可执行预览 + 事务性强制删除”方案。

核心变化：

1. 删除预览仍然存在，但它的职责从“决定能否删除”改为“告诉用户会发生什么”。
2. 删除接口不再因为图片关联或子标签存在而拒绝执行。
3. 删除动作在一个事务里完成：
   - 清空直接子标签的 `parent_id`
   - 删除该标签的所有图片关联
   - 删除该标签的别名
   - 删除该标签本身
4. 删除完成后继续执行必要的 FTS 同步与前端状态刷新。

这是最符合当前清理诉求的方案，因为它不引入新的标签语义，也不会把被删除标签偷偷迁移到其他标签上。

## 数据与行为语义

### 删除预览

删除预览继续返回以下信息：

- `tag_id`
- `preferred_label`
- `affected_image_count`：直接持有该标签的图片数量
- `child_count`：直接子标签数量
- `child_labels`（可选）：用于前端展示受影响层级结构

预览不再承担“禁止删除”的职责，因此不再以 `CanDelete = false` 阻止后续删除。

为兼容现有前端模型，短期可以保留 `canDelete` 字段，但其值应始终为 `true`，或者前端改为不再依赖该字段控制按钮展示。

### 图片关联清理范围

删除动作仅删除当前 `tag_id` 在 `image_tags` 中的直接关联。

不做以下联动：

- 不删除子标签与图片的关联
- 不删除其他同义标签或其他标签 ID 的图片关联
- 不自动将图片迁移到父标签或子标签

### 层级处理

删除某个标签时，只处理它的“直接子标签”：

- 将这些子标签的 `parent_id = NULL`
- 保留它们当前 `level`

这意味着：

- 删除 `parent` 标签后，原 `child` 标签会变成“无父级 child”
- 删除 `root` 标签后，原 `parent` 标签会变成“无父级 parent”

本次不额外重写 `level`。理由是当前代码已经允许孤立层级存在，且这次需求的核心是“快速清理标签”，不是“重塑整棵树”。

## 后端设计

### 1. `GetDeletePreview` 改为信息预览

文件：`internal/service/tag_admin_service.go`

调整点：

- 保留查找标签、统计直接图片关联数、查询直接子标签的逻辑。
- 新增 `child_count`（以及必要时的子标签摘要）返回值。
- 不再将“有图片关联”或“有子标签”作为阻断条件。

### 2. 新增事务性删除服务

文件：`internal/service/tag_admin_service.go`

新增或重写标签删除服务逻辑，推荐形态：

- `DeleteTagAndCleanup(ctx, tagID)`

事务步骤：

1. 校验标签存在。
2. 查询该标签的直接子标签。
3. 将这些直接子标签的 `parent_id` 置空。
4. 删除 `image_tags` 中该 `tag_id` 的所有直接关联。
5. 同步受影响图片的 FTS / 标签搜索索引。
6. 删除该标签的 alias 记录。
7. 删除标签本身。

所有步骤必须在一个事务中完成；任何一步失败都要回滚。

### 3. 复用现有能力

已存在的后端能力可以直接复用：

- `tag_repository.Update(...)` 已支持更新 `parent_id`
- `TagAdminService.ReparentTag(...)` / `ChangeLevel(...)` 已体现了父级置空的语义
- `image_tag_repository` 已有删除图片标签关联与 FTS 同步相关能力
- `tag_handler.DeleteTag(...)` 已是标签删除入口，可直接改造为调用新的清理型删除服务

### 4. API 语义

`DELETE /tags/{id}` 保持原路径不变，但行为改变为：

- 不再因为标签被使用或有子标签而返回 `409`
- 成功时返回清理结果摘要，建议包括：
  - `success`
  - `deleted_tag_id`
  - `affected_image_count`
  - `detached_child_count`

这样前端后续如果要加 toast 或操作结果提示，不需要再次查询。

## 前端设计

### 1. 删除按钮策略

文件：

- `flutter_app/lib/widgets/tag_management/tag_management_list.dart`
- `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`

删除入口保留在列表行级操作中，不再受“是否可删”控制。

### 2. 删除确认弹窗

当前弹窗要从“阻止型弹窗”改为“影响提示型弹窗”。

建议文案结构：

- 标题：`删除标签`
- 主文案：`标签：xxx`
- 说明 1：`将从 N 张图片中移除此标签`
- 说明 2：`M 个直接子标签将变为顶级标签`
- 风险提示：`此操作不可撤销`

无论 `affected_image_count` 或 `child_count` 是否大于零，都允许显示最终“删除”按钮。

### 3. 删除后的刷新

删除成功后，前端继续刷新：

- governance 列表
- tag tree

这样能确保：

- 删除的标签从治理页消失
- 原子标签的树结构马上变成顶级节点
- 后续筛选与统计读取到最新状态

## 失败场景与处理

### 标签不存在

返回 `404`，前端提示资源不存在。

### 事务中途失败

事务整体回滚，不允许出现：

- 图片关联删了但标签还在
- 子标签父级清空了但标签还没删
- alias 删了但主标签没删

### FTS 同步失败

视为整个删除失败，事务回滚。

原因：搜索索引与标签实际状态不一致，会比删除失败更难排查。

## 测试设计

### 后端服务测试

优先扩展：`internal/service/tag_admin_service_test.go`

新增/修改用例：

1. 删除预览可返回受影响图片数与子标签数。
2. 有图片关联时仍允许执行删除。
3. 有子标签时仍允许执行删除，并将直接子标签 `parent_id` 置空。
4. 删除后，`image_tags` 中不再存在该 `tag_id` 的关联。
5. 删除后，标签本身与 alias 均被移除。
6. 任一步骤失败时事务回滚。

### 后端 Handler 测试

优先扩展：`internal/handler/tag_handler_test.go`

新增/修改用例：

1. `DELETE /tags/{id}` 对已使用标签返回 `200` 而不是 `409`。
2. 响应体包含 `affected_image_count` 与 `detached_child_count`。
3. 删除后再次查询返回 `404`。

### Flutter 服务 / Provider / Widget 测试

优先扩展：

- `flutter_app/test/services/tag_service_test.dart`
- `flutter_app/test/providers/tag_provider_test.dart`
- `flutter_app/test/widgets/tag_management_workspace_test.dart`

新增/修改用例：

1. 删除预览接口解析新的提示字段。
2. 删除确认弹窗在有影响图片和子标签时仍显示“删除”按钮。
3. 删除成功后刷新治理列表与标签树。

## 不做的事

本次设计明确不包含以下内容：

- 不自动把图片迁移到父标签、子标签或“未分类”标签
- 不批量联动删除子标签
- 不重写孤立子标签/父标签的 `level`
- 不增加回收站或软删除语义
- 不改变标签树筛选语义

## 实施边界

本次只改“标签治理页删除标签”的语义与其后端清理逻辑。

不涉及：

- 图像详情页其他标签编辑交互
- AI 自动打标策略
- 标签层级创建、合并、改级别的既有规则
- 额外的批量迁移工具
