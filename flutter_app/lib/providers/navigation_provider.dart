import 'package:flutter/foundation.dart';

/// 全局导航状态管理
/// 管理 FluentApp 和 MaterialApp 共享的导航索引
/// Updated for Android 3-item navigation (09-01)
class NavigationProvider extends ChangeNotifier {
  // Navigation indices for easy reference
  static const int galleryIndex = 0;
  static const int searchIndex = 1;
  static const int tagManagementIndex = 2;

  // Total number of navigation items
  static const int itemCount = 3;

  static const List<String> _pageTitles = [
    '图库',
    '搜索',
    '标签管理',
  ];

  int _selectedIndex = 0;

  int get selectedIndex => _selectedIndex;

  String get currentPageTitle => _pageTitles[_selectedIndex];

  void setSelectedIndex(int index) {
    if (index < 0 || index >= itemCount) {
      throw RangeError('Invalid navigation index: $index');
    }
    if (_selectedIndex != index) {
      _selectedIndex = index;
      notifyListeners();
    }
  }
}
