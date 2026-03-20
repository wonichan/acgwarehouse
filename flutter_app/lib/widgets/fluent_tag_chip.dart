import 'package:fluent_ui/fluent_ui.dart';
import '../models/tag.dart';

/// Tag chip style variants
enum FluentTagChipStyle {
  confirmed, // Blue accent
  pending, // Orange warning
  rejected, // Grey subdued
}

/// Fluent-styled tag chip widget
class FluentTagChip extends StatelessWidget {
  final Tag tag;
  final FluentTagChipStyle style;
  final VoidCallback? onTap;
  final VoidCallback? onDelete;
  final VoidCallback? onConfirm;
  final VoidCallback? onReject;

  const FluentTagChip({
    super.key,
    required this.tag,
    this.style = FluentTagChipStyle.confirmed,
    this.onTap,
    this.onDelete,
    this.onConfirm,
    this.onReject,
  });

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final colors = _getColors(theme);

    return MouseRegion(
      cursor: onTap != null ? SystemMouseCursors.click : MouseCursor.defer,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            color: colors.backgroundColor,
            borderRadius: BorderRadius.circular(4),
            border: Border.all(color: colors.borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                tag.preferredLabel,
                style: TextStyle(color: colors.textColor),
              ),
              if (style == FluentTagChipStyle.pending) ...[
                const SizedBox(width: 8),
                // Confirm button
                IconButton(
                  icon: const Icon(FluentIcons.check_mark, size: 12),
                  onPressed: onConfirm,
                ),
                // Reject button
                IconButton(
                  icon: const Icon(FluentIcons.clear, size: 12),
                  onPressed: onReject,
                ),
              ],
              if (onDelete != null) ...[
                const SizedBox(width: 4),
                IconButton(
                  icon: const Icon(FluentIcons.delete, size: 12),
                  onPressed: onDelete,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  _ChipColors _getColors(FluentThemeData theme) {
    switch (style) {
      case FluentTagChipStyle.confirmed:
        return _ChipColors(
          backgroundColor: theme.accentColor.withOpacity(0.1),
          borderColor: theme.accentColor,
          textColor: theme.accentColor,
        );
      case FluentTagChipStyle.pending:
        return _ChipColors(
          backgroundColor: const Color(0xFFFFF3E0),
          borderColor: const Color(0xFFFF9800),
          textColor: const Color(0xFFE65100),
        );
      case FluentTagChipStyle.rejected:
        return _ChipColors(
          backgroundColor: theme.resources.cardBackgroundFillColorSecondary,
          borderColor: theme.resources.cardStrokeColorDefault,
          textColor: theme.resources.textFillColorSecondary,
        );
    }
  }
}

class _ChipColors {
  final Color backgroundColor;
  final Color borderColor;
  final Color textColor;

  _ChipColors({
    required this.backgroundColor,
    required this.borderColor,
    required this.textColor,
  });
}
