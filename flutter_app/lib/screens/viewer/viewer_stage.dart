import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import 'package:gallery/models/viewer_session.dart';

class ViewerStage extends StatelessWidget {
  final ViewerSessionItem item;

  const ViewerStage({super.key, required this.item});

  @override
  Widget build(BuildContext context) {
    final largeUrl = item.thumbnailLargeUrl;
    if (largeUrl == null || largeUrl.isEmpty) {
      return const Center(
        child: Icon(Icons.image, size: 64, color: Colors.grey),
      );
    }

    return ExtendedImage.network(
      largeUrl,
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
