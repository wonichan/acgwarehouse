// lib/utils/responsive_breakpoint.dart
/// Material Design 3 standard breakpoints
/// https://m3.material.io/foundations/layout/applying-layout/window-size-classes
enum Breakpoint {
  compact,   // 0-600dp - phones
  medium,    // 600-840dp - tablets
  expanded,  // >840dp - large tablets/desktop
}

/// Utility class for responsive breakpoint calculations
class ResponsiveBreakpoint {
  static const double compactMax = 600;
  static const double mediumMax = 840;

  /// Returns the breakpoint category for given width
  static Breakpoint getBreakpoint(double width) {
    if (width <= compactMax) return Breakpoint.compact;
    if (width <= mediumMax) return Breakpoint.medium;
    return Breakpoint.expanded;
  }

  /// Returns true if width is in compact range (phones)
  static bool isCompact(double width) => getBreakpoint(width) == Breakpoint.compact;

  /// Returns true if width is in medium range (tablets)
  static bool isMedium(double width) => getBreakpoint(width) == Breakpoint.medium;

  /// Returns true if width is in expanded range (large tablets)
  static bool isExpanded(double width) => getBreakpoint(width) == Breakpoint.expanded;

  /// Returns true if width is >= 600 (tablet or larger - for NavigationRail)
  static bool isTabletOrLarger(double width) => width >= compactMax;

  /// Returns recommended grid columns for breakpoint
  static int getGridColumns(Breakpoint breakpoint) {
    return switch (breakpoint) {
      Breakpoint.compact => 2,
      Breakpoint.medium => 3,
      Breakpoint.expanded => 4,
    };
  }

  /// Returns recommended grid spacing for breakpoint
  static double getGridSpacing(Breakpoint breakpoint) {
    return switch (breakpoint) {
      Breakpoint.compact => 4.0,
      Breakpoint.medium => 8.0,
      Breakpoint.expanded => 12.0,
    };
  }
}