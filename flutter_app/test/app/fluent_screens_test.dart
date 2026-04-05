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

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => ImageListProvider(ApiService(client: mockClient)),
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
}
