import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/material.dart' show MaterialPageRoute;
import 'package:provider/provider.dart';

import '../screens/duplicate_screen.dart';
import '../screens/tag_management_screen.dart';
import '../screens/image_detail_screen.dart';
import '../providers/image_provider.dart';
import '../providers/search_provider.dart';
import '../providers/navigation_provider.dart';
import '../providers/tag_provider.dart';
import '../widgets/fluent_gallery_content.dart';
import '../widgets/fluent_search_content.dart';
import '../models/image.dart';
import '../models/tag.dart';

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
                  label: Text(imageProvider.viewMode == ViewMode.grid ? '网格' : '瀑布流'),
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

  Widget _buildTitle(BuildContext context, ImageListProvider imageProvider, TagProvider tagProvider) {
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
                },
              ),
            ...selectedTags.map((tag) => Button(
              child: Text('${tag.preferredLabel} ×'),
              onPressed: () {
                tagProvider.toggleTag(tag.id);
                imageProvider.setTagFilter(tagProvider.selectedTagIds.toList());
              },
            )),
          ],
        ),
      ],
    );
  }

  void _showTagFilterDialog(BuildContext context) {
    final imageProvider = context.read<ImageListProvider>();
    final tagProvider = context.read<TagProvider>();
    
    showDialog(
      context: context,
      builder: (dialogContext) => _TagFilterDialogContent(
        imageProvider: imageProvider,
        tagProvider: tagProvider,
      ),
    );
  }

  void _showImageDetail(BuildContext context, ImageModel image) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (context) => ImageDetailScreen(image: image),
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
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (context) => ImageDetailScreen(image: image),
      ),
    );
  }
}

/// Fluent UI Tag Filter Dialog Content
/// A ContentDialog-based tag filter UI integrated with TagProvider and ImageListProvider
class _TagFilterDialogContent extends StatefulWidget {
  final ImageListProvider imageProvider;
  final TagProvider tagProvider;

  const _TagFilterDialogContent({
    required this.imageProvider,
    required this.tagProvider,
  });

  @override
  State<_TagFilterDialogContent> createState() => _TagFilterDialogContentState();
}

class _TagFilterDialogContentState extends State<_TagFilterDialogContent> {
  final TextEditingController _searchController = TextEditingController();
  late List<int> _selectedTagIds;
  bool? _hasTagsFilter;

  @override
  void initState() {
    super.initState();
    // Initialize local state from providers
    _selectedTagIds = widget.tagProvider.selectedTagIds.toList();
    _hasTagsFilter = widget.imageProvider.hasTagsFilter;
    
    // Load tags if not already loaded
    WidgetsBinding.instance.addPostFrameCallback((_) {
      widget.tagProvider.loadTags();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _applyFilters() {
    // 同步 TagProvider 的选择状态
    widget.tagProvider.clearSelection();
    for (final tagId in _selectedTagIds) {
      widget.tagProvider.toggleTag(tagId);
    }
    // 互斥逻辑：根据用户选择只调用一个筛选方法
    if (_hasTagsFilter != null) {
      // 用户选择了"显示未打标签的图片"
      widget.imageProvider.setHasTagsFilter(_hasTagsFilter);
    } else {
      // 用户选择了标签筛选或清空
      widget.imageProvider.setTagFilter(_selectedTagIds);
    }
    Navigator.pop(context);
  }

  void _clearSelection() {
    setState(() {
      _selectedTagIds = [];
      _hasTagsFilter = null;
    });
    widget.tagProvider.clearSelection();
  }

  @override
  Widget build(BuildContext context) {
    return ContentDialog(
      title: const Text('标签筛选'),
      content: SizedBox(
        width: 400,
        height: 450,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Search box
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: TextBox(
                controller: _searchController,
                placeholder: '搜索标签...',
                prefix: const Padding(
                  padding: EdgeInsets.only(left: 8),
                  child: Icon(FluentIcons.search, size: 16),
                ),
                onChanged: (value) {
                  widget.tagProvider.searchTags(value);
                },
              ),
            ),
            // Show untagged toggle
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Row(
                children: [
                  ToggleSwitch(
                    content: const Text('显示未打标签的图片'),
                    checked: _hasTagsFilter == false,
                    onChanged: (value) {
                      setState(() {
                        _hasTagsFilter = value ? false : null;
                      });
                    },
                  ),
                ],
              ),
            ),
            // Selected count and clear button
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    '已选择 ${_selectedTagIds.length} 个标签',
                    style: const TextStyle(
                      fontSize: 13,
                      color: Colors.grey,
                    ),
                  ),
                  Button(
                    onPressed: _clearSelection,
                    child: const Text('清空选择'),
                  ),
                ],
              ),
            ),
            const Divider(),
            // Tag list
            Expanded(
              child: Consumer<TagProvider>(
                builder: (context, tagProvider, _) {
                  if (tagProvider.isLoading) {
                    return const Center(
                      child: ProgressRing(),
                    );
                  }
                  if (tagProvider.error != null) {
                    return Center(
                      child: Text('加载失败: ${tagProvider.error}'),
                    );
                  }
                  if (tagProvider.filteredTags.isEmpty) {
                    return const Center(
                      child: Text('暂无标签'),
                    );
                  }
                  return ListView.builder(
                    itemCount: tagProvider.filteredTags.length,
                    itemBuilder: (context, index) {
                      final tag = tagProvider.filteredTags[index];
                      final isSelected = _selectedTagIds.contains(tag.id);
                      return Padding(
                        padding: const EdgeInsets.symmetric(vertical: 2),
                        child: Checkbox(
                          content: Row(
                            children: [
                              Text(tag.preferredLabel),
                              const SizedBox(width: 8),
                              Text(
                                '${tag.usageCount} 张',
                                style: const TextStyle(
                                  fontSize: 12,
                                  color: Colors.grey,
                                ),
                              ),
                            ],
                          ),
                          checked: isSelected,
                          onChanged: (checked) {
                            setState(() {
                              if (checked == true) {
                                _selectedTagIds.add(tag.id);
                              } else {
                                _selectedTagIds.remove(tag.id);
                              }
                            });
                          },
                        ),
                      );
                    },
                  );
                },
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
          onPressed: _applyFilters,
          child: const Text('应用筛选'),
        ),
      ],
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
