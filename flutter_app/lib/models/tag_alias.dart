class TagAlias {
  final int id;
  final int tagId;
  final String label;
  final String normalizedLabel;
  final String? locale;
  final String aliasType;
  final bool isPreferred;

  const TagAlias({
    required this.id,
    required this.tagId,
    required this.label,
    required this.normalizedLabel,
    this.locale,
    required this.aliasType,
    required this.isPreferred,
  });

  factory TagAlias.fromJson(Map<String, dynamic> json) {
    return TagAlias(
      id: json['id'] as int,
      tagId: json['tag_id'] as int,
      label: json['label'] as String,
      normalizedLabel: json['normalized_label'] as String,
      locale: json['locale'] as String?,
      aliasType: json['alias_type'] as String,
      isPreferred: json['is_preferred'] as bool,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'tag_id': tagId,
      'label': label,
      'normalized_label': normalizedLabel,
      'locale': locale,
      'alias_type': aliasType,
      'is_preferred': isPreferred,
    };
  }

  TagAlias copyWith({
    int? id,
    int? tagId,
    String? label,
    String? normalizedLabel,
    String? locale,
    String? aliasType,
    bool? isPreferred,
  }) {
    return TagAlias(
      id: id ?? this.id,
      tagId: tagId ?? this.tagId,
      label: label ?? this.label,
      normalizedLabel: normalizedLabel ?? this.normalizedLabel,
      locale: locale ?? this.locale,
      aliasType: aliasType ?? this.aliasType,
      isPreferred: isPreferred ?? this.isPreferred,
    );
  }

  @override
  String toString() {
    return 'TagAlias(id: $id, tagId: $tagId, label: $label)';
  }
}
