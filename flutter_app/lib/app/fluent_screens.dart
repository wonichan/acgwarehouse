import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/material.dart' show MaterialPageRoute;
import 'package:provider/provider.dart';

import '../screens/duplicate_screen.dart';
import '../screens/image_detail_screen.dart';
import '../providers/image_provider.dart';
import '../providers/search_provider.dart';
import '../providers/tag_provider.dart';
import '../widgets/fluent_gallery_content.dart';
import '../widgets/gallery_filter_panel.dart';
import '../widgets/fluent_search_content.dart';
import '../widgets/monitoring/monitoring_workspace.dart';
import '../widgets/log_viewer/log_viewer_workspace.dart';
import '../widgets/tag_management/tag_management_workspace.dart';
import '../models/image.dart';
import '../models/viewer_window_context.dart';
import '../services/viewer_window_service.dart';

/// Fluent 风格图库页面
/// 包含 CommandBar 工具栏和图库内容
class FluentGalleryPage extends StatelessWidget {
  const FluentGalleryPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer2<ImageListProvider, TagProvider>(
      builder: (context, imageProvider, tagProvider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: _buildTitle(context, imageProvider, tagProvider),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                // View toggle
                CommandBarButton(
                  icon: Icon(
                    imageProvider.viewMode == ViewMode.grid
                        ? FluentIcons.bulleted_list_text
                        : FluentIcons.tiles,
                  ),
                  label: Text(
                    imageProvider.viewMode == ViewMode.grid ? '网格' : '瀑布流',
                  ),
                  onPressed: () {
                    imageProvider.setViewMode(
                      imageProvider.viewMode == ViewMode.grid
                          ? ViewMode.masonry
                          : ViewMode.grid,
                    );
                  },
                ),
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
    return Row(
      children: [
        Expanded(
          child: FluentGalleryContent(
            onImageTap: null, // Removed single click routing
            onImageDoubleTap: (image) =>
                _showGalleryImageDetail(context, image),
          ),
        ),
        const SizedBox(width: 1, child: ColoredBox(color: Color(0x22000000))),
        const GalleryFilterPanel(),
      ],
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

    if (selectedTags.isEmpty && !hasUntaggedFilter) {
      return const Text('图库');
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        const Text('图库'),
        const SizedBox(height: 4),
        Wrap(
          spacing: 6,
          runSpacing: 4,
          children: [
            if (hasUntaggedFilter)
              Button(
                child: const Text('未打标签 ×'),
                onPressed: () {
                  imageProvider.setHasTagsFilter(null);
                  tagProvider.clearSelection();
                },
              ),
            ...selectedTags.map(
              (tag) => Button(
                child: Text('${tag.preferredLabel} ×'),
                onPressed: () {
                  tagProvider.toggleTag(tag.id);
                  imageProvider.setTagFilter(
                    tagProvider.selectedTagIds.toList(),
                  );
                },
              ),
            ),
          ],
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

class FluentDuplicatePage extends StatelessWidget {
  const FluentDuplicatePage({super.key});

  @override
  Widget build(BuildContext context) {
    return const DuplicateScreen();
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
