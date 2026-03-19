import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/tag_provider.dart';

class TagFilterDrawer extends StatefulWidget {
  final Function(List<int> tagIds)? onFilterChanged;
  final Function(bool? hasTags)? onHasTagsChanged;
  final bool? hasTagsFilter;

  const TagFilterDrawer({
    super.key,
    this.onFilterChanged,
    this.onHasTagsChanged,
    this.hasTagsFilter,
  });

  @override
  State<TagFilterDrawer> createState() => _TagFilterDrawerState();
}

class _TagFilterDrawerState extends State<TagFilterDrawer> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<TagProvider>().loadTags();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // 标题
        Container(
          color: Theme.of(context).primaryColor,
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('标签筛选',
                  style: Theme.of(context)
                      .textTheme
                      .titleLarge
                      ?.copyWith(color: Colors.white)),
              const SizedBox(height: 8),
              Consumer<TagProvider>(
                builder: (context, provider, _) {
                  final count = provider.selectedTagIds.length;
                  return Text('已选择 $count 个标签',
                      style: const TextStyle(color: Colors.white70));
                },
              ),
            ],
          ),
        ),
        // 搜索框
        Padding(
          padding: const EdgeInsets.all(8.0),
          child: TextField(
            controller: _searchController,
            decoration: const InputDecoration(
              hintText: '搜索标签...',
              prefixIcon: Icon(Icons.search),
              border: OutlineInputBorder(),
              contentPadding:
                  EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
            onChanged: (query) {
              context.read<TagProvider>().searchTags(query);
            },
          ),
        ),
        // 未打标签筛选开关
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8.0, vertical: 4.0),
          child: SwitchListTile(
            title: const Text('未打标签'),
            subtitle: const Text('显示没有标签的图片'),
            value: widget.hasTagsFilter == false,
            onChanged: (value) {
              widget.onHasTagsChanged?.call(value ? false : null);
            },
            secondary: const Icon(Icons.label_off_outlined),
          ),
        ),
        // 清空按钮
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8.0),
          child: Row(
            children: [
              TextButton.icon(
                icon: const Icon(Icons.clear_all),
                label: const Text('清空选择'),
                onPressed: () {
                  context.read<TagProvider>().clearSelection();
                  widget.onFilterChanged?.call([]);
                  widget.onHasTagsChanged?.call(null);
                },
              ),
            ],
          ),
        ),
        const Divider(),
        // 标签列表
        Expanded(
          child: Consumer<TagProvider>(
            builder: (context, provider, _) {
              if (provider.isLoading) {
                return const Center(child: CircularProgressIndicator());
              }
              if (provider.error != null) {
                return Center(child: Text('加载失败: ${provider.error}'));
              }
              if (provider.filteredTags.isEmpty) {
                return const Center(child: Text('暂无标签'));
              }
              return ListView.builder(
                itemCount: provider.filteredTags.length,
                itemBuilder: (itemContext, index) {
                  final tag = provider.filteredTags[index];
                  final isSelected = provider.selectedTagIds.contains(tag.id);
                  return CheckboxListTile(
                    title: Text(tag.preferredLabel),
                    subtitle: Text('${tag.usageCount} 张图片'),
                    value: isSelected,
                    onChanged: (checked) {
                      debugPrint('标签点击: ${tag.preferredLabel} (ID: ${tag.id}), 选中状态: $checked');
                      provider.toggleTag(tag.id);
                      final selectedIds = provider.selectedTagIds.toList();
                      debugPrint('选中标签IDs: $selectedIds');
                      widget.onFilterChanged?.call(selectedIds);
                    },
                    secondary: Chip(
                      label: Text('${tag.usageCount}'),
                      visualDensity: VisualDensity.compact,
                    ),
                  );
                },
              );
            },
          ),
        ),
      ],
    );
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }
}
