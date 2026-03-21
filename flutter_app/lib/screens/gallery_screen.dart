import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/image_provider.dart';
import '../providers/selection_provider.dart';
import '../providers/tag_provider.dart';
import '../services/api_service.dart';
import '../services/tag_service.dart';
import '../widgets/batch_operation_sheet.dart';
import '../widgets/batch_tag_dialog.dart';
import '../widgets/responsive_image_grid.dart';
import '../widgets/tag_filter_drawer.dart';
import 'image_detail_screen.dart';
import 'tag_management_screen.dart';

class GalleryScreen extends StatelessWidget {
  final ImageListProvider? imageListProvider;
  final TagProvider? tagProvider;
  final SelectionProvider? selectionProvider;
  final TagService? tagService;

  const GalleryScreen({
    super.key,
    this.imageListProvider,
    this.tagProvider,
    this.selectionProvider,
    this.tagService,
  });

  @override
  Widget build(BuildContext context) {
    final resolvedTagService = tagService ?? TagService();
    final resolvedImageListProvider = imageListProvider ?? _tryRead<ImageListProvider>(context);
    final resolvedTagProvider = tagProvider ?? _tryRead<TagProvider>(context);
    final resolvedSelectionProvider = selectionProvider ?? _tryRead<SelectionProvider>(context);

    if (resolvedImageListProvider != null &&
        resolvedTagProvider != null &&
        resolvedSelectionProvider != null &&
        imageListProvider == null &&
        tagProvider == null &&
        selectionProvider == null) {
      return const _GalleryContent();
    }

    return MultiProvider(
      providers: [
        if (resolvedImageListProvider != null && imageListProvider != null)
          ChangeNotifierProvider<ImageListProvider>.value(value: resolvedImageListProvider),
        if (resolvedImageListProvider == null)
          ChangeNotifierProvider(create: (_) => ImageListProvider(ApiService())..loadImages()),
        if (resolvedTagProvider != null && tagProvider != null)
          ChangeNotifierProvider<TagProvider>.value(value: resolvedTagProvider),
        if (resolvedTagProvider == null)
          ChangeNotifierProvider(create: (_) => TagProvider(resolvedTagService)),
        if (resolvedSelectionProvider != null && selectionProvider != null)
          ChangeNotifierProvider<SelectionProvider>.value(value: resolvedSelectionProvider),
        if (resolvedSelectionProvider == null)
          ChangeNotifierProvider(create: (_) => SelectionProvider()),
      ],
      child: const _GalleryContent(),
    );
  }

  T? _tryRead<T>(BuildContext context) {
    try {
      return context.read<T>();
    } catch (_) {
      return null;
    }
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
    if (_scrollController.position.pixels >= _scrollController.position.maxScrollExtent - 200) {
      context.read<ImageListProvider>().loadImages();
    }
  }

  @override
  Widget build(BuildContext context) {
    final selectionProvider = context.watch<SelectionProvider>();
    final inSelectionMode = selectionProvider.isSelectionMode;

    return Scaffold(
      appBar: AppBar(
        title: Text(inSelectionMode ? '已选 ${selectionProvider.selectedCount} 张' : 'ACGWarehouse'),
        actions: inSelectionMode
            ? [
                TextButton(
                  onPressed: selectionProvider.exitSelectionMode,
                  child: const Text('完成'),
                ),
              ]
            : [
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
                provider.setTagFilter(tagIds);
              },
              onHasTagsChanged: (hasTags) {
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
            final hasTagFilter = provider.selectedTagIds.isNotEmpty;
            final hasUntaggedFilter = provider.hasTagsFilter == false;
            final hasFilters = hasTagFilter || hasUntaggedFilter;
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
                        provider.setHasTagsFilter(null);
                      },
                      icon: const Icon(Icons.clear_all),
                      label: const Text('清除筛选'),
                    ),
                  ],
                ],
              ),
            );
          }

          final galleryWidget = ResponsiveImageGrid(
            images: provider.images,
            viewMode: provider.viewMode,
            onImageTap: (img) => _navigateToDetail(context, img),
            scrollController: _scrollController,
            selectionProvider: selectionProvider,
          );

          return Stack(
            children: [
              RefreshIndicator(
                onRefresh: () async {
                  await provider.loadImages(refresh: true);
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('已刷新'),
                        duration: Duration(seconds: 1),
                      ),
                    );
                  }
                },
                displacement: 40,
                child: galleryWidget,
              ),
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
      floatingActionButton: selectionProvider.hasSelection
          ? FloatingActionButton.extended(
              onPressed: () => _showBatchOperations(context),
              icon: const Icon(Icons.more_horiz),
              label: Text('批量操作 (${selectionProvider.selectedCount})'),
            )
          : null,
    );
  }

  Widget _buildTagFilterButton(BuildContext context) {
    return Builder(
      builder: (buttonContext) {
        return IconButton(
          icon: const Icon(Icons.filter_list),
          onPressed: () {
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
          icon: Icon(provider.viewMode == ViewMode.grid ? Icons.view_agenda : Icons.grid_view),
          onPressed: () {
            provider.setViewMode(
              provider.viewMode == ViewMode.grid ? ViewMode.masonry : ViewMode.grid,
            );
          },
          tooltip: provider.viewMode == ViewMode.grid ? '切换到瀑布流' : '切换到网格',
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
            final asc = value.endsWith('_asc');
            final field = value.replaceAll('_asc', '').replaceAll('_desc', '');

            SortField? sortField;
            if (field == 'createdAt') {
              sortField = SortField.createdAt;
            } else if (field == 'filename') {
              sortField = SortField.filename;
            } else if (field == 'fileSize') {
              sortField = SortField.fileSize;
            }

            if (sortField != null) {
              provider.setSort(sortField, asc);
            }
          },
          itemBuilder: (context) => const [
            PopupMenuItem(value: 'createdAt_desc', child: Text('最新导入')),
            PopupMenuItem(value: 'createdAt_asc', child: Text('最早导入')),
            PopupMenuItem(value: 'filename_asc', child: Text('文件名 A-Z')),
            PopupMenuItem(value: 'filename_desc', child: Text('文件名 Z-A')),
            PopupMenuItem(value: 'fileSize_desc', child: Text('文件最大')),
            PopupMenuItem(value: 'fileSize_asc', child: Text('文件最小')),
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

  Future<void> _showBatchOperations(BuildContext context) {
    final selectionProvider = context.read<SelectionProvider>();
    return BatchOperationSheet.show(
      context,
      selectionProvider: selectionProvider,
      onAddTags: () => _batchAddTags(context),
      onRemoveTags: () => _batchRemoveTags(context),
      onGenerateAITags: () => _generateAITags(context),
    );
  }

  Future<void> _batchAddTags(BuildContext context) async {
    final selectionProvider = context.read<SelectionProvider>();
    final imageIds = selectionProvider.selectedImageIds.toList();

    // Close bottom sheet
    Navigator.pop(context);

    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => BatchAddTagDialog(imageIds: imageIds),
    );

    // Handle result
    if (result is Map && result['success'] == true && mounted) {
      final successCount = result['successCount'] as int? ?? 0;
      final failCount = result['failCount'] as int? ?? 0;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('已为 $successCount 张图片添加标签${failCount > 0 ? '，$failCount 张失败' : ''}'),
        ),
      );
      selectionProvider.exitSelectionMode();
    } else if (result is Map && result['success'] == false && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('添加失败: ${result['error']}')),
      );
    }
  }

  Future<void> _batchRemoveTags(BuildContext context) async {
    final selectionProvider = context.read<SelectionProvider>();
    final imageIds = selectionProvider.selectedImageIds.toList();

    // Close bottom sheet
    Navigator.pop(context);

    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => BatchRemoveTagDialog(imageIds: imageIds),
    );

    // Handle result
    if (result is Map && result['success'] == true && mounted) {
      final successCount = result['successCount'] as int? ?? 0;
      final failCount = result['failCount'] as int? ?? 0;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('已从 $successCount 张图片移除标签${failCount > 0 ? '，$failCount 张失败' : ''}'),
        ),
      );
      selectionProvider.exitSelectionMode();
    }
  }

  Future<void> _generateAITags(BuildContext context) async {
    final selectionProvider = context.read<SelectionProvider>();
    final tagService = context.read<TagProvider>().tagService;
    final imageIds = selectionProvider.selectedImageIds.toList();
    final count = imageIds.length;

    // Close bottom sheet immediately
    Navigator.pop(context);
    
    // Show immediate feedback
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('AI标签生成任务已在后台启动 ($count张图片)')),
      );
    }
    
    // Exit selection mode
    selectionProvider.exitSelectionMode();

    // Fire API call asynchronously without blocking
    _triggerAITagsAsync(tagService, imageIds);
  }

  void _triggerAITagsAsync(TagService tagService, List<int> imageIds) async {
    try {
      await tagService.batchTriggerAITags(imageIds);
      // Success - no need to show another message
    } catch (e) {
      // Log error but don't show to user since we're async
      debugPrint('AI tag generation error: $e');
    }
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
