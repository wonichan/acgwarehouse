# Phase 2: 缩略图、基础浏览与 AI 复核界面底座 - Context

**Gathered:** 2026-03-14
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付缩略图生成与 COS 云存储、感知哈希计算、Flutter 图片浏览界面（网格/瀑布流）、图片详情页，以及为后续 AI 标签复核提供的界面底座。

本阶段不包含 AI 标签生成与归并流程本身（Phase 3）、相似图片检测与去重功能（Phase 4）、收藏夹与批量操作（Phase 5）。

</domain>

<decisions>
## Implementation Decisions

### 缩略图生成策略
- 两档尺寸：小缩略图（~200px）用于网格展示，大缩略图（~600px）用于详情页预览
- 存储位置：上传到腾讯云 COS 云存储（存储桶：acgwarehouse-1301393037），使用 Go SDK 上传
- 生成时机：首次访问时按需生成，不阻塞导入流程
- 输出格式：JPEG（有损压缩，体积小，适合照片类二次元图片）
- Go 库推荐：`disintegration/imaging`（Lanczos 滤镜，高质量）

### 感知哈希计算策略
- 主算法：pHash（对压缩、缩放、颜色变化鲁棒性强）
- 计算时机：图片导入时同步计算，确保哈希值立即可用
- 存储方式：存入 `images.phash` 字段（已存在于 domain）
- 相似阈值：默认阈值 12，用户可在设置中调整
- Go 库推荐：`corona10/goimagehash`

### 浏览视图体验
- 视图类型：网格视图和瀑布流视图同时实现，用户可切换
- 默认视图：网格视图（更稳定，加载更快）
- 排序方式：支持多种排序（导入时间、文件名、文件大小），默认按导入时间降序
- Flutter 组件：网格用 `GridView.builder`，瀑布流用 `flutter_staggered_grid_view`
- 图片缓存：使用 `cached_network_image` 组件

### 图片详情页内容
- 元数据展示：文件名、尺寸、格式、文件大小、原始路径
- 标签区域：显示标签区域，为 Phase 3 AI 标签功能预留占位
- 大图交互：支持双指缩放和拖动查看大图
- Flutter 组件推荐：`extended_image`（支持缩放、拖动）

### AI 复核界面底座
- 入口位置：详情页标签区域
- 状态展示：显示状态文本（"AI 标签生成中"、"待复核 X 个"、"已确认"等）
- 与 Phase 3 衔接：界面底座完成后，Phase 3 实现标签数据填充和确认/合并交互

### 前后端接口设计
- 接口风格：RESTful API（与 Phase 1 已有框架一致）
- 分页方式：游标分页（适合无限滚动，不支持跳页）
- 缩略图 URL：API 返回完整 COS URL，前端直接使用（CDN 友好）

### OpenCode's Discretion
- 缩略图 JPEG 质量参数（建议小图 85，大图 90）
- 游标分页的具体实现细节（cursor 字段格式）
- COS 上传失败时的重试策略
- 缩略图命中时是否更新访问时间
- 网格/瀑布流的列数（建议根据屏幕宽度自适应）

</decisions>

<specifics>
## Specific Ideas

- 用户已购买腾讯云 COS 云存储服务，存储桶 acgwarehouse-1301393037 已创建就绪
- 缩略图上传使用 Go SDK：https://cloud.tencent.com/document/product/436/65644
- 二次元图片高度不一，瀑布流展示效果更好，但网格视图作为默认更稳定
- 感知哈希在导入时同步计算，为后续相似检测做好准备

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/domain/image.go`: Image 结构体已有 `PHash` 字段（int64），可直接使用
- `internal/service/metadata_service.go`: MetadataService 可提取图片尺寸/格式，可扩展支持缩略图生成
- `internal/worker/job_manager.go`: JobManager 支持注册异步任务 handler，可用于缩略图生成任务
- `internal/handler/routes.go`: Gin 路由框架已搭建，可添加新的 API 端点
- `internal/repository/schema.go`: 数据库 schema 已有 images 表和 async_jobs 表

### Established Patterns
- 项目采用 Go 后端 + Flutter 前端架构
- 数据库支持 SQLite（开发）和 PostgreSQL（生产）双模式
- 图片文件不存入数据库，只存路径与元数据
- 标签系统采用"原始观测 → 标准标签"分层治理
- 异步任务底座已建立，可注册新任务类型

### Integration Points
- 缩略图生成需要与 COS SDK 集成
- 图片详情页 API 需要返回图片元数据和 COS 缩略图 URL
- 感知哈希计算需要在导入流程中集成
- Flutter 前端需要新项目初始化，连接后端 API

</code_context>

<deferred>
## Deferred Ideas
- 受管库存储目录设计（当前保留原路径索引模式）
- 缩略图的 CDN 加速配置（COS 自带 CDN 支持）
- 图片预加载策略优化（可后续迭代）
- 网格视图的虚拟化优化（大规模图片库场景）

</deferred>

---

*Phase: 02-ai*
*Context gathered: 2026-03-14*