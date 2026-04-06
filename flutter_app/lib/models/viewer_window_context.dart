import 'viewer_session.dart';

enum ViewerWindowSource { gallery, search }

ViewerWindowSource _viewerWindowSourceFromJson(String value) {
  return ViewerWindowSource.values.firstWhere(
    (source) => source.name == value,
    orElse: () => ViewerWindowSource.gallery,
  );
}

ViewerWindowSource? _tryViewerWindowSourceFromJson(Object? value) {
  if (value is! String) {
    return null;
  }

  for (final source in ViewerWindowSource.values) {
    if (source.name == value) {
      return source;
    }
  }

  return null;
}

abstract class ViewerWindowSnapshot {
  const ViewerWindowSnapshot();

  Map<String, dynamic> toJson();
}

class ViewerWindowGallerySnapshot extends ViewerWindowSnapshot {
  final String sortBy;
  final String sortDir;
  final List<int> tagIds;
  final bool? hasTags;

  const ViewerWindowGallerySnapshot({
    required this.sortBy,
    required this.sortDir,
    required this.tagIds,
    required this.hasTags,
  });

  factory ViewerWindowGallerySnapshot.fromJson(Map<String, dynamic> json) {
    return ViewerWindowGallerySnapshot(
      sortBy: json['sort_by'] as String? ?? 'created_at',
      sortDir: json['sort_dir'] as String? ?? 'desc',
      tagIds: (json['tag_ids'] as List<dynamic>? ?? const [])
          .map((item) => item as int)
          .toList(growable: false),
      hasTags: json['has_tags'] as bool?,
    );
  }

  @override
  Map<String, dynamic> toJson() {
    return {
      'sort_by': sortBy,
      'sort_dir': sortDir,
      'tag_ids': tagIds,
      'has_tags': hasTags,
    };
  }
}

class ViewerWindowSearchSnapshot extends ViewerWindowSnapshot {
  final String query;
  final List<int> tagIds;
  final String sortBy;
  final String sortOrder;

  const ViewerWindowSearchSnapshot({
    required this.query,
    required this.tagIds,
    required this.sortBy,
    required this.sortOrder,
  });

  factory ViewerWindowSearchSnapshot.fromJson(Map<String, dynamic> json) {
    return ViewerWindowSearchSnapshot(
      query: json['q'] as String? ?? '',
      tagIds: (json['tag_ids'] as List<dynamic>? ?? const [])
          .map((item) => item as int)
          .toList(growable: false),
      sortBy: json['sort_by'] as String? ?? 'relevance',
      sortOrder: json['sort_order'] as String? ?? 'desc',
    );
  }

  @override
  Map<String, dynamic> toJson() {
    return {
      'q': query,
      'tag_ids': tagIds,
      'sort_by': sortBy,
      'sort_order': sortOrder,
    };
  }
}

class ViewerWindowContext {
  final ViewerWindowSource source;
  final int selectedIndex;
  final int selectedImageId;
  final ViewerWindowSnapshot snapshot;

  const ViewerWindowContext._({
    required this.source,
    required this.selectedIndex,
    required this.selectedImageId,
    required this.snapshot,
  });

  const ViewerWindowContext.gallery({
    required int selectedIndex,
    required int selectedImageId,
    required ViewerWindowGallerySnapshot snapshot,
  }) : this._(
         source: ViewerWindowSource.gallery,
         selectedIndex: selectedIndex,
         selectedImageId: selectedImageId,
         snapshot: snapshot,
       );

  const ViewerWindowContext.search({
    required int selectedIndex,
    required int selectedImageId,
    required ViewerWindowSearchSnapshot snapshot,
  }) : this._(
         source: ViewerWindowSource.search,
         selectedIndex: selectedIndex,
         selectedImageId: selectedImageId,
         snapshot: snapshot,
       );

  factory ViewerWindowContext.fromJson(Map<String, dynamic> json) {
    final source = _viewerWindowSourceFromJson(
      json['source'] as String? ?? 'gallery',
    );
    final snapshotJson = json['snapshot'] as Map<String, dynamic>? ?? const {};
    return ViewerWindowContext._(
      source: source,
      selectedIndex: json['selected_index'] as int? ?? 0,
      selectedImageId: json['selected_image_id'] as int? ?? 0,
      snapshot: source == ViewerWindowSource.search
          ? ViewerWindowSearchSnapshot.fromJson(snapshotJson)
          : ViewerWindowGallerySnapshot.fromJson(snapshotJson),
    );
  }

  static ViewerWindowContext? tryFromJson(Map<String, dynamic> json) {
    final source = _tryViewerWindowSourceFromJson(json['source']);
    final selectedIndex = json['selected_index'];
    final selectedImageId = json['selected_image_id'];
    final snapshotJson = json['snapshot'];

    if (source == null ||
        selectedIndex is! int ||
        selectedImageId is! int ||
        snapshotJson is! Map<String, dynamic>) {
      return null;
    }

    return ViewerWindowContext._(
      source: source,
      selectedIndex: selectedIndex,
      selectedImageId: selectedImageId,
      snapshot: source == ViewerWindowSource.search
          ? ViewerWindowSearchSnapshot.fromJson(snapshotJson)
          : ViewerWindowGallerySnapshot.fromJson(snapshotJson),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'source': source.name,
      'selected_index': selectedIndex,
      'selected_image_id': selectedImageId,
      'snapshot': snapshot.toJson(),
    };
  }
}

ViewerSession buildLegacyViewerSession({
  required ViewerWindowContext context,
  required String title,
}) {
  final filename = title.startsWith('ACGWarehouse Viewer — ')
      ? title.substring('ACGWarehouse Viewer — '.length)
      : title;
  return ViewerSession(
    source: context.source == ViewerWindowSource.search
        ? ViewerSessionSource.search
        : ViewerSessionSource.gallery,
    items: [
      ViewerSessionItem(
        imageId: context.selectedImageId,
        path: '',
        filename: filename.isEmpty ? 'loading...' : filename,
        sourceRoot: '',
        fileSize: 0,
        width: 0,
        height: 0,
        format: '',
        thumbnailSmallUrl: null,
        thumbnailLargeUrl: null,
        createdAtIso8601: '',
        updatedAtIso8601: '',
      ),
    ],
    initialSelectedIndex: 0,
  );
}
