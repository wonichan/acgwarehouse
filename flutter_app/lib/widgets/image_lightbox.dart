import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';

/// A fullscreen image preview/lightbox widget with gesture support.
///
/// Provides Weibo/Bilibili-style fullscreen image viewing with:
/// - Pinch to zoom
/// - Drag to pan
/// - Swipe down to dismiss
/// - Hero animation support
class ImageLightbox {
  /// Shows a fullscreen image preview dialog.
  ///
  /// [context] - The build context to show the dialog in.
  /// [imageUrl] - The URL of the image to display.
  /// [heroTag] - Optional tag for Hero animation transitions.
  static Future<void> show(
    BuildContext context, {
    required String imageUrl,
    String? heroTag,
  }) {
    return showGeneralDialog(
      context: context,
      barrierDismissible: true,
      barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
      barrierColor: Colors.transparent,
      transitionDuration: const Duration(milliseconds: 200),
      pageBuilder: (context, animation, secondaryAnimation) {
        return _ImageLightboxContent(
          imageUrl: imageUrl,
          heroTag: heroTag,
          animation: animation,
        );
      },
    );
  }
}

/// The actual content of the lightbox dialog.
class _ImageLightboxContent extends StatefulWidget {
  final String imageUrl;
  final String? heroTag;
  final Animation<double> animation;

  const _ImageLightboxContent({
    required this.imageUrl,
    this.heroTag,
    required this.animation,
  });

  @override
  State<_ImageLightboxContent> createState() => _ImageLightboxContentState();
}

class _ImageLightboxContentState extends State<_ImageLightboxContent> {
  double _dragOffset = 0;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onVerticalDragStart: (_) {},
      onVerticalDragUpdate: (details) {
        setState(() {
          _dragOffset += details.delta.dy;
        });
      },
      onVerticalDragEnd: (details) {
        // Dismiss if dragged down enough or with enough velocity
        if (_dragOffset > 100 || (details.primaryVelocity ?? 0) > 500) {
          Navigator.of(context).pop();
        } else {
          setState(() {
            _dragOffset = 0;
          });
        }
      },
      onTap: () {
        // Allow tap on background to dismiss
      },
      child: AnimatedBuilder(
        animation: widget.animation,
        builder: (context, child) {
          return Container(
            color: Colors.black.withOpacity(0.9 * widget.animation.value),
            child: Transform.translate(
              offset: Offset(0, _dragOffset),
              child: child,
            ),
          );
        },
        child: Stack(
          children: [
            // Centered image with gesture support
            Center(
              child: Hero(
                tag: widget.heroTag ?? 'lightbox-${widget.imageUrl}',
                child: ExtendedImage.network(
                  widget.imageUrl,
                  fit: BoxFit.contain,
                  mode: ExtendedImageMode.gesture,
                  enableLoadState: true,
                  initGestureConfigHandler: (state) {
                    return GestureConfig(
                      minScale: 0.5,
                      animationMinScale: 0.3,
                      maxScale: 3.0,
                      animationMaxScale: 3.5,
                      speed: 1.0,
                      inertialSpeed: 100.0,
                      initialScale: 1.0,
                      inPageView: false,
                    );
                  },
                  loadStateChanged: (state) {
                    if (state.extendedImageLoadState == LoadState.loading) {
                      return const Center(
                        child: CircularProgressIndicator(color: Colors.white),
                      );
                    }
                    if (state.extendedImageLoadState == LoadState.failed) {
                      return const Center(
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(Icons.error, color: Colors.red, size: 48),
                            SizedBox(height: 8),
                            Text(
                              '加载失败',
                              style: TextStyle(color: Colors.white70),
                            ),
                          ],
                        ),
                      );
                    }
                    return null;
                  },
                ),
              ),
            ),

            // Close button
            Positioned(
              top: 0,
              left: 0,
              right: 0,
              child: AnimatedBuilder(
                animation: widget.animation,
                builder: (context, child) {
                  return SafeArea(
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      height: 56,
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topCenter,
                          end: Alignment.bottomCenter,
                          colors: [
                            Colors.black.withOpacity(
                              0.5 * widget.animation.value,
                            ),
                            Colors.transparent,
                          ],
                        ),
                      ),
                      child: Row(
                        children: [
                          const Spacer(),
                          IconButton(
                            icon: const Icon(Icons.close, color: Colors.white),
                            onPressed: () => Navigator.of(context).pop(),
                            tooltip: '关闭',
                          ),
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),

            // Swipe hint at bottom
            Positioned(
              bottom: 0,
              left: 0,
              right: 0,
              child: AnimatedBuilder(
                animation: widget.animation,
                builder: (context, child) {
                  return SafeArea(
                    child: Container(
                      padding: const EdgeInsets.only(bottom: 16),
                      child: Center(
                        child: Text(
                          '向下滑动关闭',
                          style: TextStyle(
                            color: Colors.white.withOpacity(
                              0.5 * widget.animation.value,
                            ),
                            fontSize: 12,
                          ),
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}
