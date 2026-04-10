import 'package:flutter/material.dart';
import '../models/tag.dart';

enum TagChipStyle {
  confirmed, // 绿色，已确认
  pending, // 橙色，待确认
  rejected, // 红色，已拒绝
  neutral, // 默认灰色
}

class TagChip extends StatefulWidget {
  final Tag tag;
  final TagChipStyle style;
  final VoidCallback? onTap;
  final VoidCallback? onConfirm;
  final VoidCallback? onReject;
  final VoidCallback? onDelete;
  final VoidCallback? onMerge;
  final VoidCallback? onEdit;
  final bool showActions;

  const TagChip({
    super.key,
    required this.tag,
    this.style = TagChipStyle.neutral,
    this.onTap,
    this.onConfirm,
    this.onReject,
    this.onDelete,
    this.onMerge,
    this.onEdit,
    this.showActions = false,
  });

  @override
  State<TagChip> createState() => _TagChipState();
}

class _TagChipState extends State<TagChip> {
  bool _isHovered = false;

  Color _dotColor(BuildContext context) {
    switch (widget.style) {
      case TagChipStyle.confirmed:
        return Colors.green.shade500;
      case TagChipStyle.pending:
        return Colors.orange.shade400;
      case TagChipStyle.rejected:
        return Colors.red.shade400;
      default:
        return Theme.of(context).colorScheme.outline;
    }
  }

  Color _textColor(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    switch (widget.style) {
      case TagChipStyle.rejected:
        return colorScheme.onSurface.withOpacity(0.5);
      default:
        return colorScheme.onSurface;
    }
  }

  FontWeight get _fontWeight {
    switch (widget.style) {
      case TagChipStyle.confirmed:
        return FontWeight.w600;
      case TagChipStyle.pending:
        return FontWeight.w500;
      default:
        return FontWeight.w400;
    }
  }

  @override
  Widget build(BuildContext context) {
    final hasActions =
        widget.showActions ||
        widget.onConfirm != null ||
        widget.onReject != null ||
        widget.onDelete != null ||
        widget.onMerge != null ||
        widget.onEdit != null;

    final isRejected = widget.style == TagChipStyle.rejected;
    final showActionButtons = hasActions && (_isHovered || widget.showActions);

    return MouseRegion(
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      cursor: widget.onTap != null
          ? SystemMouseCursors.click
          : MouseCursor.defer,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          color: Colors.transparent, // To catch hover and click events
          padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 2),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(
                  color: _dotColor(context),
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 6),
              Text(
                widget.tag.preferredLabel,
                style: TextStyle(
                  color: _textColor(context),
                  fontSize: 14,
                  fontWeight: _fontWeight,
                  decoration: isRejected ? TextDecoration.lineThrough : null,
                ),
              ),
              if (hasActions) ...[
                const SizedBox(width: 4),
                AnimatedOpacity(
                  duration: const Duration(milliseconds: 150),
                  opacity: showActionButtons ? 1.0 : 0.0,
                  child: IgnorePointer(
                    ignoring: !showActionButtons,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (widget.onConfirm != null)
                          _buildIcon(
                            Icons.check,
                            Colors.green.shade600,
                            widget.onConfirm!,
                          ),
                        if (widget.onReject != null)
                          _buildIcon(
                            Icons.close,
                            Colors.red.shade400,
                            widget.onReject!,
                          ),
                        if (widget.onMerge != null)
                          _buildIcon(
                            Icons.merge_type,
                            Colors.blue.shade400,
                            widget.onMerge!,
                          ),
                        if (widget.onEdit != null)
                          _buildIcon(
                            Icons.edit,
                            Theme.of(context).colorScheme.outline,
                            widget.onEdit!,
                          ),
                        if (widget.onDelete != null)
                          _buildIcon(
                            Icons.delete_outline,
                            Theme.of(context).colorScheme.outlineVariant,
                            widget.onDelete!,
                          ),
                      ],
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildIcon(IconData icon, Color color, VoidCallback onTap) {
    return Padding(
      padding: const EdgeInsets.only(left: 4),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(4),
        child: Icon(icon, size: 16, color: color),
      ),
    );
  }
}
