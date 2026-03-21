import 'package:flutter/material.dart';
import '../utils/responsive_breakpoint.dart';

/// Builder function type for BreakpointObserver
typedef BreakpointWidgetBuilder = Widget Function(BuildContext context, Breakpoint breakpoint);

/// Widget that observes screen size and provides breakpoint information to children.
/// 
/// Usage:
/// ```dart
/// BreakpointObserver(
///   builder: (context, breakpoint) {
///     final isTablet = breakpoint != Breakpoint.compact;
///     return isTablet ? NavigationRail(...) : NavigationBar(...);
///   },
/// )
/// ```
class BreakpointObserver extends StatelessWidget {
  final BreakpointWidgetBuilder builder;

  const BreakpointObserver({
    super.key,
    required this.builder,
  });

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final breakpoint = ResponsiveBreakpoint.getBreakpoint(constraints.maxWidth);
        return builder(context, breakpoint);
      },
    );
  }
}

/// Extension to easily access breakpoint from BuildContext
extension BreakpointContext on BuildContext {
  /// Gets the current breakpoint based on screen width
  Breakpoint get breakpoint {
    final width = MediaQuery.of(this).size.width;
    return ResponsiveBreakpoint.getBreakpoint(width);
  }

  /// True if current screen is compact (phone)
  bool get isCompact => breakpoint == Breakpoint.compact;

  /// True if current screen is medium or larger (tablet+)
  bool get isTabletOrLarger => breakpoint != Breakpoint.compact;
}