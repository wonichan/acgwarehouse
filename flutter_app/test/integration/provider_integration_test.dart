import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/duplicate_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/duplicate_service.dart';

void main() {
  group('Provider Integration Tests', () {
    testWidgets('NavigationProvider 状态在 Material 导航中正常工作',
        (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider(create: (_) => NavigationProvider()),
          ],
          child: MaterialApp(
            home: Builder(
              builder: (context) {
                final navProvider = context.watch<NavigationProvider>();
                return Scaffold(
                  body: Text('Page ${navProvider.selectedIndex}'),
                  bottomNavigationBar: NavigationBar(
                    selectedIndex: navProvider.selectedIndex,
                    onDestinationSelected: (index) {
                      navProvider.setSelectedIndex(index);
                    },
                    destinations: const [
                      NavigationDestination(
                        icon: Icon(Icons.photo_library_outlined),
                        label: '图库',
                      ),
                      NavigationDestination(
                        icon: Icon(Icons.search_outlined),
                        label: '搜索',
                      ),
                      NavigationDestination(
                        icon: Icon(Icons.content_copy_outlined),
                        label: '重复检测',
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ),
      );

      // 验证初始状态
      expect(find.text('Page 0'), findsOneWidget);

      // 点击第二个导航项
      await tester.tap(find.text('搜索'));
      await tester.pumpAndSettle();

      expect(find.text('Page 1'), findsOneWidget);
    });

    testWidgets('NavigationProvider 状态在 Fluent 导航中正常工作',
        (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider(create: (_) => NavigationProvider()),
          ],
          child: fluent.FluentApp(
            home: Builder(
              builder: (context) {
                final navProvider = context.watch<NavigationProvider>();
                return fluent.NavigationView(
                  pane: fluent.NavigationPane(
                    selected: navProvider.selectedIndex,
                    onChanged: (index) {
                      navProvider.setSelectedIndex(index);
                    },
                    displayMode: fluent.PaneDisplayMode.auto,
                    items: [
                      fluent.PaneItem(
                        icon: const Icon(fluent.FluentIcons.photo2),
                        title: const Text('图库'),
                        body: Text('Page ${navProvider.selectedIndex}'),
                      ),
                      fluent.PaneItem(
                        icon: const Icon(fluent.FluentIcons.search),
                        title: const Text('搜索'),
                        body: Text('Page ${navProvider.selectedIndex}'),
                      ),
                      fluent.PaneItem(
                        icon: const Icon(fluent.FluentIcons.copy),
                        title: const Text('重复检测'),
                        body: Text('Page ${navProvider.selectedIndex}'),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ),
      );

      // 验证初始状态
      expect(find.text('Page 0'), findsOneWidget);
    });

    testWidgets('ImageListProvider 可被创建和访问', (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            Provider(create: (_) => ApiService()),
            ChangeNotifierProvider(
              create: (context) =>
                  ImageListProvider(context.read<ApiService>()),
            ),
          ],
          child: MaterialApp(
            home: Builder(
              builder: (context) {
                final provider = context.watch<ImageListProvider>();
                return Text('Images: ${provider.images.length}');
              },
            ),
          ),
        ),
      );

      expect(find.textContaining('Images:'), findsOneWidget);
    });

    testWidgets('所有 Provider 可同时初始化', (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            Provider(create: (_) => ApiService()),
            Provider(create: (_) => TagService()),
            Provider(create: (_) => DuplicateService()),
            Provider(create: (_) => SearchService()),
            ChangeNotifierProvider(
              create: (context) =>
                  ImageListProvider(context.read<ApiService>()),
            ),
            ChangeNotifierProvider(
              create: (context) => TagProvider(context.read<TagService>()),
            ),
            ChangeNotifierProvider(
              create: (context) =>
                  DuplicateProvider(service: context.read<DuplicateService>()),
            ),
            ChangeNotifierProvider(
              create: (context) =>
                  SearchProvider(service: context.read<SearchService>()),
            ),
            ChangeNotifierProvider(create: (_) => NavigationProvider()),
          ],
          child: const MaterialApp(
            home: Scaffold(body: Text('All Providers Ready')),
          ),
        ),
      );

      expect(find.text('All Providers Ready'), findsOneWidget);
    });
  });
}