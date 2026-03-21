import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import '../models/image.dart';

/// Fullscreen image viewer with swipe navigation.
///
/// Features:
/// - Swipe left/right to navigate between images
/// - Pinch-to-zoom
/// - Double-tap to zoom
/// - Shows current position ("3 / 25")
///
/// Usage:
/// ```dart
/// Navigator.push(context, MaterialPageRoute(
///   builder: (_) => ImageGalleryViewer(
///     images: imageList,
///     initialIndex: currentIndex,
///   ),
/// ));
/// ```
class ImageGalleryViewer extends StatefulWidget {
  final List<ImageModel> images;
  final int initialIndex;

  const ImageGalleryViewer({
    super.key,
    required this.images,
    required this.initialIndex,
  });

  @override
  State<ImageGalleryViewer> createState() => _ImageGalleryViewerState();
}

class _ImageGalleryViewerState extends State<ImageGalleryViewer> {
  late PageController _pageController;
  late int _currentIndex;

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex;
    _pageController = PageController(initialPage: _currentIndex);
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  void _onPageChanged(int index) {
    setState(() {
      _currentIndex = index;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black.withOpacity(0.5),
        iconTheme: const IconThemeData(color: Colors.white),
        title: Text(
          '${_currentIndex + 1} / ${widget.images.length}',
          style: const TextStyle(color: Colors.white),
        ),
      ),
      body: PageView.builder(
        controller: _pageController,
        itemCount: widget.images.length,
        onPageChanged: _onPageChanged,
        itemBuilder: (context, index) {
          final image = widget.images[index];
          return _buildImagePage(image);
        },
      ),
    );
  }

  Widget _buildImagePage(ImageModel image) {
    final largeUrl = image.thumbnailLargeUrl;

    if (largeUrl == null || largeUrl.isEmpty) {
      return const Center(
        child: Icon(Icons.error, color: Colors.white, size: 48),
      );
    }

    return ExtendedImage.network(
      largeUrl,
      fit: BoxFit.contain,
      mode: ExtendedImageMode.gesture,
      initGestureConfigHandler: (state) {
        return GestureConfig(
          minScale: 1.0,
          maxScale: 3.0,
          animationMaxScale: 3.5,
          initialScale: 1.0,
          inPageView: true, // Important for PageView compatibility
          initialAlignment: InitialAlignment.center,
        );
      },
      onDoubleTap: (state) {
        final begin = state.gestureDetails?.totalScale ?? 1.0;
        final end = begin == 1.0 ? 2.0 : 1.0;
        state.handleDoubleTap(scale: end);
      },
      loadStateChanged: (state) {
        if (state.extendedImageLoadState == LoadState.loading) {
          return const Center(child: CircularProgressIndicator());
        }
        if (state.extendedImageLoadState == LoadState.failed) {
          return const Center(
            child: Icon(Icons.error, color: Colors.white, size: 48),
          );
        }
        return null;
      },
    );
  }
}