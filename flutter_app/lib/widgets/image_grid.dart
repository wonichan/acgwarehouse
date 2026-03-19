import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';
import '../models/image.dart';
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
