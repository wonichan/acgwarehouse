import 'package:flutter/material.dart';
import '../models/tag.dart';

enum TagChipStyle {
  confirmed, // 绿色，已确认
  pending, // 橙色，待确认
  rejected, // 红色，已拒绝
  neutral, // 默认灰色
}

class TagChip extends StatelessWidget {
  final Tag tag;
  final TagChipStyle style;
  final VoidCallback? onTap;
  final VoidCallback? onConfirm;
  final VoidCallback? onReject;
  final VoidCallback? onDelete;
  final bool showActions;

  const TagChip({
    super.key,
    required this.tag,
    this.style = TagChipStyle.neutral,
    this.onTap,
    this.onConfirm,
    this.onReject,
    this.onDelete,
    this.showActions = false,
  });

  Color get _backgroundColor {
    switch (style) {
      case TagChipStyle.confirmed:
        return Colors.green.shade100;
      case TagChipStyle.pending:
        return Colors.orange.shade100;
      case TagChipStyle.rejected:
        return Colors.red.shade100;
      default:
        return Colors.grey.shade200;
    }
  }

  Color get _textColor {
    switch (style) {
      case TagChipStyle.confirmed:
        return Colors.green.shade800;
      case TagChipStyle.pending:
        return Colors.orange.shade800;
      case TagChipStyle.rejected:
        return Colors.red.shade800;
      default:
        return Colors.grey.shade800;
    }
  }

  @override
  Widget build(BuildContext context) {
    final hasActions =
        showActions || onConfirm != null || onReject != null || onDelete != null;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: _backgroundColor,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              tag.preferredLabel,
              style: TextStyle(color: _textColor, fontSize: 14),
            ),
            if (hasActions) ...[
              const SizedBox(width: 4),
              if (onConfirm != null)
                InkWell(
                  onTap: onConfirm,
                  child: const Icon(Icons.check, size: 16, color: Colors.green),
                ),
              if (onReject != null)
                InkWell(
                  onTap: onReject,
                  child: const Icon(Icons.close, size: 16, color: Colors.red),
                ),
              if (onDelete != null)
                InkWell(
                  onTap: onDelete,
                  child: const Icon(Icons.delete_outline,
                      size: 16, color: Colors.grey),
                ),
            ],
          ],
        ),
      ),
    );
  }
}
