# Phase 5: 收藏夹与批量操作 - Context

**Gathered:** 2026-03-17
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段实现收藏夹/相册管理、批量操作功能。用户可创建/重命名/删除收藏夹，添加/移除收藏夹中的图片，设置收藏夹封面，批量选择图片进行标签操作或删除。

本阶段不包含 PostgreSQL 迁移与部署（Phase 6）。

</domain>

<decisions>
## Implementation Decisions

### 收藏夹入口与导航

- **入口位置**: 侧边栏抽屉
- **抽屉布局**: 双区域分区 — 上半部分是标签筛选（现有功能），下半部分是收藏夹列表
- **列表形式**: 简洁列表 — 每行显示收藏夹名称 + 图片数量 + 小缩略图封面
- **点击行为**: 过滤图库内容 — 点击后主内容区显示该收藏夹的图片
- **新建入口**: 列表底部按钮 "+ 新建收藏夹"
- **排序方式**: 按更新时间降序
- **添加图片入口**: 图片详情页按钮 — 点击后弹出收藏夹列表选择器

### 批量选择交互

- **进入模式**: 两种都支持 — AppBar 编辑按钮 + 长按图片卡片触发
- **选择反馈**: 蓝色边框 + 勾选图标（类似 Google Photos）
- **操作面板**: 底部 Bottom Sheet — 显示"添加标签""移至收藏夹""删除"等操作
- **批量标签**: 两种都支持 — 标签选择器（选择现有标签）+ 输入框直接输入（新标签）
- **移至收藏夹**: 收藏夹列表选择器

### 收藏夹封面策略

- **封面来源**: 自动选择最新添加的图片
- **更新时机**: 添加时立即更新
- **封面移除处理**: 当封面图片被移出收藏夹时，自动更换为最新图片

### 批量删除安全设计

- **确认机制**: 二次确认对话框 — 显示删除数量，提示"此操作不可恢复"
- **撤销机制**: 不提供撤销
- **删除范围**: 让用户选择 — 仅从收藏夹移除 / 从图库删除文件

### 收藏夹管理操作

- **管理入口**: 列表项菜单按钮 — 收藏夹列表项右侧"..."按钮，点击弹出菜单（重命名、删除、编辑描述）
- **删除收藏夹确认**: 确认对话框 — 提示"图片不会被删除"

### 移除图片操作

- **移除入口**: 两种都支持 — 批量选择模式 + 图片详情页按钮
- **移除确认**: 确认对话框

### OpenCode's Discretion

- 收藏夹列表的缩略图尺寸
- 收藏夹描述字段是否必填
- Bottom Sheet 的动画效果
- 确认对话框的具体文案
- 批量操作的最大数量限制（如有）

</decisions>

<specifics>
## Specific Ideas

- 收藏夹入口应与标签筛选并列，在同一个侧边栏抽屉中分区展示，保持界面简洁
- 批量选择模式应支持多种进入方式，兼顾不同用户习惯
- 删除操作应有明确的确认提示，防止误删重要图片
- 收藏夹封面自动更新，减少用户手动维护的工作量

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain/collection.go`: Collection 结构体已存在（ID, Name, Description, CoverImageID, ImageCount, CreatedAt, UpdatedAt）— 可直接使用
- `internal/repository/image_repository.go`: ImageRepository 模式可参考实现 CollectionRepository
- `internal/service/tag_governance_service.go`: Service 层模式可参考实现 CollectionService
- `internal/handler/image_handler.go`: Handler 层模式可参考实现 CollectionHandler
- `flutter_app/lib/widgets/tag_filter_drawer.dart`: 侧边栏抽屉组件，可扩展支持收藏夹列表
- `flutter_app/lib/widgets/image_grid.dart`: 网格视图组件，可扩展支持批量选择模式
- `flutter_app/lib/widgets/image_masonry.dart`: 瀑布流视图组件，可扩展支持批量选择模式
- `flutter_app/lib/providers/image_provider.dart`: Provider 模式可参考实现 CollectionProvider

### Established Patterns

- 项目采用 Go 后端 + Flutter 前端架构
- 数据库支持 SQLite（开发）和 PostgreSQL（生产）双模式
- 图片文件不存入数据库，只存路径与元数据
- API 采用 RESTful 风格，使用 Gin 框架
- Flutter 使用 Provider 模式管理状态
- Repository/Service/Handler 三层架构模式

### Integration Points

- 需要添加 `collections` 和 `collection_images` 数据表到 schema.go
- 需要实现 CollectionRepository、CollectionService、CollectionHandler
- 需要扩展 tag_filter_drawer.dart 支持收藏夹列表展示
- 需要实现 Flutter 批量选择模式（image_grid/image_masonry 扩展）
- 需要实现 Flutter Bottom Sheet 操作面板
- 需要实现 CollectionProvider 状态管理

### Gaps to Address

- 没有 `collections` 和 `collection_images` 数据表
- 没有 CollectionRepository/CollectionService/CollectionHandler
- 没有 Flutter 收藏夹相关界面
- 没有批量选择 UI 组件
- 没有 Bottom Sheet 操作面板组件

</code_context>

<deferred>
## Deferred Ideas

- 收藏夹搜索/筛选功能 — 可作为后续增强
- 收藏夹导出功能 — 可作为后续增强
- 批量操作的历史记录 — 可作为后续增强
- 回收站机制 — 用户选择不实现，后续如需要可重新讨论

</deferred>

---

*Phase: 05-collection-batch-operations*
*Context gathered: 2026-03-17*