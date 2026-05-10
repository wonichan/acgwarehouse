class ImageMoveRequest {
  final List<String> sourceDirs;
  final int tagId;
  final String targetDir;
  final String conflict;
  final int limit;
  final bool allowTargetInsideSource;

  const ImageMoveRequest({
    required this.sourceDirs,
    required this.tagId,
    required this.targetDir,
    this.conflict = 'skip',
    this.limit = 1000,
    this.allowTargetInsideSource = false,
  });

  Map<String, dynamic> toJson() {
    return {
      'source_dirs': sourceDirs,
      'tag_id': tagId,
      'target_dir': targetDir,
      'conflict': conflict,
      'limit': limit,
      'allow_target_inside_source': allowTargetInsideSource,
    };
  }
}

class ImageMoveItem {
  final int imageId;
  final String filename;
  final String sourcePath;
  final String targetPath;
  final String status;
  final String? reason;
  final bool retryable;
  final bool overwritten;

  const ImageMoveItem({
    required this.imageId,
    required this.filename,
    required this.sourcePath,
    required this.targetPath,
    required this.status,
    this.reason,
    this.retryable = false,
    this.overwritten = false,
  });

  factory ImageMoveItem.fromJson(Map<String, dynamic> json) {
    return ImageMoveItem(
      imageId: json['image_id'] as int,
      filename: json['filename'] as String? ?? '',
      sourcePath: json['source_path'] as String? ?? '',
      targetPath: json['target_path'] as String? ?? '',
      status: json['status'] as String? ?? '',
      reason: json['reason'] as String?,
      retryable: json['retryable'] as bool? ?? false,
      overwritten: json['overwritten'] as bool? ?? false,
    );
  }
}

class ImageMovePreview {
  final int totalMatched;
  final int movable;
  final int skipped;
  final List<ImageMoveItem> items;

  const ImageMovePreview({
    required this.totalMatched,
    required this.movable,
    required this.skipped,
    required this.items,
  });

  factory ImageMovePreview.fromJson(Map<String, dynamic> json) {
    return ImageMovePreview(
      totalMatched: json['total_matched'] as int? ?? 0,
      movable: json['movable'] as int? ?? 0,
      skipped: json['skipped'] as int? ?? 0,
      items: (json['items'] as List? ?? const [])
          .map((item) => ImageMoveItem.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }
}

class ImageMoveResult {
  final int totalMatched;
  final int moved;
  final int skipped;
  final int failed;
  final List<ImageMoveItem> items;

  const ImageMoveResult({
    required this.totalMatched,
    required this.moved,
    required this.skipped,
    required this.failed,
    required this.items,
  });

  factory ImageMoveResult.fromJson(Map<String, dynamic> json) {
    return ImageMoveResult(
      totalMatched: json['total_matched'] as int? ?? 0,
      moved: json['moved'] as int? ?? 0,
      skipped: json['skipped'] as int? ?? 0,
      failed: json['failed'] as int? ?? 0,
      items: (json['items'] as List? ?? const [])
          .map((item) => ImageMoveItem.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }
}

class ImageMoveProgress {
  final int total;
  final int processed;
  final int moved;
  final int skipped;
  final int failed;
  final String? currentPath;

  const ImageMoveProgress({
    required this.total,
    required this.processed,
    required this.moved,
    required this.skipped,
    required this.failed,
    this.currentPath,
  });

  factory ImageMoveProgress.fromJson(Map<String, dynamic>? json) {
    return ImageMoveProgress(
      total: json?['total'] as int? ?? 0,
      processed: json?['processed'] as int? ?? 0,
      moved: json?['moved'] as int? ?? 0,
      skipped: json?['skipped'] as int? ?? 0,
      failed: json?['failed'] as int? ?? 0,
      currentPath: json?['current_path'] as String?,
    );
  }
}

class ImageMoveBatch {
  final int id;
  final int tagId;
  final List<String> sourceDirs;
  final String targetDir;
  final String conflictStrategy;
  final int totalMatched;
  final int moved;
  final int skipped;
  final int failed;
  final String status;
  final String createdAt;
  final String? finishedAt;
  final List<ImageMoveItem> items;
  final ImageMoveProgress progress;

  const ImageMoveBatch({
    required this.id,
    required this.tagId,
    required this.sourceDirs,
    required this.targetDir,
    required this.conflictStrategy,
    required this.totalMatched,
    required this.moved,
    required this.skipped,
    required this.failed,
    required this.status,
    required this.createdAt,
    this.finishedAt,
    required this.items,
    required this.progress,
  });

  factory ImageMoveBatch.fromJson(Map<String, dynamic> json) {
    return ImageMoveBatch(
      id: json['id'] as int? ?? 0,
      tagId: json['tag_id'] as int? ?? 0,
      sourceDirs: (json['source_dirs'] as List? ?? const [])
          .map((item) => item.toString())
          .toList(),
      targetDir: json['target_dir'] as String? ?? '',
      conflictStrategy: json['conflict_strategy'] as String? ?? 'skip',
      totalMatched: json['total_matched'] as int? ?? 0,
      moved: json['moved'] as int? ?? 0,
      skipped: json['skipped'] as int? ?? 0,
      failed: json['failed'] as int? ?? 0,
      status: json['status'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
      finishedAt: json['finished_at'] as String?,
      items: (json['items'] as List? ?? const [])
          .map((item) => ImageMoveItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      progress: ImageMoveProgress.fromJson(
        json['progress'] as Map<String, dynamic>?,
      ),
    );
  }
}
