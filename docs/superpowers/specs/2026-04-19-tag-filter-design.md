# 标签治理页筛选设计

- 日期：2026-04-19
- 主题：为前端标签治理界面增加复合筛选能力，并让筛选条件在治理列表查询链路中统一执行 AND 逻辑

## 目标

在现有标签治理页中增加一组可组合筛选条件，让用户能稳定筛出目标标签，而不是继续依赖全文搜索与人工滚动查找。

本次新增筛选条件包括：

1. 祖级 / 父级 / 子级（按**当前标签自身 `level`** 筛选）
2. 无父级（严格定义为 `parent_id IS NULL AND level != root`）
3. 使用量区间
4. AI 生成
5. 手动生成

所有条件组之间统一采用 **AND** 逻辑；但组内规则不是统一 OR：

1. `levels` 组内采用 **OR**。
2. `source_ai` 与 `source_manual` 是两个独立布尔约束；二者同时开启时，语义是 **AND**，不是 OR。

## 已确认产品规则

1. `祖级 / 父级 / 子级` 的语义，是按当前标签自己的 `level` 过滤，不按祖先链过滤。
2. `无父级` 不包含 root 标签；它只匹配 `parent_id IS NULL AND level != 'root'` 的孤立标签。
3. `AI 生成` 与 `手动生成` 可以同时开启；同时开启时，标签必须同时满足 `ai_count > 0` 且 `manual_count > 0`。
4. `使用量区间` 本次默认按**直接使用量**过滤，不按树总使用量过滤。
5. 筛选面板采用**草稿态 / 已应用态**双态结构，点击“应用筛选”后才会真正刷新列表。
6. 治理列表的分页、搜索与筛选必须共用同一条后端查询链，避免前端本地筛选与后端分页结果不一致。

## 当前实现与约束

### 前端

- `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart`
  - 已有治理页主工作区、统计卡、搜索入口与列表容器。
- `flutter_app/lib/providers/tag_provider.dart`
  - 已管理治理列表、分页、搜索、批量操作与刷新。
  - 当前 `loadGovernanceTags({String? search})` 只支持搜索，不支持复合筛选。
- `flutter_app/lib/services/tag_service.dart`
  - 当前 `fetchGovernanceTags()` 只透传 `search/limit/offset`。
- `flutter_app/lib/models/tag_governance.dart`
  - 当前治理行数据已经具备本次筛选所需字段：`level`、`parentId`、`usageCount`、`aiCount`、`manualCount`。

### 后端

- `internal/handler/tag_handler.go`
  - `GetGovernanceTags()` 当前只解析 `search/limit/offset`。
- `internal/service/tag_admin_service.go`
  - `ListGovernanceTags(ctx, search, limit, offset)` 是治理查询总入口。
  - 当前先解析标签切片，再计算 direct/tree 统计。
- `internal/domain/tag.go`
  - 已有 `level`、`parent_id`、`usage_count`。
- `internal/domain/image_tag.go`
  - 已有 `source = ai | manual`。

这些基础字段已足够支撑本次筛选，不需要额外改数据库表结构。

## 方案比较

### 方案一：扩展现有 `/tags/governance` 查询参数（推荐）

在现有 `GET /api/v1/tags/governance` 上直接扩展筛选参数。

优点：

1. 复用现有治理页分页与查询链路。
2. 改动范围集中在现有 handler / service / provider / service 层。
3. 与当前代码结构最一致，风险最低。

缺点：

1. 查询参数会变长。
2. `ListGovernanceTags` 需要新增过滤法典。

### 方案二：前端本地筛选

优点：

1. 后端改动少。

缺点：

1. 与现有分页加载模式冲突。
2. 数据量上涨后会失真。
3. AI / 手动 / 使用量等统计本来就依赖后端聚合，前端接管会让语义漂移。

### 方案三：新建专用查询接口或 POST 查询体

优点：

1. 协议最清晰。
2. 未来扩展空间最大。

缺点：

1. 超出本次最小闭环。
2. 会引入新的接口面与更大测试成本。

### 结论

采用**方案一**：直接扩展现有 `GET /api/v1/tags/governance`。

## 推荐设计

## 前端设计

### 1. 新增治理筛选状态模型

建议新增独立筛选模型，例如：

- `TagGovernanceFilterState`

建议字段：

- `levels: Set<String>`
- `orphanOnly: bool`
- `minUsageCount: int?`
- `maxUsageCount: int?`
- `sourceAi: bool`
- `sourceManual: bool`
- `search: String`

同时建议提供：

- `isEmpty`
- `copyWith`
- `toQueryParameters`
- `summaryChips`

### 2. Provider 采用双态统御

在 `flutter_app/lib/providers/tag_provider.dart` 中新增：

- `governanceDraftFilters`
- `governanceAppliedFilters`

行为规则：

1. 用户修改任一控件时，只更新 `draft`。
2. 点击“应用筛选”时，将 `draft` 复制到 `applied`，再从第一页重载治理列表。
3. 点击“重置”时，清空 `draft` 与 `applied`，再重载默认列表。
4. 无限滚动加载更多时，始终沿用 `applied`。

### 3. 筛选面板布局

在 `flutter_app/lib/widgets/tag_management/tag_management_workspace.dart` 中，将筛选面板放在：

- 统计卡下方
- 治理列表上方

筛选面板包含：

1. 层级多选 chips：祖级 / 父级 / 子级
2. 无父级开关
3. 使用量最小值 / 最大值输入
4. AI 生成 / 手动生成开关
5. 重置按钮
6. 应用筛选按钮
7. 已生效条件摘要区

同时，`flutter_app/lib/widgets/tag_management/tag_management_list.dart` 也属于本次改动范围。

该文件当前负责治理列表的渲染与局部过滤行为；本次必须保证：

1. 治理列表的显示结果以**后端已过滤结果**为准。
2. 不允许再叠加会改变结果集语义的前端本地过滤逻辑。
3. 如果仍保留前端局部搜索高亮，只能作为展示增强，不能改变最终列表命中集合。

### 4. 交互法典

1. 任何筛选控件的即时变化，不直接触发请求。
2. 只有“应用筛选”才会刷新列表。
3. 搜索也纳入同一个草稿 / 应用体系，避免搜索与筛选脱节。
4. 治理列表上方显示当前已生效条件摘要，例如：
   - `层级: root,parent`
   - `无父级`
   - `使用量: 10~500`
   - `来源: AI+手动`

### 5. 选择态法典

当前治理页存在多选、批量操作与合并源标签状态，因此必须明确筛选切换后的选择态规则。

推荐规则：

1. 点击“应用筛选”时，清空 `selectedGovernanceIds`。
2. 点击“重置”时，清空 `selectedGovernanceIds` 与 `activeMergeSource`。
3. 任意治理动作（删除、合并、批量清理、批量编辑）成功后触发列表重载时，也清空选择态。
4. 不保留“当前已不可见但仍被选中”的标签，避免对隐藏项执行批量操作。

这是为了保证筛选切换后，用户看到的就是实际操作对象。

## 后端设计

### 1. 保持原路径不变

继续使用：

- `GET /api/v1/tags/governance`

新增查询参数：

- `search`
- `levels=root,parent,child`
- `orphan_only=true`
- `min_usage_count`
- `max_usage_count`
- `source_ai=true`
- `source_manual=true`
- `limit`
- `offset`

### 2. 查询语义

后端统一遵循：

1. 不同筛选组之间全部 **AND**。
2. `levels` 组内部是 `IN (...)`，即组内 OR。
3. 来源组不是“组内 OR”，而是两个独立布尔条件：
   - `source_ai=true` 时要求 `ai_count > 0`
   - `source_manual=true` 时要求 `manual_count > 0`
4. `source_ai=true && source_manual=true` 时，同时要求：
   - `ai_count > 0`
   - `manual_count > 0`
5. 当两个来源都未开启时，不对来源做过滤。

### 3. 无父级语义

`orphan_only=true` 必须严格解释为：

`parent_id IS NULL AND level != 'root'`

这条规则必须由后端保证，不能让前端自行推断。

### 4. 使用量区间语义

本次按**直接使用量**过滤。

推荐行为：

- 最小值存在时：`usage_count >= min_usage_count`
- 最大值存在时：`usage_count <= max_usage_count`

若用户仅填写一端，则按单边区间处理。

### 5. 来源语义

推荐直接基于现有治理统计字段过滤：

- `source_ai=true` → `ai_count > 0`
- `source_manual=true` → `manual_count > 0`

如果两个来源同时开启，则两个条件同时生效。

不新增表字段，不单独维护来源布尔列。

### 6. 实现落点

主要调整点：

- `internal/handler/tag_handler.go`
  - 扩展 `GetGovernanceTags()` 参数解析
- `internal/service/tag_admin_service.go`
  - 扩展 `ListGovernanceTags(...)` 或引入新的过滤输入模型
- `internal/handler/tag_handler.go` 中的 `tagAdminService` 接口
  - 同步迁移到新的过滤输入签名
- `internal/handler/tag_handler.go` 中其他 `ListGovernanceTags(...)` 调用点
  - 同步更新参数与测试
- 视实现方式而定，可在 service 内部直接过滤，或把筛选进一步下沉到 repository

推荐新建一个过滤输入结构，例如：

```go
type GovernanceTagFilter struct {
    Search         string
    Levels         []string
    OrphanOnly     bool
    MinUsageCount  *int
    MaxUsageCount  *int
    SourceAI       bool
    SourceManual   bool
    Limit          int
    Offset         int
}
```

然后把它作为治理查询的统一载体，替代继续堆叠函数参数。

### 7. 分页前过滤法典

这条规则必须写死：

**所有筛选条件都必须在 `total` 计算与 `limit/offset` 分页之前生效。**

原因：当前治理查询链路是“先取标签切片，再计算统计”。如果把 `orphan/source/usage` 等条件放在已分页结果上再过滤，会直接导致：

1. `total` 错误
2. 页面结果数量不稳定
3. 翻页漏数据或重数据
4. AND 语义被破坏

因此实现时必须保证：

1. 先得到完整命中集合
2. 再计算过滤后的 `total`
3. 最后才应用 `limit/offset`

### 8. 参数校验法典

后端必须统一校验以下输入：

1. `levels`
   - 仅允许 `root,parent,child`
   - 逗号分隔后需要 `trim + 去重`
2. `source_ai` / `source_manual` / `orphan_only`
   - 仅接受 `true/false`
   - 参数缺失等价于 `false`
3. `min_usage_count` / `max_usage_count`
   - 必须为非负整数
4. `min_usage_count > max_usage_count`
   - 直接返回 `400 Bad Request`

推荐策略：任何非法筛选参数都返回 `400`，不要静默吞掉。

同时需要明确错误责任：

1. 参数格式校验优先在 `handler` 层完成。
2. `handler` 能确定的非法输入直接返回 `400`。
3. `service` 只处理已通过基础解析的结构化输入；若发现语义非法，也返回可被 `handler` 映射为 `400` 的校验错误。
4. 真正的内部执行失败才返回 `500`。

### 9. 搜索命中集法典

“先过滤再分页”不仅适用于 `levels/orphan/source/usage`，也必须适用于 `search`。

当前实现里，搜索路径存在“先按 `limit+offset` 取结果，再做后续处理”的倾向；本次设计必须禁止这种做法。

实现要求：

1. `search` 必须先得到**完整命中集合**（包括标签名与别名命中）。
2. 然后与 `levels/orphan/source/usage` 做统一 AND 过滤。
3. 再计算过滤后的 `total`。
4. 最后才应用 `limit/offset`。

否则 `search + filters + pagination` 的 `total` 与翻页结果都会失真。

## 测试与验收

### 前端测试

优先扩展：

- `flutter_app/test/providers/tag_provider_test.dart`
- `flutter_app/test/services/tag_service_test.dart`
- `flutter_app/test/widgets/tag_management_workspace_test.dart`
- `flutter_app/test/widgets/tag_management_list_test.dart`（如当前项目尚无，可新增）

新增用例：

1. 默认无筛选时仍可正常分页加载。
2. `levels` 任意组合应用后，参数与摘要正确。
3. `orphanOnly=true` 时，请求参数正确传递并能刷新列表。
4. 使用量区间单边 / 双边输入都能正确透传。
5. `AI生成`、`手动生成` 单开与双开时，摘要和请求参数都正确。
6. 草稿态变化不会立即触发请求。
7. 点击“应用筛选”后才触发第一页重载。
8. 点击“重置”后，draft/applied 都回到默认态。
9. `loadMoreGovernanceTags()` 会继续沿用已应用筛选。
10. 搜索输入只更新草稿态，不立即请求；点击“应用筛选”后才把搜索词与其他筛选一起提交。
11. 应用筛选、重置、治理动作成功后，选择态会被清空。
12. 列表不会因前端本地过滤而把后端已命中的结果再次过滤掉。
13. 刷新、编辑、删除、合并后重载，仍沿用 `appliedFilters`。

### 后端测试

优先扩展：

- `internal/handler/tag_handler_test.go`
- `internal/service/tag_admin_service_test.go`

新增用例：

1. 仅 `levels` 过滤。
2. 仅 `orphan_only` 过滤，且 root 不会被带出。
3. 仅最小使用量、仅最大使用量、双边使用量区间。
4. 仅 `source_ai`。
5. 仅 `source_manual`。
6. `source_ai + source_manual` 同时开启。
7. `search + filters + pagination` 联合场景。
8. 多条件复合时确认整体为 AND 逻辑。
9. 过滤后的 `total` 等于全部命中数，而不是“当前页过滤后剩余数”。
10. 翻页时不会因过滤顺序错误导致漏项或重复项。
11. 非法 query 参数返回 `400`，而不是 `500`。
12. 别名命中搜索与其他筛选组合时，`total` 与结果仍正确。

### 失败判定

出现以下任一情况，即判定设计落地失败：

1. 条件之间被错误实现成 OR。
2. `orphan_only` 错误包含 root。
3. 草稿态变化直接触发请求。
4. 应用筛选后分页丢失筛选条件。
5. 重置后仍残留已生效筛选。
6. `AI生成 + 手动生成` 同时开启时，没有要求双命中。

## 不做的事

本次明确不包含：

1. 不引入基于祖先链的单独筛选条件。
2. 不新增“按树总使用量过滤”模式。
3. 不新建专用 POST 查询接口。
4. 不改数据库 schema。
5. 不重写现有治理列表的基础排序与分页机制。

## 实施边界

本次是对现有标签治理查询能力的增强，不是重构整个标签系统。

因此所有改动都应尽量围绕现有治理页与现有 `/tags/governance` 接口完成，避免把一次筛选增强演化成新的治理子系统。

## 后续计划入口

在该设计确认后，下一步应进入实施计划，细化为：

1. 前端模型与 Provider 变更
2. 前端筛选面板与摘要 UI
3. 服务层 query 参数透传
4. 后端过滤输入结构与 handler 扩展
5. `TagAdminService` 过滤实现
6. 前后端测试补齐
