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
      brightness: brightness,
      useMaterial3: true,
      scaffoldBackgroundColor: colorScheme.surface,
      appBarTheme: AppBarTheme(
        backgroundColor: colorScheme.surface,
        foregroundColor: colorScheme.onSurface,
        elevation: 0,
        centerTitle: true,
      ),
      cardTheme: CardThemeData(
        color: colorScheme.surfaceContainerHighest,
        shadowColor: colorScheme.shadow,
        surfaceTintColor: colorScheme.surfaceTint,
        elevation: 0,
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: colorScheme.surface,
        indicatorColor: colorScheme.primaryContainer,
      ),
      floatingActionButtonTheme: FloatingActionButtonThemeData(
        backgroundColor: colorScheme.primaryContainer,
        foregroundColor: colorScheme.onPrimaryContainer,
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: colorScheme.primary,
          foregroundColor: colorScheme.onPrimary,
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: colorScheme.primary,
          foregroundColor: colorScheme.onPrimary,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: colorScheme.primary,
          side: BorderSide(color: colorScheme.primary),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: colorScheme.primary,
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colorScheme.surfaceContainerHighest,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: colorScheme.outlineVariant),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: colorScheme.primary, width: 2),
        ),
      ),
    );
  }

  static fluent.FluentThemeData getFluentTheme(Brightness brightness) {
    final primary = AppColors.primaryFor(brightness);
    final background = AppColors.backgroundFor(brightness);

    return fluent.FluentThemeData(
      brightness: brightness,
      accentColor: primary.toAccentColor(),
      scaffoldBackgroundColor: background,
      cardColor: background,
    );
  }
}
