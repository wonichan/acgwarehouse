import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag_governance_filter.dart';

void main() {
  test('empty filter has isEmpty=true and isNotEmpty=false', () {
    final filter = TagGovernanceFilterState();
    expect(filter.isEmpty, isTrue);
    expect(filter.isNotEmpty, isFalse);
  });

  test('toQueryParameters omits defaults', () {
    final filter = TagGovernanceFilterState();
    final params = filter.toQueryParameters();
    expect(params.containsKey('levels'), isFalse);
    expect(params.containsKey('orphan_only'), isFalse);
    expect(params.containsKey('min_usage_count'), isFalse);
    expect(params.containsKey('source_ai'), isFalse);
    expect(params.containsKey('search'), isFalse);
  });

  test('toQueryParameters includes active filters', () {
    final filter = TagGovernanceFilterState(
      levels: {'root', 'parent'},
      orphanOnly: true,
      minUsageCount: 10,
      sourceAI: true,
      search: '发色',
    );
    final params = filter.toQueryParameters();
    expect(params['levels'], 'parent,root');
    expect(params['orphan_only'], 'true');
    expect(params['min_usage_count'], '10');
    expect(params['source_ai'], 'true');
    expect(params['search'], '发色');
  });

  test('toQueryParameters includes maxUsageCount', () {
    final filter = TagGovernanceFilterState(maxUsageCount: 500);
    final params = filter.toQueryParameters();
    expect(params['max_usage_count'], '500');
  });

  test('toQueryParameters includes sourceManual', () {
    final filter = TagGovernanceFilterState(sourceManual: true);
    final params = filter.toQueryParameters();
    expect(params['source_manual'], 'true');
  });

  test('summaryChips returns correct descriptions for combined filters', () {
    final filter = TagGovernanceFilterState(
      levels: {'root'},
      orphanOnly: true,
      sourceAI: true,
      sourceManual: true,
    );
    final chips = filter.summaryChips;
    expect(chips, containsAll(['层级: root', '无父级', '来源: AI+手动']));
  });

  test('summaryChips shows usage range with both bounds', () {
    final filter = TagGovernanceFilterState(
      minUsageCount: 10,
      maxUsageCount: 500,
    );
    expect(filter.summaryChips, contains('使用量: 10~500'));
  });

  test('summaryChips shows usage range with min only', () {
    final filter = TagGovernanceFilterState(minUsageCount: 5);
    expect(filter.summaryChips, contains('使用量: 5+'));
  });

  test('summaryChips shows usage range with max only', () {
    final filter = TagGovernanceFilterState(maxUsageCount: 100);
    expect(filter.summaryChips, contains('使用量: 0~100'));
  });

  test('summaryChips shows AI only source', () {
    final filter = TagGovernanceFilterState(sourceAI: true);
    expect(filter.summaryChips, contains('来源: AI'));
  });

  test('summaryChips shows manual only source', () {
    final filter = TagGovernanceFilterState(sourceManual: true);
    expect(filter.summaryChips, contains('来源: 手动'));
  });

  test('summaryChips includes search keyword', () {
    final filter = TagGovernanceFilterState(search: 'blue');
    expect(filter.summaryChips, contains('关键词: blue'));
  });

  test('copyWith preserves unmodified fields', () {
    final filter = TagGovernanceFilterState(
      levels: {'root'},
      sourceAI: true,
    );
    final modified = filter.copyWith(orphanOnly: true);
    expect(modified.levels, {'root'});
    expect(modified.sourceAI, isTrue);
    expect(modified.orphanOnly, isTrue);
  });

  test('copyWith clearMinUsage clears the field', () {
    final filter = TagGovernanceFilterState(minUsageCount: 10);
    final cleared = filter.copyWith(clearMinUsage: true);
    expect(cleared.minUsageCount, isNull);
  });

  test('copyWith clearMaxUsage clears the field', () {
    final filter = TagGovernanceFilterState(maxUsageCount: 100);
    final cleared = filter.copyWith(clearMaxUsage: true);
    expect(cleared.maxUsageCount, isNull);
  });

  test('filter with only search is not empty', () {
    final filter = TagGovernanceFilterState(search: 'test');
    expect(filter.isEmpty, isFalse);
  });

  test('filter with only levels is not empty', () {
    final filter = TagGovernanceFilterState(levels: {'child'});
    expect(filter.isEmpty, isFalse);
  });
}
