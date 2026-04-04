# Phase 19: Tag Management - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

管理员可以在桌面端直接管理标签，而不必切换到旧入口。Phase 19 的交付边界包括：在独立标签管理视图中查看全部标签及其统计信息、就地编辑标签、合并重复标签、删除未使用标签，并在删除确认中明确展示受影响图片数量。

本阶段继续沿用 Phase 8 已确定的“独立标签管理页”方向，以及 Phase 17 已完成的桌面壳层/导航结构；不扩展到完整 taxonomy 体系、颜色语义体系或新的重型治理工作台。

</domain>

<decisions>
## Implementation Decisions

### 主工作流与页面结构
- **D-01:** 标签管理继续保留为桌面壳层中的独立导航页，不回退到旧入口，也不把治理能力塞回图库页头部。
- **D-02:** 页面主工作流采用“标签列表主视图 + 合并面板”模式：列表承担浏览、筛选、统计与单条入口；点击“合并”后进入目标标签选择/确认面板完成重复标签合并。
- **D-03:** 编辑、删除、合并统一作为列表行级治理动作呈现，而不是拆成独立重型治理台或独立向导首页。

### 删除与安全约束
- **D-04:** 仅允许删除未使用标签；有引用的标签不走直接删除路径，应提示使用合并/整理等更安全的治理方式。
- **D-05:** 删除确认必须显式展示受影响图片数量；对未使用标签该数值应清晰呈现为 `0`，不能只用模糊风险文案。
- **D-06:** 删除语义以“安全治理”优先，不保留“强制删除并连带清空所有图片关联”的普通主路径。

### 标签模型边界
- **D-07:** Phase 19 除核心标签治理（查看统计、改名、合并、删除）外，还正式纳入 `primaryCategory` 与 alias 的治理能力。
- **D-08:** alias 被视为标签治理的一部分，应与重命名/合并流程协同考虑，而不是留在纯后端能力或隐藏状态中。
- **D-09:** taxonomy、颜色语义、完整层级分类体系不纳入本阶段锁定范围，后续若需要可独立成新的治理增强阶段。

### 批量治理
- **D-10:** 本阶段锁定“完整批量治理”方向，而不只停留在单条操作或“批量清理未使用标签”。
- **D-11:** 批量治理至少覆盖批量清理、批量分类/别名整理，以及批量合并候选处理；规划时应把多选状态、批量确认和失败反馈纳入任务范围。

### 跨页面联动
- **D-12:** 标签治理页不仅显示统计数字，还应支持从标签行跳转到预设筛选后的受影响图片集合，帮助管理员把治理动作与实际图片影响联系起来。
- **D-13:** 联动方式优先采用“跳转到图库/搜索结果并带预设标签筛选”而不是在治理页内嵌重型图片预览面板。

### the agent's Discretion
- 合并面板采用侧面板、对话框还是嵌入式详情区，只要仍满足“列表主视图 + 合并面板”语义即可。
- 列表中的统计信息密度、排序方式、搜索框/筛选器布局可按现有桌面 Fluent 壳层风格做细化。
- 批量治理的交互细节（勾选栏、批量工具条、确认文案层级）可在不改变已锁定能力边界的前提下由 planner/implementer 决定。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Scope & Requirements
- `.planning/ROADMAP.md` — Phase 19 的目标、依赖关系与 success criteria（查看统计、原地编辑/合并、删除未使用标签并显示影响数）
- `.planning/REQUIREMENTS.md` — `DSK-04` 的正式要求；确认 Phase 19 是桌面端标签管理的归属阶段
- `.planning/PROJECT.md` — v4.0 里程碑目标、Windows Photos 风格桌面重构方向，以及标签归并“精确匹配优先”的项目级决策
- `.planning/STATE.md` — 当前项目状态与已知 Flutter 基线问题说明

### Prior Phase Decisions
- `.planning/phases/08-windows-ui/08-CONTEXT.md` — 独立标签管理页、页面复用策略、标签管理与详情联动的早期产品决策
- `.planning/phases/17-desktop-shell-foundation/17-CONTEXT.md` — 桌面壳层/导航已收口，标签治理明确留到 Phase 19；图库页不再拥有重复的全局入口

### Existing Desktop Tag Flow
- `flutter_app/lib/screens/tag_management_screen.dart` — 当前桌面标签管理页基线：统计卡片、单条编辑/删除、清理无用标签按钮
- `flutter_app/lib/providers/tag_provider.dart` — 前端标签治理状态模型：统计、更新、删除、清理、图片标签合并接口
- `flutter_app/lib/services/tag_service.dart` — 前端标签治理 API 契约：统计、更新、删除、清理、图片级合并接口

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `flutter_app/lib/screens/tag_management_screen.dart`：已经有桌面端独立标签管理页，可直接演进为 Phase 19 的主界面，而不是从零开始搭治理入口。
- `flutter_app/lib/providers/tag_provider.dart`：已经封装 `loadStatistics()`、`cleanUnusedTags()`、`updateTag()`、`deleteTag()` 与图片级合并能力，可作为桌面治理状态层基础。
- `flutter_app/lib/services/tag_service.dart`：已有统计、更新、删除、清理、图片标签合并等 API 封装，能帮助 planner 识别“已有契约”与“仍需补齐的契约缺口”。
- `internal/handler/tag_handler.go`：后端已有标签列表、搜索、创建、更新、删除、alias 查询等基础治理接口，是 Phase 19 的服务基线。
- `internal/service/tag_governance_service.go`：已有标签归并/规范化相关后端治理逻辑与 alias 解析线索，可复用于重复标签治理。

### Established Patterns
- 桌面端能力继续沿用“独立导航页 + Fluent 壳层工作区”模式，不应回退到 legacy 入口或页面内临时弹窗主导的治理方式。
- 现有标签治理已经区分标签实体、alias、image-tag 关联、统计信息与 review state，说明 Phase 19 适合在现有模型上补齐桌面治理 UX，而不是重写领域模型。
- 项目已有“破坏性动作必须强确认并返回影响数量”的运营设计传统，这与本阶段删除确认显示影响图片数的需求一致。

### Integration Points
- 标签治理页与桌面导航的接入点在 `flutter_app/lib/app/fluent_app_shell.dart` / 现有导航结构，无需新增独立入口体系。
- “跳转到受影响图片集合”应复用现有图库/搜索视图与标签筛选状态，而不是新建孤立图片结果页。
- 批量治理将直接影响 `TagProvider` 状态模型、列表选择模式与后端批量契约，需要 planner 在前后端任务拆分中显式覆盖。

</code_context>

<specifics>
## Specific Ideas

- 标签治理主体验应保持“列表视图先行”，不要为了治理能力把页面直接升级成重型后台工作台。
- 重复标签合并应是一条明确、可确认的工作流，而不是借用普通“编辑标签”流程偷偷完成。
- 删除确认不能只写“会删除关联”，而要把受影响图片数量明确展示出来，保证可解释性。
- 管理员在治理完标签后，应能快速跳到受影响图片集合验证结果，而不是停留在抽象统计层。

</specifics>

<deferred>
## Deferred Ideas

- 完整 taxonomy / 层级分类体系治理 — 超出本阶段 success criteria，后续可独立扩展
- 标签颜色语义、分组视觉体系、统一色板规范 — 可在后续桌面体验增强阶段讨论
- 在治理页内直接嵌入重型图片预览工作区 — 当前优先采用跳转联动而非页内预览
- “强制删除任意标签并连带移除所有图片关联” — 不作为本阶段主路径

None — discussion stayed within phase scope

</deferred>

---

*Phase: 19-tag-management*
*Context gathered: 2026-04-05*
