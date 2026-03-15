import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/image_provider.dart';
import '../services/api_service.dart';
import '../widgets/image_grid.dart';
import '../widgets/image_masonry.dart';
import '../widgets/tag_filter_drawer.dart';
import 'image_detail_screen.dart';
import 'tag_management_screen.dart';

class GalleryScreen extends StatelessWidget {
  const GalleryScreen({super.key});
  
  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => ImageListProvider(ApiService())..loadImages(),
      child: const _GalleryContent(),
    );
  }
}

class _GalleryContent extends StatelessWidget {
  const _GalleryContent();
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ACGWarehouse'),
        actions: [
          _buildTagFilterButton(context),
          _buildViewModeToggle(context),
          _buildSortButton(context),
          _buildManageTagsButton(context),
        ],
      ),
      drawer: TagFilterDrawer(
        initialSelectedIds: context.watch<ImageListProvider>().selectedTagIds,
        onFilterChanged: (tagIds) {
          context.read<ImageListProvider>().setTagFilter(tagIds);
        },
      ),
      body: Consumer<ImageListProvider>(
        builder: (context, provider, child) {
          if (provider.images.isEmpty && provider.isLoading) {
            return const Center(child: CircularProgressIndicator());
          }
          
          if (provider.images.isEmpty) {
            return const Center(child: Text('暂无图片'));
          }
          
          final widget = provider.viewMode == ViewMode.grid
              ? ImageGrid(
                  images: provider.images,
                  onImageTap: (img) => _navigateToDetail(context, img),
                )
              : ImageMasonry(
                  images: provider.images,
                  onImageTap: (img) => _navigateToDetail(context, img),
                );
          
          return RefreshIndicator(
            onRefresh: () => provider.loadImages(refresh: true),
            child: widget,
          );
        },
      ),
    );
  }
  
  Widget _buildTagFilterButton(BuildContext context) {
    return IconButton(
      icon: const Icon(Icons.filter_list),
      onPressed: () => Scaffold.of(context).openDrawer(),
      tooltip: '标签筛选',
    );
  }
  
  Widget _buildViewModeToggle(BuildContext context) {
    return Consumer<ImageListProvider>(
      builder: (context, provider, _) {
        return IconButton(
          icon: Icon(
            provider.viewMode == ViewMode.grid
                ? Icons.view_agenda
                : Icons.grid_view,
          ),
          onPressed: () {
            provider.setViewMode(
              provider.viewMode == ViewMode.grid
                  ? ViewMode.masonry
                  : ViewMode.grid,
            );
          },
          tooltip: provider.viewMode == ViewMode.grid
              ? '切换到瀑布流'
              : '切换到网格',
        );
      },
    );
  }
  
  Widget _buildSortButton(BuildContext context) {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.sort),
      onSelected: (value) {
        final provider = context.read<ImageListProvider>();
        final asc = value.endsWith('_asc');
        final field = value.replaceAll('_asc', '').replaceAll('_desc', '');
        provider.setSort(
          SortField.values.firstWhere((f) => f.name == field),
          asc,
        );
      },
      itemBuilder: (context) => [
        const PopupMenuItem(value: 'createdAt_desc', child: Text('最新导入')),
        const PopupMenuItem(value: 'createdAt_asc', child: Text('最早导入')),
        const PopupMenuItem(value: 'filename_asc', child: Text('文件名 A-Z')),
        const PopupMenuItem(value: 'filename_desc', child: Text('文件名 Z-A')),
        const PopupMenuItem(value: 'fileSize_desc', child: Text('文件最大')),
        const PopupMenuItem(value: 'fileSize_asc', child: Text('文件最小')),
      ],
    );
  }
  
  Widget _buildManageTagsButton(BuildContext context) {
    return IconButton(
      icon: const Icon(Icons.label_outline),
      onPressed: () => _navigateToTagManagement(context),
      tooltip: '标签管理',
    );
  }
  
  void _navigateToDetail(BuildContext context, image) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => ImageDetailScreen(image: image),
      ),
    );
  }
  
  void _navigateToTagManagement(BuildContext context) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => const TagManagementScreen(),
      ),
    );
  }
}