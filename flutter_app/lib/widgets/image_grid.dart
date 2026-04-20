import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/image.dart';
import '../providers/config_provider.dart';
import '../providers/selection_provider.dart';

typedef ImageTapCallback = void Function(ImageModel image);

class ImageGrid extends StatelessWidget {
  final List<ImageModel> images;
  final ImageTapCallback? onImageTap;
  final SelectionProvider? selectionProvider;
  final int crossAxisCount;
  final ScrollController? scrollController;

  const ImageGrid({
    super.key,
    required this.images,
    this.onImageTap,
    this.selectionProvider,
    this.crossAxisCount = 3,
    this.scrollController,
  });

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      controller: scrollController,
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: crossAxisCount,
        mainAxisSpacing: 4,
        crossAxisSpacing: 4,
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
                  selectionProvider!.handleImageTap(
                    image.id,
                    longPress: true,
                    index: index,
                  );
                },
          child: Stack(
            fit: StackFit.expand,
            children: [
              _buildImageTile(context, image),
              if (inSelectionMode)
                Container(
                  color: isSelected
                      ? Colors.black.withOpacity(0.25)
                      : Colors.transparent,
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
