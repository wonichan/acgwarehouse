# Phase 5: 收藏夹与批量操作 - Research

**Created:** 2026-03-17
**Status:** Ready for planning

---

## Executive Summary

本文档研究 Phase 5（收藏夹与批量操作）的技术实现方案。该阶段需覆盖需求 COLL-01~05（收藏夹管理）和 BTCH-01~04（批量操作）。

**核心挑战：**
1. 收藏夹与图片的多对多关系管理
2. 批量选择 UI 模式实现
3. 批量操作的 API 设计与事务处理
4. 收藏夹封面自动更新机制

---

## 1. Technology Stack Recommendations

### 1.1 后端技术栈（沿用现有模式）

| 层级 | 技术 | 参考实现 |
|------|------|----------|
| 数据模型 | `internal/domain/collection.go` | 已存在 Collection 结构体 |
| Repository | `internal/repository/collection_repository.go` | 参考 `image_repository.go` |
| Service | `internal/service/collection_service.go` | 参考 `tag_governance_service.go` |
| Handler | `internal/handler/collection_handler.go` | 参考 `image_tag_handler.go` |
| 数据库 | SQLite (开发) / PostgreSQL (生产) | 现有 schema.go 模式 |

### 1.2 前端技术栈（沿用现有模式）

| 层级 | 技术 | 参考实现 |
|------|------|----------|
| 状态管理 | Provider + ChangeNotifier | `providers/image_provider.dart` |
| 服务层 | `services/collection_service.dart` | `services/tag_service.dart` |
| UI 组件 | StatefulWidget + Consumer | `widgets/tag_filter_drawer.dart` |

### 1.3 关键依赖

**后端：** 无新增依赖，使用现有 Gin + database/sql

**前端：** 无新增依赖，使用现有：
- `provider` — 状态管理
- `cached_network_image` — 图片缓存
- `flutter_staggered_grid_view` — 瀑布流布局

---

## 2. Data Model Design

### 2.1 数据库表结构

```sql
-- ============================================================
-- collections: 收藏夹表
-- ============================================================
CREATE TABLE IF NOT EXISTS collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    cover_image_id INTEGER,
    image_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cover_image_id) REFERENCES images(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_collections_updated_at ON collections(updated_at);

-- ============================================================
-- collection_images: 收藏夹-图片关联表（多对多）
-- ============================================================
CREATE TABLE IF NOT EXISTS collection_images (
    collection_id INTEGER NOT NULL,
    image_id INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, image_id),
    FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_collection_images_collection ON collection_images(collection_id);
CREATE INDEX IF NOT EXISTS idx_collection_images_image ON collection_images(image_id);
CREATE INDEX IF NOT EXISTS idx_collection_images_added_at ON collection_images(added_at);
```

### 2.2 Domain 模型

**Collection（已存在于 `internal/domain/collection.go`）：**
```go
type Collection struct {
    ID           int64     `json:"id"`
    Name         string    `json:"name"`
    Description  string    `json:"description"`
    CoverImageID *int64    `json:"cover_image_id"`
    ImageCount   int       `json:"image_count"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

**CollectionImage（新增）：**
```go
type CollectionImage struct {
    CollectionID int64     `json:"collection_id"`
    ImageID      int64     `json:"image_id"`
    AddedAt      time.Time `json:"added_at"`
}
```

**批量操作请求模型：**
```go
type BatchTagRequest struct {
    ImageIDs []int64 `json:"image_ids"`
    TagIDs   []int64 `json:"tag_ids"`   // 要添加的标签
    Action   string  `json:"action"`    // "add" | "remove"
}

type BatchMoveRequest struct {
    ImageIDs     []int64 `json:"image_ids"`
    CollectionID int64   `json:"collection_id"`
}

type BatchDeleteRequest struct {
    ImageIDs []int64 `json:"image_ids"`
}
```

### 2.3 关系说明

```
images (1) ←→ (N) collection_images (N) ←→ (1) collections
                       ↑
                  中间关联表
                  
设计决策：
- collection_images.added_at 用于追踪添加时间，支持"最新添加"排序
- 删除图片时级联删除 collection_images 记录
- 删除收藏夹时级联删除 collection_images 记录，不影响图片
- image_count 冗余存储，避免每次 COUNT 查询
```

---

## 3. API Endpoint Design

### 3.1 收藏夹 CRUD 端点

| 方法 | 端点 | 描述 | 需求映射 |
|------|------|------|----------|
| GET | `/api/v1/collections` | 获取收藏夹列表 | COLL-05 |
| POST | `/api/v1/collections` | 创建收藏夹 | COLL-01 |
| GET | `/api/v1/collections/:id` | 获取收藏夹详情 | — |
| PUT | `/api/v1/collections/:id` | 更新收藏夹（重命名/描述） | COLL-03 |
| DELETE | `/api/v1/collections/:id` | 删除收藏夹 | COLL-03 |
| GET | `/api/v1/collections/:id/images` | 获取收藏夹内图片 | COLL-02 |

### 3.2 收藏夹图片操作端点

| 方法 | 端点 | 描述 | 需求映射 |
|------|------|------|----------|
| POST | `/api/v1/collections/:id/images` | 添加图片到收藏夹 | COLL-02 |
| DELETE | `/api/v1/collections/:id/images/:image_id` | 从收藏夹移除图片 | COLL-02 |
| PUT | `/api/v1/collections/:id/cover` | 设置收藏夹封面 | COLL-04 |

### 3.3 批量操作端点

| 方法 | 端点 | 描述 | 需求映射 |
|------|------|------|----------|
| POST | `/api/v1/images/batch/tags` | 批量添加/删除标签 | BTCH-02 |
| POST | `/api/v1/images/batch/move` | 批量移动到收藏夹 | BTCH-03 |
| POST | `/api/v1/images/batch/delete` | 批量删除图片 | BTCH-04 |

### 3.4 API 详细设计

#### 3.4.1 创建收藏夹

```
POST /api/v1/collections
Content-Type: application/json

Request:
{
    "name": "我的收藏",
    "description": "精选图片"  // 可选
}

Response (201):
{
    "id": 1,
    "name": "我的收藏",
    "description": "精选图片",
    "cover_image_id": null,
    "image_count": 0,
    "created_at": "2026-03-17T10:00:00Z",
    "updated_at": "2026-03-17T10:00:00Z"
}
```

#### 3.4.2 添加图片到收藏夹

```
POST /api/v1/collections/:id/images
Content-Type: application/json

Request:
{
    "image_ids": [1, 2, 3]
}

Response (200):
{
    "success": true,
    "added_count": 3,
    "collection": {
        "id": 1,
        "image_count": 5,
        "cover_image_id": 3  // 自动更新为最新添加的图片
    }
}
```

#### 3.4.3 批量添加标签

```
POST /api/v1/images/batch/tags
Content-Type: application/json

Request:
{
    "image_ids": [1, 2, 3],
    "tag_ids": [10, 20],
    "action": "add"  // 或 "remove"
}

Response (200):
{
    "success": true,
    "affected_images": 3,
    "affected_tags": 6  // 3 images × 2 tags
}
```

#### 3.4.4 批量删除图片

```
POST /api/v1/images/batch/delete
Content-Type: application/json

Request:
{
    "image_ids": [1, 2, 3]
}

Response (200):
{
    "success": true,
    "deleted_count": 3,
    "affected_collections": [
        {"id": 1, "new_count": 5}
    ]
}
```

---

## 4. Frontend Component Design

### 4.1 批量选择模式

**设计参考：** Google Photos 批量选择交互

**组件扩展：** `ImageGrid` 和 `ImageMasonry`

```dart
// 新增参数
class ImageGrid extends StatelessWidget {
  final List<ImageModel> images;
  final ImageTapCallback? onImageTap;
  final int crossAxisCount;
  
  // 新增：批量选择模式
  final bool selectionMode;
  final Set<int> selectedImageIds;
  final void Function(int imageId)? onSelectionToggle;
  
  // ...
}

// 选中状态视觉效果
Widget _buildImageTile(ImageModel image) {
  final isSelected = selectedImageIds.contains(image.id);
  
  return Stack(
    children: [
      // 原有缩略图
      CachedNetworkImage(...),
      
      // 选择模式下显示覆盖层
      if (selectionMode)
        Positioned.fill(
          child: Container(
            decoration: BoxDecoration(
              border: isSelected 
                  ? Border.all(color: Colors.blue, width: 3)
                  : null,
              color: isSelected 
                  ? Colors.blue.withOpacity(0.2)
                  : Colors.transparent,
            ),
            child: isSelected
                ? const Align(
                    alignment: Alignment.topRight,
                    child: Padding(
                      padding: EdgeInsets.all(4),
                      child: Icon(Icons.check_circle, color: Colors.blue),
                    ),
                  )
                : null,
          ),
        ),
    ],
  );
}
```

**进入选择模式：**
1. AppBar 编辑按钮点击
2. 图片卡片长按

**退出选择模式：**
1. AppBar 取消按钮
2. 选择数量归零时自动退出
3. 执行批量操作后自动退出

### 4.2 Bottom Sheet 操作面板

**触发时机：** 选择模式下选中任意图片后显示

```dart
class BatchOperationSheet extends StatelessWidget {
  final Set<int> selectedImageIds;
  final VoidCallback onOperationComplete;
  
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text('已选择 ${selectedImageIds.length} 张图片'),
          const SizedBox(height: 16),
          ListTile(
            leading: const Icon(Icons.label),
            title: const Text('添加标签'),
            onTap: () => _showTagSelector(context),
          ),
          ListTile(
            leading: const Icon(Icons.folder),
            title: const Text('移至收藏夹'),
            onTap: () => _showCollectionSelector(context),
          ),
          ListTile(
            leading: const Icon(Icons.delete, color: Colors.red),
            title: const Text('删除', style: TextStyle(color: Colors.red)),
            onTap: () => _showDeleteConfirmation(context),
          ),
        ],
      ),
    );
  }
}
```

### 4.3 侧边栏收藏夹列表

**扩展现有 `TagFilterDrawer`：**

```dart
class TagFilterDrawer extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: Column(
        children: [
          // 现有标签筛选区域
          Expanded(
            flex: 2,
            child: TagFilterSection(),
          ),
          
          // 新增：收藏夹列表区域
          const Divider(),
          Expanded(
            flex: 1,
            child: CollectionListSection(),
          ),
        ],
      ),
    );
  }
}

class CollectionListSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Consumer<CollectionProvider>(
      builder: (context, provider, _) {
        return Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8.0),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Text('收藏夹', style: TextStyle(fontWeight: FontWeight.bold)),
                  IconButton(
                    icon: const Icon(Icons.add),
                    onPressed: () => _showCreateDialog(context),
                    tooltip: '新建收藏夹',
                  ),
                ],
              ),
            ),
            Expanded(
              child: ListView.builder(
                itemCount: provider.collections.length,
                itemBuilder: (context, index) {
                  final collection = provider.collections[index];
                  return ListTile(
                    leading: _buildCoverThumbnail(collection),
                    title: Text(collection.name),
                    trailing: Text('${collection.imageCount}'),
                    onTap: () => provider.filterByCollection(collection.id),
                    trailing: PopupMenuButton(
                      itemBuilder: (context) => [
                        const PopupMenuItem(value: 'rename', child: Text('重命名')),
                        const PopupMenuItem(value: 'delete', child: Text('删除')),
                      ],
                      onSelected: (value) => _handleMenuAction(context, collection, value),
                    ),
                  );
                },
              ),
            ),
          ],
        );
      },
    );
  }
}
```

---

## 5. Implementation Strategy

### 5.1 后端实现顺序

1. **数据层** — 添加 collections 和 collection_images 表到 schema.go
2. **Repository 层** — 实现 CollectionRepository（CRUD + 关联操作）
3. **Service 层** — 实现 CollectionService（业务逻辑 + 封面自动更新）
4. **Handler 层** — 实现 CollectionHandler（API 端点）
5. **批量操作** — 实现 BatchHandler（批量标签/移动/删除）

### 5.2 前端实现顺序

1. **状态管理** — 实现 CollectionProvider 和 SelectionProvider
2. **服务层** — 实现 CollectionService（HTTP 客户端）
3. **UI 组件** — 扩展现有组件支持批量选择
4. **Bottom Sheet** — 实现批量操作面板
5. **侧边栏扩展** — 添加收藏夹列表

---

## 6. Common Pitfalls

| 陷阱 | 解决方案 |
|------|----------|
| 封面图片被删除 | 自动更换为最新图片（查询 added_at DESC LIMIT 1） |
| 批量操作性能 | 使用事务 + 批量 INSERT/DELETE，避免 N+1 |
| 选中状态丢失 | 使用 Provider 持久化选中状态，页面切换时保留 |
| 收藏夹列表性能 | 只加载名称和数量，封面缩略图懒加载 |
| 并发修改冲突 | 使用数据库事务保证一致性 |
| 删除确认误操作 | 二次确认对话框，显示删除数量 |

---

## 7. Validation Architecture

### 7.1 测试策略总览

| 层级 | 测试类型 | 工具 | 覆盖目标 |
|------|----------|------|----------|
| Repository | 单元测试 | Go testing | CRUD 操作、关联查询、事务 |
| Service | 单元测试 | Go testing | 业务逻辑、封面更新、批量操作 |
| Handler | 集成测试 | httptest | API 端点、请求验证、响应格式 |
| Flutter Widget | Widget 测试 | flutter_test | UI 交互、状态管理、导航 |

### 7.2 后端验证点

**收藏夹 CRUD：**
- [ ] 创建收藏夹 → 返回 ID，name/description 正确
- [ ] 重命名收藏夹 → 名称更新，updated_at 更新
- [ ] 删除收藏夹 → 收藏夹消失，图片保留
- [ ] 获取收藏夹列表 → 按更新时间降序

**收藏夹图片操作：**
- [ ] 添加图片到收藏夹 → image_count 增加，cover_image_id 更新
- [ ] 移除图片 → image_count 减少，封面自动更换（如需要）
- [ ] 设置封面 → cover_image_id 更新

**批量操作：**
- [ ] 批量添加标签 → 所有图片获得指定标签
- [ ] 批量移除标签 → 所有图片移除指定标签
- [ ] 批量移动到收藏夹 → 所有图片出现在目标收藏夹
- [ ] 批量删除 → 图片从数据库删除，文件系统文件删除

**边界条件：**
- [ ] 创建空名称收藏夹 → 返回 400 错误
- [ ] 添加不存在的图片 → 忽略或返回错误
- [ ] 批量操作空列表 → 返回 400 错误
- [ ] 删除不存在的收藏夹 → 返回 404

### 7.3 前端验证点

**收藏夹 UI：**
- [ ] 侧边栏显示收藏夹列表
- [ ] 点击收藏夹过滤图库
- [ ] 收藏夹列表项显示名称、数量、缩略图
- [ ] 收藏夹菜单（重命名/删除）正常工作

**批量选择 UI：**
- [ ] 长按进入批量选择模式
- [ ] AppBar 编辑按钮进入批量选择模式
- [ ] 选中图片显示蓝色边框 + 勾选图标
- [ ] AppBar 显示选中数量
- [ ] 取消按钮退出选择模式

**批量操作 UI：**
- [ ] Bottom Sheet 显示操作选项
- [ ] 批量添加标签对话框正常工作
- [ ] 批量移动收藏夹选择器正常工作
- [ ] 删除确认对话框显示数量和警告

### 7.4 集成测试场景

```gherkin
Feature: 收藏夹管理

Scenario: 创建收藏夹并添加图片
  Given 用户在图库页面
  When 用户创建收藏夹 "我的收藏"
  And 用户选择 3 张图片
  And 用户将图片移至 "我的收藏"
  Then 收藏夹显示 3 张图片
  And 收藏夹封面为最后添加的图片

Scenario: 批量添加标签
  Given 用户选择 5 张图片
  When 用户添加标签 "风景"
  Then 5 张图片都有 "风景" 标签

Scenario: 删除收藏夹保护图片
  Given 收藏夹 "测试" 包含 3 张图片
  When 用户删除收藏夹 "测试"
  Then 收藏夹列表不再显示 "测试"
  And 3 张图片仍然存在于图库中

Scenario: 封面自动更新
  Given 收藏夹 "封面测试" 封面为图片 A
  When 用户将图片 B 添加到收藏夹
  Then 封面更新为图片 B
  When 用户移除图片 B
  Then 封面恢复为图片 A
```

### 7.5 自动化验证命令

**后端测试：**
```bash
# 运行所有收藏夹相关测试
go test ./internal/repository/... -run Collection -v
go test ./internal/service/... -run Collection -v
go test ./internal/handler/... -run Collection -v

# 运行批量操作测试
go test ./internal/handler/... -run Batch -v
```

**前端测试：**
```bash
# 运行 Widget 测试
cd flutter_app && flutter test test/widgets/collection_test.dart
flutter test test/widgets/batch_selection_test.dart
flutter test test/providers/selection_provider_test.dart
```

### 7.6 验收标准

Phase 5 完成必须满足：

1. **功能验收**
   - 所有 6 条成功标准达成
   - 所有需求 COLL-01~05 和 BTCH-01~04 覆盖

2. **测试验收**
   - 后端单元测试覆盖率 > 80%
   - 前端 Widget 测试覆盖关键交互
   - 所有测试通过

3. **集成验收**
   - API 端点可从 Flutter 调用
   - 批量选择模式与现有图库兼容
   - 侧边栏收藏夹列表与标签筛选共存

---

*研究完成时间：2026-03-17*
*状态：Ready for planning* 
