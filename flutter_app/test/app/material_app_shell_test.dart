import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/material_app_shell.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:provider/provider.dart';

void main() {
  group('MaterialAppShell', () {
    Widget createTestWidget() {
      return MultiProvider(
        providers: [
          ChangeNotifierProvider(create: (_) => NavigationProvider()),
          ChangeNotifierProvider(create: (_) => ImageListProvider(ApiService())),
          ChangeNotifierProvider(create: (_) => TagProvider(TagService())),
          ChangeNotifierProvider(create: (_) => SelectionProvider()),
        ],
        child: const MaterialApp(
          home: MaterialAppShell(),
        ),
      );
    }

    testWidgets('shows NavigationBar on compact screen', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationBar), findsOneWidget);
    });

    testWidgets('shows 3 navigation destinations', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationDestination), findsNWidgets(3));
    });

    testWidgets('shows correct navigation labels', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.text('图库'), findsOneWidget);
      expect(find.text('搜索'), findsOneWidget);
      expect(find.text('标签管理'), findsOneWidget);
    });
  });
}