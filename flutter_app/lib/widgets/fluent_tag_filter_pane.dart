import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../models/gallery_filter_state.dart';
import '../providers/tag_provider.dart';

/// Fluent 风格的标签筛选面板
/// 从侧边弹出，提供标签多选筛选功能
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

  @override
  void initState() {
    super.initState();
    _draftFilter = widget.initialFilter.normalized();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TagProvider>().loadTagTree();
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

  List<TreeViewItem> _buildTreeNodes(List<dynamic> nodes) {
    final theme = FluentTheme.of(context);
    return nodes.map((node) {
      final tagId = (node['tag_id'] ?? node['id']) as int;
      final isSelected = _draftFilter.exactTagIds.contains(tagId);
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
                final nextIds = _draftFilter.exactTagIds.toSet();
                if (nextIds.contains(tagId)) {
                  nextIds.remove(tagId);
                } else {
                  nextIds.add(tagId);
                }

                setState(() {
                  _draftFilter = _draftFilter
                      .copyWith(exactTagIds: nextIds, hasTags: null)
                      .normalized();
                });
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
        children: children.isNotEmpty ? _buildTreeNodes(children) : [],
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
                  final count = _draftFilter.exactTagIds.length;
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
              if (_draftFilter.isEmpty) {
                return const SizedBox.shrink();
              }
              return Row(
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
                  items: _buildTreeNodes(visibleNodes),
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

  Widget _buildPendingTagsToggle(BuildContext context, bool isDark) {
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
