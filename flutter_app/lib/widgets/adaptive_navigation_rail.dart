import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../app/nav_item_definitions.dart';
import '../providers/navigation_provider.dart';

/// Side navigation rail for tablet screens (>= 600px).
///
/// Features:
/// - 72px icon-only mode (standard Material Rail width)
/// - Shows label only for selected item
/// - Core items defined once in [coreNavigationItems]
/// - Smooth selection indicator animation
class AdaptiveNavigationRail extends StatelessWidget {
  const AdaptiveNavigationRail({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        final selectedIndex =
            navProvider.selectedIndex < coreNavigationItems.length
            ? navProvider.selectedIndex
            : NavigationProvider.galleryIndex;
        return NavigationRail(
          selectedIndex: selectedIndex,
          onDestinationSelected: navProvider.setSelectedIndex,
          labelType: NavigationRailLabelType.selected,
          minWidth: 72,
          destinations: coreNavigationItems
              .map(
                (item) => NavigationRailDestination(
                  icon: Icon(item.icon),
                  selectedIcon: Icon(item.selectedIcon),
                  label: Text(item.label),
                ),
              )
              .toList(),
        );
      },
    );
  }
}
