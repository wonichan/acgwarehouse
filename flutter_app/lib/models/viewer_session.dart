import 'image.dart';

enum ViewerSessionSource { gallery, search }

ViewerSessionSource _viewerSessionSourceFromJson(String value) {
  return ViewerSessionSource.values.firstWhere(
    (source) => source.name == value,
    orElse: () => ViewerSessionSource.gallery,
  );
}

class ViewerSessionItem {
  final int imageId;
  final String path;
  final String filename;
  final String sourceRoot;
  final int fileSize;
  final int width;
  final int height;
  final String format;
  final String? thumbnailSmallUrl;
  final String? thumbnailLargeUrl;
  final String createdAtIso8601;
  final String updatedAtIso8601;

  const ViewerSessionItem({
    required this.imageId,
    required this.path,
    required this.filename,
    required this.sourceRoot,
    required this.fileSize,
    required this.width,
    required this.height,
    required this.format,
    required this.thumbnailSmallUrl,
    required this.thumbnailLargeUrl,
    required this.createdAtIso8601,
    required this.updatedAtIso8601,
  });

  factory ViewerSessionItem.fromImage(ImageModel image) {
    return ViewerSessionItem(
      imageId: image.id,
      path: image.path,
      filename: image.filename,
      sourceRoot: image.sourceRoot,
      fileSize: image.fileSize,
      width: image.width,
      height: image.height,
      format: image.format,
      thumbnailSmallUrl: image.thumbnailSmallUrl,
      thumbnailLargeUrl: image.thumbnailLargeUrl,
      createdAtIso8601: image.createdAt.toUtc().toIso8601String(),
      updatedAtIso8601: image.updatedAt.toUtc().toIso8601String(),
    );
  }

  factory ViewerSessionItem.fromJson(Map<String, dynamic> json) {
    return ViewerSessionItem(
      imageId: json['image_id'] as int,
      path: json['path'] as String,
      filename: json['filename'] as String,
      sourceRoot: json['source_root'] as String,
      fileSize: json['file_size'] as int,
      width: json['width'] as int,
      height: json['height'] as int,
      format: json['format'] as String,
      thumbnailSmallUrl: json['thumbnail_small_url'] as String?,
      thumbnailLargeUrl: json['thumbnail_large_url'] as String?,
      createdAtIso8601: json['created_at'] as String,
      updatedAtIso8601: json['updated_at'] as String,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'image_id': imageId,
      'path': path,
      'filename': filename,
      'source_root': sourceRoot,
      'file_size': fileSize,
      'width': width,
      'height': height,
      'format': format,
      'thumbnail_small_url': thumbnailSmallUrl,
      'thumbnail_large_url': thumbnailLargeUrl,
      'created_at': createdAtIso8601,
      'updated_at': updatedAtIso8601,
    };
  }
}

class ViewerSession {
  final ViewerSessionSource source;
  final List<ViewerSessionItem> items;
  final int initialSelectedIndex;

  const ViewerSession({
    required this.source,
    required this.items,
    required this.initialSelectedIndex,
  });

  factory ViewerSession.fromResultSet({
    required ViewerSessionSource source,
    required List<ImageModel> images,
    required int selectedImageId,
  }) {
    final index = images.indexWhere((image) => image.id == selectedImageId);
    if (index < 0) {
      throw ArgumentError.value(
        selectedImageId,
        'selectedImageId',
        'must exist in the result-set snapshot',
      );
    }

    return ViewerSession(
      source: source,
      items: images.map(ViewerSessionItem.fromImage).toList(growable: false),
      initialSelectedIndex: index,
    );
  }

  factory ViewerSession.fromJson(Map<String, dynamic> json) {
    final itemsJson = json['items'] as List<dynamic>;
    return ViewerSession(
      source: _viewerSessionSourceFromJson(json['source'] as String),
      items: itemsJson
          .map(
            (item) => ViewerSessionItem.fromJson(item as Map<String, dynamic>),
          )
          .toList(growable: false),
      initialSelectedIndex: json['initial_selected_index'] as int,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'source': source.name,
      'items': items.map((item) => item.toJson()).toList(growable: false),
      'initial_selected_index': initialSelectedIndex,
    };
  }

  ViewerSessionItem get selectedItem => items[initialSelectedIndex];
}
