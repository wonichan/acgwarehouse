import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/navigation_provider.dart';
import '../utils/responsive_breakpoint.dart';
import '../widgets/breakpoint_observer.dart';
import '../widgets/adaptive_navigation_bar.dart';
import '../widgets/adaptive_navigation_rail.dart';
import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/tag_management_screen.dart';
import '../screens/settings_screen.dart';

/// MaterialApp Shell - Android/Web 平台
///
/// Features:
/// - NavigationBar on phones (< 600px)
/// - NavigationRail on tablets (>= 600px)
/// - 4-item navigation: Gallery, Search, Tag Management, Settings
/// - Responsive layout with smooth transitions
/// - Fully driven by Material 3 theme colors
///
/// Uses BreakpointObserver from 09-05 for responsive detection.
class MaterialAppShell extends StatelessWidget {
  const MaterialAppShell({super.key});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return BreakpointObserver(
      builder: (context, breakpoint) {
        final isCompact = breakpoint == Breakpoint.compact;

        return Consumer<NavigationProvider>(
          builder: (context, navProvider, child) {
            const screens = <Widget>[
              GalleryScreen(),
              SearchScreen(),
              TagManagementScreen(),
              SettingsScreen(),
            ];

            final content = screens[navProvider.selectedIndex];

            // Responsive layout: NavigationBar vs NavigationRail
            if (isCompact) {
              // Phone: Bottom navigation
              return Scaffold(
                backgroundColor: colorScheme.surface,
                body: content,
                bottomNavigationBar: const AdaptiveNavigationBar(),
              );
            } else {
              // Tablet: Side navigation
              return Scaffold(
                backgroundColor: colorScheme.surface,
                body: Row(
                  children: [
                    const AdaptiveNavigationRail(),
                    VerticalDivider(
                      width: 1,
                      thickness: 1,
                      color: colorScheme.outlineVariant,
                    ),
                    Expanded(child: content),
                  ],
                ),
              );
            }
          },
        );
      },
    );
  }
}
