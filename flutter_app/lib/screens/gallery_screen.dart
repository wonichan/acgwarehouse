import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/image_provider.dart';
import '../providers/tag_provider.dart';
import '../services/api_service.dart';
import '../services/tag_service.dart';
import '../widgets/image_grid.dart';
import '../widgets/image_masonry.dart';
import '../widgets/tag_filter_drawer.dart';
import 'image_detail_screen.dart';
import 'tag_management_screen.dart';

class GalleryScreen extends StatelessWidget {
  const GalleryScreen({super.key});
  
  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => ImageListProvider(ApiService())..loadImages()),
        ChangeNotifierProvider(create: (_) => TagProvider(TagService())),
      ],
      child: const _GalleryContent(),
    );
  }
}

class _GalleryContent extends StatefulWidget {
  const _GalleryContent();
  
  @override
  State<_GalleryContent> createState() => _GalleryContentState();
}

class _GalleryContentState extends State<_GalleryContent> {
  final ScrollController _scrollController = ScrollController();
  
  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }
  
  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }
  
  void _onScroll() {
    if (_scrollController.position.pixels >= 
        _scrollController.position.maxScrollExtent - 200) {
      // Load more when within 200 pixels of bottom
      context.read<ImageListProvider>().loadImages();
    }
  }
  
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
        onFilterChanged: (tagIds) {
          // Update both providers
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
                  scrollController: _scrollController,
                )
              : ImageMasonry(
                  images: provider.images,
                  onImageTap: (img) => _navigateToDetail(context, img),
                  scrollController: _scrollController,
                );
          
          return Stack(
            children: [
              RefreshIndicator(
                onRefresh: () => provider.loadImages(refresh: true),
                child: widget,
              ),
              // Loading indicator at bottom when fetching more
              if (provider.isLoading && provider.hasMore)
                const Positioned(
                  bottom: 16,
                  left: 0,
                  right: 0,
                  child: Center(child: CircularProgressIndicator()),
                ),
            ],
          );
        },
      ),
    );
  }
  
  Widget _buildTagFilterButton(BuildContext context) {
    return Builder(
      builder: (context) {
        return IconButton(
          icon: const Icon(Icons.filter_list),
          onPressed: () {
            debugPrint('标签筛选按钮被点击');
            try {
              Scaffold.of(context).openDrawer();
              debugPrint('Drawer 打开成功');
            } catch (e) {
              debugPrint('打开 Drawer 失败: $e');
            }
          },
          tooltip: '标签筛选',
        );
      },
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