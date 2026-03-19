import 'package:flutter/material.dart';
import '../providers/selection_provider.dart';

class BatchOperationSheet extends StatelessWidget {
  final SelectionProvider selectionProvider;
  final VoidCallback? onAddTags;
  final VoidCallback? onRemoveTags;
  final VoidCallback? onGenerateAITags;
  final VoidCallback? onMoveToCollection;
  final VoidCallback? onDelete;

  const BatchOperationSheet({
    super.key,
    required this.selectionProvider,
    this.onAddTags,
    this.onRemoveTags,
    this.onGenerateAITags,
    this.onMoveToCollection,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
          // Header
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                '已选择 ${selectionProvider.selectedCount} 张图片',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              TextButton(
                onPressed: () {
                  selectionProvider.exitSelectionMode();
                  Navigator.pop(context);
                },
                child: const Text('取消'),
              ),
            ],
          ),
          const Divider(),
          const SizedBox(height: 8),

          // Operation buttons
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
            children: [
              _buildOperationButton(
                context,
                icon: Icons.add_circle_outline,
                label: '添加标签',
                onTap: onAddTags,
              ),
              _buildOperationButton(
                context,
                icon: Icons.remove_circle_outline,
                label: '移除标签',
                onTap: onRemoveTags,
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
            children: [
              _buildOperationButton(
                context,
                icon: Icons.folder_outlined,
                label: '移至收藏夹',
                onTap: onMoveToCollection,
              ),
              _buildOperationButton(
                context,
                icon: Icons.delete_outline,
                label: '删除',
                onTap: onDelete,
                isDestructive: true,
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              _buildOperationButton(
                context,
                icon: Icons.auto_awesome,
                label: 'AI生成标签',
                onTap: onGenerateAITags,
                color: const Color(0xFF5E35B1),
              ),
            ],
          ),
          const SizedBox(height: 16),
        ],
      ),
    ),
  );
  }

  Widget _buildOperationButton(
    BuildContext context, {
    required IconData icon,
    required String label,
    VoidCallback? onTap,
    bool isDestructive = false,
    Color? color,
  }) {
    final resolvedColor = color ?? (isDestructive ? Colors.red : Theme.of(context).primaryColor);

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        width: 80,
        padding: const EdgeInsets.symmetric(vertical: 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, color: resolvedColor, size: 28),
            const SizedBox(height: 8),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: resolvedColor,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  /// Shows the batch operation sheet
  static Future<void> show(
    BuildContext context, {
    required SelectionProvider selectionProvider,
    VoidCallback? onAddTags,
    VoidCallback? onRemoveTags,
    VoidCallback? onGenerateAITags,
    VoidCallback? onMoveToCollection,
    VoidCallback? onDelete,
  }) {
    return showModalBottomSheet(
      context: context,
      builder: (context) => BatchOperationSheet(
        selectionProvider: selectionProvider,
        onAddTags: onAddTags,
        onRemoveTags: onRemoveTags,
        onGenerateAITags: onGenerateAITags,
        onMoveToCollection: onMoveToCollection,
        onDelete: onDelete,
      ),
    );
  }
}
