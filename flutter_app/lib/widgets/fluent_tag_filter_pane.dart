import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/tag_provider.dart';
import '../providers/image_provider.dart';

/// Fluent 风格的标签筛选面板
/// 从侧边弹出，提供标签多选筛选功能
class FluentTagFilterPane extends StatefulWidget {
  final bool? hasTagsFilter;
  final Function(bool? hasTags)? onHasTagsChanged;
  final bool? hasPendingTagsFilter;
  final Function(bool? hasPendingTags)? onHasPendingTagsChanged;

  const FluentTagFilterPane({
    super.key,
    this.hasTagsFilter,
    this.onHasTagsChanged,
    this.hasPendingTagsFilter,
    this.onHasPendingTagsChanged,
  });

  @override
  State<FluentTagFilterPane> createState() => _FluentTagFilterPaneState();
}

class _FluentTagFilterPaneState extends State<FluentTagFilterPane> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TagProvider>().loadTagTree();
      context.read<TagProvider>().loadTags();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

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

  List<TreeViewItem> _buildTreeNodes(
    List<dynamic> nodes,
    TagProvider provider,
  ) {
    final theme = FluentTheme.of(context);
    return nodes.map((node) {
      final tagId = (node['tag_id'] ?? node['id']) as int;
      final isSelected = provider.selectedTagIds.contains(tagId);
      final children = node['children'] as List<dynamic>? ?? [];
      final usageCount = node['tree_usage_count'] ?? node['usage_count'] ?? 0;
      final level = node['level'] as String?;

      return TreeViewItem(
        content: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Checkbox(
              checked: isSelected,
              onChanged: (checked) {
                provider.toggleTag(tagId);
                context.read<ImageListProvider>().setTagFilter(
                  provider.selectedTagIds.toList(),
                );
              },
            ),
            const SizedBox(width: 8),
            Text(node['preferred_label'] as String? ?? ''),
            _buildLevelBadge(level, theme),
            const SizedBox(width: 2),
            Text(
              '$usageCount',
              style: TextStyle(
                fontSize: 11,
                color: theme.resources.textFillColorSecondary,
              ),
            ),
          ],
        ),
        value: tagId,
        children: children.isNotEmpty
            ? _buildTreeNodes(children, provider)
            : [],
      );
    }).toList();
  }

  List<dynamic> _filterTreeNodes(List<dynamic> nodes, String query) {
    final normalized = query.trim().toLowerCase();
    if (normalized.isEmpty) {
      return nodes;
    }

    final filtered = <dynamic>[];
    for (final rawNode in nodes) {
      final node = Map<String, dynamic>.from(rawNode as Map);
      final label = (node['preferred_label'] as String? ?? '').toLowerCase();
      final children = _filterTreeNodes(
        node['children'] as List<dynamic>? ?? const [],
        query,
      );
      if (label.contains(normalized) || children.isNotEmpty) {
        node['children'] = children;
        filtered.add(node);
      }
    }
    return filtered;
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Padding(
      padding: const EdgeInsets.all(12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 标题行
          Row(
            children: [
              Text('标签筛选', style: theme.typography.title),
              const Spacer(),
              Consumer<TagProvider>(
                builder: (context, provider, _) {
                  final count = provider.selectedTagIds.length;
                  if (count == 0) return const SizedBox.shrink();
                  return Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: theme.accentColor.withValues(alpha: 0.2),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      '$count 个',
                      style: TextStyle(fontSize: 12, color: theme.accentColor),
                    ),
                  );
                },
              ),
            ],
          ),
          const SizedBox(height: 12),

          // 搜索框
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

          // 未打标签开关
          _buildUntaggedToggle(context, isDark),
          const SizedBox(height: 12),

          // 标签未确认开关
          _buildPendingTagsToggle(context, isDark),
          const SizedBox(height: 12),

          // 清空按钮
          Consumer<TagProvider>(
            builder: (context, provider, _) {
              if (provider.selectedTagIds.isEmpty &&
                  widget.hasTagsFilter != false &&
                  widget.hasPendingTagsFilter != true) {
                return const SizedBox.shrink();
              }
              return Button(
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(FluentIcons.clear, size: 14),
                    const SizedBox(width: 6),
                    const Text('清空筛选'),
                  ],
                ),
                onPressed: () {
                  provider.clearSelection();
                  context.read<ImageListProvider>().setTagFilter([]);
                  widget.onHasTagsChanged?.call(null);
                  widget.onHasPendingTagsChanged?.call(null);
                },
              );
            },
          ),
          const SizedBox(height: 12),
          const Divider(),
          const SizedBox(height: 8),

          // 标签列表/树
          Expanded(
            child: Consumer<TagProvider>(
              builder: (context, provider, _) {
                if (provider.isLoading) {
                  return const Center(child: ProgressRing());
                }
                if (provider.error != null) {
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
                        Text('加载失败: ${provider.error}'),
                      ],
                    ),
                  );
                }

                final treeData =
                    provider.tagTree?['tree'] as List<dynamic>? ?? [];
                final visibleNodes = _filterTreeNodes(
                  treeData,
                  _searchController.text,
                );
                if (visibleNodes.isEmpty) {
                  return Center(
                    child: Text(
                      '暂无标签',
                      style: TextStyle(
                        color: theme.resources.textFillColorSecondary,
                      ),
                    ),
                  );
                }

                return TreeView(
                  items: _buildTreeNodes(visibleNodes, provider),
                  selectionMode: TreeViewSelectionMode.multiple,
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildUntaggedToggle(BuildContext context, bool isDark) {
    return Row(
      children: [
        ToggleSwitch(
          checked: widget.hasTagsFilter == false,
          onChanged: (value) {
            if (value) {
              context.read<TagProvider>().clearSelection();
            }
            widget.onHasTagsChanged?.call(value ? false : null);
          },
        ),
        const SizedBox(width: 8),
        const Icon(FluentIcons.tag_unknown, size: 16),
        const SizedBox(width: 6),
        const Expanded(child: Text('未打标签', style: TextStyle(fontSize: 13))),
      ],
    );
  }

  Widget _buildPendingTagsToggle(BuildContext context, bool isDark) {
    return Row(
      children: [
        ToggleSwitch(
          checked: widget.hasPendingTagsFilter == true,
          onChanged: (value) {
            widget.onHasPendingTagsChanged?.call(value ? true : null);
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
