import 'image.dart';
import 'viewer_window_context.dart';

class ViewerWindowRequest {
  final ViewerWindowContext context;
  final int limit;

  const ViewerWindowRequest({required this.context, this.limit = 10});

  Map<String, dynamic> toJson() {
    return {
      'source': context.source.name,
      'selected_index': context.selectedIndex,
      'selected_image_id': context.selectedImageId,
      'limit': limit,
      'snapshot': context.snapshot.toJson(),
    };
  }
}

class ViewerWindowResult {
  final List<ImageModel> items;
  final int windowStartIndex;
  final int selectedIndex;
  final int selectedIndexInWindow;
  final int total;
  final bool hasPrevious;
  final bool hasNext;

  const ViewerWindowResult({
    required this.items,
    required this.windowStartIndex,
    required this.selectedIndex,
    required this.selectedIndexInWindow,
    required this.total,
    required this.hasPrevious,
    required this.hasNext,
  });

  factory ViewerWindowResult.fromJson(Map<String, dynamic> json) {
    return ViewerWindowResult(
      items: (json['items'] as List<dynamic>? ?? const [])
          .map((item) => ImageModel.fromJson(item as Map<String, dynamic>))
          .toList(growable: false),
      windowStartIndex: json['window_start_index'] as int? ?? 0,
      selectedIndex: json['selected_index'] as int? ?? 0,
      selectedIndexInWindow: json['selected_index_in_window'] as int? ?? 0,
      total: json['total'] as int? ?? 0,
      hasPrevious: json['has_previous'] as bool? ?? false,
      hasNext: json['has_next'] as bool? ?? false,
    );
  }
}
