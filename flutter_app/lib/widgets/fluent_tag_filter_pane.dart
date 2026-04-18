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
  final Set<int> _expandedParents = {};

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
    super.dispose();
  }

  bool _isTagSelected(int tagId, String? level, int? parentId) {
    if (level == 'root' || level == 'parent') {
      return _draftFilter.subtreeRootTagIds.contains(tagId);
    }
    return _draftFilter.exactTagIds.contains(tagId);
  }

  void _toggleTag(int tagId, String? level, int? parentId) {
    setState(() {
      if (level == 'root' || level == 'parent') {
        final nextIds = _draftFilter.subtreeRootTagIds.toSet();
        if (nextIds.contains(tagId)) {
          nextIds.remove(tagId);
        } else {
          nextIds.add(tagId);
        }
        _draftFilter = _draftFilter
            .copyWith(subtreeRootTagIds: nextIds, hasTags: null)
            .normalized();
      } else {
        final nextIds = _draftFilter.exactTagIds.toSet();
        if (nextIds.contains(tagId)) {
          nextIds.remove(tagId);
        } else {
          nextIds.add(tagId);
        }
        _draftFilter = _draftFilter
            .copyWith(exactTagIds: nextIds, hasTags: null)
            .normalized();
      }
    });
  }

  void _loadChildrenIfNeeded(int parentId) {
    final provider = context.read<TagProvider>();
    if (!provider.treeChildrenByParent.containsKey(parentId)) {
      provider.loadTreeChildren(parentId);
    }
  }

  int get _selectedCount =>
      _draftFilter.exactTagIds.length +
      _draftFilter.subtreeRootTagIds.length;

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

  Widget _buildTagRow(TagBrowseNode node, {double indent = 0}) {
    final theme = FluentTheme.of(context);
    final isSelected = _isTagSelected(node.id, node.level, node.parentId);
    final isExpandable = node.hasChildren;

    return Padding(
      padding: EdgeInsets.only(left: indent),
      child: Row(
        children: [
          if (isExpandable)
            GestureDetector(
              onTap: () {
                setState(() {
                  if (_expandedParents.contains(node.id)) {
                    _expandedParents.remove(node.id);
                  } else {
                    _expandedParents.add(node.id);
                    _loadChildrenIfNeeded(node.id);
                  }
                });
              },
              child: Padding(
                padding: const EdgeInsets.all(4),
                child: Icon(
                  _expandedParents.contains(node.id)
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
            onChanged: (_) => _toggleTag(node.id, node.level, node.parentId),
          ),
          const SizedBox(width: 6),
          Expanded(child: Text(node.preferredLabel)),
          _buildLevelBadge(node.level, theme),
        ],
      ),
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
        for (final root in roots) ...[
          _buildTagRow(root),
          if (_expandedParents.contains(root.id)) ...[
            Builder(builder: (context) {
              final children = provider.childrenOf(root.id);
              if (children.isEmpty && !provider.treeChildrenByParent.containsKey(root.id)) {
                return const Padding(
                  padding: EdgeInsets.only(left: 40),
                  child: ProgressRing(),
                );
              }
              return Column(
                children: [
                  for (final child in children)
                    _buildTagRow(child, indent: 20),
                ],
              );
            }),
          ],
        ],
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
        for (final orphan in orphans) _buildTagRow(orphan),
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

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);

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
                    onPressed: () {
                      _searchController.clear();
                      setState(() {});
                    },
                  )
                : null,
            onChanged: (query) {
              setState(() {});
            },
          ),
          const SizedBox(height: 12),

          _buildUntaggedToggle(context),
          const SizedBox(height: 12),

          _buildPendingTagsToggle(context),
          const SizedBox(height: 12),

          if (!_draftFilter.isEmpty)
            Row(
              children: [
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
                const SizedBox(width: 8),
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
                if (provider.treeBrowseError != null) {
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

                final roots = provider.treeRoots;
                final orphans = provider.orphanTags;

                if (roots.isEmpty && orphans.isEmpty && !provider.isLoadingTreeRoots && !provider.isLoadingOrphans) {
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
                  ? GalleryFilterState(hasTags: false).normalized()
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
