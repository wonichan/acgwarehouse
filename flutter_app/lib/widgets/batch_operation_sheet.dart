import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/selection_provider.dart';

class BatchOperationSheet extends StatelessWidget {
  final VoidCallback? onAddTags;
  final VoidCallback? onRemoveTags;
  final VoidCallback? onMoveToCollection;
  final VoidCallback? onDelete;

  const BatchOperationSheet({
    super.key,
    this.onAddTags,
    this.onRemoveTags,
    this.onMoveToCollection,
    this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<SelectionProvider>(
      builder: (context, selectionProvider, _) {
        return Container(
          padding: const EdgeInsets.all(16),
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
            ],
          ),
        );
      },
    );
  }

  Widget _buildOperationButton(
    BuildContext context, {
    required IconData icon,
    required String label,
    VoidCallback? onTap,
    bool isDestructive = false,
  }) {
    final color = isDestructive ? Colors.red : Theme.of(context).primaryColor;

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        width: 80,
        padding: const EdgeInsets.symmetric(vertical: 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 8),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                color: color,
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
    VoidCallback? onAddTags,
    VoidCallback? onRemoveTags,
    VoidCallback? onMoveToCollection,
    VoidCallback? onDelete,
  }) {
    return showModalBottomSheet(
      context: context,
      builder: (context) => BatchOperationSheet(
        onAddTags: onAddTags,
        onRemoveTags: onRemoveTags,
        onMoveToCollection: onMoveToCollection,
        onDelete: onDelete,
      ),
    );
  }
}