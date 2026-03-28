# Phase 14: 补跑恢复与运营收尾 - Context

**Gathered:** 2026-03-29
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段交付“未打标签图片”批量补入队入口、单图失败不阻塞同批次的恢复闭环，以及面向运营的失败原因摘要与恢复引导。

范围严格限定在 AIQ-03、SAFE-01、SAFE-02：补齐运营补跑入口、失败隔离的产品闭环与失败可见性，不扩张为已有 AI 标签图片重生成、来源/批次快捷补跑入口、高级调度策略或分布式恢复体系。

</domain>

<decisions>
## Implementation Decisions

### 补跑入口范围
- **D-01:** “未打标签图片补跑”主入口放在全局 / 当前筛选工具栏，默认作用于当前筛选结果，而不是隐式全库补跑。
- **D-02:** 当没有任何筛选条件时，不允许直接补跑；管理员需要先收窄结果集，再执行补跑动作。
- **D-03:** “当前筛选结果”指筛选命中的完整结果集，而不是仅当前页或当前已加载图片。
- **D-04:** 第 14 阶段不新增批次级或来源级快捷补跑入口，避免把补跑动作和失败恢复路径混在一起。
- **D-05:** 当当前筛选结果中混有已有 AI 标签图片时，系统自动过滤，只处理当前仍未打标签的图片，并显式提示会跳过的部分。

### 未打标签口径
- **D-06:** “未打标签图片”统一定义为：当前没有 AI 标签结果，而不是历史上从未生成过 AI 标签。
- **D-07:** 用户曾经删除过 AI 标签的图片，只要当前无 AI 标签，仍应重新纳入补跑候选。
- **D-08:** 历史上失败过但当前仍无 AI 标签的图片，仍然属于补跑候选；失败历史影响恢复判断，不影响候选资格。
- **D-09:** 已经存在 `pending` / `queued` / `running` AI 任务的图片不能重复入队，必须在补跑反馈中明确说明这部分因已有在途任务而被跳过。

### 补跑反馈
- **D-10:** 执行前确认必须展示至少三类数量：命中数、新建任务数、跳过数。
- **D-11:** 补跑成功后沿用现有重试体验：toast 反馈结果，并自动跳转 / 定位到新创建的批次。
- **D-12:** 跳过数需要细分原因，至少区分“已有 AI 标签”和“已有在途任务”两类。
- **D-13:** 如果本次补跑最终没有创建任何任务，系统必须明确说明“当前结果集里没有可补跑图片”，并附带主要原因统计，而不是把 0 创建伪装成普通成功。

### 失败可见性
- **D-14:** 批次层失败摘要默认按原因聚合展示，而不是只显示最新一条失败。
- **D-15:** 聚合失败摘要默认优先展示“原因 + 数量”，帮助运营先判断是否值得恢复。
- **D-16:** 失败摘要需要直接给出“是否适合重试”的恢复提示，降低人工判断成本。
- **D-17:** 聚合失败摘要默认直接显示在批次列表层，而不是要求管理员先下钻到任务明细。

### 恢复入口分层
- **D-18:** `failed` / `partial_failed` 批次的主恢复动作继续保持为“重试失败任务”，不把“补跑未打标签图片”提升为失败批次主动作。
- **D-19:** 当失败摘要判断该批次不适合直接重试时，界面仍保留重试动作，但要给出明显警示或不推荐提示，由运营最终决策。
- **D-20:** 单任务重试保留为次级入口，用于少量例外处理；批次级恢复继续是主路径，避免退回逐图运营。

### OpenCode's Discretion
- 补跑确认弹层与成功 toast 的具体文案、字段顺序与视觉层级。
- 跳过原因与“适合重试”提示的具体分类规则、图标和颜色表达。
- 全局 / 当前筛选工具栏中补跑按钮的具体摆放位置与响应式细节。
- 聚合失败摘要是以内联标签、紧凑列表还是折叠摘要形式呈现。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase contract
- `.planning/ROADMAP.md` — Phase 14 的目标、成功标准与 14-01 / 14-02 / 14-03 计划边界。
- `.planning/REQUIREMENTS.md` — AIQ-03、SAFE-01、SAFE-02 的正式需求定义。
- `.planning/PROJECT.md` — v3.0 当前活跃范围、里程碑目标与继承性关键决策。
- `.planning/STATE.md` — 当前阶段位置与前序阶段沉淀的最近决策摘要。

### Inherited platform semantics
- `.planning/phases/11-task-platform-batch-model/11-CONTEXT.md` — “一次触发动作 = 一个批次”、失败隔离、批次状态与去重保护语义。
- `.planning/phases/12-import-task-auto-scheduling/12-CONTEXT.md` — “当前无 AI 标签”候选口径、删除 AI 标签后允许重新纳入，以及 Phase 14 推迟项。
- `.planning/phases/13-backend-monitoring-queue-control/13-CONTEXT.md` — batch-first 监控台、失败重试主入口、信息密度与反馈语义。
- `.planning/phases/13-backend-monitoring-queue-control/13-04-PLAN.md` — 重试创建新批次、返回重试数量与跳转新批次的恢复语义。
- `.planning/phases/13-backend-monitoring-queue-control/13-04-SUMMARY.md` — Phase 13 已落地的失败重试体验，Phase 14 需要延续的一致性基线。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/handler/ai_tag_handler.go`：已具备单图与按 `image_ids` 的批量 AI 入队入口，且统一通过 `enqueueAITagBatch()` 复用任务平台创建与入队逻辑。
- `internal/repository/image_repository.go`：`FindImagesWithoutAITags()` 已实现当前候选查询，包含“当前无 AI 标签”以及“排除已有在途 AI 任务”的口径。
- `internal/service/task_platform_service.go`：已提供 `PlanBatch()` 与 `QueueTask()`，适合复用为补跑入口的批次创建与任务入队骨架。
- `internal/service/admin_service.go`：已具备批次级 / 任务级失败重试，并明确以新批次承载恢复动作。
- `internal/service/task_read_service.go`：已向后台暴露 `FailureSummary` 与 `ErrorSummary` 字段，是 Phase 14 扩展聚合失败摘要的直接读模型基础。
- `web/admin/app.js`：已具备重试成功 toast、跳转到新批次、失败 / 部分失败批次主动作按钮等交互基线。
- `web/admin/index.html`：批次监控区已有全局控制区与筛选栏，适合承载“当前筛选结果补跑”的主入口。

### Established Patterns
- 运营后台继续遵循 batch-first：先看批次，再下钻任务，而不是回到逐任务首页。
- 失败恢复继续遵循“重试形成新批次”，而不是复活旧任务或静默覆写旧状态。
- 候选图片口径已经偏向“当前缺 AI 结果”而非“历史从未生成过结果”，Phase 14 应与现有自动补偿逻辑保持一致。
- 对已有在途任务的图片不重复入队，系统更偏向幂等补缺而不是并发重复轰炸。

### Integration Points
- 需要在后台管理页的全局 / 当前筛选工具栏增加“未打标签图片补跑”入口，而不是挂到批次行动作里。
- 需要在现有批量 AI 入队链路之上增加“候选预览 / 数量反馈 / 跳过原因”能力，而不是只接受显式 `image_ids`。
- 需要扩展批次读模型和后台展示层，把现有单条 `failure_summary` 提升为按原因聚合的运营摘要，并附带重试建议。
- 需要保持补跑入口与现有重试动作的反馈一致：都形成新批次、都给数量反馈、都能把管理员带到新批次。

</code_context>

<specifics>
## Specific Ideas

- 补跑不是隐藏的“全库扫一遍”，而是面向当前筛选结果的显式运营动作。
- 系统负责自动过滤当前筛选中不符合补跑条件的图片，但必须把跳过原因说清楚。
- 失败摘要应该在批次列表层就帮助管理员回答两个问题：这批失败主要因为什么、现在值不值得直接重试。
- “补跑”和“重试失败任务”是两条不同心智路径：前者解决当前缺结果，后者解决已有失败任务的恢复。

</specifics>

<deferred>
## Deferred Ideas

- 批次级或来源级的快捷补跑入口 —— 暂不纳入 Phase 14，后续如运营确有需要再评估。
- “历史从未生成过 AI 标签”与“当前无 AI 标签”双口径切换 —— 暂不纳入本阶段，避免与 Phase 12 自动补偿语义分叉。
- 在失败批次上把“补跑未打标签图片”做成并列主动作 —— 暂不纳入本阶段，恢复主路径仍保持“重试失败任务”。

</deferred>

---

*Phase: 14-backfill-recovery-operations*
*Context gathered: 2026-03-29*
