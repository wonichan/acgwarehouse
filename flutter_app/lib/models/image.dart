class ImageModel {
  final int id;
  final int? collectionId;
  final String path;
  final String filename;
  final String sourceRoot;
  final int fileSize;
  final int width;
  final int height;
  final String format;
  final int phash;
  final String? thumbnailSmallUrl;
  final String? thumbnailLargeUrl;
  final DateTime createdAt;
  final DateTime updatedAt;

  const ImageModel({
    required this.id,
    this.collectionId,
    required this.path,
    required this.filename,
    required this.sourceRoot,
    required this.fileSize,
    required this.width,
    required this.height,
    required this.format,
    required this.phash,
    this.thumbnailSmallUrl,
    this.thumbnailLargeUrl,
    required this.createdAt,
    required this.updatedAt,
  });

  factory ImageModel.fromJson(Map<String, dynamic> json) {
    return ImageModel(
      id: json['id'] as int,
      collectionId: json['collection_id'] as int?,
      path: json['path'] as String,
      filename: json['filename'] as String,
      sourceRoot: json['source_root'] as String,
      fileSize: json['file_size'] as int,
      width: json['width'] as int,
      height: json['height'] as int,
      format: json['format'] as String,
      phash: json['phash'] as int,
      thumbnailSmallUrl: json['thumbnail_small_url'] as String?,
      thumbnailLargeUrl: json['thumbnail_large_url'] as String?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  bool get isFavorited => (collectionId ?? 0) != 0;

  String get displaySize => '$width}x$height';
  String get displayFileSize => '${(fileSize / 1024).toStringAsFixed(1)} KB';
}
