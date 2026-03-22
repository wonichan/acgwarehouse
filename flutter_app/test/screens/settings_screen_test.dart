import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/screens/settings_screen.dart';

void main() {
  testWidgets('SettingsScreen shows radio theme options', (tester) async {
    final themeProvider = ThemeProvider();

    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: themeProvider,
        child: const MaterialApp(
          home: SettingsScreen(),
        ),
      ),
    );

    expect(find.text('设置'), findsOneWidget);
    expect(find.text('外观'), findsOneWidget);
    expect(find.text('跟随系统'), findsOneWidget);
    expect(find.text('浅色'), findsOneWidget);
    expect(find.text('深色'), findsOneWidget);
    expect(find.byType(RadioListTile), findsNWidgets(3));

    await tester.tap(find.text('深色'));
    await tester.pumpAndSettle();

    expect(themeProvider.themeMode, ThemeMode.dark);
  });
}
