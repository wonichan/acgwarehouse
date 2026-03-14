import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';
import '../models/image.dart';

typedef ImageTapCallback = void Function(ImageModel image);

class ImageMasonry extends StatelessWidget {
  final List<ImageModel> images;
  final ImageTapCallback? onImageTap;
  final int crossAxisCount;
  
  const ImageMasonry({
    super.key,
    required this.images,
    this.onImageTap,
    this.crossAxisCount = 2,
  });
  
  @override
  Widget build(BuildContext context) {
    return MasonryGridView.count(
      crossAxisCount: crossAxisCount,
      mainAxisSpacing: 4,
      crossAxisSpacing: 4,
      itemCount: images.length,
      itemBuilder: (context, index) {
        final image = images[index];
        return GestureDetector(
          onTap: onImageTap != null ? () => onImageTap!(image) : null,
          child: _buildImageTile(image),
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
