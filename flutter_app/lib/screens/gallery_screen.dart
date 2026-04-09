import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../providers/image_provider.dart';
import '../providers/selection_provider.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';
import '../widgets/batch_operation_sheet.dart';
import '../widgets/batch_tag_dialog.dart';
import '../widgets/responsive_image_grid.dart';
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
            : _buildSearchBar(context),
        actions: inSelectionMode
            ? [
                TextButton(
                  onPressed: selectionProvider.exitSelectionMode,
                  child: const Text('完成'),
                ),
              ]
            : [_buildSortButton(context), _buildManageTagsButton(context)],
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
                    hasFilters
                        ? Icons.filter_alt_off
                        : Icons.photo_library_outlined,
                    size: 64,
                    color: Colors.grey[400],
                  ),
                  const SizedBox(height: 16),
                  Text(
                    hasFilters ? '筛选出 0 张图片' : '暂无图片',
                    style: Theme.of(
                      context,
                    ).textTheme.titleLarge?.copyWith(color: Colors.grey[600]),
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

  Widget _buildSearchBar(BuildContext context) {
    final tagProvider = context.watch<TagProvider>();
    final imageListProvider = context.watch<ImageListProvider>();
    final selectedTags = tagProvider.selectedTags;

    final isDark = Theme.of(context).brightness == Brightness.dark;
    final borderColor = isDark ? Colors.white24 : Colors.black12;
    final textColor = isDark ? Colors.white : Colors.black87;
    final hintColor = isDark ? Colors.white54 : Colors.black54;

    return Container(
      height: 40,
      decoration: BoxDecoration(
        border: Border.all(color: borderColor),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Row(
        children: [
          const SizedBox(width: 8),
          Icon(Icons.search, size: 20, color: hintColor),
          const SizedBox(width: 8),
          if (selectedTags.isNotEmpty)
            ...selectedTags.map(
              (tag) => Padding(
                padding: const EdgeInsets.only(right: 6.0),
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    border: Border.all(color: borderColor),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        tag.preferredLabel,
                        style: TextStyle(
                          fontSize: 13,
                          color: textColor,
                          height: 1.0,
                        ),
                      ),
                      const SizedBox(width: 4),
                      InkWell(
                        onTap: () {
                          tagProvider.toggleTag(tag.id);
                          imageListProvider.setTagFilter(
                            tagProvider.selectedTagIds.toList(),
                          );
                        },
                        child: Icon(Icons.close, size: 14, color: hintColor),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          Expanded(
            child: Autocomplete<Tag>(
              optionsBuilder: (TextEditingValue textEditingValue) {
                if (textEditingValue.text.isEmpty) {
                  return const Iterable<Tag>.empty();
                }
                return tagProvider.allTags.where((Tag tag) {
                  return tag.preferredLabel.toLowerCase().contains(
                    textEditingValue.text.toLowerCase(),
                  );
                });
              },
              displayStringForOption: (Tag option) => option.preferredLabel,
              onSelected: (Tag selection) {
                if (!tagProvider.selectedTagIds.contains(selection.id)) {
                  tagProvider.toggleTag(selection.id);
                  imageListProvider.setTagFilter(
                    tagProvider.selectedTagIds.toList(),
                  );
                }
              },
              fieldViewBuilder:
                  (
                    context,
                    textEditingController,
                    focusNode,
                    onFieldSubmitted,
                  ) {
                    return TextField(
                      controller: textEditingController,
                      focusNode: focusNode,
                      style: TextStyle(color: textColor, fontSize: 14),
                      decoration: InputDecoration(
                        isDense: true,
                        hintText: 'Search tags...',
                        hintStyle: TextStyle(color: hintColor, fontSize: 14),
                        border: InputBorder.none,
                        contentPadding: const EdgeInsets.symmetric(
                          vertical: 10,
                        ),
                      ),
                      onSubmitted: (String value) {
                        onFieldSubmitted();
                      },
                    );
                  },
              optionsViewBuilder: (context, onSelected, options) {
                return Align(
                  alignment: Alignment.topLeft,
                  child: Material(
                    elevation: 4.0,
                    color: Theme.of(context).cardColor,
                    borderRadius: BorderRadius.circular(4),
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(
                        maxHeight: 200,
                        maxWidth: 300,
                      ),
                      child: ListView.builder(
                        padding: EdgeInsets.zero,
                        shrinkWrap: true,
                        itemCount: options.length,
                        itemBuilder: (BuildContext context, int index) {
                          final Tag option = options.elementAt(index);
                          return InkWell(
                            onTap: () => onSelected(option),
                            child: Padding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 16.0,
                                vertical: 12.0,
                              ),
                              child: Text(
                                option.preferredLabel,
                                style: TextStyle(color: textColor),
                              ),
                            ),
                          );
                        },
                      ),
                    ),
                  ),
                );
              },
            ),
          ),
          if (selectedTags.isNotEmpty)
            InkWell(
              onTap: () {
                tagProvider.clearSelection();
                imageListProvider.setTagFilter([]);
                imageListProvider.setHasTagsFilter(null);
              },
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 8.0),
                child: Icon(Icons.clear_all, size: 20, color: hintColor),
              ),
            ),
          const SizedBox(width: 8),
          // Toggle for "Untagged Only"
          InkWell(
            onTap: () {
              final isUntagged = imageListProvider.hasTagsFilter == false;
              imageListProvider.setHasTagsFilter(isUntagged ? null : false);
            },
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: imageListProvider.hasTagsFilter == false
                    ? (isDark ? Colors.white24 : Colors.black12)
                    : Colors.transparent,
                border: Border.all(color: borderColor),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Row(
                children: [
                  Icon(
                    imageListProvider.hasTagsFilter == false
                        ? Icons.label_off
                        : Icons.label_outline,
                    size: 14,
                    color: textColor,
                  ),
                  const SizedBox(width: 4),
                  Text(
                    '未打标签',
                    style: TextStyle(
                      fontSize: 13,
                      color: textColor,
                      height: 1.0,
                    ),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(width: 8),
        ],
      ),
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
          content: Text(
            '已为 $successCount 张图片添加标签${failCount > 0 ? '，$failCount 张失败' : ''}',
          ),
        ),
      );
      selectionProvider.exitSelectionMode();
    } else if (result is Map && result['success'] == false && mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('添加失败: ${result['error']}')));
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
    final imageIds = selectionProvider.selectedImageIds.toList();
    final count = imageIds.length;

    // Close bottom sheet immediately
    Navigator.pop(context);

    // Show immediate feedback
    if (context.mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('AI标签生成任务已在后台启动 ($count张图片)')));
    }

    // Exit selection mode
    selectionProvider.exitSelectionMode();

    // Fire API call asynchronously without blocking
    _triggerAITagsAsync(tagService, imageIds);
  }

  void _triggerAITagsAsync(TagService tagService, List<int> imageIds) async {
    try {
      await tagService.batchTriggerAITags(imageIds: imageIds);
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
