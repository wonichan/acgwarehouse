import 'package:flutter/foundation.dart';

/// 全局导航状态管理
/// 管理 FluentApp 和 MaterialApp 共享的导航索引
class NavigationProvider extends ChangeNotifier {
  // Navigation indices for easy reference
  static const int galleryIndex = 0;
  static const int searchIndex = 1;
  static const int tagManagementIndex = 2;
  static const int settingsIndex = 3;
  static const int operationsMonitoringIndex = 4;
  static const int logViewerIndex = 5;
  static const int collectionsIndex = 6;

  static const int itemCount = 7;

  static const List<String> _pageTitles = [
    '图库',
    '搜索',
    '标签管理',
    '设置',
    '运营监控',
    '日志终端',
    '收藏',
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
