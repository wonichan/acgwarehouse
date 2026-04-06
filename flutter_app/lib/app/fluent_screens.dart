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
import '../models/viewer_session.dart';
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
                const CommandBarSeparator(),
                // Refresh button
                CommandBarButton(
                  icon: const Icon(FluentIcons.refresh),
                  label: const Text('刷新'),
                  onPressed: () {
                    imageProvider.loadImages(refresh: true);
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
            onImageDoubleTap: (image) => _showImageDetail(
              context,
              image,
              context.read<ImageListProvider>().images,
              ViewerSessionSource.gallery,
            ),
          ),
        ),
        const SizedBox(width: 1, child: ColoredBox(color: Color(0x22000000))),
        const GalleryFilterPanel(),
      ],
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

  void _showImageDetail(
    BuildContext context,
    ImageModel image,
    List<ImageModel> results,
    ViewerSessionSource source,
  ) {
    ViewerWindowService(
      adapter: DesktopMultiWindowViewerWindowAdapter(),
    ).openSession(
      ViewerSession.fromResultSet(
        source: source,
        images: results,
        selectedImageId: image.id,
      ),
    );
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
                  onImageDoubleTap: (image) => _showImageDetail(
                    context,
                    image,
                    provider.results,
                    ViewerSessionSource.search,
                  ),
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

  void _showImageDetail(
    BuildContext context,
    ImageModel image,
    List<ImageModel> results,
    ViewerSessionSource source,
  ) {
    ViewerWindowService(
      adapter: DesktopMultiWindowViewerWindowAdapter(),
    ).openSession(
      ViewerSession.fromResultSet(
        source: source,
        images: results,
        selectedImageId: image.id,
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
