import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/material.dart' show MaterialPageRoute;
import 'package:provider/provider.dart';

import '../screens/image_detail_screen.dart';
import '../models/tag.dart';
import '../providers/image_provider.dart';
import '../providers/search_provider.dart';
import '../providers/tag_provider.dart';
import '../widgets/fluent_gallery_content.dart';
import '../widgets/fluent_collections_content.dart';
import '../widgets/fluent_search_content.dart';
import '../widgets/monitoring/monitoring_workspace.dart';
import '../widgets/log_viewer/log_viewer_workspace.dart';
import '../widgets/tag_management/tag_management_workspace.dart';
import '../models/image.dart';
import '../models/viewer_window_context.dart';
import '../services/collection_service.dart';
import '../services/viewer_window_service.dart';
import '../services/batch_service.dart';
import '../providers/config_provider.dart';
import '../providers/selection_provider.dart';

/// Fluent 风格图库页面
/// 包含 CommandBar 工具栏和图库内容
class FluentGalleryPage extends StatefulWidget {
  const FluentGalleryPage({super.key});

  @override
  State<FluentGalleryPage> createState() => _FluentGalleryPageState();
}

class _FluentGalleryPageState extends State<FluentGalleryPage> {
  final TextEditingController _tagSearchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    // Load tags once on initialization
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TagProvider>().loadTags();
    });
  }

  @override
  void dispose() {
    _tagSearchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer3<ImageListProvider, TagProvider, SelectionProvider>(
      builder: (context, imageProvider, tagProvider, selection, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: _buildTitle(context, imageProvider, tagProvider),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                CommandBarButton(
                  icon: const Icon(FluentIcons.sort),
                  label: const Text('排序'),
                  onPressed: () {
                    _showGallerySortOptions(context, imageProvider);
                  },
                ),
                const CommandBarSeparator(),
                // Refresh button
                CommandBarButton(
                  icon: const Icon(FluentIcons.refresh),
                  label: const Text('刷新'),
                  onPressed: () {
                    imageProvider.loadImages(refresh: true);
                  },
                ),
                const CommandBarSeparator(),
                // Batch add tags — only when images are selected
                if (selection.hasSelection) ...[
                  CommandBarButton(
                    icon: const Icon(FluentIcons.tag),
                    label: Text('批量添加标签 (${selection.selectedCount})'),
                    onPressed: () {
                      _showBatchAddTagDialog(
                        context,
                        selection,
                        tagProvider,
                        imageProvider,
                      );
                    },
                  ),
                  CommandBarButton(
                    icon: const Icon(FluentIcons.clear),
                    label: const Text('取消选择'),
                    onPressed: selection.clearSelection,
                  ),
                  const CommandBarSeparator(),
                ],
                CommandBarButton(
                  icon: const Icon(FluentIcons.auto_enhance_on),
                  label: const Text('批量AI标签'),
                  onPressed: imageProvider.images.isEmpty
                      ? null
                      : () {
                          _confirmAndTriggerBatchAITags(
                            context,
                            imageProvider,
                            tagProvider,
                          );
                        },
                ),
              ],
            ),
          ),
          content: _buildGalleryWorkspace(context),
        );
      },
    );
  }

  Widget _buildGalleryWorkspace(BuildContext context) {
    return FluentGalleryContent(
      onImageTap: null, // Removed single click routing
      onImageDoubleTap: (image) => _showGalleryImageDetail(context, image),
    );
  }

  void _showGallerySortOptions(
    BuildContext context,
    ImageListProvider imageProvider,
  ) {
    showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: const Text('排序选项'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              title: const Text('源文件创建时间（新→旧）'),
              onPressed: () {
                imageProvider.setSort(SortField.createdAt, false);
                Navigator.pop(dialogContext);
              },
            ),
            ListTile(
              title: const Text('源文件创建时间（旧→新）'),
              onPressed: () {
                imageProvider.setSort(SortField.createdAt, true);
                Navigator.pop(dialogContext);
              },
            ),
            ListTile(
              title: const Text('源文件大小（大→小）'),
              onPressed: () {
                imageProvider.setSort(SortField.fileSize, false);
                Navigator.pop(dialogContext);
              },
            ),
            ListTile(
              title: const Text('源文件大小（小→大）'),
              onPressed: () {
                imageProvider.setSort(SortField.fileSize, true);
                Navigator.pop(dialogContext);
              },
            ),
            ListTile(
              title: const Text('源文件文件名（A-Z）'),
              onPressed: () {
                imageProvider.setSort(SortField.filename, true);
                Navigator.pop(dialogContext);
              },
            ),
            ListTile(
              title: const Text('源文件文件名（Z-A）'),
              onPressed: () {
                imageProvider.setSort(SortField.filename, false);
                Navigator.pop(dialogContext);
              },
            ),
          ],
        ),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(dialogContext),
          ),
        ],
      ),
    );
  }

  Widget _buildTitle(
    BuildContext context,
    ImageListProvider imageProvider,
    TagProvider tagProvider,
  ) {
    final selectedTags = tagProvider.selectedTags;
    final hasUntaggedFilter = imageProvider.hasTagsFilter == false;

    final theme = FluentTheme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final borderColor = isDark
        ? const Color(0x3DFFFFFF)
        : const Color(0x1F000000);
    final textColor = theme.resources.textFillColorPrimary;
    final hintColor = theme.resources.textFillColorSecondary;

    return Row(
      children: [
        const Text('图库'),
        const SizedBox(width: 24),
        Expanded(
          child: Container(
            height: 40,
            decoration: BoxDecoration(
              border: Border.all(color: borderColor),
              borderRadius: BorderRadius.circular(4),
              color: theme.resources.controlFillColorDefault,
            ),
            child: Row(
              children: [
                const SizedBox(width: 8),
                Icon(FluentIcons.search, size: 16, color: hintColor),
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
                          color: isDark
                              ? const Color(0x3DFFFFFF)
                              : const Color(0x0F000000),
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
                            GestureDetector(
                              onTap: () {
                                tagProvider.toggleTag(tag.id);
                                imageProvider.setTagFilter(
                                  tagProvider.selectedTagIds.toList(),
                                );
                              },
                              child: Icon(
                                FluentIcons.clear,
                                size: 12,
                                color: hintColor,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                Expanded(
                  child: AutoSuggestBox<Tag>(
                    controller: _tagSearchController,
                    items: tagProvider.allTags
                        .map(
                          (tag) => AutoSuggestBoxItem<Tag>(
                            value: tag,
                            label: tag.preferredLabel,
                          ),
                        )
                        .toList(),
                    placeholder: '搜索标签...',
                    style: TextStyle(fontSize: 14, color: textColor),
                    decoration: WidgetStateProperty.all(
                      const BoxDecoration(color: Colors.transparent),
                    ),
                    clearButtonEnabled: false,
                    onSelected: (item) {
                      if (item.value != null &&
                          !tagProvider.selectedTagIds.contains(
                            item.value!.id,
                          )) {
                        tagProvider.toggleTag(item.value!.id);
                        imageProvider.setTagFilter(
                          tagProvider.selectedTagIds.toList(),
                        );
                        Future.delayed(const Duration(milliseconds: 50), () {
                          _tagSearchController.clear();
                        });
                      }
                    },
                  ),
                ),
                if (selectedTags.isNotEmpty)
                  GestureDetector(
                    onTap: () {
                      tagProvider.clearSelection();
                      imageProvider.setTagFilter([]);
                      imageProvider.setHasTagsFilter(null);
                    },
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8.0),
                      child: Icon(
                        FluentIcons.clear,
                        size: 16,
                        color: hintColor,
                      ),
                    ),
                  ),
                const SizedBox(width: 8),
                GestureDetector(
                  onTap: () {
                    final isUntagged = imageProvider.hasTagsFilter == false;
                    imageProvider.setHasTagsFilter(isUntagged ? null : false);
                  },
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 4,
                    ),
                    decoration: BoxDecoration(
                      color: hasUntaggedFilter
                          ? (isDark
                                ? const Color(0x3DFFFFFF)
                                : const Color(0x1F000000))
                          : Colors.transparent,
                      border: Border.all(color: borderColor),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Row(
                      children: [
                        Icon(
                          hasUntaggedFilter
                              ? FluentIcons.tag
                              : FluentIcons.tag_unknown,
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
          ),
        ),
      ],
    );
  }

  void _showGalleryImageDetail(BuildContext context, ImageModel image) {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => ImageDetailScreen(image: image)));
  }

  Future<void> _confirmAndTriggerBatchAITags(
    BuildContext context,
    ImageListProvider imageProvider,
    TagProvider tagProvider,
  ) async {
    final totalByFilter = imageProvider.total > 0
        ? imageProvider.total
        : imageProvider.images.length;
    if (totalByFilter <= 0) {
      return;
    }

    final sortBy = _toApiSortBy(imageProvider.sortField);
    final sortDir = imageProvider.sortAsc ? 'asc' : 'desc';
    final selectedTagIDs = imageProvider.selectedTagIds.isNotEmpty
        ? List<int>.from(imageProvider.selectedTagIds)
        : null;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: const Text('批量触发 AI 标签'),
        content: Text('将按当前筛选条件为 $totalByFilter 张图片创建 AI 标签生成任务。'),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(dialogContext, false),
          ),
          FilledButton(
            child: const Text('确认'),
            onPressed: () => Navigator.pop(dialogContext, true),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) {
      return;
    }

    try {
      await tagProvider.tagService.batchTriggerAITags(
        tagIds: selectedTagIDs,
        hasTags: imageProvider.hasTagsFilter,
        sortBy: sortBy,
        sortDir: sortDir,
      );
      if (!context.mounted) {
        return;
      }
      await showDialog<void>(
        context: context,
        builder: (dialogContext) => ContentDialog(
          title: const Text('任务已提交'),
          content: Text('已按当前筛选条件提交 $totalByFilter 张图片的 AI 标签生成任务。'),
          actions: [
            FilledButton(
              child: const Text('知道了'),
              onPressed: () => Navigator.pop(dialogContext),
            ),
          ],
        ),
      );
    } catch (error) {
      if (!context.mounted) {
        return;
      }
      await showDialog<void>(
        context: context,
        builder: (dialogContext) => ContentDialog(
          title: const Text('提交失败'),
          content: Text('批量 AI 标签任务提交失败：$error'),
          actions: [
            FilledButton(
              child: const Text('关闭'),
              onPressed: () => Navigator.pop(dialogContext),
            ),
          ],
        ),
      );
    }
  }

  String _toApiSortBy(SortField sortField) {
    switch (sortField) {
      case SortField.createdAt:
        return 'created_at';
      case SortField.filename:
        return 'filename';
      case SortField.fileSize:
        return 'file_size';
    }
  }

  /// Dialog for batch-adding tags to selected images
  Future<void> _showBatchAddTagDialog(
    BuildContext context,
    SelectionProvider selection,
    TagProvider tagProvider,
    ImageListProvider imageProvider,
  ) async {
    final selectedIds = selection.selectedImageIds.toList();
    if (selectedIds.isEmpty) return;

    await tagProvider.loadTags();
    if (!context.mounted) return;

    final result = await showDialog<List<int>>(
      context: context,
      builder: (dialogContext) => _BatchAddTagDialog(
        tagProvider: tagProvider,
        selectedCount: selectedIds.length,
      ),
    );

    if (result == null || !context.mounted) return;

    try {
      final batchService = BatchService(
        baseUrl: context.read<ConfigProvider>().baseUrl,
      );
      final updated = await batchService.batchAddTags(selectedIds, result);
      if (!context.mounted) return;
      selection.clearSelection();
      imageProvider.loadImages(refresh: true);
      await showDialog<void>(
        context: context,
        builder: (ctx) => ContentDialog(
          title: const Text('批量添加成功'),
          content: Text('已为 $updated 张图片添加所选标签'),
          actions: [
            FilledButton(
              child: const Text('知道了'),
              onPressed: () => Navigator.pop(ctx),
            ),
          ],
        ),
      );
    } catch (e) {
      if (!context.mounted) return;
      await showDialog<void>(
        context: context,
        builder: (ctx) => ContentDialog(
          title: const Text('批量添加失败'),
          content: Text('批量添加标签失败：$e'),
          actions: [
            FilledButton(
              child: const Text('关闭'),
              onPressed: () => Navigator.pop(ctx),
            ),
          ],
        ),
      );
    }
  }
}

/// Fluent 风格搜索页面
/// 包含 AutoSuggestBox、搜索结果和排序选项
class FluentSearchPage extends StatefulWidget {
  const FluentSearchPage({super.key});

  @override
  State<FluentSearchPage> createState() => _FluentSearchPageState();
}

class _FluentSearchPageState extends State<FluentSearchPage> {
  final TextEditingController _searchController = TextEditingController();
  final FocusNode _searchFocusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SearchProvider>().loadSearchHistory();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    _searchFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<SearchProvider>(
      builder: (context, provider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('搜索'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                // Sort dropdown
                CommandBarButton(
                  icon: const Icon(FluentIcons.sort),
                  label: const Text('排序'),
                  onPressed: () {
                    _showSortOptions(context, provider);
                  },
                ),
              ],
            ),
          ),
          content: Column(
            children: [
              // Search bar
              _buildSearchBar(context, provider),
              // Content
              Expanded(
                child: FluentSearchContent(
                  onImageTap: null, // Removed single click routing
                  onImageDoubleTap: (image) =>
                      _showSearchImageDetail(context, image, provider),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildSearchBar(BuildContext context, SearchProvider provider) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(
            child: TextBox(
              controller: _searchController,
              focusNode: _searchFocusNode,
              placeholder: '搜索图片...',
              prefix: const Padding(
                padding: EdgeInsets.symmetric(horizontal: 8),
                child: Icon(FluentIcons.search, size: 16),
              ),
              suffix: _searchController.text.isNotEmpty
                  ? IconButton(
                      icon: const Icon(FluentIcons.clear, size: 16),
                      onPressed: () {
                        _searchController.clear();
                        provider.clearSearch();
                      },
                    )
                  : null,
              onSubmitted: (value) {
                if (value.isNotEmpty) {
                  provider.search(query: value);
                }
              },
            ),
          ),
          const SizedBox(width: 8),
          FilledButton(
            onPressed: () {
              if (_searchController.text.isNotEmpty) {
                provider.search(query: _searchController.text);
              }
            },
            child: const Text('搜索'),
          ),
        ],
      ),
    );
  }

  void _showSortOptions(BuildContext context, SearchProvider provider) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('排序选项'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              title: const Text('相关度'),
              onPressed: () {
                provider.setSort('relevance', provider.sortOrder);
                Navigator.pop(context);
              },
            ),
            ListTile(
              title: const Text('时间'),
              onPressed: () {
                provider.setSort('created_at', provider.sortOrder);
                Navigator.pop(context);
              },
            ),
            ListTile(
              title: const Text('文件名'),
              onPressed: () {
                provider.setSort('filename', provider.sortOrder);
                Navigator.pop(context);
              },
            ),
            ListTile(
              title: const Text('大小'),
              onPressed: () {
                provider.setSort('file_size', provider.sortOrder);
                Navigator.pop(context);
              },
            ),
          ],
        ),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }

  void _showSearchImageDetail(
    BuildContext context,
    ImageModel image,
    SearchProvider provider,
  ) {
    final selectedIndex = provider.indexOfResult(image.id);
    if (selectedIndex < 0) {
      return;
    }

    ViewerWindowService(
      adapter: DesktopMultiWindowViewerWindowAdapter(),
    ).openWindow(
      selectedFilename: image.filename,
      context: ViewerWindowContext.search(
        selectedIndex: selectedIndex,
        selectedImageId: image.id,
        snapshot: provider.viewerWindowSnapshot,
      ),
    );
  }
}

/// Fluent desktop tag-governance destination.
/// Hosts TagManagementWorkspace directly (retires legacy TagManagementScreen wrapper).
class FluentTagManagementPage extends StatelessWidget {
  const FluentTagManagementPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const TagManagementWorkspace();
  }
}

class FluentOperationsMonitoringPage extends StatelessWidget {
  const FluentOperationsMonitoringPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const OperationsMonitoringWorkspace();
  }
}

class FluentLogViewerPage extends StatelessWidget {
  const FluentLogViewerPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const LogViewerWorkspace();
  }
}

class FluentCollectionsPage extends StatelessWidget {
  final CollectionService? collectionService;

  const FluentCollectionsPage({super.key, this.collectionService});

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: const PageHeader(title: Text('收藏')),
      content: FluentCollectionsContent(
        collectionService: collectionService,
        onImageDoubleTap: (image) => _showImageDetail(context, image),
      ),
    );
  }

  void _showImageDetail(BuildContext context, ImageModel image) {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => ImageDetailScreen(image: image)));
  }
}

/// Dialog for selecting tags to batch-add to images
class _BatchAddTagDialog extends StatefulWidget {
  final TagProvider tagProvider;
  final int selectedCount;

  const _BatchAddTagDialog({
    required this.tagProvider,
    required this.selectedCount,
  });

  @override
  State<_BatchAddTagDialog> createState() => _BatchAddTagDialogState();
}

class _BatchAddTagDialogState extends State<_BatchAddTagDialog> {
  final Set<int> _selectedTagIds = {};
  final TextEditingController _searchController = TextEditingController();
  List<Tag> _filteredTags = [];

  @override
  void initState() {
    super.initState();
    _filteredTags = widget.tagProvider.allTags;
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _filterTags(String query) {
    if (query.isEmpty) {
      setState(() => _filteredTags = widget.tagProvider.allTags);
      return;
    }
    setState(() {
      _filteredTags = widget.tagProvider.allTags
          .where(
            (t) => t.preferredLabel.toLowerCase().contains(query.toLowerCase()),
          )
          .toList();
    });
  }

  @override
  Widget build(BuildContext context) {
    return ContentDialog(
      title: const Text('批量添加标签'),
      content: SizedBox(
        width: 400,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('将为 ${widget.selectedCount} 张图片添加以下标签：'),
            const SizedBox(height: 12),
            TextBox(
              controller: _searchController,
              placeholder: '搜索标签...',
              prefix: const Padding(
                padding: EdgeInsets.symmetric(horizontal: 8),
                child: Icon(FluentIcons.search, size: 14),
              ),
              onChanged: _filterTags,
            ),
            const SizedBox(height: 12),
            ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 250),
              child: SingleChildScrollView(
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: _filteredTags.map((tag) {
                    final isSelected = _selectedTagIds.contains(tag.id);
                    return ToggleButton(
                      checked: isSelected,
                      onChanged: (v) {
                        setState(() {
                          if (v) {
                            _selectedTagIds.add(tag.id);
                          } else {
                            _selectedTagIds.remove(tag.id);
                          }
                        });
                      },
                      child: Text(tag.preferredLabel),
                    );
                  }).toList(),
                ),
              ),
            ),
            if (_selectedTagIds.isEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  '请至少选择一个标签',
                  style: TextStyle(
                    color: FluentTheme.of(
                      context,
                    ).resources.textFillColorSecondary,
                    fontSize: 12,
                  ),
                ),
              ),
          ],
        ),
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.pop(context),
        ),
        FilledButton(
          onPressed: _selectedTagIds.isEmpty
              ? null
              : () => Navigator.pop(context, _selectedTagIds.toList()),
          child: const Text('确认添加'),
        ),
      ],
    );
  }
}
