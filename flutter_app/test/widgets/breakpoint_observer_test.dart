import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/breakpoint_observer.dart';
import 'package:gallery/utils/responsive_breakpoint.dart';

void main() {
  group('BreakpointObserver', () {
    testWidgets('provides compact breakpoint on small screen', (tester) async {
      Breakpoint? capturedBreakpoint;

      // Set test view size to compact (phone)
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(
        MaterialApp(
          home: BreakpointObserver(
            builder: (context, breakpoint) {
              capturedBreakpoint = breakpoint;
              return Container();
            },
          ),
        ),
      );

      expect(capturedBreakpoint, Breakpoint.compact);
    });

    testWidgets('provides medium breakpoint on tablet screen', (tester) async {
      Breakpoint? capturedBreakpoint;

      // Set test view size to medium (tablet)
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.reset());

      await tester.pumpWidget(
        MaterialApp(
          home: BreakpointObserver(
            builder: (context, breakpoint) {
              capturedBreakpoint = breakpoint;
              return Container();
            },
          ),
        ),
      );

      expect(capturedBreakpoint, Breakpoint.medium);
    });

    testWidgets('rebuilds when size changes', (tester) async {
      int buildCount = 0;

      // Start with compact size
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;

      await tester.pumpWidget(
        MaterialApp(
          home: BreakpointObserver(
            builder: (context, breakpoint) {
              buildCount++;
              return Text(breakpoint.toString());
            },
          ),
        ),
      );

      expect(buildCount, 1);

      // Change to tablet size
      tester.view.physicalSize = const Size(700, 1000);
      await tester.pump();

      expect(buildCount, 2);

      addTearDown(() => tester.view.reset());
    });
  });
}