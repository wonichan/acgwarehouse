class TagGovernanceRow {
  final int tagId;
  final String preferredLabel;
  final String? primaryCategory;
  final List<String> aliases;
  final int usageCount;
  final int pendingCount;
  final int confirmedCount;
  final int rejectedCount;
  final int aiCount;
  final int manualCount;
  final int affectedImageCount;
  final bool canDelete;
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

  const TagGovernanceRow({
    required this.tagId,
    required this.preferredLabel,
    required this.primaryCategory,
    required this.aliases,
    required this.usageCount,
    required this.pendingCount,
    required this.confirmedCount,
    required this.rejectedCount,
    required this.aiCount,
    required this.manualCount,
    required this.affectedImageCount,
    required this.canDelete,
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

  factory TagGovernanceRow.fromJson(Map<String, dynamic> json) {
    return TagGovernanceRow(
      tagId: json['tag_id'] as int,
      preferredLabel: json['preferred_label'] as String,
      primaryCategory: json['primary_category'] as String?,
      aliases: (json['aliases'] as List? ?? [])
          .map((e) => e.toString())
          .toList(),
      usageCount: json['usage_count'] as int? ?? 0,
      pendingCount: json['pending_count'] as int? ?? 0,
      confirmedCount: json['confirmed_count'] as int? ?? 0,
      rejectedCount: json['rejected_count'] as int? ?? 0,
      aiCount: json['ai_count'] as int? ?? 0,
      manualCount: json['manual_count'] as int? ?? 0,
      affectedImageCount: json['affected_image_count'] as int? ?? 0,
      canDelete: json['can_delete'] as bool? ?? false,
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

class TagDeletePreview {
  final int tagId;
  final String preferredLabel;
  final int affectedImageCount;
  final bool canDelete;
  final String? blockingReason;

  const TagDeletePreview({
    required this.tagId,
    required this.preferredLabel,
    required this.affectedImageCount,
    required this.canDelete,
    required this.blockingReason,
  });

  factory TagDeletePreview.fromJson(Map<String, dynamic> json) {
    return TagDeletePreview(
      tagId: json['tag_id'] as int,
      preferredLabel: json['preferred_label'] as String? ?? '',
      affectedImageCount: json['affected_image_count'] as int? ?? 0,
      canDelete: json['can_delete'] as bool? ?? false,
      blockingReason: json['blocking_reason'] as String?,
    );
  }
}

class TagGovernanceFailure {
  final int tagId;
  final String preferredLabel;
  final String message;

  const TagGovernanceFailure({
    required this.tagId,
    required this.preferredLabel,
    required this.message,
  });

  factory TagGovernanceFailure.fromJson(Map<String, dynamic> json) {
    return TagGovernanceFailure(
      tagId: json['tag_id'] as int,
      preferredLabel: json['preferred_label'] as String? ?? '',
      message:
          json['message'] as String? ??
          json['blocking_reason'] as String? ??
          json['error'] as String? ??
          '',
    );
  }
}

class TagGovernanceBatchResult {
  final List<int> deletedTagIds;
  final List<TagGovernanceFailure> failures;

  const TagGovernanceBatchResult({
    required this.deletedTagIds,
    required this.failures,
  });

  factory TagGovernanceBatchResult.fromJson(Map<String, dynamic> json) {
    final deleted = (json['deleted'] as List? ?? [])
        .map((entry) => (entry as Map<String, dynamic>)['tag_id'] as int)
        .toList();

    final blockedFailures = (json['blocked'] as List? ?? []).map(
      (entry) => TagGovernanceFailure.fromJson(entry as Map<String, dynamic>),
    );
    final failedFailures = (json['failed'] as List? ?? []).map(
      (entry) => TagGovernanceFailure.fromJson(entry as Map<String, dynamic>),
    );

    return TagGovernanceBatchResult(
      deletedTagIds: deleted,
      failures: [...blockedFailures, ...failedFailures],
    );
  }
}

class TagMergeRequest {
  final int targetTagId;

  const TagMergeRequest({required this.targetTagId});

  Map<String, dynamic> toJson() {
    return {'target_tag_id': targetTagId};
  }
}
