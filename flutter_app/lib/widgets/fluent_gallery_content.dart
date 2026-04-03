import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/material.dart' show RefreshIndicator;
import 'package:provider/provider.dart';
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';

import '../providers/image_provider.dart';
import '../models/image.dart';
import 'fluent_image_card.dart';

/// Fluent 风格图库内容区域
/// 支持网格视图和瀑布流视图切换
/// 支持滚动分页加载和下拉刷新
class FluentGalleryContent extends StatefulWidget {
  final void Function(ImageModel)? onImageTap;
  final ScrollController? scrollController;

  const FluentGalleryContent({
    super.key,
    this.onImageTap,
    this.scrollController,
  });

  @override
  State<FluentGalleryContent> createState() => _FluentGalleryContentState();
}

class _FluentGalleryContentState extends State<FluentGalleryContent> {
  late ScrollController _internalScrollController;
  bool _disposed = false;

  ScrollController get _scrollController =>
      widget.scrollController ?? _internalScrollController;

  @override
  void initState() {
    super.initState();
    _internalScrollController = ScrollController();
    _internalScrollController.addListener(_onScroll);
  }

  @override
  void didUpdateWidget(FluentGalleryContent oldWidget) {
    super.didUpdateWidget(oldWidget);
    // 如果外部传入的 scrollController 发生变化，需要更新监听
    if (oldWidget.scrollController != widget.scrollController) {
      if (oldWidget.scrollController == null) {
        _internalScrollController.removeListener(_onScroll);
      }
      if (widget.scrollController == null && !_disposed) {
        _internalScrollController.addListener(_onScroll);
      }
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _internalScrollController.removeListener(_onScroll);
    _internalScrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    // 当滚动到接近底部时触发加载更多
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final provider = context.read<ImageListProvider>();
      if (!provider.isLoading && provider.hasMore) {
        provider.loadImages();
      }
    }
  }

  Future<void> _onRefresh() async {
    final provider = context.read<ImageListProvider>();
    await provider.loadImages(refresh: true);
  }

  void _ensureScrollableContent(ImageListProvider provider) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) {
        return;
      }

      if (provider.isLoading || !provider.hasMore || provider.images.isEmpty) {
        return;
      }

      if (_scrollController.position.maxScrollExtent <= 0) {
        provider.loadImages();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<ImageListProvider>(
      builder: (context, provider, child) {
        // Empty state
        if (provider.images.isEmpty && !provider.isLoading) {
          return _buildEmptyState(context, provider);
        }

        // Loading state
        if (provider.images.isEmpty && provider.isLoading) {
          return const Center(child: ProgressRing());
        }

        _ensureScrollableContent(provider);

        // Image grid/masonry with RefreshIndicator
        return Stack(
          children: [
            RefreshIndicator(
              onRefresh: _onRefresh,
              displacement: 40,
              child: _buildImageList(provider),
            ),
            // Loading indicator at bottom for pagination
            if (provider.isLoading && provider.hasMore)
              const Positioned(
                bottom: 16,
                left: 0,
                right: 0,
                child: Center(child: ProgressRing()),
              ),
          ],
        );
      },
    );
  }

  Widget _buildEmptyState(BuildContext context, ImageListProvider provider) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            FluentIcons.photo2,
            size: 64,
            color: FluentTheme.of(context).resources.textFillColorSecondary,
          ),
          const SizedBox(height: 16),
          Text('暂无图片', style: FluentTheme.of(context).typography.subtitle),
        ],
      ),
    );
  }

  Widget _buildImageList(ImageListProvider provider) {
    final images = provider.images;

    if (provider.viewMode == ViewMode.grid) {
      return _buildGridView(images);
    } else {
      return _buildMasonryView(images);
    }
  }

  Widget _buildGridView(List<ImageModel> images) {
    return GridView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.all(8),
      gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
        maxCrossAxisExtent: 200,
        mainAxisSpacing: 8,
        crossAxisSpacing: 8,
      ),
      itemCount: images.length,
      itemBuilder: (context, index) {
        return FluentImageCard(image: images[index], onTap: widget.onImageTap);
      },
    );
  }

  Widget _buildMasonryView(List<ImageModel> images) {
    return MasonryGridView.count(
      controller: _scrollController,
      padding: const EdgeInsets.all(8),
      crossAxisCount: 4,
      mainAxisSpacing: 8,
      crossAxisSpacing: 8,
      itemCount: images.length,
      itemBuilder: (context, index) {
        return FluentImageCard(image: images[index], onTap: widget.onImageTap);
      },
    );
  }
}
