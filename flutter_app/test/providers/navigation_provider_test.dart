import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/config/api_config.dart';
import 'package:gallery/providers/navigation_provider.dart';

void main() {
  group('NavigationProvider - 7 item navigation', () {
    late NavigationProvider provider;

    setUp(() {
      provider = NavigationProvider();
    });

    test('has correct 7 page titles', () {
      expect(provider.currentPageTitle, '图库');

      provider.setSelectedIndex(0);
      expect(provider.currentPageTitle, '图库');

      provider.setSelectedIndex(1);
      expect(provider.currentPageTitle, '搜索');

      provider.setSelectedIndex(2);
      expect(provider.currentPageTitle, '标签管理');

      provider.setSelectedIndex(3);
      expect(provider.currentPageTitle, '设置');

      provider.setSelectedIndex(4);
      expect(provider.currentPageTitle, '运营监控');

      provider.setSelectedIndex(5);
      expect(provider.currentPageTitle, '日志终端');

      provider.setSelectedIndex(6);
      expect(provider.currentPageTitle, '收藏');
    });

    test('throws error for invalid index', () {
      expect(() => provider.setSelectedIndex(7), throwsRangeError);
      expect(() => provider.setSelectedIndex(-1), throwsRangeError);
    });

    test('navigation indices are correct', () {
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

  group('ApiConfig monitoring endpoints', () {
    test('builds admin overview and monitoring websocket endpoints', () {
      const testBase = 'http://127.0.0.1:8088';
      expect(
        ApiConfig.adminOverview(testBase),
        'http://127.0.0.1:8088/admin/api/task-platform/overview',
      );
      expect(
        ApiConfig.monitoringWs(testBase),
        'ws://127.0.0.1:8088/admin/api/monitoring/ws',
      );
    });
  });
}
