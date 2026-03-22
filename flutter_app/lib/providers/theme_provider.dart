import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

class ThemeProvider extends ChangeNotifier {
  ThemeMode _themeMode = ThemeMode.system;

  ThemeMode get themeMode => _themeMode;

  void setThemeMode(ThemeMode mode) {
    if (_themeMode == mode) {
      return;
    }

    _themeMode = mode;
    notifyListeners();
  }

  bool get isSystemMode => _themeMode == ThemeMode.system;
  bool get isLightMode => _themeMode == ThemeMode.light;
  bool get isDarkMode => _themeMode == ThemeMode.dark;
}
