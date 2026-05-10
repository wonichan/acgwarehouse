class ImageMoveRequest {
  final List<String> sourceDirs;
  final int tagId;
  final String targetDir;
  final String conflict;
  final int limit;

  const ImageMoveRequest({
    required this.sourceDirs,
    required this.tagId,
    required this.targetDir,
    this.conflict = 'skip',
    this.limit = 1000,
  });

  Map<String, dynamic> toJson() {
    return {
      'source_dirs': sourceDirs,
      'tag_id': tagId,
      'target_dir': targetDir,
      'conflict': conflict,
      'limit': limit,
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

  const ImageMoveItem({
    required this.imageId,
    required this.filename,
    required this.sourcePath,
    required this.targetPath,
    required this.status,
    this.reason,
  });

  factory ImageMoveItem.fromJson(Map<String, dynamic> json) {
    return ImageMoveItem(
      imageId: json['image_id'] as int,
      filename: json['filename'] as String? ?? '',
      sourcePath: json['source_path'] as String? ?? '',
      targetPath: json['target_path'] as String? ?? '',
      status: json['status'] as String? ?? '',
      reason: json['reason'] as String?,
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
