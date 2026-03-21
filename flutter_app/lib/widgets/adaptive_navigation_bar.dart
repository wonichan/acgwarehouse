import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/navigation_provider.dart';

/// Bottom navigation bar for phone screens (< 600px).
///
/// Displays 3 navigation items:
/// - Gallery (图库)
/// - Search (搜索)
/// - Tag Management (标签管理)
///
/// Usage:
/// ```dart
/// Scaffold(
///   bottomNavigationBar: const AdaptiveNavigationBar(),
///   body: // content
/// )
/// ```
class AdaptiveNavigationBar extends StatelessWidget {
  const AdaptiveNavigationBar({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationBar(
          selectedIndex: navProvider.selectedIndex,
          onDestinationSelected: (index) {
            navProvider.setSelectedIndex(index);
          },
          destinations: const [
            NavigationDestination(
              icon: Icon(Icons.photo_library_outlined),
              selectedIcon: Icon(Icons.photo_library),
              label: '图库',
            ),
            NavigationDestination(
              icon: Icon(Icons.search_outlined),
              selectedIcon: Icon(Icons.search),
              label: '搜索',
            ),
            NavigationDestination(
              icon: Icon(Icons.label_outlined),
              selectedIcon: Icon(Icons.label),
              label: '标签管理',
            ),
          ],
        );
      },
    );
  }
}