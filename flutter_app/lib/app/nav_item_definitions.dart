import 'package:flutter/material.dart';

/// Canonical navigation item definition shared across Material shells.
/// Defines icon pairs and label once; NavigationBar and NavigationRail
/// each consume this list to build their own destination widgets.
class NavItemDefinition {
  final IconData icon;
  final IconData selectedIcon;
  final String label;

  const NavItemDefinition({
    required this.icon,
    required this.selectedIcon,
    required this.label,
  });
}

/// Core navigation items used by both NavigationBar (phone) and
/// NavigationRail (tablet). Desktop Fluent shell maintains its own
/// 7-item list with platform-specific bodies.
const List<NavItemDefinition> coreNavigationItems = [
  NavItemDefinition(
    icon: Icons.photo_library_outlined,
    selectedIcon: Icons.photo_library,
    label: '图库',
  ),
  NavItemDefinition(
    icon: Icons.search_outlined,
    selectedIcon: Icons.search,
    label: '搜索',
  ),
  NavItemDefinition(
    icon: Icons.label_outlined,
    selectedIcon: Icons.label,
    label: '标签管理',
  ),
  NavItemDefinition(
    icon: Icons.settings_outlined,
    selectedIcon: Icons.settings,
    label: '设置',
  ),
];
