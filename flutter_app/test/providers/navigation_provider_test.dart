import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/navigation_provider.dart';

void main() {
  group('NavigationProvider - 5 item navigation', () {
    late NavigationProvider provider;

    setUp(() {
      provider = NavigationProvider();
    });

    test('has correct 5 page titles', () {
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
    });

    test('throws error for invalid index', () {
      expect(() => provider.setSelectedIndex(5), throwsRangeError);
      expect(() => provider.setSelectedIndex(-1), throwsRangeError);
    });

    test('navigation indices are correct', () {
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
