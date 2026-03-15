/// Represents a tag in the system
class Tag {
  final int id;
  final String label;
  final String? category;
  final String? normalizedLabel;
  final int? usageCount;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Tag({
    required this.id,
    required this.label,
    this.category,
    this.normalizedLabel,
    this.usageCount,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Tag.fromJson(Map<String, dynamic> json) {
    return Tag(
      id: json['id'] as int,
      label: json['label'] as String,
      category: json['category'] as String?,
      normalizedLabel: json['normalized_label'] as String?,
      usageCount: json['usage_count'] as int?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'label': label,
      'category': category,
      'normalized_label': normalizedLabel,
      'usage_count': usageCount,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }
}

/// Represents an image-tag association with review state
class ImageTag {
  final int id;
  final int imageId;
  final int tagId;
  final Tag tag;
  final String reviewState; // 'pending', 'confirmed', 'rejected'
  final double? confidence;
  final int? sourceObservationId;
  final DateTime createdAt;
  final DateTime updatedAt;

  const ImageTag({
    required this.id,
    required this.imageId,
    required this.tagId,
    required this.tag,
    this.reviewState = 'pending',
    this.confidence,
    this.sourceObservationId,
    required this.createdAt,
    required this.updatedAt,
  });

  factory ImageTag.fromJson(Map<String, dynamic> json) {
    return ImageTag(
      id: json['id'] as int,
      imageId: json['image_id'] as int,
      tagId: json['tag_id'] as int,
      tag: Tag.fromJson(json['tag'] as Map<String, dynamic>),
      reviewState: json['review_state'] as String? ?? 'pending',
      confidence: (json['confidence'] as num?)?.toDouble(),
      sourceObservationId: json['source_observation_id'] as int?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  bool get isPending => reviewState == 'pending';
  bool get isConfirmed => reviewState == 'confirmed';
  bool get isRejected => reviewState == 'rejected';
  bool get isAIGenerated => sourceObservationId != null;
}

/// AI tag job status
enum AIJobStatus {
  queued,
  running,
  completed,
  failed,
}

/// AI tag job status response
class AITagStatus {
  final int imageId;
  final AIJobStatus status;
  final int? observationId;
  final String? error;
  final List<Tag>? generatedTags;
  final DateTime? completedAt;

  AITagStatus({
    required this.imageId,
    required this.status,
    this.observationId,
    this.error,
    this.generatedTags,
    this.completedAt,
  });

  factory AITagStatus.fromJson(Map<String, dynamic> json) {
    AIJobStatus status;
    final statusStr = json['status'] as String? ?? 'queued';
    switch (statusStr) {
      case 'running':
        status = AIJobStatus.running;
        break;
      case 'completed':
        status = AIJobStatus.completed;
        break;
      case 'failed':
        status = AIJobStatus.failed;
        break;
      default:
        status = AIJobStatus.queued;
    }

    return AITagStatus(
      imageId: json['image_id'] as int,
      status: status,
      observationId: json['observation_id'] as int?,
      error: json['error'] as String?,
      generatedTags: (json['generated_tags'] as List?)
          ?.map((t) => Tag.fromJson(t as Map<String, dynamic>))
          .toList(),
      completedAt: json['completed_at'] != null
          ? DateTime.parse(json['completed_at'] as String)
          : null,
    );
  }
}

/// Tag governance statistics
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
      label: json['label'] as String,
      usageCount: json['usage_count'] as int? ?? 0,
      pendingCount: json['pending_count'] as int? ?? 0,
      confirmedCount: json['confirmed_count'] as int? ?? 0,
      aiCount: json['ai_count'] as int? ?? 0,
      manualCount: json['manual_count'] as int? ?? 0,
    );
  }
}