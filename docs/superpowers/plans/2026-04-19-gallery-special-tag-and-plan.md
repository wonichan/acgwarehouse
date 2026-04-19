# 图库特殊标签 AND/互斥筛选修复 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 Flutter 桌面图库中的特殊标签筛选语义，使“未打标签”与“标签未确认”彼此互斥，但它们各自与普通标签保持 AND，且只有“清空筛选”才会整体清空状态。

**Architecture:** 仅在 `flutter_app` 前端层完成修复。把互斥/AND 语义统一收口到 `GalleryFilterState.normalized()`、`ImageListProvider` setter 与 `FluentTagFilterPane` 交互逻辑，避免任一 UI 分支擅自清空普通标签。通过模型测试、Provider 测试、API 序列化测试、Widget 测试和桌面入口回归测试建立完整防线。

**Tech Stack:** Flutter, Provider, flutter_test, MockClient/http, existing gallery filter state and API service.

---

## File Structure

### Production files
- Modify: `flutter_app/lib/models/gallery_filter_state.dart`
  - 收口特殊标签互斥与普通标签保留规则。
- Modify: `flutter_app/lib/providers/image_provider.dart`
  - 统一 setter / applyFilter 的状态更新语义。
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`
  - 修复“未打标签”“标签未确认”开关与普通标签切换逻辑。
- Verify/Modify if needed: `flutter_app/lib/services/api_service.dart`
  - 确认 query 参数在组合条件下被同时正确序列化。

### Test files
- Modify: `flutter_app/test/models/gallery_filter_state_test.dart`
- Modify: `flutter_app/test/providers/image_provider_filter_state_test.dart`
- Modify: `flutter_app/test/providers/image_provider_has_tags_test.dart`
- Modify/Create: `flutter_app/test/services/api_service_test.dart`
- Modify: `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`
- Modify: `flutter_app/test/app/fluent_screens_test.dart`

---

## Chunk 1: 状态模型法典收口

### Task 1: 为 GalleryFilterState 写失败测试并实现确定性归一化

**Files:**
- Modify: `flutter_app/test/models/gallery_filter_state_test.dart`
- Modify: `flutter_app/lib/models/gallery_filter_state.dart`

- [ ] **Step 1: 写失败测试，定义特殊标签互斥但保留普通标签**

```dart
test('normalized keeps normal tags when hasTags is false', () {
  final state = GalleryFilterState(
    exactTagIds: {1, 2},
    subtreeRootTagIds: {3},
    hasTags: false,
  );

  final normalized = state.normalized();

  expect(normalized.exactTagIds, {1, 2});
  expect(normalized.subtreeRootTagIds, {3});
  expect(normalized.hasTags, isFalse);
});

test('normalized resolves conflicting special flags deterministically', () {
  final state = GalleryFilterState(
    exactTagIds: {9},
    hasTags: false,
    hasPendingTags: true,
  );

  final normalized = state.normalized();

  expect(normalized.exactTagIds, {9});
  expect(normalized.hasTags, isNull);
  expect(normalized.hasPendingTags, isTrue);
});
```

- [ ] **Step 2: 运行模型测试，确认当前失败**

Run:
```bash
flutter test test/models/gallery_filter_state_test.dart
```
Expected: FAIL because current `normalized()` clears tag selections for `hasTags == false`.

- [ ] **Step 3: 最小实现 `normalized()` 与必要辅助逻辑**

```dart
GalleryFilterState normalized() {
  if (hasTags == false && hasPendingTags == true) {
    return GalleryFilterState(
      exactTagIds: exactTagIds,
      subtreeRootTagIds: subtreeRootTagIds,
      hasPendingTags: true,
    );
  }

  return GalleryFilterState(
    exactTagIds: exactTagIds,
    subtreeRootTagIds: subtreeRootTagIds,
    hasTags: hasTags,
    hasPendingTags: hasPendingTags,
  );
}
```

- [ ] **Step 4: 重新运行模型测试，确认通过**

Run:
```bash
flutter test test/models/gallery_filter_state_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/models/gallery_filter_state.dart flutter_app/test/models/gallery_filter_state_test.dart
git commit -m "fix: normalize gallery special tag filters"
```

---

## Chunk 2: Provider 语义修复

### Task 2: 用 TDD 修复 applyFilter 与 setter 的互斥/AND 语义

**Files:**
- Modify: `flutter_app/test/providers/image_provider_filter_state_test.dart`
- Modify: `flutter_app/test/providers/image_provider_has_tags_test.dart`
- Modify: `flutter_app/lib/providers/image_provider.dart`

- [ ] **Step 1: 为 applyFilter 写失败测试，锁定组合透传**

```dart
test('applyFilter forwards normal tags with pending special filter', () async {
  final api = RecordingApiService();
  final provider = ImageListProvider(api);

  await provider.applyFilter(
    GalleryFilterState(exactTagIds: {2}, hasPendingTags: true),
  );

  expect(api.lastExactTagIds, [2]);
  expect(api.lastHasPendingTags, isTrue);
  expect(api.lastHasTags, isNull);
});

test('applyFilter resolves conflicting special flags before API call', () async {
  final api = RecordingApiService();
  final provider = ImageListProvider(api);

  await provider.applyFilter(
    GalleryFilterState(exactTagIds: {2}, hasTags: false, hasPendingTags: true),
  );

  expect(api.lastExactTagIds, [2]);
  expect(api.lastHasTags, isNull);
  expect(api.lastHasPendingTags, isTrue);
});
```

- [ ] **Step 2: 为 setter 写失败测试，锁定“特殊标签彼此互斥但不清普通标签”**

```dart
test('setHasTagsFilter(false) keeps selected normal tags and clears pending flag', () async {
  final api = RecordingApiService();
  final provider = ImageListProvider(api);

  await provider.applyFilter(
    GalleryFilterState(exactTagIds: {7}, hasPendingTags: true),
  );
  await provider.setHasTagsFilter(false);

  expect(provider.filter.exactTagIds, {7});
  expect(provider.filter.hasTags, isFalse);
  expect(provider.filter.hasPendingTags, isNull);
});

test('setHasPendingTagsFilter(true) keeps selected normal tags and clears hasTags', () async {
  final api = RecordingApiService();
  final provider = ImageListProvider(api);

  await provider.applyFilter(
    GalleryFilterState(exactTagIds: {7}, hasTags: false),
  );
  await provider.setHasPendingTagsFilter(true);

  expect(provider.filter.exactTagIds, {7});
  expect(provider.filter.hasTags, isNull);
  expect(provider.filter.hasPendingTags, isTrue);
});
```

- [ ] **Step 3: 运行 Provider 测试，确认当前失败**

Run:
```bash
flutter test test/providers/image_provider_filter_state_test.dart
flutter test test/providers/image_provider_has_tags_test.dart
```
Expected: FAIL because current setters clear the wrong fields and current `applyFilter` semantics normalize to legacy untagged mode.

- [ ] **Step 4: 最小修改 Provider 实现**

```dart
Future<void> setTagFilter(List<int> tagIds) async {
  await applyFilter(_filter.copyWith(exactTagIds: tagIds.toSet()));
}

Future<void> setHasTagsFilter(bool? hasTags) async {
  await applyFilter(
    _filter.copyWith(
      hasTags: hasTags,
      hasPendingTags: hasTags == false ? null : _filter.hasPendingTags,
    ),
  );
}

Future<void> setHasPendingTagsFilter(bool? hasPendingTags) async {
  await applyFilter(
    _filter.copyWith(
      hasPendingTags: hasPendingTags,
      hasTags: hasPendingTags == true ? null : _filter.hasTags,
    ),
  );
}
```

- [ ] **Step 5: 重新运行 Provider 测试，确认通过**

Run:
```bash
flutter test test/providers/image_provider_filter_state_test.dart
flutter test test/providers/image_provider_has_tags_test.dart
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add flutter_app/lib/providers/image_provider.dart flutter_app/test/providers/image_provider_filter_state_test.dart flutter_app/test/providers/image_provider_has_tags_test.dart
git commit -m "fix: preserve normal tags in gallery special filters"
```

---

## Chunk 3: API 序列化与面板交互

### Task 3: 为 API 序列化写测试并修正参数并存

**Files:**
- Modify/Create: `flutter_app/test/services/api_service_test.dart`
- Verify/Modify: `flutter_app/lib/services/api_service.dart`

- [ ] **Step 1: 写失败测试，覆盖四种 query 组合**

```dart
test('fetchImages serializes exact tags with has_tags', () async {
  // expect exact_tag_ids and has_tags together in request URL
});

test('fetchImages serializes exact tags with has_pending_tags', () async {
  // expect exact_tag_ids and has_pending_tags together in request URL
});

test('fetchImages serializes subtree roots with has_tags', () async {
  // expect subtree_root_tag_ids and has_tags together in request URL
});

test('fetchImages serializes subtree roots with has_pending_tags', () async {
  // expect subtree_root_tag_ids and has_pending_tags together in request URL
});
```

- [ ] **Step 2: 运行 API service 测试，确认失败或缺失**

Run:
```bash
flutter test test/services/api_service_test.dart
```
Expected: FAIL or missing test coverage.

- [ ] **Step 3: 最小修改 `ApiService.fetchImages()` 序列化逻辑**

重点检查：

```dart
if (exactTagIds != null && exactTagIds.isNotEmpty) {
  params['exact_tag_ids'] = exactTagIds.join(',');
}
if (subtreeRootTagIds != null && subtreeRootTagIds.isNotEmpty) {
  params['subtree_root_tag_ids'] = subtreeRootTagIds.join(',');
}
if (hasTags != null) {
  params['has_tags'] = hasTags.toString();
}
if (hasPendingTags != null) {
  params['has_pending_tags'] = hasPendingTags.toString();
}
```

- [ ] **Step 4: 重新运行 API service 测试，确认通过**

Run:
```bash
flutter test test/services/api_service_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/services/api_service.dart flutter_app/test/services/api_service_test.dart
git commit -m "test: cover gallery filter query serialization"
```

### Task 4: 用 Widget 测试修复筛选面板交互

**Files:**
- Modify: `flutter_app/test/widgets/fluent_tag_filter_pane_test.dart`
- Modify: `flutter_app/lib/widgets/fluent_tag_filter_pane.dart`

- [ ] **Step 1: 写失败测试，锁定三类面板行为**

```dart
testWidgets('untagged toggle keeps normal tags on apply', (tester) async {
  // initialFilter has exactTagIds: {1}
  // toggle untagged, apply
  // expect appliedFilter.exactTagIds == {1}, hasTags == false
});

testWidgets('pending then untagged keeps normal tags and clears pending', (tester) async {
  // enable pending, then enable untagged, apply
  // expect hasTags == false, hasPendingTags == null, tags preserved
});

testWidgets('changing normal tags does not clear active special tag', (tester) async {
  // start with hasPendingTags true, tap a tag checkbox, apply
  // expect tag selected and hasPendingTags still true
});

testWidgets('clear filter clears normal and special filters together', (tester) async {
  // set tags + special filter, tap clear, apply
  // expect all filter fields empty/null
});
```

- [ ] **Step 2: 运行 Widget 测试，确认当前失败**

Run:
```bash
flutter test test/widgets/fluent_tag_filter_pane_test.dart
```
Expected: FAIL because current widget rebuilds single-field untagged state and `_toggleTag()` clears `hasTags`.

- [ ] **Step 3: 最小修改面板交互实现**

```dart
_draftFilter = _draftFilter.copyWith(hasTags: false, hasPendingTags: null).normalized();
_draftFilter = _draftFilter.copyWith(hasPendingTags: true, hasTags: null).normalized();

// In _toggleTag():
_draftFilter = _draftFilter.copyWith(exactTagIds: nextIds).normalized();
// or subtreeRootTagIds: nextIds without forcing hasTags/hasPendingTags null
```

- [ ] **Step 4: 重新运行 Widget 测试，确认通过**

Run:
```bash
flutter test test/widgets/fluent_tag_filter_pane_test.dart
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add flutter_app/lib/widgets/fluent_tag_filter_pane.dart flutter_app/test/widgets/fluent_tag_filter_pane_test.dart
git commit -m "fix: keep normal tags in filter pane special toggles"
```

---

## Chunk 4: 桌面入口回归与整体验证

### Task 5: 覆盖 fluent_screens 顶部入口并完成回归验证

**Files:**
- Modify: `flutter_app/test/app/fluent_screens_test.dart`

- [ ] **Step 1: 写失败测试，锁定顶部“未打标签”入口不会抹掉普通标签**

```dart
testWidgets('top untagged entry keeps selected normal tags in provider', (tester) async {
  // seed provider with exactTagIds {1}
  // trigger top untagged action
  // expect provider.filter.exactTagIds == {1}
  // expect provider.filter.hasTags == false
});
```

- [ ] **Step 2: 运行桌面入口测试，确认失败或需要适配**

Run:
```bash
flutter test test/app/fluent_screens_test.dart
```
Expected: FAIL or expose missing regression coverage.

- [ ] **Step 3: 按最小需要调整测试夹具或入口行为**

说明：如果入口本身不需要生产代码改动，只补测试即可；若它有独立清空逻辑，再做最小修复。

- [ ] **Step 4: 运行完整 Flutter 回归集**

Run:
```bash
flutter test test/models/gallery_filter_state_test.dart
flutter test test/providers/image_provider_filter_state_test.dart
flutter test test/providers/image_provider_has_tags_test.dart
flutter test test/services/api_service_test.dart
flutter test test/widgets/fluent_tag_filter_pane_test.dart
flutter test test/app/fluent_screens_test.dart
```
Expected: PASS.

- [ ] **Step 5: 运行聚合回归（可选但推荐）**

Run:
```bash
flutter test
```
Expected: PASS or only unrelated pre-existing failures; if unrelated failures exist, document them clearly before proceeding.

- [ ] **Step 6: Commit**

```bash
git add flutter_app/test/app/fluent_screens_test.dart
git commit -m "test: cover gallery special filter desktop entry"
```

---

## Plan Review Loop

For each chunk above:

1. Dispatch a plan reviewer with the chunk content and spec path `docs/superpowers/specs/2026-04-19-gallery-special-tag-and-design.md`.
2. If issues are found, fix the plan chunk before execution.
3. Only execute once the chunk review passes.

## Execution Notes

- Follow strict TDD: test first, verify red, implement minimal code, verify green.
- Do not widen scope into backend schema or API contract redesign.
- Do not preserve the old “untagged mode clears all filters” behavior in any helper path.
- The only allowed global clear is the explicit “清空筛选” interaction.

Plan complete and saved to `docs/superpowers/plans/2026-04-19-gallery-special-tag-and-plan.md`. Ready to execute?
