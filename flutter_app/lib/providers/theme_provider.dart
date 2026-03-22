import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

class ThemeProvider extends ChangeNotifier {
  static const String _themeModeKey = 'theme_mode';

  ThemeMode _themeMode = ThemeMode.system;
  SharedPreferences? _prefs;
  bool _initialized = false;

  ThemeMode get themeMode => _themeMode;
  bool get isInitialized => _initialized;

  Future<void> initialize() async {
    if (_initialized) return;

    _prefs = await SharedPreferences.getInstance();
    final savedMode = _prefs?.getString(_themeModeKey);
    if (savedMode != null) {
      _themeMode = _parseThemeMode(savedMode);
    }

    _initialized = true;
    notifyListeners();
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    if (!_initialized) {
      _prefs ??= await SharedPreferences.getInstance();
      final savedMode = _prefs?.getString(_themeModeKey);
      if (savedMode != null) {
        _themeMode = _parseThemeMode(savedMode);
      }
      _initialized = true;
    }

    if (_themeMode == mode) return;

    _themeMode = mode;
    _prefs ??= await SharedPreferences.getInstance();
    await _prefs!.setString(_themeModeKey, _themeModeToString(mode));
    notifyListeners();
  }

  bool get isSystemMode => _themeMode == ThemeMode.system;
  bool get isLightMode => _themeMode == ThemeMode.light;
  bool get isDarkMode => _themeMode == ThemeMode.dark;

  ThemeMode _parseThemeMode(String value) {
    switch (value) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
    }
  }

  String _themeModeToString(ThemeMode mode) {
    switch (mode) {
      case ThemeMode.light:
        return 'light';
      case ThemeMode.dark:
        return 'dark';
      case ThemeMode.system:
        return 'system';
    }
  }
}
