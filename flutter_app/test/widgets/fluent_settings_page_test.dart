import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/widgets/fluent_settings_page.dart';

void main() {
  testWidgets('FluentSettingsPage shows theme mode options', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final themeProvider = ThemeProvider();

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider.value(value: themeProvider),
          ChangeNotifierProvider(create: (_) => ConfigProvider()),
        ],
        child: const fluent.FluentApp(home: FluentSettingsPage()),
      ),
    );

    expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
    expect(find.text('设置'), findsWidgets);
    expect(find.text('外观'), findsOneWidget);
    expect(find.text('跟随系统'), findsOneWidget);
    expect(find.text('浅色'), findsOneWidget);
    expect(find.text('深色'), findsOneWidget);

    await tester.tap(find.text('浅色'));
    await tester.pumpAndSettle();

    expect(themeProvider.themeMode, ThemeMode.light);
  });
}
