import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../providers/tag_provider.dart';

/// Screen for tag governance and statistics
class TagManagementScreen extends StatelessWidget {
  const TagManagementScreen({super.key});

  @override
  Widget build(BuildContext context) {
    // Load statistics on first build if not already loaded
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final provider = context.read<TagProvider>();
      if (provider.statistics.isEmpty && !provider.isLoadingStatistics) {
        provider.loadStatistics();
      }
    });

    return const _TagManagementContent();
  }
}

class _TagManagementContent extends StatelessWidget {
  const _TagManagementContent();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('标签管理'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => context.read<TagProvider>().loadStatistics(),
            tooltip: '刷新',
          ),
        ],
      ),
      body: Consumer<TagProvider>(
        builder: (context, provider, child) {
          if (provider.isLoadingStatistics && provider.statistics.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (provider.error != null) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    '加载失败: ${provider.error}',
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: () => provider.loadStatistics(),
                    child: const Text('重试'),
                  ),
                ],
              ),
            );
          }

          if (provider.statistics.isEmpty) {
            return const Center(child: Text('暂无标签统计数据'));
          }

          return Column(
            children: [
              // Summary cards
              _buildSummaryCards(context, provider),

              // Action buttons
              _buildActionButtons(context, provider),

              // Statistics list
              Expanded(
                child: ListView.builder(
                  itemCount: provider.statistics.length,
                  itemBuilder: (context, index) {
                    final stat = provider.statistics[index];
                    return _buildStatTile(context, provider, stat);
                  },
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildSummaryCards(BuildContext context, TagProvider provider) {
    final totals = provider.totals;

    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(
            child: _buildSummaryCard(
              context,
              '总使用量',
              totals['usageCount'] ?? 0,
              Icons.label,
              Theme.of(context).colorScheme.primary,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildSummaryCard(
              context,
              '待复核',
              totals['pendingCount'] ?? 0,
              Icons.pending_actions,
              Theme.of(context).colorScheme.secondary,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildSummaryCard(
              context,
              'AI 生成',
              totals['aiCount'] ?? 0,
              Icons.auto_awesome,
              Theme.of(context).colorScheme.tertiary,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSummaryCard(
    BuildContext context,
    String title,
    int value,
    IconData icon,
    Color color,
  ) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            Icon(icon, color: color, size: 24),
            const SizedBox(height: 4),
            Text(
              '$value',
              style: Theme.of(
                context,
              ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold),
            ),
            Text(
              title,
              style: Theme.of(context).textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButtons(BuildContext context, TagProvider provider) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: FilledButton.icon(
              onPressed: provider.isLoadingStatistics
                  ? null
                  : () => _showCleanUnusedTagsDialog(context, provider),
              icon: const Icon(Icons.cleaning_services),
              label: const Text('清理无用标签'),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _showCleanUnusedTagsDialog(
    BuildContext context,
    TagProvider provider,
  ) async {
    // 计算无用标签数量
    final unusedCount = provider.statistics
        .where((s) => s.usageCount == 0)
        .length;

    if (unusedCount == 0) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('没有无用标签需要清理')));
      return;
    }

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('确认清理'),
        content: Text('将删除 $unusedCount 个无用标签（未被任何图片使用）。此操作不可撤销。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('确认清理'),
          ),
        ],
      ),
    );

    if (confirmed == true && context.mounted) {
      try {
        final result = await provider.cleanUnusedTags();
        final deletedCount = result['deleted_count'] as int? ?? 0;

        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('成功清理 $deletedCount 个无用标签')));
        }
      } catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('清理失败: $e')));
        }
      }
    }
  }

  Widget _buildStatTile(
    BuildContext context,
    TagProvider provider,
    TagStatistics stat,
  ) {
    return ListTile(
      leading: CircleAvatar(
        child: Text(stat.label.isNotEmpty ? stat.label[0].toUpperCase() : '?'),
      ),
      title: Text(stat.label),
      subtitle: Text(
        'AI: ${stat.aiCount} | 手动: ${stat.manualCount} | 待复核: ${stat.pendingCount}',
        style: const TextStyle(fontSize: 12),
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Chip(
            label: Text('${stat.usageCount}'),
            backgroundColor: stat.usageCount == 0
                ? Theme.of(context).colorScheme.errorContainer
                : Theme.of(context).colorScheme.surfaceContainerHighest,
            side: BorderSide.none,
          ),
          PopupMenuButton<String>(
            onSelected: (value) {
              if (value == 'edit') {
                _showEditTagDialog(context, provider, stat);
              } else if (value == 'delete') {
                _showDeleteTagDialog(context, provider, stat);
              }
            },
            itemBuilder: (context) => [
              const PopupMenuItem(
                value: 'edit',
                child: Row(
                  children: [
                    Icon(Icons.edit, size: 20),
                    SizedBox(width: 8),
                    Text('编辑标签'),
                  ],
                ),
              ),
              PopupMenuItem(
                value: 'delete',
                child: Row(
                  children: [
                    Icon(
                      Icons.delete,
                      size: 20,
                      color: Theme.of(context).colorScheme.error,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      '删除标签',
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _showEditTagDialog(
    BuildContext context,
    TagProvider provider,
    TagStatistics stat,
  ) async {
    final controller = TextEditingController(text: stat.label);

    final result = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('编辑标签'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(
            labelText: '标签名称',
            hintText: '输入新的标签名称',
          ),
          autofocus: true,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () {
              final newLabel = controller.text.trim();
              if (newLabel.isNotEmpty && newLabel != stat.label) {
                Navigator.of(context).pop(newLabel);
              } else {
                Navigator.of(context).pop();
              }
            },
            child: const Text('保存'),
          ),
        ],
      ),
    );

    if (result != null && result.isNotEmpty && context.mounted) {
      try {
        await provider.updateTag(stat.tagId, preferredLabel: result);
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(const SnackBar(content: Text('标签更新成功')));
        }
      } catch (e) {
        // 检查是否是重名错误 (409 Conflict)
        final errorMsg = e.toString();
        if (errorMsg.contains('409') || errorMsg.contains('标签名已存在')) {
          if (context.mounted) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(const SnackBar(content: Text('标签名已存在，请使用其他名称')));
            // 重新打开编辑对话框让用户修改
            _showEditTagDialog(context, provider, stat);
          }
        } else {
          if (context.mounted) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text('更新失败: $e')));
          }
        }
      }
    }
  }

  Future<void> _showDeleteTagDialog(
    BuildContext context,
    TagProvider provider,
    TagStatistics stat,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('确认删除'),
        content: Text('确定要删除标签 "${stat.label}" 吗？此操作将同时删除该标签的所有图片关联，且不可撤销。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.error,
              foregroundColor: Theme.of(context).colorScheme.onError,
            ),
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('删除'),
          ),
        ],
      ),
    );

    if (confirmed == true && context.mounted) {
      try {
        await provider.deleteTag(stat.tagId);
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('标签 "${stat.label}" 已删除')));
        }
      } catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('删除失败: $e')));
        }
      }
    }
  }
}
