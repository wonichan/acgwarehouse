import 'package:fluent_ui/fluent_ui.dart';
import '../models/tag.dart';

/// Tag chip style variants
enum FluentTagChipStyle {
  confirmed, // Blue accent / Green
  pending, // Orange warning
  rejected, // Grey subdued
}

/// Fluent-styled tag chip widget
class FluentTagChip extends StatefulWidget {
  final Tag tag;
  final FluentTagChipStyle style;
  final VoidCallback? onTap;
  final VoidCallback? onDelete;
  final VoidCallback? onConfirm;
  final VoidCallback? onReject;
  final VoidCallback? onMerge;
  final VoidCallback? onEdit;
  final bool showActions;

  const FluentTagChip({
    super.key,
    required this.tag,
    this.style = FluentTagChipStyle.confirmed,
    this.onTap,
    this.onDelete,
    this.onConfirm,
    this.onReject,
    this.onMerge,
    this.onEdit,
    this.showActions = false,
  });

  @override
  State<FluentTagChip> createState() => _FluentTagChipState();
}

class _FluentTagChipState extends State<FluentTagChip> {
  bool _isHovered = false;

  Color _getDotColor(FluentThemeData theme) {
    switch (widget.style) {
      case FluentTagChipStyle.confirmed:
        return Colors.green;
      case FluentTagChipStyle.pending:
        return Colors.orange;
      case FluentTagChipStyle.rejected:
        return Colors.red;
    }
  }

  Color _getTextColor(FluentThemeData theme) {
    switch (widget.style) {
      case FluentTagChipStyle.rejected:
        return theme.resources.textFillColorSecondary;
      default:
        return theme.resources.textFillColorPrimary;
    }
  }

  FontWeight get _fontWeight {
    switch (widget.style) {
      case FluentTagChipStyle.confirmed:
        return FontWeight.w600;
      case FluentTagChipStyle.pending:
        return FontWeight.w500;
      default:
        return FontWeight.normal;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final hasActions =
        widget.showActions ||
        widget.onConfirm != null ||
        widget.onReject != null ||
        widget.onDelete != null ||
        widget.onMerge != null ||
        widget.onEdit != null;

    final isRejected = widget.style == FluentTagChipStyle.rejected;

    return MouseRegion(
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      cursor: widget.onTap != null
          ? SystemMouseCursors.click
          : MouseCursor.defer,
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          color: Colors.transparent, // Ensure hit test catches hover
          padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 2),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(
                  color: _getDotColor(theme),
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 6),
              Text(
                widget.tag.preferredLabel,
                style: TextStyle(
                  color: _getTextColor(theme),
                  fontWeight: _fontWeight,
                  decoration: isRejected ? TextDecoration.lineThrough : null,
                ),
              ),
              if (hasActions && (_isHovered || widget.showActions)) ...[
                const SizedBox(width: 4),
                if (widget.onConfirm != null)
                  _buildIcon(
                    FluentIcons.check_mark,
                    Colors.green,
                    widget.onConfirm!,
                  ),
                if (widget.onReject != null)
                  _buildIcon(FluentIcons.clear, Colors.red, widget.onReject!),
                if (widget.onMerge != null)
                  _buildIcon(FluentIcons.merge, Colors.blue, widget.onMerge!),
                if (widget.onEdit != null)
                  _buildIcon(
                    FluentIcons.edit,
                    theme.resources.textFillColorTertiary,
                    widget.onEdit!,
                  ),
                if (widget.onDelete != null)
                  _buildIcon(
                    FluentIcons.delete,
                    theme.resources.textFillColorSecondary,
                    widget.onDelete!,
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
      child: IconButton(
        icon: Icon(icon, size: 12, color: color),
        onPressed: onTap,
        style: ButtonStyle(
          padding: WidgetStateProperty.all(const EdgeInsets.all(4)),
        ),
      ),
    );
  }
}
