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

    test('currentPageTitle 随索引切换更新 (7-item navigation)', () {
      final provider = NavigationProvider();

      // Index 0: Gallery
      expect(provider.currentPageTitle, equals('图库'));

      // Index 1: Search
      provider.setSelectedIndex(1);
      expect(provider.currentPageTitle, equals('搜索'));

      // Index 2: Tag Management
      provider.setSelectedIndex(2);
      expect(provider.currentPageTitle, equals('标签管理'));

      // Index 3: Settings
      provider.setSelectedIndex(3);
      expect(provider.currentPageTitle, equals('设置'));

      // Index 4: Operations monitoring
      provider.setSelectedIndex(4);
      expect(provider.currentPageTitle, equals('运营监控'));

      // Index 5: Log viewer
      provider.setSelectedIndex(5);
      expect(provider.currentPageTitle, equals('日志终端'));

      // Index 6: Collections
      provider.setSelectedIndex(6);
      expect(provider.currentPageTitle, equals('收藏'));
    });

    test('throws error for invalid index', () {
      final provider = NavigationProvider();

      expect(() => provider.setSelectedIndex(7), throwsRangeError);
      expect(() => provider.setSelectedIndex(-1), throwsRangeError);
    });

    test('navigation indices constants are correct', () {
      expect(NavigationProvider.galleryIndex, 0);
      expect(NavigationProvider.searchIndex, 1);
      expect(NavigationProvider.tagManagementIndex, 2);
      expect(NavigationProvider.settingsIndex, 3);
      expect(NavigationProvider.operationsMonitoringIndex, 4);
      expect(NavigationProvider.logViewerIndex, 5);
      expect(NavigationProvider.collectionsIndex, 6);
    });

    test('itemCount is 7', () {
      expect(NavigationProvider.itemCount, 7);
    });
  });
}
