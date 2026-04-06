import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/widgets/fluent_image_card.dart';
import 'package:gallery/widgets/tag_management/tag_management_workspace.dart';

void main() {
  testWidgets(
    'FluentTagManagementPage hosts TagManagementWorkspace in ScaffoldPage',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>(
              create: (_) => TagProvider(TagService(client: mockClient)),
            ),
          ],
          child: const fluent.FluentApp(home: FluentTagManagementPage()),
        ),
      );

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.byType(fluent.PageHeader), findsOneWidget);
      expect(find.text('Tag Governance'), findsOneWidget);
      expect(find.byType(TagManagementWorkspace), findsOneWidget);
    },
  );

  testWidgets(
    'FluentGalleryPage keeps page content but not shell-owned command actions',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/images')) {
          return http.Response('{"images":[],"total":0,"has_more":false}', 200);
        }
        if (request.url.path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      final imageProvider = ImageListProvider(ApiService(client: mockClient));

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => imageProvider,
            ),
            ChangeNotifierProvider<TagProvider>(
              create: (_) => TagProvider(TagService(client: mockClient)),
            ),
            ChangeNotifierProvider<NavigationProvider>(
              create: (_) => NavigationProvider(),
            ),
          ],
          child: const fluent.FluentApp(home: FluentGalleryPage()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.text('图库'), findsWidgets);
      expect(find.byIcon(fluent.FluentIcons.filter), findsNothing);

      await tester.tap(find.text('排序').first);
      await tester.pumpAndSettle();

      expect(find.text('源文件创建时间（新→旧）'), findsOneWidget);
      expect(find.text('源文件创建时间（旧→新）'), findsOneWidget);
      expect(find.text('源文件大小（大→小）'), findsOneWidget);
      expect(find.text('源文件大小（小→大）'), findsOneWidget);
      expect(find.text('源文件文件名（A-Z）'), findsOneWidget);
      expect(find.text('源文件文件名（Z-A）'), findsOneWidget);

      await tester.tap(find.text('源文件大小（小→大）'));
      await tester.pumpAndSettle();

      expect(imageProvider.sortField, SortField.fileSize);
      expect(imageProvider.sortAsc, isTrue);
    },
  );

  testWidgets(
    'FluentSearchPage still renders search body inside ScaffoldPage',
    (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<SearchProvider>(
              create: (_) => SearchProvider(
                service: SearchService(
                  client: MockClient((_) async => http.Response('{}', 200)),
                ),
              ),
            ),
          ],
          child: const fluent.FluentApp(home: FluentSearchPage()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.byType(fluent.TextBox), findsOneWidget);
    },
  );

  testWidgets('FluentGalleryPage opens in-window detail on image double tap', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      if (request.url.path.endsWith('/api/v1/images/1/tags')) {
        return http.Response(
          '{"confirmed":[],"pending":[],"rejected":[]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(ApiService(client: mockClient));
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => imageProvider,
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
          ),
          ChangeNotifierProvider<NavigationProvider>(
            create: (_) => NavigationProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: FluentGalleryPage()),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.byType(FluentImageCard), findsOneWidget);

    await tester.tap(find.byType(FluentImageCard));
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(find.byType(FluentImageCard));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('图片详情'), findsOneWidget);
  });
}
