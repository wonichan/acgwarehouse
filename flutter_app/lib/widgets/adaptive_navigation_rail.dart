import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/navigation_provider.dart';

/// Side navigation rail for tablet screens (>= 600px).
///
/// Features:
/// - 72px icon-only mode (standard Material Rail width)
/// - Shows label only for selected item
/// - 4 navigation items: Gallery, Search, Tag Management, Settings
/// - Smooth selection indicator animation
///
/// Usage:
/// ```dart
/// Scaffold(
///   body: Row(
///     children: [
///       const AdaptiveNavigationRail(),
///       Expanded(child: content),
///     ],
///   ),
/// )
/// ```
class AdaptiveNavigationRail extends StatelessWidget {
  const AdaptiveNavigationRail({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationRail(
          selectedIndex: navProvider.selectedIndex,
          onDestinationSelected: (index) {
            navProvider.setSelectedIndex(index);
          },
          labelType: NavigationRailLabelType.selected,
          minWidth: 72,
          destinations: const [
            NavigationRailDestination(
              icon: Icon(Icons.photo_library_outlined),
              selectedIcon: Icon(Icons.photo_library),
              label: Text('图库'),
            ),
            NavigationRailDestination(
              icon: Icon(Icons.search_outlined),
              selectedIcon: Icon(Icons.search),
              label: Text('搜索'),
            ),
            NavigationRailDestination(
              icon: Icon(Icons.label_outlined),
              selectedIcon: Icon(Icons.label),
              label: Text('标签管理'),
            ),
            NavigationRailDestination(
              icon: Icon(Icons.settings_outlined),
              selectedIcon: Icon(Icons.settings),
              label: Text('设置'),
            ),
          ],
        );
      },
    );
  }
}
