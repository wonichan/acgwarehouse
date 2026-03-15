# Phase 3: AI 开放标签与治理 - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段集成千问/豆包等多模态 AI 生成开放描述标签，建立标签治理能力。用户可查看、确认、修改、合并 AI 生成的标签，并按标签筛选图片。

本阶段不包含相似图片检测与去重（Phase 4）、收藏夹与批量操作（Phase 5）、以图搜图功能（Phase 4）。

</domain>

<decisions>
## Implementation Decisions

### AI 服务选择与提示词设计

- **提供商支持**：同时支持千问 VL 和豆包，用户可在配置中切换（需要设计提供商抽象层）
- **输出格式**：简洁标签列表（10-20 个短标签），不使用自然语言描述
- **标签数量**：精简核心模式，每张图片生成 8-12 个高置信度标签
- **限流策略**：固定速率限制，用户可设置每分钟请求数，超出后排队等待
- **处理时机**：图片导入后自动加入 AI 处理队列，后台自动处理
- **失败处理**：自动重试 3 次，间隔递增，仍失败则记录错误并跳过
- **模型升级**：升级模型后只处理新图片，历史图片保持原标签
- **NSFW 检测**：不自动检测 NSFW 内容
- **并发策略**：用户可配置并发数（默认 1，串行处理）
- **结果缓存**：按图片哈希缓存标签结果，相同图片不重复调用 AI

### 标签归并与别名策略

- **归并触发**：AI 返回标签后自动匹配现有标签，匹配成功则关联，否则创建新标签
- **匹配规则**：精确匹配（标签文本完全一致才匹配），不使用模糊相似度
- **别名来源**：仅用户手动添加别名，不支持自动创建
- **标签分类**：无预设分类，采用自由标签模式（`PrimaryCategory` 字段暂不使用）
- **状态管理**：复核状态流（待确认/已确认/已拒绝），AI 新建标签默认待确认
- **冲突处理**：同一图片多次 AI 打标时，保留所有观测记录，用户可选择采纳哪条
- **批量操作**：用户可将多个标签合并为一个，被合并的标签转为别名

### 用户复核界面体验

- **复核入口**：图片详情页内嵌标签区域，不使用独立复核页面
- **标签展示**：分区域展示（已确认标签一个区域，待确认标签另一个区域）
- **确认交互**：每个标签旁有确认/拒绝按钮，单击即生效
- **修改能力**：用户可修改 AI 生成的标签文本，修改后自动创建新标签
- **批量确认**：用户可选择多个标签后批量确认（多选模式）
- **撤销操作**：操作历史页面，可查看并撤销之前的操作
- **新增标签**：用户可在详情页直接输入添加新标签，添加后进入已确认区域

### 标签筛选与搜索体验

- **筛选器位置**：侧边栏抽屉，点击展开显示所有标签
- **标签排序**：按使用频次降序排序，高频标签优先展示
- **组合逻辑**：AND 交集模式，选择多个标签时图片需同时包含所有选中标签
- **排除标签**：不支持排除模式
- **标签搜索**：筛选器内置搜索框，可快速查找标签
- **别名匹配**：搜索时自动匹配别名，如搜索"夜雨"能找到"雨夜"
- **标签统计**：每个标签旁显示对应图片数量

### OpenCode's Discretion

- 提示词模板的具体内容和优化
- AI 响应的解析逻辑（处理不同提供商的响应格式差异）
- 侧边栏抽屉的具体宽度、动画效果
- 标签多选的交互细节
- 历史记录撤销的具体实现

</decisions>

<specifics>
## Specific Ideas

- AI 服务应该像一个"黑盒"，用户只需配置 Provider/APIKey，系统自动处理调用和错误
- 标签复核应该是"轻量"操作，用户可以在浏览图片的同时快速确认标签
- 标签筛选器应该显示图片数量，让用户知道筛选结果规模
- 同一图片多次打标时保留所有观测，这是 Phase 1 已锁定的设计

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain/tag.go`: Tag 结构体（ID, PreferredLabel, Slug, PrimaryCategory, ReviewState, TrustScore, UsageCount）- ReviewState 字段可直接使用
- `internal/domain/tag_observation.go`: TagObservation 结构体（含 Provider, ModelName, PromptVersion, Confidence, EvidenceType）- 完整支持多提供商追踪
- `internal/domain/tag_alias.go`: TagAlias 结构体（含 AliasType, IsPreferred, NormalizedLabel）- 支持别名归一化查询
- `internal/domain/async_job.go`: AsyncJob 结构体（含 Status, Payload, Progress, Error）- 可用于 AI 任务队列
- `internal/worker/job_manager.go`: JobManager 支持 `RegisterHandler` 注册新任务类型 - 可注册 `ai_tag_generation` 任务
- `internal/config/config.go`: AIConfig 结构体已有 Provider/APIKey/Model 字段，支持环境变量覆盖

### Established Patterns

- 项目采用 Go 后端 + Flutter 前端架构
- 数据库支持 SQLite（开发）和 PostgreSQL（生产）双模式
- 标签系统采用"原始观测 → 标准标签"分层治理（Phase 1 已锁定）
- 异步任务采用队列模式，通过 `JobManager.AddJob()` 添加任务
- 配置采用 YAML 文件 + 环境变量覆盖模式

### Integration Points

- AI 服务客户端需要新建 `internal/ai/` 目录，实现提供商抽象层
- 标签归并逻辑需要在 `internal/service/tag_governance_service.go` 中实现
- 标签管理 API 需要在 `internal/handler/tag_handler.go` 中添加端点
- Flutter 标签筛选组件需要在 `lib/presentation/tags/` 中实现
- 数据库需要添加 tags、tag_observations、tag_aliases、image_tags 表的 migration

### Gaps to Address

- `image_tags` 关联表 domain 模型缺失
- AI 客户端实现完全缺失（需要 HTTP 调用外部 API）
- 标签 Repository 层缺失
- 标签相关 API 端点缺失
- Flutter 标签组件缺失

</code_context>

<deferred>
## Deferred Ideas

- NSFW 自动检测 - 可作为后续增强功能
- 标签排除模式筛选 - 用户反馈后可考虑添加
- 标签分类/大类 - 当前采用自由标签模式，后续如需要可重新设计
- AI 标签的批量重新处理 - 模型升级时历史标签重打标
- Booru 标签同步 - 作为独立功能在后续版本考虑

</deferred>

---

*Phase: 03-ai*
*Context gathered: 2026-03-15*