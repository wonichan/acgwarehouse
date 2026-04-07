import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:gallery/providers/theme_provider.dart';

void main() {
  group('ThemeProvider', () {
    setUp(() {
      SharedPreferences.setMockInitialValues({});
    });

    test('默认主题模式为 system', () {
      final provider = ThemeProvider();

      expect(provider.themeMode, ThemeMode.system);
    });

    test('setThemeMode(light) 会更新状态并通知监听器', () async {
      final provider = ThemeProvider();
      var notified = false;
      provider.addListener(() {
        notified = true;
      });

      await provider.setThemeMode(ThemeMode.light);

      expect(provider.themeMode, ThemeMode.light);
      expect(notified, isTrue);
    });

    test('setThemeMode(dark) 会更新状态并通知监听器', () async {
      final provider = ThemeProvider();
      var notified = false;
      provider.addListener(() => notified = true);

      await provider.setThemeMode(ThemeMode.dark);
      expect(provider.themeMode, ThemeMode.dark);
      expect(notified, isTrue);
    });

    test('连续设置相同模式不会重复通知', () async {
      final provider = ThemeProvider();
      var calls = 0;
      provider.addListener(() => calls++);

      await provider.setThemeMode(ThemeMode.system);

      expect(calls, 0);
    });

    test('themeMode getter 返回当前模式', () async {
      final provider = ThemeProvider();

      await provider.setThemeMode(ThemeMode.dark);

      expect(provider.themeMode, ThemeMode.dark);
    });

    test('initialize 会加载已保存的主题模式', () async {
      SharedPreferences.setMockInitialValues({'theme_mode': 'dark'});

      final provider = ThemeProvider();

      await provider.initialize();

      expect(provider.themeMode, ThemeMode.dark);
      expect(provider.isInitialized, isTrue);
    });

    test('setThemeMode 会持久化保存主题模式', () async {
      final provider = ThemeProvider();

      await provider.initialize();
      await provider.setThemeMode(ThemeMode.light);

      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('theme_mode'), 'light');
      expect(provider.themeMode, ThemeMode.light);
    });
  });
}
