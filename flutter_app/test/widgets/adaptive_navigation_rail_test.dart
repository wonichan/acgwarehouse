import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/adaptive_navigation_rail.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:provider/provider.dart';

void main() {
  group('AdaptiveNavigationRail', () {
    Widget createTestWidget({int selectedIndex = 0}) {
      return MaterialApp(
        home: ChangeNotifierProvider(
          create: (_) => NavigationProvider()..setSelectedIndex(selectedIndex),
          child: const Scaffold(body: AdaptiveNavigationRail()),
        ),
      );
    }

    testWidgets('displays 4 navigation destinations', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pumpAndSettle();

      expect(find.byType(NavigationRail), findsOneWidget);
      // Check for the icons that represent each destination
      // NavigationRail shows selected icon for selected item, outlined for others
      expect(find.byIcon(Icons.photo_library), findsOneWidget); // Selected icon
      expect(find.byIcon(Icons.search_outlined), findsOneWidget);
      expect(find.byIcon(Icons.label_outlined), findsOneWidget);
      expect(find.byIcon(Icons.settings_outlined), findsOneWidget);
    });

    testWidgets('has 72px width in icon-only mode', (tester) async {
      await tester.pumpWidget(createTestWidget());

      final rail = tester.widget<NavigationRail>(find.byType(NavigationRail));
      expect(rail.minWidth, 72);
      expect(rail.minExtendedWidth, isNull);
    });

    testWidgets('shows labels in selected item', (tester) async {
      await tester.pumpWidget(createTestWidget(selectedIndex: 1));

      final rail = tester.widget<NavigationRail>(find.byType(NavigationRail));
      expect(rail.labelType, NavigationRailLabelType.selected);
    });

    testWidgets('navigates on tap', (tester) async {
      await tester.pumpWidget(createTestWidget());

      // Find and tap second destination (Search)
      await tester.tap(find.byIcon(Icons.search_outlined));
      await tester.pump();

      final context = tester.element(find.byType(NavigationRail));
      final provider = context.read<NavigationProvider>();
      expect(provider.selectedIndex, 1);
    });
  });
}
