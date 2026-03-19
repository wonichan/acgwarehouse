import 'package:flutter/material.dart';
import 'package:cached_network_image/cached_network_image.dart';
import '../models/image.dart';

typedef ImageTapCallback = void Function(ImageModel image);

class ImageGrid extends StatelessWidget {
  final List<ImageModel> images;
  final ImageTapCallback? onImageTap;
  final int crossAxisCount;
  
  const ImageGrid({
    super.key,
    required this.images,
    this.onImageTap,
    this.crossAxisCount = 3,
  });
  
  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: crossAxisCount,
        mainAxisSpacing: 4,
        crossAxisSpacing: 4,
      ),
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
        print('Image load error: $error');
        print('URL: $thumbnailUrl');
        return Container(
          color: Colors.grey[200],
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error, color: Colors.red, size: 20),
              Text('Error', style: TextStyle(fontSize: 10, color: Colors.red)),
            ],
          ),
        );
      },
    );
  }
}
