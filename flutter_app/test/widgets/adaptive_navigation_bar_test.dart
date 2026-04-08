import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/adaptive_navigation_bar.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:provider/provider.dart';

void main() {
  group('AdaptiveNavigationBar', () {
    Widget createTestWidget({int selectedIndex = 0}) {
      return MaterialApp(
        home: ChangeNotifierProvider(
          create: (_) => NavigationProvider()..setSelectedIndex(selectedIndex),
          child: const Scaffold(body: AdaptiveNavigationBar()),
        ),
      );
    }

    testWidgets('displays 4 navigation destinations', (tester) async {
      await tester.pumpWidget(createTestWidget());

      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.byType(NavigationDestination), findsNWidgets(4));
    });

    testWidgets('shows correct labels', (tester) async {
      await tester.pumpWidget(createTestWidget());

      expect(find.text('图库'), findsOneWidget);
      expect(find.text('搜索'), findsOneWidget);
      expect(find.text('标签管理'), findsOneWidget);
      expect(find.text('设置'), findsOneWidget);
    });

    testWidgets('highlights selected item', (tester) async {
      await tester.pumpWidget(createTestWidget(selectedIndex: 1));

      final navBar = tester.widget<NavigationBar>(find.byType(NavigationBar));
      expect(navBar.selectedIndex, 1);
    });

    testWidgets('navigates on tap', (tester) async {
      await tester.pumpWidget(createTestWidget());

      await tester.tap(find.text('搜索'));
      await tester.pump();

      final context = tester.element(find.byType(NavigationBar));
      final provider = context.read<NavigationProvider>();
      expect(provider.selectedIndex, 1);
    });
  });
}
