import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/duplicate_provider.dart';
import '../widgets/duplicate_group_card.dart';

/// Screen for managing duplicate images
class DuplicateScreen extends StatefulWidget {
  const DuplicateScreen({super.key});

  @override
  State<DuplicateScreen> createState() => _DuplicateScreenState();
}

class _DuplicateScreenState extends State<DuplicateScreen> {
  final TextEditingController _thresholdController = TextEditingController(text: '10');
  
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<DuplicateProvider>().loadGroups(refresh: true);
    });
  }

  @override
  void dispose() {
    _thresholdController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('重复检测'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              context.read<DuplicateProvider>().loadGroups(refresh: true);
            },
          ),
        ],
      ),
      body: Consumer<DuplicateProvider>(
        builder: (context, provider, child) {
          if (provider.error != null) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.error_outline, size: 48, color: Colors.red),
                  const SizedBox(height: 16),
                  Text('错误: ${provider.error}'),
                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () {
                      provider.clearError();
                      provider.loadGroups(refresh: true);
                    },
                    child: const Text('重试'),
                  ),
                ],
              ),
            );
          }

          if (provider.isDetecting) {
            return const Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  CircularProgressIndicator(),
                  SizedBox(height: 16),
                  Text('正在检测重复图片...'),
                ],
              ),
            );
          }

          if (provider.isLoading && provider.groups.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (provider.groups.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.check_circle_outline, size: 48, color: Colors.green),
                  const SizedBox(height: 16),
                  const Text('没有发现重复图片'),
                  const SizedBox(height: 24),
                  _buildDetectButton(context, provider),
                ],
              ),
            );
          }

          return Column(
            children: [
              // Header with detection button
              Container(
                padding: const EdgeInsets.all(16),
                child: Row(
                  children: [
                    Text('发现 ${provider.totalGroups} 组重复图片'),
                    const Spacer(),
                    _buildDetectButton(context, provider),
                  ],
                ),
              ),
              // Groups list
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () => provider.loadGroups(refresh: true),
                  child: ListView.builder(
                    itemCount: provider.groups.length + (provider.hasMore ? 1 : 0),
                    itemBuilder: (context, index) {
                      if (index == provider.groups.length) {
                        // Load more indicator
                        provider.loadMore();
                        return const Center(
                          child: Padding(
                            padding: EdgeInsets.all(16),
                            child: CircularProgressIndicator(),
                          ),
                        );
                      }

                      final group = provider.groups[index];
                      return DuplicateGroupCard(
                        group: group,
                        onDelete: () => _confirmDeleteGroup(context, provider, group.id),
                      );
                    },
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildDetectButton(BuildContext context, DuplicateProvider provider) {
    return ElevatedButton.icon(
      onPressed: () => _showDetectDialog(context, provider),
      icon: const Icon(Icons.search),
      label: const Text('检测重复'),
    );
  }

  void _showDetectDialog(BuildContext context, DuplicateProvider provider) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('检测重复图片'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('设置相似度阈值（汉明距离）：'),
            const SizedBox(height: 8),
            TextField(
              controller: _thresholdController,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(
                labelText: '阈值',
                hintText: '10',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              '阈值越小，检测越严格。推荐值：10',
              style: TextStyle(fontSize: 12, color: Colors.grey),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('取消'),
          ),
          ElevatedButton(
            onPressed: () {
              Navigator.pop(context);
              final threshold = int.tryParse(_thresholdController.text) ?? 10;
              provider.detectDuplicates(threshold: threshold);
            },
            child: const Text('开始检测'),
          ),
        ],
      ),
    );
  }

  void _confirmDeleteGroup(BuildContext context, DuplicateProvider provider, int groupId) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('确认删除'),
        content: const Text('确定要删除此重复组记录吗？\n\n注意：这只会删除重复组记录，不会删除图片文件。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('取消'),
          ),
          ElevatedButton(
            onPressed: () {
              Navigator.pop(context);
              provider.deleteGroup(groupId);
            },
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('删除'),
          ),
        ],
      ),
    );
  }
}