import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/navigation_mode_switcher.dart';

void main() {
  group('NavigationModeSwitcher', () {
    testWidgets('shows compact child when isCompact is true', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: NavigationModeSwitcher(
            isCompact: true,
            compactChild: const Text('Compact'),
            expandedChild: const Text('Expanded'),
          ),
        ),
      );

      expect(find.text('Compact'), findsOneWidget);
      expect(find.text('Expanded'), findsNothing);
    });

    testWidgets('shows expanded child when isCompact is false', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: NavigationModeSwitcher(
            isCompact: false,
            compactChild: const Text('Compact'),
            expandedChild: const Text('Expanded'),
          ),
        ),
      );

      expect(find.text('Compact'), findsNothing);
      expect(find.text('Expanded'), findsOneWidget);
    });

    testWidgets('animates transition on mode change', (tester) async {
      final ValueNotifier<bool> isCompact = ValueNotifier(true);

      await tester.pumpWidget(
        MaterialApp(
          home: ValueListenableBuilder<bool>(
            valueListenable: isCompact,
            builder: (context, compact, _) {
              return NavigationModeSwitcher(
                isCompact: compact,
                compactChild: const Text('Compact'),
                expandedChild: const Text('Expanded'),
                duration: const Duration(milliseconds: 100),
              );
            },
          ),
        ),
      );

      // Initial state
      expect(find.text('Compact'), findsOneWidget);

      // Change mode
      isCompact.value = false;
      await tester.pump();

      // Animation started
      expect(find.byType(FadeTransition), findsOneWidget);

      // Complete animation
      await tester.pumpAndSettle();
      expect(find.text('Expanded'), findsOneWidget);
    });
  });
}