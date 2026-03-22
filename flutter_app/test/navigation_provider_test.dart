import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/navigation_provider.dart';

void main() {
  group('NavigationProvider', () {
    test('初始 selectedIndex 为 0', () {
      final provider = NavigationProvider();
      expect(provider.selectedIndex, equals(0));
    });

    test('setSelectedIndex(2) 后 selectedIndex 变为 2', () {
      final provider = NavigationProvider();
      provider.setSelectedIndex(2);
      expect(provider.selectedIndex, equals(2));
    });

    test('setSelectedIndex 触发 notifyListeners', () {
      final provider = NavigationProvider();
      var notified = false;
      provider.addListener(() {
        notified = true;
      });
      
      provider.setSelectedIndex(1);
      expect(notified, isTrue);
    });

    test('相同索引不触发 notifyListeners', () {
      final provider = NavigationProvider();
      var callCount = 0;
      provider.addListener(() {
        callCount++;
      });
      
      provider.setSelectedIndex(0); // 相同索引
      expect(callCount, equals(0));
      
      provider.setSelectedIndex(1); // 不同索引
      expect(callCount, equals(1));
    });

    test('currentPageTitle 对应图库', () {
      final provider = NavigationProvider();
      expect(provider.currentPageTitle, equals('图库'));
    });

    test('currentPageTitle 随索引切换更新 (5-item navigation)', () {
      final provider = NavigationProvider();

      // Index 0: Gallery
      expect(provider.currentPageTitle, equals('图库'));

      // Index 1: Duplicate
      provider.setSelectedIndex(1);
      expect(provider.currentPageTitle, equals('重复检测'));

      // Index 2: Search
      provider.setSelectedIndex(2);
      expect(provider.currentPageTitle, equals('搜索'));

      // Index 3: Tag Management
      provider.setSelectedIndex(3);
      expect(provider.currentPageTitle, equals('标签管理'));

      // Index 4: Settings
      provider.setSelectedIndex(4);
      expect(provider.currentPageTitle, equals('设置'));
    });

    test('throws error for invalid index', () {
      final provider = NavigationProvider();
      
      expect(() => provider.setSelectedIndex(5), throwsRangeError);
      expect(() => provider.setSelectedIndex(-1), throwsRangeError);
    });

    test('navigation indices constants are correct', () {
      expect(NavigationProvider.galleryIndex, 0);
      expect(NavigationProvider.duplicateIndex, 1);
      expect(NavigationProvider.searchIndex, 2);
      expect(NavigationProvider.tagManagementIndex, 3);
      expect(NavigationProvider.settingsIndex, 4);
    });

    test('itemCount is 5', () {
      expect(NavigationProvider.itemCount, 5);
    });
  });
}
