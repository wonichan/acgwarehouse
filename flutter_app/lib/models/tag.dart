class Tag {
  final int id;
  final String preferredLabel;
  final String slug;
  final String? primaryCategory;
  final String reviewState;
  final double trustScore;
  final int usageCount;
  final DateTime createdAt;

  const Tag({
    required this.id,
    required this.preferredLabel,
    required this.slug,
    this.primaryCategory,
    required this.reviewState,
    required this.trustScore,
    required this.usageCount,
    required this.createdAt,
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
    );
  }

  @override
  String toString() {
    return 'Tag(id: $id, label: $preferredLabel, state: $reviewState)';
  }

  static String _slugFromLabel(String label) {
    return label.trim().toLowerCase().replaceAll(' ', '-');
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

  TagStatistics({
    required this.tagId,
    required this.label,
    required this.usageCount,
    this.pendingCount = 0,
    this.confirmedCount = 0,
    this.aiCount = 0,
    this.manualCount = 0,
  });

  factory TagStatistics.fromJson(Map<String, dynamic> json) {
    return TagStatistics(
      tagId: json['tag_id'] as int,
      label: json['label'] as String? ?? json['preferred_label'] as String? ?? '',
      usageCount: json['usage_count'] as int? ?? 0,
      pendingCount: json['pending_count'] as int? ?? 0,
      confirmedCount: json['confirmed_count'] as int? ?? 0,
      aiCount: json['ai_count'] as int? ?? 0,
      manualCount: json['manual_count'] as int? ?? 0,
    );
  }
}
