import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';
import '../models/image.dart';
import '../providers/selection_provider.dart';

typedef ImageTapCallback = void Function(ImageModel image);

class ImageMasonry extends StatelessWidget {
  final List<ImageModel> images;
  final ImageTapCallback? onImageTap;
  final SelectionProvider? selectionProvider;
  final int crossAxisCount;
  final ScrollController? scrollController;
  
  const ImageMasonry({
    super.key,
    required this.images,
    this.onImageTap,
    this.selectionProvider,
    this.crossAxisCount = 2,
    this.scrollController,
  });
  
  @override
  Widget build(BuildContext context) {
    return MasonryGridView.count(
      controller: scrollController,
      crossAxisCount: crossAxisCount,
      mainAxisSpacing: 4,
      crossAxisSpacing: 4,
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
  
  Widget _buildImageTile(ImageModel image) {
    final thumbnailUrl = image.thumbnailSmallUrl;
    
    if (thumbnailUrl == null || thumbnailUrl.isEmpty) {
      return Container(
        height: 150,
        color: Colors.grey[200],
        child: const Icon(Icons.image, color: Colors.grey),
      );
    }
    
    return CachedNetworkImage(
      imageUrl: thumbnailUrl,
      fit: BoxFit.cover,
      memCacheWidth: 300,
      placeholder: (context, url) => Container(
        height: 150,
        color: Colors.grey[200],
        child: const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      ),
      errorWidget: (context, url, error) => Container(
        height: 150,
        color: Colors.grey[200],
        child: const Icon(Icons.error, color: Colors.red),
      ),
    );
  }
}
