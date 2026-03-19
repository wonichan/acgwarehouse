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
      drawer: Drawer(
        child: Consumer<ImageListProvider>(
          builder: (context, provider, _) {
            return TagFilterDrawer(
              hasTagsFilter: provider.hasTagsFilter,
              onFilterChanged: (tagIds) {
                // Update image list with selected tags
                provider.setTagFilter(tagIds);
              },
              onHasTagsChanged: (hasTags) {
                // Update image list with hasTags filter
                provider.setHasTagsFilter(hasTags);
              },
            );
          },
        ),
      ),
      body: Consumer<ImageListProvider>(
        builder: (context, provider, child) {
          if (provider.images.isEmpty && provider.isLoading) {
            return const Center(child: CircularProgressIndicator());
          }
          
          if (provider.images.isEmpty) {
            // Check if filters are applied
            final hasFilters = provider.selectedTagIds.isNotEmpty;
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(
                    hasFilters ? Icons.filter_alt_off : Icons.photo_library_outlined,
                    size: 64,
                    color: Colors.grey[400],
                  ),
                  const SizedBox(height: 16),
                  Text(
                    hasFilters ? '筛选出 0 张图片' : '暂无图片',
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: Colors.grey[600],
                    ),
                  ),
                  if (hasFilters) ...[
                    const SizedBox(height: 8),
                    TextButton.icon(
                      onPressed: () {
                        context.read<TagProvider>().clearSelection();
                        provider.setTagFilter([]);
                      },
                      icon: const Icon(Icons.clear_all),
                      label: const Text('清除筛选'),
                    ),
                  ],
                ],
              ),
            );
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
      builder: (buttonContext) {
        return IconButton(
          icon: const Icon(Icons.filter_list),
          onPressed: () {
            // 打开左侧抽屉
            Scaffold.of(buttonContext).openDrawer();
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
    return Consumer<ImageListProvider>(
      builder: (context, provider, _) {
        return PopupMenuButton<String>(
          icon: const Icon(Icons.sort),
          tooltip: '排序',
          onSelected: (value) {
            debugPrint('排序菜单被选中: $value');
            final asc = value.endsWith('_asc');
            final field = value.replaceAll('_asc', '').replaceAll('_desc', '');
            
            // 安全地匹配 SortField 枚举
            SortField? sortField;
            if (field == 'createdAt') {
              sortField = SortField.createdAt;
            } else if (field == 'filename') {
              sortField = SortField.filename;
            } else if (field == 'fileSize') {
              sortField = SortField.fileSize;
            }
            
            debugPrint('解析结果: field=$field, asc=$asc, sortField=$sortField');
            
            if (sortField != null) {
              provider.setSort(sortField, asc);
            }
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
      },
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