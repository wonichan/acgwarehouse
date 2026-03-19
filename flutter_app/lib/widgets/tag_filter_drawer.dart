import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/tag_provider.dart';

class TagFilterDrawer extends StatefulWidget {
  final Function(List<int> tagIds)? onFilterChanged;

  const TagFilterDrawer({super.key, this.onFilterChanged});

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
    return Drawer(
      child: Column(
        children: [
          // 标题
          DrawerHeader(
            decoration: BoxDecoration(color: Theme.of(context).primaryColor),
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
                  itemBuilder: (context, index) {
                    final tag = provider.filteredTags[index];
                    final isSelected = provider.selectedTagIds.contains(tag.id);
                    return CheckboxListTile(
                      title: Text(tag.preferredLabel),
                      subtitle: Text('${tag.usageCount} 张图片'),
                      value: isSelected,
                      onChanged: (checked) {
                        // 使用 context.read 确保获取到正确的 provider 实例
                        final tagProvider = context.read<TagProvider>();
                        tagProvider.toggleTag(tag.id);
                        widget.onFilterChanged
                            ?.call(tagProvider.selectedTagIds.toList());
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
      ),
    );
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }
}
