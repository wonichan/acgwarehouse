import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';

/// Screen for tag governance and statistics
class TagManagementScreen extends StatelessWidget {
  const TagManagementScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => TagProvider(TagService())..loadStatistics(),
      child: const _TagManagementContent(),
    );
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
          if (provider.isLoading && provider.statistics.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (provider.error != null) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    '加载失败: ${provider.error}',
                    style: const TextStyle(color: Colors.red),
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
              
              // Statistics list
              Expanded(
                child: ListView.builder(
                  itemCount: provider.statistics.length,
                  itemBuilder: (context, index) {
                    final stat = provider.statistics[index];
                    return _buildStatTile(stat);
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
              Colors.blue,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildSummaryCard(
              context,
              '待复核',
              totals['pendingCount'] ?? 0,
              Icons.pending_actions,
              Colors.orange,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildSummaryCard(
              context,
              'AI 生成',
              totals['aiCount'] ?? 0,
              Icons.auto_awesome,
              Colors.purple,
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
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                fontWeight: FontWeight.bold,
              ),
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

  Widget _buildStatTile(dynamic stat) {
    return ListTile(
      leading: CircleAvatar(
        child: Text(stat.label[0].toUpperCase()),
      ),
      title: Text(stat.label),
      subtitle: Text(
        'AI: ${stat.aiCount} | 手动: ${stat.manualCount} | 待复核: ${stat.pendingCount}',
        style: const TextStyle(fontSize: 12),
      ),
      trailing: Chip(
        label: Text('${stat.usageCount}'),
        backgroundColor: Colors.grey[200],
      ),
    );
  }
}