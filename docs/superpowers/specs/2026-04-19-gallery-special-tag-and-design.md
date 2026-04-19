# 图库特殊标签 AND 筛选修复设计

- 日期：2026-04-19
- 主题：修复 Flutter 桌面图库中“标签未确认”“未打标签”与普通标签脱钩的问题，统一纳入 AND 筛选法典

## 目标

当前图库筛选面板中，普通标签与两个特殊标签存在语义断层：

1. `未打标签` 当前被实现为排他模式，开启后会清空普通标签与未确认状态。
2. `标签未确认` 与普通标签的关系未被明确定义为统一 AND 法则。

本次必须将三类约束统一为同一筛选体系：

- 普通标签（`exactTagIds` / `subtreeRootTagIds`）
- `标签未确认`（`hasPendingTags == true`）
- `未打标签`（`hasTags == false`）

其中：

- 普通标签 与 特殊标签：**AND**
- `未打标签` 与 `标签未确认`：**互斥**

不允许前端擅自把特殊标签与普通标签改写成互斥模式。

## 已裁定产品规则

1. 普通标签与特殊标签之间统一采用 **AND**。
2. `普通标签 + 标签未确认`：返回同时命中普通标签且存在待确认标签的图片。
3. `普通标签 + 未打标签`：按严格 AND 透传，命中结果交由后端真实数据裁定。
4. `标签未确认` 与 `未打标签` 两个特殊标签彼此 **互斥**，不得同时开启。
5. 当用户开启其中一个特殊标签时，前端必须自动关闭另一个特殊标签，但不得清除普通标签选择。
6. 只有用户主动清空筛选时，普通标签与特殊标签才整体清除。

## 根因审判

### 1. 状态模型将 `未打标签` 视为排他模式

文件：`flutter_app/lib/models/gallery_filter_state.dart`

当前 `normalized()` 逻辑：

- 一旦 `hasTags == false`，直接返回 `GalleryFilterState(hasTags: false)`
- 这会清空：
  - `exactTagIds`
  - `subtreeRootTagIds`
  - `hasPendingTags`

结果：`未打标签` 不再是普通筛选条件，而是篡夺整个筛选状态的特殊模式。

### 2. 面板交互直接重建为单字段状态

文件：`flutter_app/lib/widgets/fluent_tag_filter_pane.dart`

当前 `_buildUntaggedToggle()` 逻辑在打开开关时执行：

- `GalleryFilterState(hasTags: false).normalized()`

这意味着 UI 层在状态提交前就提前完成了排他清洗，进一步放大了断层。

### 3. Provider / Widget 测试已把错误行为固化为契约

相关测试文件：

- `flutter_app/test/models/gallery_filter_state_test.dart`
- `flutter_app/test/providers/image_provider_filter_state_test.dart`
- `flutter_app/test/providers/image_provider_has_tags_test.dart`
- `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`

这些测试将“切到未打标签就清空普通标签/未确认”或“特殊标签与普通标签不能并存”视为正确结果，导致错误语义被反复回灌。

## 方案比较

### 方案一：保留排他模式

做法：继续让 `未打标签` 清空普通标签。

优点：

1. 改动最少。

缺点：

1. 直接违背用户指令。
2. 破坏统一筛选语义。
3. 继续让 UI 接管业务规则。

结论：否决。

### 方案二：保留特殊标签互斥，仅去除其与普通标签的排他关系（推荐）

做法：

1. `GalleryFilterState` 保留普通标签与单一特殊标签共存。
2. `FluentTagFilterPane` 切换特殊标签时只对另一个特殊标签执行互斥清理，不清除普通标签组。
3. `ImageListProvider` 与 API 层照实透传所有筛选条件。

优点：

1. 直接符合当前 API 结构。
2. 改动面集中在 Flutter 前端与测试。
3. 能最小闭环修复“特殊标签不参与 AND”。

缺点：

1. 需要重写现有互斥测试。
2. 需要精确实现“特殊标签彼此互斥，但对普通标签不互斥”的细粒度法典。

### 方案三：抽象统一 specialFilters 枚举层

做法：新增 `specialFilters` 结构，重写筛选模型表达。

优点：

1. 结构长期更整齐。

缺点：

1. 超出本次最小闭环。
2. 会放大改动范围与回归成本。

## 结论

采用 **方案二**：保留特殊标签彼此互斥，但去除它们与普通标签之间的错误互斥。

## 推荐设计

### 1. 状态模型法典

文件：`flutter_app/lib/models/gallery_filter_state.dart`

调整规则：

1. `normalized()` 不再因为 `hasTags == false` 而清空普通标签字段。
2. `clear()` 仍返回空状态。
3. `isEmpty` 定义保持不变：仅在所有字段都为空时为真。

设计结果：

- `exactTagIds`
- `subtreeRootTagIds`
- `hasTags`
- `hasPendingTags`

状态约束：

- 普通标签可与 `hasTags` 或 `hasPendingTags` 之一共存。
- `hasTags == false` 与 `hasPendingTags == true` 不得共存。

归一化法典：

1. `normalized()` 必须成为最终收口点。
2. 若外部传入 `hasTags == false && hasPendingTags == true` 的冲突状态，`normalized()` 必须采用**确定性规则**收口到单一特殊标签。
3. 本次固定裁定：当冲突同时出现时，优先保留 `hasPendingTags == true`，并将 `hasTags` 归一为 `null`。
4. `normalized()` 在执行特殊标签互斥收口时，仍必须保留普通标签字段，不得再清空 `exactTagIds` 或 `subtreeRootTagIds`。

### 2. 面板交互法典

文件：`flutter_app/lib/widgets/fluent_tag_filter_pane.dart`

#### 未打标签开关

当前错误：打开时直接重建单字段状态。

修复后：

- 开启：`_draftFilter = _draftFilter.copyWith(hasTags: false, hasPendingTags: null).normalized()`
- 关闭：`_draftFilter = _draftFilter.copyWith(hasTags: null).normalized()`

只允许清理：

- `hasPendingTags`

不得清理：

- `exactTagIds`
- `subtreeRootTagIds`

#### 标签未确认开关

保持同样原则，但要执行特殊标签互斥：

- 开启：`_draftFilter = _draftFilter.copyWith(hasPendingTags: true, hasTags: null).normalized()`
- 关闭：`_draftFilter = _draftFilter.copyWith(hasPendingTags: null).normalized()`
- 不因开启/关闭而清除普通标签选择

#### 普通标签勾选

当前风险：`_toggleTag()` 仍可能在增删普通标签时顺带把特殊标签清空。

修复后法典：

1. 勾选或取消普通标签时，只更新 `exactTagIds` 或 `subtreeRootTagIds`。
2. 不得在普通标签切换时写入 `hasTags: null`。
3. 不得在普通标签切换时写入 `hasPendingTags: null`。
4. 普通标签切换必须完整保留当前单一特殊标签状态。

### 3. Provider 行为法典

文件：`flutter_app/lib/providers/image_provider.dart`

调整规则：

1. `applyFilter()` 只接收统一 filter，并将其原样透传到 API 请求层。
2. `setTagFilter()` 不再清除 `hasTags` 或 `hasPendingTags`。
3. `setHasTagsFilter()` 不再重建 `GalleryFilterState(hasTags: hasTags)`；应基于现有 `_filter.copyWith(hasTags: hasTags, hasPendingTags: hasTags == false ? null : _filter.hasPendingTags)` 更新。
4. `setHasPendingTagsFilter()` 开启时应清除 `hasTags`，关闭时仅清除自身。

结果：所有单项 setter 都变成“增删一个约束组”；但两个特殊标签之间仍保持互斥。

### 4. API 透传法典

文件：

- `flutter_app/lib/services/api_service.dart`
- `flutter_app/test/services/api_service_test.dart`
- 以及相关调用测试

要求：

1. 若 `exactTagIds` 非空，透传 `exact_tag_ids`。
2. 若 `subtreeRootTagIds` 非空，透传 `subtree_root_tag_ids`。
3. 若 `hasTags != null`，透传 `has_tags`。
4. 若 `hasPendingTags != null`，透传 `has_pending_tags`。

任何字段都不得因其他字段存在而被本地消音。

验收时必须分别覆盖：

1. `exact_tag_ids + has_tags`
2. `exact_tag_ids + has_pending_tags`
3. `subtree_root_tag_ids + has_tags`
4. `subtree_root_tag_ids + has_pending_tags`

### 5. 特殊标签互斥语义

UI 必须实现以下法典：

1. 开启“未打标签”时，自动关闭“标签未确认”。
2. 开启“标签未确认”时，自动关闭“未打标签”。
3. 两者关闭任意一个时，不影响普通标签。
4. 不需要额外弹提示；状态切换本身即为法典表达。

## 测试设计

### 模型测试

文件：`flutter_app/test/models/gallery_filter_state_test.dart`

新增/改写覆盖：

1. `normalized()` 在 `hasTags == false` 时保留普通标签。
2. `normalized()` 在 `hasPendingTags == true` 时保留普通标签。
3. `normalized()` 对 `hasTags == false && hasPendingTags == true` 的冲突输入会按确定性规则收口为 `hasPendingTags == true`，且不清空普通标签。
4. `copyWith()` 支持普通标签与单一特殊标签共存。

### Provider 测试

文件：

- `flutter_app/test/providers/image_provider_filter_state_test.dart`
- `flutter_app/test/providers/image_provider_has_tags_test.dart`
- `flutter_app/test/services/api_service_test.dart`

新增/改写覆盖：

1. `applyFilter()` 会向 API 透传 `普通标签 + 单一特殊标签` 的组合。
2. `setTagFilter()` 不再清除 `hasTags` / `hasPendingTags`。
3. `setHasTagsFilter(false)` 不再清除已选普通标签，但会清除 `hasPendingTags`。
4. `setHasPendingTagsFilter(true)` 不再清除普通标签，但会清除 `hasTags`。
5. `ApiService` 在四种组合下都能正确序列化 query 参数。

### Widget 测试

文件：`flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`

新增/改写覆盖：

1. 面板初始带普通标签时，勾选“未打标签”后应用，普通标签仍保留。
2. 面板先勾选“标签未确认”再勾选“未打标签”时，应用后只保留 `hasTags=false`，普通标签仍保留。
3. 面板先勾选“未打标签”再勾选“标签未确认”时，应用后只保留 `hasPendingTags=true`，普通标签仍保留。
4. 已开启特殊标签时再增删普通标签，应用后特殊标签状态仍保留。
5. 旧的“未打标签会清空标签”的断言必须删除。

### 桌面入口回归

文件：`flutter_app/test/app/fluent_screens_test.dart`

新增/改写覆盖：

1. 顶部“未打标签”入口在已有普通标签选择时，不会把普通标签从 Provider 状态中抹掉。
2. 桌面入口调用 `setHasTagsFilter(false)` 后，仍符合“特殊标签与普通标签 AND、特殊标签之间互斥”的总法典。

### 清空筛选回归

文件：

- `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`
- `flutter_app/test/providers/image_provider_filter_state_test.dart`

新增/改写覆盖：

1. 点击“清空筛选”后，普通标签、`hasTags`、`hasPendingTags` 全部回到空状态。
2. 除“清空筛选”以外的交互，不得触发同等级别的整体清空。

## 验收标准

1. 选择普通标签后再勾选“标签未确认”，请求同时携带标签参数与 `has_pending_tags=true`。
2. 选择普通标签后再勾选“未打标签”，请求同时携带标签参数与 `has_tags=false`。
3. 开启一个特殊标签时，另一个特殊标签会被自动关闭。
4. 前端不再自动清除普通标签组。
5. 普通标签切换不会抹掉当前特殊标签状态。
6. 只有点击“清空筛选”时，普通标签与特殊标签才会整体清空。
7. 相关 Flutter 单元测试、组件测试全部通过。

## 实施范围

主要文件：

- `flutter_app/lib/models/gallery_filter_state.dart`
- `flutter_app/lib/providers/image_provider.dart`
- `flutter_app/lib/services/api_service.dart`
- `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
- `flutter_app/test/app/fluent_screens_test.dart`
- `flutter_app/test/models/gallery_filter_state_test.dart`
- `flutter_app/test/providers/image_provider_filter_state_test.dart`
- `flutter_app/test/providers/image_provider_has_tags_test.dart`
- `flutter_app/test/services/api_service_test.dart`
- `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`

本次不需要改后端数据库结构；也不要求新增接口。
