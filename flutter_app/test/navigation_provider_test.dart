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
  });
}