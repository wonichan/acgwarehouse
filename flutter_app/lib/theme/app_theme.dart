import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';

import 'app_colors.dart';

class AppTheme {
  static ThemeData getMaterialTheme(Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: AppColors.seedColor,
      brightness: brightness,
    );

    return ThemeData(
      colorScheme: colorScheme,
      useMaterial3: true,
      scaffoldBackgroundColor: AppColors.backgroundFor(brightness),
      appBarTheme: AppBarTheme(
        backgroundColor: colorScheme.surface,
        foregroundColor: colorScheme.onSurface,
        elevation: 0,
      ),
      cardTheme: CardThemeData(
        color: colorScheme.surface,
        shadowColor: colorScheme.shadow,
      ),
    );
  }

  static fluent.FluentThemeData getFluentTheme(Brightness brightness) {
    final primary = AppColors.primaryFor(brightness);

    return fluent.FluentThemeData(
      brightness: brightness,
      accentColor: primary.toAccentColor(),
      scaffoldBackgroundColor: AppColors.backgroundFor(brightness),
    );
  }
}
