import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:gallery/app/fluent_app_shell.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/log_viewer_provider.dart';
import 'package:gallery/providers/monitoring_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/services/monitoring_service.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/collection_service.dart';
import 'package:gallery/services/log_stream_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/fluent_settings_page.dart';

class _ShellMonitoringProvider extends MonitoringProvider {
  _ShellMonitoringProvider(http.Client client)
    : super(
        service: MonitoringService(client: client),
        wsUriFactory: () =>
            Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
      );

  @override
  Future<void> connect() async {}

  @override
  Future<void> disconnect() async {}
}

class _ShellLogViewerProvider extends LogViewerProvider {
  _ShellLogViewerProvider(http.Client client)
    : super(
        service: LogStreamService(client: client),
        wsUriFactory: ({required source, tail = 200}) => Uri.parse(
          'ws://localhost:8080/admin/api/logs/stream?source=${source.name}&tail=$tail',
        ),
      );

  @override
  Future<void> connect() async {}

  @override
  Future<void> disconnect() async {}
}

void main() {
  testWidgets('FluentAppShell exposes 收藏 navigation and matching pages', (
    tester,
  ) async {
    final navProvider = NavigationProvider();
    final mockClient = MockClient((request) async {
      final path = request.url.path;
      if (path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (path.endsWith('/api/v1/tags/stats')) {
        return http.Response('{"stats":[]}', 200);
      }
      if (path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      if (path.endsWith('/api/v1/collections')) {
        return http.Response(
          '{"collections":[{"id":1,"name":"默认合集","description":null,"cover_image_id":null,"image_count":1,"created_at":"2026-04-07T00:00:00.000Z","updated_at":"2026-04-07T00:00:00.000Z"}]}',
          200,
        );
      }
      if (path.endsWith('/api/v1/collections/1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });
    final collectionService = CollectionService(client: mockClient);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<NavigationProvider>.value(value: navProvider),
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => ImageListProvider(ApiService(client: mockClient)),
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
          ),
          ChangeNotifierProvider<SearchProvider>(
            create: (_) =>
                SearchProvider(service: SearchService(client: mockClient)),
          ),
          ChangeNotifierProvider<MonitoringProvider>(
            create: (_) => _ShellMonitoringProvider(mockClient),
          ),
          ChangeNotifierProvider<LogViewerProvider>(
            create: (_) => _ShellLogViewerProvider(mockClient),
          ),
          ChangeNotifierProvider<ThemeProvider>(create: (_) => ThemeProvider()),
          ChangeNotifierProvider<ConfigProvider>(
            create: (_) => ConfigProvider(),
          ),
        ],
        child: fluent.FluentApp(
          home: FluentAppShell(collectionService: collectionService),
        ),
      ),
    );

    expect(find.byType(FluentGalleryPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);
    expect(find.byIcon(fluent.FluentIcons.filter), findsNothing);
    expect(NavigationProvider.itemCount, 7);

    navProvider.setSelectedIndex(1);
    await tester.pumpAndSettle();
    expect(find.byType(FluentSearchPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);

    navProvider.setSelectedIndex(2);
    await tester.pumpAndSettle();
    expect(find.byType(FluentTagManagementPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);

    navProvider.setSelectedIndex(3);
    await tester.pumpAndSettle();
    expect(find.byType(FluentSettingsPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);

    expect(find.byIcon(fluent.FluentIcons.diagnostic), findsOneWidget);

    navProvider.setSelectedIndex(4);
    await tester.pumpAndSettle();
    expect(find.byType(FluentOperationsMonitoringPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);

    navProvider.setSelectedIndex(5);
    await tester.pumpAndSettle();
    expect(find.byType(FluentLogViewerPage), findsOneWidget);
    expect(find.text('搜索图片和标签'), findsOneWidget);

    navProvider.setSelectedIndex(6);
    await tester.pumpAndSettle();
    expect(find.byType(FluentCollectionsPage), findsOneWidget);
    expect(navProvider.currentPageTitle, '收藏');
    expect(find.text('ACGWarehouse - 收藏'), findsOneWidget);
  });
}
