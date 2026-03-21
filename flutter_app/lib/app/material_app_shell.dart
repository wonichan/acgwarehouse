import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/navigation_provider.dart';
import '../utils/responsive_breakpoint.dart';
import '../widgets/breakpoint_observer.dart';
import '../widgets/adaptive_navigation_bar.dart';
import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/tag_management_screen.dart';

/// MaterialApp Shell - Android/Web 平台
/// 
/// Features:
/// - NavigationBar on phones (< 600px)
/// - NavigationRail on tablets (>= 600px) - implemented in 09-02
/// - 3-item navigation: Gallery, Search, Tag Management
/// - Responsive layout with smooth transitions
///
/// Uses BreakpointObserver from 09-05 for responsive detection.
class MaterialAppShell extends StatelessWidget {
  const MaterialAppShell({super.key});

  @override
  Widget build(BuildContext context) {
    return BreakpointObserver(
      builder: (context, breakpoint) {
        final isCompact = breakpoint == Breakpoint.compact;
        
        return Consumer<NavigationProvider>(
          builder: (context, navProvider, child) {
            final screens = const <Widget>[
              GalleryScreen(),
              SearchScreen(),
              TagManagementScreen(),
            ];

            return Scaffold(
              body: screens[navProvider.selectedIndex],
              bottomNavigationBar: isCompact 
                  ? const AdaptiveNavigationBar() 
                  : null,
              // NavigationRail will be added in 09-02
            );
          },
        );
      },
    );
  }
}