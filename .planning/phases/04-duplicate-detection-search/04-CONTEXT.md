# Phase 4: 重复检测与搜索 - Context

**Gathered:** 2026-03-16
**Status:** Ready for planning

<domain>
## Phase Boundary

实现相似图片检测、去重、以图搜图功能。系统可检测完全相同图片（文件哈希）和相似图片（感知哈希），用户可设置相似度阈值、管理重复图片组，并支持按文件名/标签搜索和以图搜图。

本阶段不包含收藏夹与批量操作（Phase 5）、PostgreSQL 迁移与部署（Phase 6）。

</domain>

<decisions>
## Implementation Decisions

### 重复检测策略

- **检测算法**：组合检测 — pHash 计算相似度 + 文件哈希（SHA256）检测完全相同
- **执行时机**：定时批量检测，后台静默运行，不阻塞用户操作
- **相似度阈值**：默认 pHash 汉明距离 ≤ 10 为相似，平衡精度和召回率
- **分组定义**：传递性分组 — A-B 相似且 A-C 相似时，B 和 C 归为同一组
- **推荐保留**：优先推荐分辨率最高的图片作为保留候选
- **增量检测**：支持增量检测，只检测新增图片与已有图片的相似关系
- **进度展示**：显示进度条和百分比，用户可看到实时进度

### 去重交互设计

- **展示方式**：分组展示 — 显示重复组列表，每组内标记推荐保留（分辨率最高）
- **删除处理**：实际删除文件，释放存储空间（操作不可逆，需确认）
- **结果持久化**：检测结果存储在数据库，支持增量检测和历史查询

### 搜索功能实现

- **实现方式**：使用 SQLite FTS5 全文索引，支持高效搜索
- **UI 入口**：AppBar 标题区域变为搜索框，点击后进入搜索模式
- **搜索范围**：文件名 + 标签文本（包括别名），一个搜索框搜多处
- **结果展示**：分类展示 — 文件名匹配和标签匹配分开展示，各有分组标题
- **中文分词**：使用 jieba 或简单分词，支持中文关键词搜索
- **触发时机**：手动触发搜索 — 用户输入后需点击搜索或回车
- **搜索历史**：保存最近 10 条搜索记录，点击可快速重搜
- **空结果处理**：显示"未找到结果"提示，推荐相似标签或热门标签
- **组合搜索**：搜索词 + 标签筛选器可同时使用，组合过滤结果
- **结果排序**：默认相关度排序，用户可切换为时间/名称排序
- **多词逻辑**：多个关键词用空格分隔，AND 逻辑（必须全部匹配）
- **大小写**：搜索不区分大小写
- **模糊匹配**：支持拼音搜索、前缀匹配、后缀通配符 *

### 以图搜图交互

- **交互方式**：多方式支持 — 文件选择对话框 + 拖拽 + 剪贴板粘贴
- **结果数量**：用户可配置最大返回数量，默认 20 张
- **相似度显示**：每张结果图显示相似度百分比，如 95% 相似
- **支持格式**：支持 JPG/PNG/WebP/GIF 所有已入库格式
- **原图展示**：结果页顶部显示上传的原图缩略图，方便对比
- **快捷操作**：点击结果图可快速添加标签或加入收藏夹
- **历史记录**：不保存以图搜图历史，每次都是新搜索
- **网络图片**：支持粘贴图片 URL，后端下载后计算哈希搜索
- **移动端**：仅支持从相册选择图片，不支持拍照搜索
- **错误处理**：上传失败/格式不支持时显示友好提示，引导用户重试

### OpenCode's Discretion

- FTS5 虚拟表的具体结构设计
- 分词器的选择和配置（jieba vs simple tokenizer）
- 搜索结果缓存的实现细节
- 以图搜图上传图片的临时存储位置
- 检测任务的具体调度频率
- 重复检测结果的数据库表结构设计

</decisions>

<specifics>
## Specific Ideas

- 重复检测应该像"杀毒软件扫描"，后台静默运行，完成后通知用户
- 相似度百分比让用户直观了解匹配程度，比单纯的排序更有价值
- 搜索词和标签筛选组合使用，满足高级用户的精确搜索需求
- 以图搜图支持网络 URL，方便用户从网页直接搜索

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain/image.go` — Image 结构体已有 `PHash int64` 字段，可直接使用
- `internal/repository/image_repository.go` — 已支持 PHash 存取，有 `idx_images_phash` 索引
- `flutter_app/lib/widgets/image_grid.dart` — 网格视图组件，可用于搜索结果展示
- `flutter_app/lib/widgets/image_masonry.dart` — 瀑布流视图组件，可用于搜索结果展示
- `flutter_app/lib/widgets/tag_filter_drawer.dart` — 标签筛选抽屉，可复用搜索框样式
- `flutter_app/lib/screens/gallery_screen.dart` — Gallery 页面模式，可作为搜索页面参考
- `internal/handler/image_handler.go` — 图片列表 API 模式，可扩展为搜索 API
- `internal/repository/tag_repository.go` — `FindByLabelLike()` 搜索模式可参考

### Established Patterns

- 项目采用 Go 后端 + Flutter 前端架构
- 数据库支持 SQLite（开发）和 PostgreSQL（生产）双模式
- 图片文件不存入数据库，只存路径与元数据
- 标签系统采用"原始观测 → 标准标签"分层治理
- API 采用 RESTful 风格，使用 Gin 框架
- Flutter 使用 Provider 模式管理状态
- 分页采用 limit/offset 模式

### Integration Points

- PHash 计算需集成 `github.com/corona10/goimagehash` 库
- FTS5 需创建虚拟表，扩展数据库 schema
- 搜索 API 需在 `internal/handler/` 添加新端点
- 以图搜图需创建图片上传端点
- 重复检测结果需新建数据表存储
- Flutter 搜索界面需在 `flutter_app/lib/screens/` 创建新页面
- 以图搜图界面可复用搜索结果展示组件

### Gaps to Address

- PHash 计算逻辑未实现 — `metadata_service.go` 需扩展
- 文件哈希（SHA256）计算未实现
- 相似度比较（汉明距离）计算未实现
- 无 FTS5 全文索引表
- 无搜索 API 端点
- 无以图搜图上传 API
- 无重复检测服务
- 无搜索界面

</code_context>

<deferred>
## Deferred Ideas

- 向量索引实现 — ROADMAP 提示"如果图片量 > 100k 需要深入研究"，当前 Phase 4 使用 pHash 即可
- 搜索结果预加载优化 — 后续迭代
- 以图搜图结果导出功能 — 可作为后续增强

</deferred>

---

*Phase: 04-duplicate-detection-search*
*Context gathered: 2026-03-16*