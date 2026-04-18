class Tag {
  final int id;
  final String preferredLabel;
  final String slug;
  final String? primaryCategory;
  final String reviewState;
  final double trustScore;
  final int usageCount;
  final DateTime createdAt;
  final String? level;
  final int? parentId;

  const Tag({
    required this.id,
    required this.preferredLabel,
    required this.slug,
    this.primaryCategory,
    required this.reviewState,
    required this.trustScore,
    required this.usageCount,
    required this.createdAt,
    this.level,
    this.parentId,
  });

  factory Tag.fromJson(Map<String, dynamic> json) {
    return Tag(
      id: json['id'] as int,
      preferredLabel: json['preferred_label'] as String,
      slug: json['slug'] as String,
      primaryCategory: json['primary_category'] as String?,
      reviewState: json['review_state'] as String,
      trustScore: (json['trust_score'] as num).toDouble(),
      usageCount: json['usage_count'] as int,
      createdAt: DateTime.parse(json['created_at'] as String),
      level: json['level'] as String?,
      parentId: json['parent_id'] as int?,
    );
  }

  factory Tag.fromImageTagJson(Map<String, dynamic> json) {
    final label = (json['preferred_label'] as String?) ?? '';
    return Tag(
      id: (json['tag_id'] ?? json['id']) as int,
      preferredLabel: label,
      slug: _slugFromLabel(label),
      primaryCategory: null,
      reviewState: (json['review_state'] as String?) ?? 'pending',
      trustScore: (json['confidence'] as num?)?.toDouble() ?? 0,
      usageCount: 0,
      createdAt: DateTime.fromMillisecondsSinceEpoch(0),
      level: json['level'] as String?,
      parentId: json['parent_id'] as int?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'preferred_label': preferredLabel,
      'slug': slug,
      'primary_category': primaryCategory,
      'review_state': reviewState,
      'trust_score': trustScore,
      'usage_count': usageCount,
      'created_at': createdAt.toIso8601String(),
      if (level != null) 'level': level,
      if (parentId != null) 'parent_id': parentId,
    };
  }

  Tag copyWith({
    int? id,
    String? preferredLabel,
    String? slug,
    String? primaryCategory,
    String? reviewState,
    double? trustScore,
    int? usageCount,
    DateTime? createdAt,
    String? level,
    int? parentId,
  }) {
    return Tag(
      id: id ?? this.id,
      preferredLabel: preferredLabel ?? this.preferredLabel,
      slug: slug ?? this.slug,
      primaryCategory: primaryCategory ?? this.primaryCategory,
      reviewState: reviewState ?? this.reviewState,
      trustScore: trustScore ?? this.trustScore,
      usageCount: usageCount ?? this.usageCount,
      createdAt: createdAt ?? this.createdAt,
      level: level ?? this.level,
      parentId: parentId ?? this.parentId,
    );
  }

  @override
  String toString() {
    return 'Tag(id: $id, label: $preferredLabel, state: $reviewState, level: $level, parentId: $parentId)';
  }

  static String _slugFromLabel(String label) {
    return label.trim().toLowerCase().replaceAll(' ', '-');
  }
}

/// Lightweight tag node returned by lazy tree browse endpoints.
class TagBrowseNode {
  final int id;
  final String preferredLabel;
  final String? level;
  final int? parentId;
  final bool hasChildren;

  const TagBrowseNode({
    required this.id,
    required this.preferredLabel,
    this.level,
    this.parentId,
    required this.hasChildren,
  });

  factory TagBrowseNode.fromJson(Map<String, dynamic> json) {
    return TagBrowseNode(
      id: json['id'] as int,
      preferredLabel: json['preferred_label'] as String,
      level: json['level'] as String?,
      parentId: json['parent_id'] as int?,
      hasChildren: json['has_children'] as bool? ?? false,
    );
  }
}

/// Paged response for orphan tags endpoint.
class OrphanTagsPage {
  final List<TagBrowseNode> items;
  final int total;
  final bool hasMore;

  const OrphanTagsPage({
    required this.items,
    required this.total,
    required this.hasMore,
  });

  factory OrphanTagsPage.fromJson(Map<String, dynamic> json) {
    final items = (json['items'] as List? ?? [])
        .map((e) => TagBrowseNode.fromJson(e as Map<String, dynamic>))
        .toList();
    return OrphanTagsPage(
      items: items,
      total: json['total'] as int? ?? 0,
      hasMore: json['has_more'] as bool? ?? false,
    );
  }
}

/// Tag statistics for governance display
class TagStatistics {
  final int tagId;
  final String label;
  final int usageCount;
  final int pendingCount;
  final int confirmedCount;
  final int aiCount;
  final int manualCount;
  final String? level;
  final int? parentId;
  final int directUsageCount;
  final int treeUsageCount;
  final int directPendingCount;
  final int treePendingCount;
  final int directConfirmedCount;
  final int treeConfirmedCount;
  final int directAiCount;
  final int treeAiCount;
  final int directManualCount;
  final int treeManualCount;

  TagStatistics({
    required this.tagId,
    required this.label,
    required this.usageCount,
    this.pendingCount = 0,
    this.confirmedCount = 0,
    this.aiCount = 0,
    this.manualCount = 0,
    this.level,
    this.parentId,
    this.directUsageCount = 0,
    this.treeUsageCount = 0,
    this.directPendingCount = 0,
    this.treePendingCount = 0,
    this.directConfirmedCount = 0,
    this.treeConfirmedCount = 0,
    this.directAiCount = 0,
    this.treeAiCount = 0,
    this.directManualCount = 0,
    this.treeManualCount = 0,
  });

  factory TagStatistics.fromJson(Map<String, dynamic> json) {
    return TagStatistics(
      tagId: json['tag_id'] as int,
      label:
          json['label'] as String? ?? json['preferred_label'] as String? ?? '',
      usageCount: json['usage_count'] as int? ?? 0,
      pendingCount: json['pending_count'] as int? ?? 0,
      confirmedCount: json['confirmed_count'] as int? ?? 0,
      aiCount: json['ai_count'] as int? ?? 0,
      manualCount: json['manual_count'] as int? ?? 0,
      level: json['level'] as String?,
      parentId: json['parent_id'] as int?,
      directUsageCount: json['direct_usage_count'] as int? ?? 0,
      treeUsageCount: json['tree_usage_count'] as int? ?? 0,
      directPendingCount: json['direct_pending_count'] as int? ?? 0,
      treePendingCount: json['tree_pending_count'] as int? ?? 0,
      directConfirmedCount: json['direct_confirmed_count'] as int? ?? 0,
      treeConfirmedCount: json['tree_confirmed_count'] as int? ?? 0,
      directAiCount: json['direct_ai_count'] as int? ?? 0,
      treeAiCount: json['tree_ai_count'] as int? ?? 0,
      directManualCount: json['direct_manual_count'] as int? ?? 0,
      treeManualCount: json['tree_manual_count'] as int? ?? 0,
    );
  }
}
