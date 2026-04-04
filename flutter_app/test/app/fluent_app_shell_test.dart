import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:gallery/app/fluent_app_shell.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/duplicate_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/fluent_settings_page.dart';

void main() {
  testWidgets(
    'FluentAppShell exposes five navigation items and matching pages',
    (tester) async {
      final navProvider = NavigationProvider();
      final mockClient = MockClient((request) async {
        final path = request.url.path;
        if (path.endsWith('/api/v1/images')) {
          return http.Response('{"images":[],"total":0,"has_more":false}', 200);
        }
        if (path.endsWith('/api/v1/duplicates')) {
          return http.Response('{"groups":[]}', 200);
        }
        if (path.contains('/api/v1/duplicates/detect')) {
          return http.Response('{"message":"ok","groups_found":0}', 200);
        }
        if (path.endsWith('/api/v1/tags/stats')) {
          return http.Response('{"stats":[]}', 200);
        }
        if (path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<NavigationProvider>.value(
              value: navProvider,
            ),
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
            ChangeNotifierProvider<DuplicateProvider>(
              create: (_) => DuplicateProvider(
                service: DuplicateService(client: mockClient),
              ),
            ),
            ChangeNotifierProvider<ThemeProvider>(
              create: (_) => ThemeProvider(),
            ),
            ChangeNotifierProvider<ConfigProvider>(
              create: (_) => ConfigProvider(),
            ),
          ],
          child: const fluent.FluentApp(home: FluentAppShell()),
        ),
      );

      expect(find.byType(FluentGalleryPage), findsOneWidget);
      expect(find.text('Search images and tags'), findsOneWidget);

      navProvider.setSelectedIndex(1);
      await tester.pumpAndSettle();
      expect(find.byType(FluentDuplicatePage), findsOneWidget);
      expect(find.text('Search images and tags'), findsOneWidget);

      navProvider.setSelectedIndex(2);
      await tester.pumpAndSettle();
      expect(find.byType(FluentSearchPage), findsOneWidget);
      expect(find.text('Search images and tags'), findsOneWidget);

      navProvider.setSelectedIndex(3);
      await tester.pumpAndSettle();
      expect(find.byType(FluentTagManagementPage), findsOneWidget);
      expect(find.text('Search images and tags'), findsOneWidget);

      navProvider.setSelectedIndex(4);
      await tester.pumpAndSettle();
      expect(find.byType(FluentSettingsPage), findsOneWidget);
      expect(find.text('Search images and tags'), findsOneWidget);
    },
  );
}
