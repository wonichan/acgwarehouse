import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/image.dart';
import '../providers/config_provider.dart';
import '../providers/selection_provider.dart';
import '../providers/image_provider.dart' show ViewMode;
import '../utils/responsive_breakpoint.dart';
import 'image_grid.dart' show ImageTapCallback;

/// Responsive image grid that adapts columns and spacing to screen size.
///
/// Features:
/// - Justified image layout algorithm (Simple ListView version)
/// - Infinite scroll friendly
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

  List<List<ImageModel>> _buildRows(
    List<ImageModel> images,
    double maxWidth,
    double spacing,
  ) {
    final List<List<ImageModel>> rows = [];
    List<ImageModel> currentRow = [];
    double currentWidth = 0.0;
    const double targetHeight = 250.0;

    for (final image in images) {
      double aspect = (image.width > 0 && image.height > 0)
          ? image.width / image.height
          : 1.0;

      currentRow.add(image);
      currentWidth += aspect * targetHeight;

      int spacingCount = currentRow.length > 1 ? currentRow.length - 1 : 0;
      if (currentWidth + (spacingCount * spacing) >= maxWidth) {
        rows.add(List.from(currentRow));
        currentRow.clear();
        currentWidth = 0.0;
      }
    }

    if (currentRow.isNotEmpty) {
      rows.add(currentRow);
    }
    return rows;
  }

  @override
  Widget build(BuildContext context) {
    if (images.isEmpty) {
      return const Center(child: Text('No images to display.'));
    }

    return LayoutBuilder(
      builder: (context, constraints) {
        final breakpoint = ResponsiveBreakpoint.getBreakpoint(
          constraints.maxWidth,
        );
        final spacing = ResponsiveBreakpoint.getGridSpacing(breakpoint);
        final maxWidth = constraints.maxWidth;

        final rows = _buildRows(images, maxWidth, spacing);

        return ListView.builder(
          controller: scrollController,
          itemCount: rows.length,
          itemBuilder: (context, index) {
            final row = rows[index];
            double totalAspect = 0.0;
            for (final img in row) {
              totalAspect += (img.width > 0 && img.height > 0)
                  ? img.width / img.height
                  : 1.0;
            }

            int spacingCount = row.length > 1 ? row.length - 1 : 0;
            double availableWidth = maxWidth - (spacingCount * spacing);

            // If it's the last row, don't stretch it excessively.
            bool isLastRow = index == rows.length - 1;
            double newHeight;
            if (isLastRow && (totalAspect * 250.0 < availableWidth * 0.75)) {
              newHeight = 250.0;
            } else {
              newHeight = availableWidth / totalAspect;
            }

            return Padding(
              padding: EdgeInsets.only(bottom: spacing),
              child: Row(
                children: row.map((image) {
                  double aspect = (image.width > 0 && image.height > 0)
                      ? image.width / image.height
                      : 1.0;
                  double width = aspect * newHeight;

                  final imageIndex = images.indexOf(image);
                  final inSelectionMode =
                      selectionProvider?.isSelectionMode ?? false;
                  final isSelected =
                      selectionProvider?.isSelected(image.id) ?? false;

                  return Padding(
                    padding: EdgeInsets.only(
                      right: image != row.last ? spacing : 0,
                    ),
                    child: SizedBox(
                      width: width,
                      height: newHeight,
                      child: GestureDetector(
                        key: ValueKey('image-${image.id}'),
                        onTap: () {
                          if (selectionProvider != null &&
                              selectionProvider!.handleImageTap(
                                image.id,
                                index: imageIndex,
                              )) {
                            return;
                          }
                          if (onImageTap != null) {
                            onImageTap!(image);
                          }
                        },
                        onLongPress: selectionProvider == null
                            ? null
                            : () {
                                selectionProvider!.handleImageTap(
                                  image.id,
                                  longPress: true,
                                  index: imageIndex,
                                );
                              },
                        child: Stack(
                          fit: StackFit.expand,
                          children: [
                            _buildImageTile(context, image),
                            if (inSelectionMode)
                              Positioned.fill(
                                child: Container(
                                  color: isSelected
                                      ? Colors.black.withOpacity(0.25)
                                      : Colors.transparent,
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
                                    side: const BorderSide(
                                      color: Colors.white,
                                      width: 2,
                                    ),
                                  ),
                                ),
                              ),
                          ],
                        ),
                      ),
                    ),
                  );
                }).toList(),
              ),
            );
          },
        );
      },
    );
  }

  Widget _buildImageTile(BuildContext context, ImageModel image) {
    final thumbnailUrl = context.watch<ConfigProvider?>()?.resolveThumbnailUrl(
      image.thumbnailSmallUrl,
    );
    final colorScheme = Theme.of(context).colorScheme;

    if (thumbnailUrl == null || thumbnailUrl.isEmpty) {
      return Container(
        color: colorScheme.surfaceContainerHighest,
        child: Icon(Icons.image, color: colorScheme.outlineVariant),
      );
    }

    return Image.network(
      thumbnailUrl,
      fit: BoxFit.cover,
      loadingBuilder: (context, child, loadingProgress) {
        if (loadingProgress == null) return child;
        return Container(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          child: Center(
            child: CircularProgressIndicator(
              strokeWidth: 2,
              value: loadingProgress.expectedTotalBytes != null
                  ? loadingProgress.cumulativeBytesLoaded /
                        loadingProgress.expectedTotalBytes!
                  : null,
            ),
          ),
        );
      },
      errorBuilder: (context, error, stackTrace) {
        debugPrint('Image load error: $error, URL: $thumbnailUrl');
        return Container(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          child: Icon(Icons.error, color: Theme.of(context).colorScheme.error),
        );
      },
    );
  }
}
