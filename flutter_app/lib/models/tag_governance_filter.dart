class TagGovernanceFilterState {
  final Set<String> levels;
  final bool orphanOnly;
  final int? minUsageCount;
  final int? maxUsageCount;
  final bool sourceAI;
  final bool sourceManual;
  final String search;

  const TagGovernanceFilterState({
    this.levels = const {},
    this.orphanOnly = false,
    this.minUsageCount,
    this.maxUsageCount,
    this.sourceAI = false,
    this.sourceManual = false,
    this.search = '',
  });

  bool get isEmpty =>
      levels.isEmpty &&
      !orphanOnly &&
      minUsageCount == null &&
      maxUsageCount == null &&
      !sourceAI &&
      !sourceManual &&
      search.isEmpty;

  bool get isNotEmpty => !isEmpty;

  TagGovernanceFilterState copyWith({
    Set<String>? levels,
    bool? orphanOnly,
    int? minUsageCount,
    bool clearMinUsage = false,
    int? maxUsageCount,
    bool clearMaxUsage = false,
    bool? sourceAI,
    bool? sourceManual,
    String? search,
  }) {
    return TagGovernanceFilterState(
      levels: levels ?? this.levels,
      orphanOnly: orphanOnly ?? this.orphanOnly,
      minUsageCount:
          clearMinUsage ? null : (minUsageCount ?? this.minUsageCount),
      maxUsageCount:
          clearMaxUsage ? null : (maxUsageCount ?? this.maxUsageCount),
      sourceAI: sourceAI ?? this.sourceAI,
      sourceManual: sourceManual ?? this.sourceManual,
      search: search ?? this.search,
    );
  }

  Map<String, String> toQueryParameters() {
    final params = <String, String>{};
    if (levels.isNotEmpty) {
      final sorted = levels.toList()..sort();
      params['levels'] = sorted.join(',');
    }
    if (orphanOnly) params['orphan_only'] = 'true';
    if (minUsageCount != null) {
      params['min_usage_count'] = minUsageCount.toString();
    }
    if (maxUsageCount != null) {
      params['max_usage_count'] = maxUsageCount.toString();
    }
    if (sourceAI) params['source_ai'] = 'true';
    if (sourceManual) params['source_manual'] = 'true';
    if (search.isNotEmpty) params['search'] = search;
    return params;
  }

  List<String> get summaryChips {
    final chips = <String>[];
    if (levels.isNotEmpty) {
      final sorted = levels.toList()..sort();
      chips.add('层级: ${sorted.join(',')}');
    }
    if (orphanOnly) chips.add('无父级');
    if (minUsageCount != null || maxUsageCount != null) {
      final lo = minUsageCount ?? 0;
      final hi = maxUsageCount != null ? '~$maxUsageCount' : '+';
      chips.add('使用量: $lo$hi');
    }
    if (sourceAI && sourceManual) {
      chips.add('来源: AI+手动');
    } else if (sourceAI) {
      chips.add('来源: AI');
    } else if (sourceManual) {
      chips.add('来源: 手动');
    }
    if (search.isNotEmpty) chips.add('关键词: $search');
    return chips;
  }
}
