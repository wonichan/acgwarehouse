import 'dart:async';

import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../models/tag.dart';
import '../models/gallery_filter_state.dart';
import '../providers/tag_provider.dart';

class FluentTagFilterPane extends StatefulWidget {
  final GalleryFilterState initialFilter;
  final ValueChanged<GalleryFilterState> onApplyFilter;

  const FluentTagFilterPane({
    super.key,
    required this.initialFilter,
    required this.onApplyFilter,
  });

  @override
  State<FluentTagFilterPane> createState() => _FluentTagFilterPaneState();
}

class _FluentTagFilterPaneState extends State<FluentTagFilterPane> {
  final TextEditingController _searchController = TextEditingController();
  late GalleryFilterState _draftFilter;
  final Set<int> _expandedNodes = {};

  // Search state
  List<Tag> _searchResults = [];
  bool _isSearching = false;
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _draftFilter = widget.initialFilter.normalized();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final provider = context.read<TagProvider>();
      provider.loadTreeRoots();
      provider.loadOrphanTags();
    });
  }

  @override
  void didUpdateWidget(covariant FluentTagFilterPane oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialFilter != widget.initialFilter) {
      _draftFilter = widget.initialFilter.normalized();
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    _debounce?.cancel();
    super.dispose();
  }

  // ---- Search ----

  void _onSearchChanged(String query) {
    _debounce?.cancel();
    if (query.trim().isEmpty) {
      setState(() {
        _searchResults = [];
        _isSearching = false;
      });
      return;
    }
    setState(() {
      _isSearching = true;
    });
    _debounce = Timer(const Duration(milliseconds: 300), () {
      _performSearch(query.trim());
    });
  }

  Future<void> _performSearch(String query) async {
    final provider = context.read<TagProvider>();
    try {
      final results = await provider.tagService.searchTags(query);
      if (!mounted) return;
      // Only update if the query hasn't changed while we were waiting
      if (_searchController.text.trim().toLowerCase() ==
          query.toLowerCase()) {
        setState(() {
          _searchResults = results;
          _isSearching = false;
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _searchResults = [];
        _isSearching = false;
      });
    }
  }

  void _clearSearch() {
    _debounce?.cancel();
    _searchController.clear();
    setState(() {
      _searchResults = [];
      _isSearching = false;
    });
  }

  bool get _hasSearchQuery => _searchController.text.trim().isNotEmpty;

  // ---- Selection ----

  bool _isTagSelected(int tagId, String? level) {
    if (level == 'root' || level == 'parent') {
      return _draftFilter.subtreeRootTagIds.contains(tagId);
    }
    return _draftFilter.exactTagIds.contains(tagId);
  }

  void _toggleTag(int tagId, String? level) {
    setState(() {
      if (level == 'root' || level == 'parent') {
        final nextIds = _draftFilter.subtreeRootTagIds.toSet();
        if (nextIds.contains(tagId)) {
          nextIds.remove(tagId);
        } else {
          nextIds.add(tagId);
        }
        _draftFilter = _draftFilter
            .copyWith(subtreeRootTagIds: nextIds)
            .normalized();
      } else {
        final nextIds = _draftFilter.exactTagIds.toSet();
        if (nextIds.contains(tagId)) {
          nextIds.remove(tagId);
        } else {
          nextIds.add(tagId);
        }
        _draftFilter = _draftFilter
            .copyWith(exactTagIds: nextIds)
            .normalized();
      }
    });
  }

  // ---- Tree expansion ----

  void _toggleExpand(int nodeId, TagProvider provider) {
    setState(() {
      if (_expandedNodes.contains(nodeId)) {
        _expandedNodes.remove(nodeId);
      } else {
        _expandedNodes.add(nodeId);
        if (!provider.treeChildrenByParent.containsKey(nodeId)) {
          provider.loadTreeChildren(nodeId);
        }
      }
    });
  }

  int get _selectedCount =>
      _draftFilter.exactTagIds.length + _draftFilter.subtreeRootTagIds.length;

  // ---- Widgets ----

  Widget _buildLevelBadge(String? level, FluentThemeData theme) {
    if (level == null || level.isEmpty) return const SizedBox.shrink();

    final String label = switch (level) {
      'root' => 'R',
      'parent' => 'P',
      _ => 'C',
    };
    final Color accent = switch (level) {
      'root' => theme.accentColor,
      'parent' => Colors.orange,
      _ => theme.resources.textFillColorSecondary,
    };

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 6),
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
      decoration: BoxDecoration(
        color: accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: accent.withValues(alpha: 0.35), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 10,
          fontWeight: FontWeight.w600,
          color: accent,
        ),
      ),
    );
  }

  /// Recursively builds a tag row and its expanded children (tree view).
  Widget _buildNodeTree(TagBrowseNode node, TagProvider provider,
      {double indent = 0}) {
    final theme = FluentTheme.of(context);
    final isSelected = _isTagSelected(node.id, node.level);
    final expanded = _expandedNodes.contains(node.id);
    final children = provider.childrenOf(node.id);
    final childrenLoaded = provider.treeChildrenByParent.containsKey(node.id);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: EdgeInsets.only(left: indent),
          child: Row(
            children: [
              if (node.hasChildren)
                GestureDetector(
                  onTap: () => _toggleExpand(node.id, provider),
                  child: Padding(
                    padding: const EdgeInsets.all(4),
                    child: Icon(
                      expanded
                          ? FluentIcons.chevron_down
                          : FluentIcons.chevron_right,
                      size: 12,
                      color: theme.resources.textFillColorSecondary,
                    ),
                  ),
                )
              else
                const SizedBox(width: 20),
              Checkbox(
                checked: isSelected,
                onChanged: (_) => _toggleTag(node.id, node.level),
              ),
              const SizedBox(width: 6),
              Expanded(child: Text(node.preferredLabel)),
              _buildLevelBadge(node.level, theme),
            ],
          ),
        ),
        if (expanded) ...[
          if (!childrenLoaded && node.hasChildren)
            const Padding(
              padding: EdgeInsets.only(left: 40),
              child: ProgressRing(),
            )
          else
            for (final child in children)
              _buildNodeTree(child, provider, indent: indent + 20),
        ],
      ],
    );
  }

  /// Builds a single search result row from a Tag object.
  Widget _buildSearchResultRow(Tag tag) {
    final theme = FluentTheme.of(context);
    final isSelected = _isTagSelected(tag.id, tag.level);

    return Row(
      children: [
        Checkbox(
          checked: isSelected,
          onChanged: (_) => _toggleTag(tag.id, tag.level),
        ),
        const SizedBox(width: 6),
        Expanded(child: Text(tag.preferredLabel)),
        _buildLevelBadge(tag.level, theme),
      ],
    );
  }

  Widget _buildTreeSection(TagProvider provider) {
    final roots = provider.treeRoots;
    if (provider.isLoadingTreeRoots) {
      return const Center(child: ProgressRing());
    }
    if (roots.isEmpty) return const SizedBox.shrink();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final root in roots) _buildNodeTree(root, provider),
      ],
    );
  }

  Widget _buildOrphanSection(TagProvider provider) {
    final orphans = provider.orphanTags;
    if (provider.isLoadingOrphans && orphans.isEmpty) {
      return const Center(child: ProgressRing());
    }
    if (orphans.isEmpty) return const SizedBox.shrink();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Text(
            '未分类标签',
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: FluentTheme.of(context).resources.textFillColorSecondary,
            ),
          ),
        ),
        for (final orphan in orphans) _buildNodeTree(orphan, provider),
        if (provider.hasMoreOrphans)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: HyperlinkButton(
              onPressed: () => provider.loadOrphanTags(
                offset: provider.orphanTags.length,
              ),
              child: const Text('加载更多...'),
            ),
          ),
      ],
    );
  }

  Widget _buildSearchResults() {
    if (_isSearching) {
      return const Center(child: ProgressRing());
    }
    if (_searchResults.isEmpty) {
      return Center(
        child: Text(
          '未找到匹配标签',
          style: TextStyle(
            color: FluentTheme.of(context).resources.textFillColorSecondary,
          ),
        ),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: Text(
            '搜索结果 (${_searchResults.length})',
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color:
                  FluentTheme.of(context).resources.textFillColorSecondary,
            ),
          ),
        ),
        for (final tag in _searchResults) _buildSearchResultRow(tag),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final canApplyCurrentDraft =
        !_draftFilter.isEmpty || !widget.initialFilter.isEmpty;

    return Padding(
      padding: const EdgeInsets.all(12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text('标签筛选', style: theme.typography.title),
              const Spacer(),
              if (_selectedCount > 0)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: theme.accentColor.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    '$_selectedCount 个',
                    style: TextStyle(fontSize: 12, color: theme.accentColor),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),

          TextBox(
            controller: _searchController,
            placeholder: '搜索标签...',
            prefix: const Padding(
              padding: EdgeInsets.symmetric(horizontal: 8),
              child: Icon(FluentIcons.search, size: 14),
            ),
            suffix: _searchController.text.isNotEmpty
                ? IconButton(
                    icon: const Icon(FluentIcons.clear, size: 14),
                    onPressed: _clearSearch,
                  )
                : null,
            onChanged: _onSearchChanged,
          ),
          const SizedBox(height: 12),

          _buildUntaggedToggle(context),
          const SizedBox(height: 12),

          _buildPendingTagsToggle(context),
          const SizedBox(height: 12),

          if (canApplyCurrentDraft)
            Row(
              children: [
                if (!_draftFilter.isEmpty)
                  Button(
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(FluentIcons.clear, size: 14),
                        const SizedBox(width: 6),
                        const Text('清空筛选'),
                      ],
                    ),
                    onPressed: () {
                      setState(() {
                        _draftFilter = _draftFilter.clear();
                      });
                    },
                  ),
                if (!_draftFilter.isEmpty) const SizedBox(width: 8),
                FilledButton(
                  child: const Text('应用筛选'),
                  onPressed: () {
                    widget.onApplyFilter(_draftFilter.normalized());
                  },
                ),
              ],
            ),
          const SizedBox(height: 12),
          const Divider(),
          const SizedBox(height: 8),

          Expanded(
            child: Consumer<TagProvider>(
              builder: (context, provider, _) {
                if (provider.treeBrowseError != null && !_hasSearchQuery) {
                  return Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          FluentIcons.error,
                          size: 32,
                          color: theme.resources.systemFillColorCritical,
                        ),
                        const SizedBox(height: 8),
                        Text('加载失败: ${provider.treeBrowseError}'),
                        const SizedBox(height: 8),
                        HyperlinkButton(
                          onPressed: () {
                            provider.loadTreeRoots();
                            provider.loadOrphanTags();
                          },
                          child: const Text('重试'),
                        ),
                      ],
                    ),
                  );
                }

                // When searching, show search results from backend API
                if (_hasSearchQuery) {
                  return ListView(
                    children: [_buildSearchResults()],
                  );
                }

                // Otherwise show the tree + orphan sections
                final roots = provider.treeRoots;
                final orphans = provider.orphanTags;

                if (roots.isEmpty &&
                    orphans.isEmpty &&
                    !provider.isLoadingTreeRoots &&
                    !provider.isLoadingOrphans) {
                  return Center(
                    child: Text(
                      '暂无标签',
                      style: TextStyle(
                        color: theme.resources.textFillColorSecondary,
                      ),
                    ),
                  );
                }

                return ListView(
                  children: [
                    _buildTreeSection(provider),
                    if (roots.isNotEmpty && orphans.isNotEmpty)
                      const Divider(),
                    _buildOrphanSection(provider),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildUntaggedToggle(BuildContext context) {
    return Row(
      children: [
        ToggleSwitch(
          checked: _draftFilter.hasTags == false,
          onChanged: (value) {
            setState(() {
              _draftFilter = value
                  ? _draftFilter
                        .copyWith(hasTags: false, hasPendingTags: null)
                        .normalized()
                  : _draftFilter.copyWith(hasTags: null).normalized();
            });
          },
        ),
        const SizedBox(width: 8),
        const Icon(FluentIcons.tag_unknown, size: 16),
        const SizedBox(width: 6),
        const Expanded(child: Text('未打标签', style: TextStyle(fontSize: 13))),
      ],
    );
  }

  Widget _buildPendingTagsToggle(BuildContext context) {
    return Row(
      children: [
        ToggleSwitch(
          checked: _draftFilter.hasPendingTags == true,
          onChanged: (value) {
            setState(() {
              _draftFilter = _draftFilter
                  .copyWith(
                    hasPendingTags: value ? true : null,
                    hasTags: value ? null : _draftFilter.hasTags,
                  )
                  .normalized();
            });
          },
        ),
        const SizedBox(width: 8),
        const Icon(FluentIcons.tag_unknown, size: 16),
        const SizedBox(width: 6),
        const Expanded(child: Text('标签未确认', style: TextStyle(fontSize: 13))),
      ],
    );
  }
}
