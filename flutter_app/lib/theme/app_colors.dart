import 'package:flutter/material.dart';

/// 全局统一配色。
/// 所有平台共享同一 seedColor，明暗主题使用预定义颜色常量。
class AppColors {
  static const Color seedColor = Color(0xFFED79B5);

  static const Color lightPrimary = Color(0xFFB32C71);
  static const Color lightPrimaryContainer = Color(0xFFFFD8E8);
  static const Color lightSecondary = Color(0xFF75565F);
  static const Color lightSecondaryContainer = Color(0xFFF3DAE0);
  static const Color lightSurface = Color(0xFFFFF8F9);
  static const Color lightSurfaceVariant = Color(0xFFF3DDE3);
  static const Color lightBackground = Color(0xFFFFF8F9);
  static const Color lightOnPrimary = Color(0xFFFFFFFF);
  static const Color lightOnSurface = Color(0xFF23161C);
  static const Color lightOnBackground = Color(0xFF23161C);

  static const Color darkPrimary = Color(0xFFF1A0C7);
  static const Color darkPrimaryContainer = Color(0xFF5C1A3C);
  static const Color darkSecondary = Color(0xFFD7BFC7);
  static const Color darkSecondaryContainer = Color(0xFF4A3640);
  static const Color darkSurface = Color(0xFF1D1418);
  static const Color darkSurfaceVariant = Color(0xFF46353C);
  static const Color darkBackground = Color(0xFF140F12);
  static const Color darkOnPrimary = Color(0xFF3D0C27);
  static const Color darkOnSurface = Color(0xFFF6EAF0);
  static const Color darkOnBackground = Color(0xFFF6EAF0);

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
