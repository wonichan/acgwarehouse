# Image Detail Tag Picker UX Design

## Goal

优化图片详情页中的“添加标签 / 编辑标签”弹窗，减少重复输入成本，让用户可以优先直接选择已有标签，同时保留模糊搜索与创建新标签能力。

## Problem Summary

当前实现位于：

- `flutter_app/lib/widgets/add_tag_dialog.dart`
- `flutter_app/lib/widgets/edit_tag_dialog.dart`

现状是：

- 弹窗打开后默认只有输入框
- 用户通常需要先输入关键词，才会看到候选标签
- 已有标签虽然支持模糊搜索，但缺少“直接点选”的首屏体验
- 高频标签场景下，重复输入带来不必要操作负担

这与图片详情页的高频打标工作流不匹配。

## Approved Direction

采用 **“默认可选标签 + 搜索补充 + 滑动加载更多”** 的方案：

- 弹窗打开时先加载一批已有标签
- 默认列表按 `usageCount` 优先展示更常用标签
- 用户无需输入即可直接点击已有标签完成添加或替换
- 顶部保留搜索框，输入后切换为模糊搜索结果
- 默认列表支持继续滚动/加载更多标签，而不是只给固定一屏
- 继续保留“创建新标签”能力，避免阻断长尾标签录入

## Scope

In scope:

- `AddTagDialog` 的首屏已有标签选择体验
- `EditTagDialog` 的首屏已有标签选择体验
- 默认标签列表的排序、分页/增量加载、滚动行为
- 搜索态与默认列表态之间的切换规则
- 复用现有标签接口完成默认列表与搜索列表加载

Out of scope:

- 修改图片详情页主布局
- 修改标签确认/拒绝/合并/删除逻辑
- 新增后端标签专用搜索接口
- 重新设计标签管理后台
- 修改标签领域模型或数据库结构

## Codebase Evidence

- `flutter_app/lib/widgets/add_tag_dialog.dart` 当前在 `onChanged` 中调用 `tagService.searchTags(query)`，只有输入后才显示候选列表。
- `flutter_app/lib/widgets/edit_tag_dialog.dart` 当前也是输入驱动候选结果，没有默认已有标签区。
- `flutter_app/lib/services/tag_service.dart` 已提供：
  - `fetchTags({search, limit, offset})`
  - `searchTags(query)`
- 后端已有 `GET /api/v1/tags`，支持：
  - `limit`
  - `offset`
  - `search`

因此本次设计可以优先复用现有分页与搜索能力，无需新增 API。

## UX Principles

1. 常用操作优先：优先降低“选已有标签”的成本。
2. 搜索是补充，不是起点：搜索框保留，但不应成为默认进入路径。
3. 新建能力不断流：找不到目标标签时，仍可快速新建。
4. 渐进加载：默认列表可继续滑动加载更多，避免一次性渲染过多标签。
5. 行为一致：添加与编辑的交互模型尽量一致，只在最终动作上区分。

## Interaction Model

### 1. Shared Dialog Structure

`AddTagDialog` 与 `EditTagDialog` 统一采用以下结构：

1. 顶部说明文案（编辑态额外显示当前标签）
2. 搜索输入框
3. 默认可选标签区 / 搜索结果区
4. 底部操作区（取消、创建新标签）

### 2. Default State (No Search Input)

当输入框为空时：

- 弹窗自动请求已有标签列表
- 使用 `fetchTags(limit, offset)` 加载
- 首屏结果按服务端既有顺序展示；产品语义上视为按 `usageCount` 优先
- 结果区展示为可点击列表
- 用户滚动到底部附近时继续加载下一页
- 存在更多数据时持续增量追加，不替换已有内容

默认态目标是：**不输入也能完成大部分标签操作**。

### 3. Search State

当输入框非空时：

- 切换为搜索结果模式
- 使用 `searchTags(query)` 获取匹配结果
- 搜索结果覆盖默认列表展示区
- 清空输入后回到默认列表模式，并恢复已加载的默认列表结果

搜索态目标是：**用于长尾标签和模糊查找，而不是替代默认选择区**。

### 4. Add Tag Dialog Behavior

在 `AddTagDialog` 中：

- 点击默认列表中的已有标签：立即调用添加逻辑
- 点击搜索结果中的已有标签：立即调用添加逻辑
- 输入框内容没有匹配目标时：仍可点击“创建新标签”

### 5. Edit Tag Dialog Behavior

在 `EditTagDialog` 中：

- 保留“将当前标签更改为 xxx”的上下文提示
- 点击默认列表中的已有标签：立即返回替换目标标签
- 点击搜索结果中的已有标签：立即返回替换目标标签
- 输入新名称时：仍可通过“创建新标签”返回新标签替换信息

## Data Loading Rules

### Default List Source

- 使用现有 `TagService.fetchTags(limit: ..., offset: ...)`
- 不新增新接口
- 首屏加载数量应控制在对话框中可感知但不过载的范围
- 下一页通过 `offset` 推进

### Incremental Loading

- 仅默认列表态支持滚动加载更多
- 搜索态先不做无限滚动，保持轻量实现
- 若默认列表正在加载下一页，应显示轻量 loading 提示
- 若已无更多数据，则停止继续请求

## UI States

至少覆盖以下状态：

1. 默认列表初始加载中
2. 默认列表加载成功
3. 默认列表为空
4. 默认列表加载更多中
5. 搜索中
6. 搜索无结果
7. 请求失败（默认列表或搜索）

## Error Handling

- 默认列表加载失败时，应保留弹窗可用性，至少允许用户继续输入搜索或创建新标签
- 搜索失败时，不应关闭弹窗
- 添加/替换失败继续沿用当前调用方已有错误反馈方式

## Testing Intent

需要新增或更新测试，验证以下行为：

1. 弹窗初次打开时会请求默认标签列表
2. 默认标签列表会展示已有标签
3. 点击默认标签可以完成添加或编辑返回
4. 输入搜索词后会切换到搜索结果模式
5. 清空搜索词后恢复默认标签列表
6. 默认列表支持滚动加载更多并追加结果
7. 默认列表加载失败时仍可继续创建新标签

## Implementation Notes

- 优先抽取 `AddTagDialog` / `EditTagDialog` 共享的标签列表加载与展示逻辑，避免两份分页/搜索状态机重复分叉。
- 保持现有返回结构不变，尽量减少对 `ImageMetadataPanel` 和 `TagProvider` 的影响。
- 本次不要求改动后端排序规则；如果当前 `/tags` 默认顺序已经与 usage 优先不一致，再作为单独实现决策评估。
