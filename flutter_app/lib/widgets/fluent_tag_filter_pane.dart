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
      context.read<TagProvider>().loadTags();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
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
                      context.read<TagProvider>().searchTags('');
                    },
                  )
                : null,
            onChanged: (query) {
              context.read<TagProvider>().searchTags(query);
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

          // 标签列表
          Expanded(
            child: Consumer<TagProvider>(
              builder: (context, provider, _) {
                if (provider.isLoading && provider.allTags.isEmpty) {
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
                if (provider.filteredTags.isEmpty) {
                  return Center(
                    child: Text(
                      '暂无标签',
                      style: TextStyle(
                        color: theme.resources.textFillColorSecondary,
                      ),
                    ),
                  );
                }

                return ListView.builder(
                  itemCount: provider.filteredTags.length,
                  itemBuilder: (itemContext, index) {
                    final tag = provider.filteredTags[index];
                    final isSelected = provider.selectedTagIds.contains(tag.id);

                    return ToggleButton(
                      checked: isSelected,
                      onChanged: (checked) {
                        provider.toggleTag(tag.id);
                        context.read<ImageListProvider>().setTagFilter(
                          provider.selectedTagIds.toList(),
                        );
                      },
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Expanded(
                            child: Text(
                              tag.preferredLabel,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          const SizedBox(width: 6),
                          Text(
                            '${tag.usageCount}',
                            style: TextStyle(
                              fontSize: 11,
                              color: theme.resources.textFillColorSecondary,
                            ),
                          ),
                        ],
                      ),
                    );
                  },
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
