import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/theme_provider.dart';

void main() {
  group('ThemeProvider', () {
    test('默认主题模式为 system', () {
      final provider = ThemeProvider();

      expect(provider.themeMode, ThemeMode.system);
    });

    test('setThemeMode(light) 会更新状态并通知监听器', () {
      final provider = ThemeProvider();
      var notified = false;
      provider.addListener(() {
        notified = true;
      });

      provider.setThemeMode(ThemeMode.light);

      expect(provider.themeMode, ThemeMode.light);
      expect(notified, isTrue);
    });

    test('setThemeMode(dark) 会更新状态并通知监听器', () {
      final provider = ThemeProvider();
      var notified = false;
      provider.addListener(() {
        notified = true;
      });

      provider.setThemeMode(ThemeMode.dark);

      expect(provider.themeMode, ThemeMode.dark);
      expect(notified, isTrue);
    });

    test('连续设置相同模式不会重复通知', () {
      final provider = ThemeProvider();
      var calls = 0;
      provider.addListener(() {
        calls++;
      });

      provider.setThemeMode(ThemeMode.system);

      expect(calls, 0);
    });

    test('themeMode getter 返回当前模式', () {
      final provider = ThemeProvider();

      provider.setThemeMode(ThemeMode.dark);

      expect(provider.themeMode, ThemeMode.dark);
    });
  });
}
