import 'package:flutter/foundation.dart';

/// 全局导航状态管理
/// 管理 FluentApp 和 MaterialApp 共享的导航索引
class NavigationProvider extends ChangeNotifier {
  int _selectedIndex = 0;

  int get selectedIndex => _selectedIndex;

  void setSelectedIndex(int index) {
    if (_selectedIndex != index) {
      _selectedIndex = index;
      notifyListeners();
    }
  }
}