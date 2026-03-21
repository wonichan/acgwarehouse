// test/utils/responsive_breakpoint_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/utils/responsive_breakpoint.dart';

void main() {
  group('ResponsiveBreakpoint', () {
    test('returns compact for width <= 600', () {
      expect(ResponsiveBreakpoint.getBreakpoint(600), Breakpoint.compact);
      expect(ResponsiveBreakpoint.getBreakpoint(599), Breakpoint.compact);
      expect(ResponsiveBreakpoint.getBreakpoint(0), Breakpoint.compact);
    });

    test('returns medium for width between 600 and 840', () {
      expect(ResponsiveBreakpoint.getBreakpoint(601), Breakpoint.medium);
      expect(ResponsiveBreakpoint.getBreakpoint(700), Breakpoint.medium);
      expect(ResponsiveBreakpoint.getBreakpoint(840), Breakpoint.medium);
    });

    test('returns expanded for width > 840', () {
      expect(ResponsiveBreakpoint.getBreakpoint(841), Breakpoint.expanded);
      expect(ResponsiveBreakpoint.getBreakpoint(1200), Breakpoint.expanded);
    });

    test('helper methods work correctly', () {
      expect(ResponsiveBreakpoint.isCompact(500), true);
      expect(ResponsiveBreakpoint.isCompact(700), false);
      expect(ResponsiveBreakpoint.isMedium(700), true);
      expect(ResponsiveBreakpoint.isExpanded(900), true);
    });

    test('getGridColumns returns correct values', () {
      expect(ResponsiveBreakpoint.getGridColumns(Breakpoint.compact), 2);
      expect(ResponsiveBreakpoint.getGridColumns(Breakpoint.medium), 3);
      expect(ResponsiveBreakpoint.getGridColumns(Breakpoint.expanded), 4);
    });

    test('getGridSpacing returns correct values', () {
      expect(ResponsiveBreakpoint.getGridSpacing(Breakpoint.compact), 4.0);
      expect(ResponsiveBreakpoint.getGridSpacing(Breakpoint.medium), 8.0);
      expect(ResponsiveBreakpoint.getGridSpacing(Breakpoint.expanded), 12.0);
    });
  });
}