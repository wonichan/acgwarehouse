import 'package:flutter/material.dart';

/// 全局统一配色。
/// 所有平台共享同一 seedColor，明暗主题使用预定义颜色常量。
class AppColors {
  // Apple-style minimalist monochrome seed
  static const Color seedColor = Color(0xFF888888);

  static const Color lightPrimary = Color(0xFF000000);
  static const Color lightPrimaryContainer = Color(0xFFE5E5E5);
  static const Color lightSecondary = Color(0xFF555555);
  static const Color lightSecondaryContainer = Color(0xFFEEEEEE);
  static const Color lightSurface = Color(0xFFFFFFFF);
  static const Color lightSurfaceVariant = Color(0xFFF5F5F5);
  static const Color lightBackground = Color(0xFFFFFFFF);
  static const Color lightOnPrimary = Color(0xFFFFFFFF);
  static const Color lightOnSurface = Color(0xFF000000);
  static const Color lightOnBackground = Color(0xFF000000);

  static const Color darkPrimary = Color(0xFFFFFFFF);
  static const Color darkPrimaryContainer = Color(0xFF333333);
  static const Color darkSecondary = Color(0xFFAAAAAA);
  static const Color darkSecondaryContainer = Color(0xFF222222);
  static const Color darkSurface = Color(0xFF1C1C1E);
  static const Color darkSurfaceVariant = Color(0xFF2D2D30);
  static const Color darkBackground = Color(0xFF000000);
  static const Color darkOnPrimary = Color(0xFF000000);
  static const Color darkOnSurface = Color(0xFFFFFFFF);
  static const Color darkOnBackground = Color(0xFFFFFFFF);

  static const Map<Brightness, Color> _primaryByBrightness = {
    Brightness.light: lightPrimary,
    Brightness.dark: darkPrimary,
  };

  static const Map<Brightness, Color> _surfaceByBrightness = {
    Brightness.light: lightSurface,
    Brightness.dark: darkSurface,
  };

  static Color primaryFor(Brightness brightness) =>
      _primaryByBrightness[brightness] ?? lightPrimary;

  static Color surfaceFor(Brightness brightness) =>
      _surfaceByBrightness[brightness] ?? lightSurface;

  static Color backgroundFor(Brightness brightness) =>
      brightness == Brightness.dark ? darkBackground : lightBackground;
}
