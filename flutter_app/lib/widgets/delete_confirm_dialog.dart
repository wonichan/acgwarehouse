import 'package:flutter/material.dart';

class DeleteConfirmDialog extends StatelessWidget {
  final int count;
  final String itemType;
  final VoidCallback? onConfirm;

  const DeleteConfirmDialog({
    super.key,
    required this.count,
    this.itemType = '图片',
    this.onConfirm,
  });

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('确认删除'),
      content: Text('将删除 $count 张$itemType，此操作不可恢复。'),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: const Text('取消'),
        ),
        TextButton(
          onPressed: () {
            Navigator.pop(context, true);
            onConfirm?.call();
          },
          style: TextButton.styleFrom(foregroundColor: Colors.red),
          child: const Text('删除'),
        ),
      ],
    );
  }

  /// Shows the delete confirmation dialog
  /// Returns true if user confirmed, false otherwise
  static Future<bool> show(
    BuildContext context, {
    required int count,
    String itemType = '图片',
    VoidCallback? onConfirm,
  }) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (context) => DeleteConfirmDialog(
        count: count,
        itemType: itemType,
        onConfirm: onConfirm,
      ),
    );
    return result ?? false;
  }
}