import 'package:flutter/material.dart';

/// Widget that smoothly animates between navigation modes.
///
/// Provides fade transition when switching between:
/// - NavigationBar (compact/phone)
/// - NavigationRail (medium+/tablet)
///
/// Usage:
/// ```dart
/// NavigationModeSwitcher(
///   isCompact: isCompact,
///   compactChild: NavigationBar(...),
///   expandedChild: NavigationRail(...),
/// )
/// ```
class NavigationModeSwitcher extends StatelessWidget {
  final bool isCompact;
  final Widget compactChild;
  final Widget expandedChild;
  final Duration duration;

  const NavigationModeSwitcher({
    super.key,
    required this.isCompact,
    required this.compactChild,
    required this.expandedChild,
    this.duration = const Duration(milliseconds: 250),
  });

  @override
  Widget build(BuildContext context) {
    return AnimatedSwitcher(
      duration: duration,
      switchInCurve: Curves.easeInOut,
      switchOutCurve: Curves.easeInOut,
      transitionBuilder: (child, animation) {
        return FadeTransition(
          opacity: animation,
          child: child,
        );
      },
      child: isCompact ? compactChild : expandedChild,
    );
  }
}

/// Extension to wrap navigation widgets with smooth transitions.
extension NavigationTransitions on Widget {
  /// Wraps widget with fade transition.
  Widget withFadeTransition({
    Duration duration = const Duration(milliseconds: 250),
  }) {
    return AnimatedSwitcher(
      duration: duration,
      switchInCurve: Curves.easeInOut,
      switchOutCurve: Curves.easeInOut,
      transitionBuilder: (child, animation) =>
          FadeTransition(opacity: animation, child: child),
      child: this,
    );
  }
}