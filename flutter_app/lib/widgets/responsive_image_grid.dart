// lib/widgets/responsive_image_grid.dart
import 'package:flutter/material.dart';
import '../models/image.dart';
import '../providers/selection_provider.dart';
import '../providers/image_provider.dart' show ViewMode;
import '../utils/responsive_breakpoint.dart';
import 'image_grid.dart';
import 'image_masonry.dart' hide ImageTapCallback;
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';

/// Responsive image grid that adapts columns and spacing to screen size.
///
/// Breakpoint behavior:
/// - Compact (<= 600px): 2 columns, 4px spacing (phones)
/// - Medium (600-840px): 3 columns, 8px spacing (tablets)
/// - Expanded (> 840px): 4 columns, 12px spacing (large tablets)
///
/// Features:
/// - AnimatedSwitcher for smooth transitions when column count changes
/// - 200ms fade animation with easeInOut curve
///
/// Usage:
/// ```dart
/// ResponsiveImageGrid(
///   images: imageList,
///   viewMode: ViewMode.grid,
///   onImageTap: (image) => navigateToDetail(image),
///   selectionProvider: selectionProvider,
/// )
/// ```
class ResponsiveImageGrid extends StatelessWidget {
  final List<ImageModel> images;
  final ViewMode viewMode;
  final ImageTapCallback? onImageTap;
  final SelectionProvider? selectionProvider;
  final ScrollController? scrollController;

  const ResponsiveImageGrid({
    super.key,
    required this.images,
    this.viewMode = ViewMode.grid,
    this.onImageTap,
    this.selectionProvider,
    this.scrollController,
  });

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final breakpoint = ResponsiveBreakpoint.getBreakpoint(constraints.maxWidth);
        final crossAxisCount = ResponsiveBreakpoint.getGridColumns(breakpoint);
        final spacing = ResponsiveBreakpoint.getGridSpacing(breakpoint);

        return AnimatedSwitcher(
          duration: const Duration(milliseconds: 200),
          switchInCurve: Curves.easeInOut,
          switchOutCurve: Curves.easeInOut,
          transitionBuilder: (child, animation) {
            return FadeTransition(
              opacity: animation,
              child: child,
            );
          },
          child: KeyedSubtree(
            key: ValueKey('$viewMode-$crossAxisCount'),
            child: _buildGrid(crossAxisCount, spacing),
          ),
        );
      },
    );
  }

  Widget _buildGrid(int crossAxisCount, double spacing) {
    if (viewMode == ViewMode.masonry) {
      return MasonryGridView.count(
        controller: scrollController,
        crossAxisCount: crossAxisCount,
        mainAxisSpacing: spacing,
        crossAxisSpacing: spacing,
        itemCount: images.length,
        itemBuilder: (context, index) {
          final image = images[index];
          final inSelectionMode = selectionProvider?.isSelectionMode ?? false;
          final isSelected = selectionProvider?.isSelected(image.id) ?? false;

          return GestureDetector(
            key: ValueKey('image-${image.id}'),
            onTap: () {
              if (selectionProvider != null &&
                  selectionProvider!.handleImageTap(image.id, index: index)) {
                return;
              }
              if (onImageTap != null) {
                onImageTap!(image);
              }
            },
            onLongPress: selectionProvider == null
                ? null
                : () {
                    selectionProvider!.handleImageTap(image.id, longPress: true, index: index);
                  },
            child: Stack(
              children: [
                _buildImageTile(image),
                if (inSelectionMode)
                  Positioned.fill(
                    child: Container(
                      color: isSelected ? Colors.black.withOpacity(0.25) : Colors.transparent,
                    ),
                  ),
                if (inSelectionMode)
                  Positioned(
                    top: 8,
                    right: 8,
                    child: IgnorePointer(
                      child: Checkbox(
                        value: isSelected,
                        onChanged: (_) {},
                        shape: const CircleBorder(),
                        side: const BorderSide(color: Colors.white, width: 2),
                      ),
                    ),
                  ),
              ],
            ),
          );
        },
      );
    }

    return GridView.builder(
      controller: scrollController,
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: crossAxisCount,
        mainAxisSpacing: spacing,
        crossAxisSpacing: spacing,
      ),
      itemCount: images.length,
      itemBuilder: (context, index) {
        final image = images[index];
        final inSelectionMode = selectionProvider?.isSelectionMode ?? false;
        final isSelected = selectionProvider?.isSelected(image.id) ?? false;

        return GestureDetector(
          key: ValueKey('image-${image.id}'),
          onTap: () {
            if (selectionProvider != null &&
                selectionProvider!.handleImageTap(image.id, index: index)) {
              return;
            }
            if (onImageTap != null) {
              onImageTap!(image);
            }
          },
          onLongPress: selectionProvider == null
              ? null
              : () {
                  selectionProvider!.handleImageTap(image.id, longPress: true, index: index);
                },
          child: Stack(
            fit: StackFit.expand,
            children: [
              _buildImageTile(image),
              if (inSelectionMode)
                Container(
                  color: isSelected ? Colors.black.withOpacity(0.25) : Colors.transparent,
                ),
              if (inSelectionMode)
                Positioned(
                  top: 8,
                  right: 8,
                  child: IgnorePointer(
                    child: Checkbox(
                      value: isSelected,
                      onChanged: (_) {},
                      shape: const CircleBorder(),
                      side: const BorderSide(color: Colors.white, width: 2),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildImageTile(ImageModel image) {
    final thumbnailUrl = image.thumbnailSmallUrl;

    if (thumbnailUrl == null || thumbnailUrl.isEmpty) {
      return Container(
        color: Colors.grey[200],
        child: const Icon(Icons.image, color: Colors.grey),
      );
    }

    return Image.network(
      thumbnailUrl,
      fit: BoxFit.cover,
      loadingBuilder: (context, child, loadingProgress) {
        if (loadingProgress == null) return child;
        return Container(
          color: Colors.grey[200],
          child: Center(
            child: CircularProgressIndicator(
              strokeWidth: 2,
              value: loadingProgress.expectedTotalBytes != null
                  ? loadingProgress.cumulativeBytesLoaded / loadingProgress.expectedTotalBytes!
                  : null,
            ),
          ),
        );
      },
      errorBuilder: (context, error, stackTrace) {
        debugPrint('Image load error: $error, URL: $thumbnailUrl');
        return Container(
          color: Colors.grey[200],
          child: const Icon(Icons.error, color: Colors.red),
        );
      },
    );
  }
}