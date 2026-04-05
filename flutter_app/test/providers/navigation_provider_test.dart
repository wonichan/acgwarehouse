import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/config/api_config.dart';
import 'package:gallery/providers/navigation_provider.dart';

void main() {
  group('NavigationProvider - 6 item navigation', () {
    late NavigationProvider provider;

    setUp(() {
      provider = NavigationProvider();
    });

    test('has correct 6 page titles', () {
      expect(provider.currentPageTitle, '图库');

      provider.setSelectedIndex(0);
      expect(provider.currentPageTitle, '图库');

      provider.setSelectedIndex(1);
      expect(provider.currentPageTitle, '重复检测');

      provider.setSelectedIndex(2);
      expect(provider.currentPageTitle, '搜索');

      provider.setSelectedIndex(3);
      expect(provider.currentPageTitle, '标签管理');

      provider.setSelectedIndex(4);
      expect(provider.currentPageTitle, '设置');

      provider.setSelectedIndex(5);
      expect(provider.currentPageTitle, '运营监控');
    });

    test('throws error for invalid index', () {
      expect(() => provider.setSelectedIndex(6), throwsRangeError);
      expect(() => provider.setSelectedIndex(-1), throwsRangeError);
    });

    test('navigation indices are correct', () {
      expect(NavigationProvider.galleryIndex, 0);
      expect(NavigationProvider.duplicateIndex, 1);
      expect(NavigationProvider.searchIndex, 2);
      expect(NavigationProvider.tagManagementIndex, 3);
      expect(NavigationProvider.settingsIndex, 4);
      expect(NavigationProvider.operationsMonitoringIndex, 5);
    });

    test('itemCount is 6', () {
      expect(NavigationProvider.itemCount, 6);
    });
  });

  group('ApiConfig monitoring endpoints', () {
    tearDown(ApiConfig.resetToDefault);

    test('builds admin overview and monitoring websocket endpoints', () {
      ApiConfig.updateBaseUrl('http://127.0.0.1:8088');

      expect(
        ApiConfig.adminOverview,
        'http://127.0.0.1:8088/admin/api/task-platform/overview',
      );
      expect(
        ApiConfig.monitoringWs,
        'ws://127.0.0.1:8088/admin/api/monitoring/ws',
      );
    });
  });
}
