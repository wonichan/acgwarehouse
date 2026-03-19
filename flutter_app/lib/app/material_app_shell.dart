import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/navigation_provider.dart';
import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/duplicate_screen.dart';

/// MaterialApp Shell - Android/Web 平台
/// 使用 NavigationProvider 管理导航状态
class MaterialAppShell extends StatelessWidget {
  const MaterialAppShell({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        final screens = <Widget>[
          const GalleryScreen(),
          const SearchScreen(),
          const DuplicateScreen(),
        ];

        return Scaffold(
          body: screens[navProvider.selectedIndex],
          bottomNavigationBar: NavigationBar(
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
                icon: Icon(Icons.content_copy_outlined),
                selectedIcon: Icon(Icons.content_copy),
                label: '重复检测',
              ),
            ],
          ),
        );
      },
    );
  }
}