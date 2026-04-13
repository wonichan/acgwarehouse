import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:provider/provider.dart';
import '../providers/image_provider.dart';
import '../providers/selection_provider.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';
import '../widgets/batch_operation_sheet.dart';
import '../widgets/batch_tag_dialog.dart';
import '../widgets/responsive_image_grid.dart';
import '../widgets/fluent_tag_filter_pane.dart';
import 'image_detail_screen.dart';
import 'tag_management_screen.dart';

class GalleryScreen extends StatelessWidget {
  const GalleryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    // All providers should be injected from main.dart
    return const _GalleryContent();
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
    // Load tags once on initialization
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TagProvider>().loadTags();
    });
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
      context.read<ImageListProvider>().loadImages();
    }
  }

  @override
  Widget build(BuildContext context) {
    final selectionProvider = context.watch<SelectionProvider>();
    final inSelectionMode = selectionProvider.isSelectionMode;

    return Scaffold(
      appBar: AppBar(
        title: inSelectionMode
            ? Text('已选 ${selectionProvider.selectedCount} 张')
            : const Text('图库'),
        actions: inSelectionMode
            ? [
                TextButton(
                  onPressed: selectionProvider.exitSelectionMode,
                  child: const Text('完成'),
                ),
              ]
            : [
                _buildSortButton(context),
                Builder(
                  builder: (context) => IconButton(
                    icon: const Icon(Icons.filter_list),
                    tooltip: '高级标签筛选',
                    onPressed: () => Scaffold.of(context).openEndDrawer(),
                  ),
                ),
                _buildManageTagsButton(context),
              ],
      ),
      endDrawer: Drawer(
        width: 320,
        child: fluent.FluentTheme(
          data: fluent.FluentThemeData(
            brightness: Theme.of(context).brightness == Brightness.dark
                ? fluent.Brightness.dark
                : fluent.Brightness.light,
          ),
          child: FluentTagFilterPane(
            hasTagsFilter: context.watch<ImageListProvider>().hasTagsFilter,
            onHasTagsChanged: (value) {
              context.read<ImageListProvider>().setHasTagsFilter(value);
            },
            hasPendingTagsFilter: context
                .watch<ImageListProvider>()
                .hasPendingTagsFilter,
            onHasPendingTagsChanged: (value) {
              context.read<ImageListProvider>().setHasPendingTagsFilter(value);
            },
          ),
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
            final hasPendingTagsFilter = provider.hasPendingTagsFilter == true;
            final hasFilters =
                hasTagFilter || hasUntaggedFilter || hasPendingTagsFilter;
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(
                    hasFilters
                        ? Icons.filter_alt_off
                        : Icons.photo_library_outlined,
                    size: 64,
                    color: Theme.of(context).colorScheme.outlineVariant,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    hasFilters ? '筛选出 0 张图片' : '暂无图片',
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: Theme.of(context).colorScheme.outline,
                    ),
                  ),
                  if (hasFilters) ...[
                    const SizedBox(height: 8),
                    TextButton.icon(
                      onPressed: () {
                        context.read<TagProvider>().clearSelection();
                        provider.setTagFilter([]);
                        provider.setHasTagsFilter(null);
                        provider.setHasPendingTagsFilter(null);
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
    final messenger = ScaffoldMessenger.of(context);
    final imageIds = selectionProvider.selectedImageIds.toList();

    // Close bottom sheet
    Navigator.pop(context);

    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => BatchAddTagDialog(imageIds: imageIds),
    );

    // Handle result
    if (!mounted) return;

    if (result is Map && result['success'] == true) {
      final successCount = result['successCount'] as int? ?? 0;
      final failCount = result['failCount'] as int? ?? 0;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            '已为 $successCount 张图片添加标签${failCount > 0 ? '，$failCount 张失败' : ''}',
          ),
        ),
      );
      selectionProvider.exitSelectionMode();
    } else if (result is Map && result['success'] == false) {
      messenger.showSnackBar(
        SnackBar(content: Text('添加失败: ${result['error']}')),
      );
    }
  }

  Future<void> _batchRemoveTags(BuildContext context) async {
    final selectionProvider = context.read<SelectionProvider>();
    final messenger = ScaffoldMessenger.of(context);
    final imageIds = selectionProvider.selectedImageIds.toList();

    // Close bottom sheet
    Navigator.pop(context);

    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => BatchRemoveTagDialog(imageIds: imageIds),
    );

    // Handle result
    if (!mounted) return;

    if (result is Map && result['success'] == true) {
      final successCount = result['successCount'] as int? ?? 0;
      final failCount = result['failCount'] as int? ?? 0;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            '已从 $successCount 张图片移除标签${failCount > 0 ? '，$failCount 张失败' : ''}',
          ),
        ),
      );
      selectionProvider.exitSelectionMode();
    }
  }

  Future<void> _generateAITags(BuildContext context) async {
    final selectionProvider = context.read<SelectionProvider>();
    final tagService = context.read<TagProvider>().tagService;
    final messenger = ScaffoldMessenger.of(context);
    final imageIds = selectionProvider.selectedImageIds.toList();
    final count = imageIds.length;

    // Close bottom sheet immediately
    Navigator.pop(context);

    // Show immediate feedback
    messenger.showSnackBar(
      SnackBar(content: Text('AI标签生成任务已在后台启动 ($count张图片)')),
    );

    // Exit selection mode
    selectionProvider.exitSelectionMode();

    // Fire API call asynchronously without blocking
    _triggerAITagsAsync(tagService, imageIds);
  }

  void _triggerAITagsAsync(TagService tagService, List<int> imageIds) async {
    try {
      await tagService.batchRegenerateAITags(imageIds: imageIds);
      // Success - no need to show another message
    } catch (e) {
      // Log error but don't show to user since we're async
      debugPrint('AI tag generation error: $e');
    }
  }

  void _navigateToDetail(BuildContext context, image) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => ImageDetailScreen(image: image)),
    );
  }

  void _navigateToTagManagement(BuildContext context) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => const TagManagementScreen()),
    );
  }
}
