import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

/// Unified image widget with caching, loading state, and error handling.
///
/// Uses [flutter_cache_manager] for disk caching with automatic stale object
/// cleanup (30 days) and a 500-object LRU limit.
class CachedImageWidget extends StatelessWidget {
  final String imageUrl;
  final double? width;
  final double? height;
  final BoxFit fit;
  final Widget? placeholder;
  final Widget? errorBuilder;
  final BorderRadius? borderRadius;

  const CachedImageWidget({
    super.key,
    required this.imageUrl,
    this.width,
    this.height,
    this.fit = BoxFit.cover,
    this.placeholder,
    this.errorBuilder,
    this.borderRadius,
  });

  /// Shared cache manager for all ACGWarehouse images.
  static final CacheManager cacheManager = CacheManager(
    Config(
      'acgwarehouse-images',
      stalePeriod: const Duration(days: 30),
      maxNrOfCacheObjects: 500,
    ),
  );

  @override
  Widget build(BuildContext context) {
    Widget child = CachedNetworkImage(
      imageUrl: imageUrl,
      width: width,
      height: height,
      fit: fit,
      cacheManager: cacheManager,
      fadeInDuration: const Duration(milliseconds: 300),
      fadeOutDuration: const Duration(milliseconds: 200),
      placeholder: (context, url) =>
          placeholder ??
          const Center(child: CircularProgressIndicator(strokeWidth: 2)),
      errorWidget: (context, url, error) =>
          errorBuilder ??
          Icon(
            Icons.broken_image,
            size: 48,
            color: Theme.of(context).colorScheme.outlineVariant,
          ),
    );

    if (borderRadius != null) {
      child = ClipRRect(borderRadius: borderRadius!, child: child);
    }

    return child;
  }
}
