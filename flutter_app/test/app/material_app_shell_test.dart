import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/material_app_shell.dart';
import 'package:gallery/screens/settings_screen.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:provider/provider.dart';

void main() {
  group('MaterialAppShell', () {
    Widget createTestWidget() {
      return MultiProvider(
        providers: [
          ChangeNotifierProvider(create: (_) => NavigationProvider()),
          ChangeNotifierProvider(
            create: (_) => ImageListProvider(ApiService()),
          ),
          ChangeNotifierProvider(create: (_) => TagProvider(TagService())),
          ChangeNotifierProvider(create: (_) => SelectionProvider()),
          ChangeNotifierProvider(create: (_) => ThemeProvider()),
          ChangeNotifierProvider(create: (_) => ConfigProvider()),
        ],
        child: const MaterialApp(home: MaterialAppShell()),
      );
    }

    testWidgets('shows NavigationBar on compact screen', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.byType(NavigationRail), findsNothing);
    });

    testWidgets('shows NavigationRail on medium screen (tablet)', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.byType(NavigationBar), findsNothing);
    });

    testWidgets('shows NavigationRail on expanded screen', (tester) async {
      tester.view.physicalSize = const Size(1000, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.byType(NavigationBar), findsNothing);
    });

    testWidgets('shows correct navigation labels on phone', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.text('图库'), findsOneWidget);
      expect(find.text('搜索'), findsOneWidget);
      expect(find.text('标签管理'), findsOneWidget);
    });

    testWidgets('shows correct navigation labels on tablet', (tester) async {
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      // NavigationRail shows label for selected item
      expect(find.text('图库'), findsOneWidget);
    });

    testWidgets('has VerticalDivider on tablet layout', (tester) async {
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(VerticalDivider), findsOneWidget);
    });

    testWidgets('navigates to settings screen from navigation destination', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      await tester.tap(find.text('设置'));
      await tester.pumpAndSettle();

      expect(find.byType(SettingsScreen), findsOneWidget);
    });
  });
}
