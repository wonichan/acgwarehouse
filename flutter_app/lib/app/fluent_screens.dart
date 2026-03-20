import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/duplicate_screen.dart';
import '../screens/tag_management_screen.dart';
import '../providers/image_provider.dart';
import '../providers/search_provider.dart';
import '../providers/navigation_provider.dart';
import '../widgets/fluent_gallery_content.dart';
import '../widgets/fluent_search_content.dart';
import '../models/image.dart';

/// Fluent 风格图库页面
/// 包含 CommandBar 工具栏和图库内容
class FluentGalleryPage extends StatelessWidget {
  const FluentGalleryPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<ImageListProvider>(
      builder: (context, provider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('图库'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                // View toggle
                CommandBarButton(
                  icon: Icon(
                    provider.viewMode == ViewMode.grid
                        ? FluentIcons.bulleted_list_text
                        : FluentIcons.tiles,
                  ),
                  label: Text(provider.viewMode == ViewMode.grid ? '网格' : '瀑布流'),
                  onPressed: () {
                    provider.setViewMode(
                      provider.viewMode == ViewMode.grid
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
                    provider.loadImages(refresh: true);
                  },
                ),
                // Tag filter
                CommandBarButton(
                  icon: const Icon(FluentIcons.filter),
                  label: const Text('筛选'),
                  onPressed: () {
                    _showTagFilterDialog(context);
                  },
                ),
                // Tag management
                CommandBarButton(
                  icon: const Icon(FluentIcons.tag),
                  label: const Text('标签管理'),
                  onPressed: () {
                    context.read<NavigationProvider>().setSelectedIndex(3);
                  },
                ),
              ],
            ),
          ),
          content: FluentGalleryContent(
            onImageTap: (image) => _showImageDetail(context, image),
          ),
        );
      },
    );
  }

  void _showTagFilterDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('标签筛选'),
        content: const Text('标签筛选功能将在后续实现'),
        actions: [
          FilledButton(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }

  void _showImageDetail(BuildContext context, ImageModel image) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text(image.filename),
        content: Text('图片详情功能将在 08-03 中实现'),
        actions: [
          FilledButton(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
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
                  onImageTap: (image) => _showImageDetail(context, image),
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

  void _showImageDetail(BuildContext context, ImageModel image) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text(image.filename),
        content: Text('图片详情功能将在 08-03 中实现'),
        actions: [
          FilledButton(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }
}

/// Fluent 风格重复检测页面容器
class FluentDuplicatePage extends StatelessWidget {
  const FluentDuplicatePage({super.key});

  @override
  Widget build(BuildContext context) {
    return const DuplicateScreen();
  }
}

/// Fluent 风格标签管理页面容器
/// 包装共享的 TagManagementScreen Widget
class FluentTagManagementPage extends StatelessWidget {
  const FluentTagManagementPage({super.key});

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: const PageHeader(
        title: Text('标签管理'),
      ),
      content: const TagManagementScreen(),
    );
  }
}
