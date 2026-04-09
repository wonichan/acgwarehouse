import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../app/nav_item_definitions.dart';
import '../providers/navigation_provider.dart';

/// Bottom navigation bar for phone screens (< 600px).
///
/// Displays core navigation items defined once in [coreNavigationItems].
/// Driven entirely by Material 3 theme colors.
class AdaptiveNavigationBar extends StatelessWidget {
  const AdaptiveNavigationBar({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationBar(
          selectedIndex: navProvider.selectedIndex,
          onDestinationSelected: navProvider.setSelectedIndex,
          destinations: coreNavigationItems
              .map(
                (item) => NavigationDestination(
                  icon: Icon(item.icon),
                  selectedIcon: Icon(item.selectedIcon),
                  label: item.label,
                ),
              )
              .toList(),
        );
      },
    );
  }
}
