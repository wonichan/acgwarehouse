import 'package:flutter/material.dart';

class ImageMetadataPaneTheme {
  final BuildContext context;

  ImageMetadataPaneTheme.of(this.context);

  ThemeData get theme => Theme.of(context);
  ColorScheme get colorScheme => theme.colorScheme;

  // Overall panel surface
  Color get panelSurface => colorScheme.surface;

  // Background for individual sections (AI, Tags, etc.) replacing hardcoded white Cards
  Color get sectionBackground => colorScheme.surface;

  // Borders for sections and inputs
  Color get borderColor => colorScheme.outlineVariant;

  // Status badge background replacing 0xFFEFEFEF
  Color get statusBackground => colorScheme.surfaceContainerHighest;

  // Status badge text replacing black87
  Color get statusForeground => colorScheme.onSurfaceVariant;

  // TextField fill color replacing 0xFFFAFAFA
  Color get inputFillColor =>
      colorScheme.surfaceContainerHighest.withValues(alpha: 0.3);

  // Pending tag chip background replacing 0xFFF8F8F8
  Color get pendingTagBackground =>
      colorScheme.surfaceContainerHighest.withValues(alpha: 0.5);

  // Divider color for pending tags replacing 0xFFD0D0D0
  Color get pendingTagDivider => colorScheme.outlineVariant;

  // Primary text color replacing explicit black/white estimation
  Color get textForeground => colorScheme.onSurface;

  // Muted text color replacing black54/white70
  Color get textMuted => colorScheme.onSurfaceVariant;

  // Icon color replacing Colors.blueGrey
  Color get iconColor => colorScheme.onSurfaceVariant;

  // Shared section decoration to replace Card
  BoxDecoration get sectionDecoration => BoxDecoration(
    color: sectionBackground,
    borderRadius: BorderRadius.circular(8),
    border: Border.all(color: borderColor),
  );
}
