import 'package:flutter/foundation.dart';

/// 全局导航状态管理
/// 管理 FluentApp 和 MaterialApp 共享的导航索引
class NavigationProvider extends ChangeNotifier {
  static const List<String> _pageTitles = [
    '图库',
    '重复检测',
    '搜索',
    '标签管理',
    '设置',
  ];

  int _selectedIndex = 0;

  int get selectedIndex => _selectedIndex;

  String get currentPageTitle => _pageTitles[_selectedIndex];

  void setSelectedIndex(int index) {
    if (_selectedIndex != index) {
      _selectedIndex = index;
      notifyListeners();
    }
  }
}
