import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import 'package:gallery/models/viewer_session.dart';

class ViewerStage extends StatelessWidget {
  final ViewerSessionItem item;

  const ViewerStage({super.key, required this.item});

  @override
  Widget build(BuildContext context) {
    // 优先使用缩略图，如果缩略图为空则回退到原始图片路径
    final largeUrl = item.thumbnailLargeUrl;
    final originalPath = item.path;
    final displayUrl = (largeUrl != null && largeUrl.isNotEmpty)
        ? largeUrl
        : originalPath;

    if (displayUrl.isEmpty) {
      return const Center(
        child: Icon(Icons.image, size: 64, color: Colors.grey),
      );
    }

    return ExtendedImage.network(
      displayUrl,
      key: ValueKey(item.imageId),
      fit: BoxFit.contain,
      mode: ExtendedImageMode.gesture,
      initGestureConfigHandler: (state) {
        return GestureConfig(
          minScale: 1.0,
          maxScale: 3.0,
          animationMaxScale: 3.5,
          initialScale: 1.0,
          inPageView: false,
          initialAlignment: InitialAlignment.center,
          cacheGesture: false,
        );
      },
      onDoubleTap: (state) {
        final pointerDownPosition = state.pointerDownPosition;
        final begin = state.gestureDetails?.totalScale ?? 1.0;
        final end = begin == 1.0 ? 2.0 : 1.0;

        state.handleDoubleTap(
          scale: end,
          doubleTapPosition: pointerDownPosition,
        );
      },
      loadStateChanged: (state) {
        if (state.extendedImageLoadState == LoadState.loading) {
          return const Center(child: CircularProgressIndicator());
        }
        if (state.extendedImageLoadState == LoadState.failed) {
          return const Center(child: Icon(Icons.error, color: Colors.red));
        }
        return null;
      },
    );
  }
}
