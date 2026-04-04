import 'dart:convert';

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/fluent_app_shell.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/duplicate_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/providers/theme_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _RecordingSearchProvider extends SearchProvider {
  String? submittedQuery;

  _RecordingSearchProvider({required super.service});

  @override
  Future<void> search({
    String? query,
    List<int>? tagIds,
    String? sortBy,
    String? sortOrder,
    bool refresh = true,
  }) async {
    submittedQuery = query;
  }
}

void main() {
  group('Desktop shell top bar contract', () {
    late NavigationProvider navProvider;
    late _RecordingSearchProvider searchProvider;

    Widget createShell() {
      final mockClient = MockClient((request) async {
        final path = request.url.path;
        if (path.endsWith('/api/v1/images')) {
          return http.Response(
            jsonEncode({'images': [], 'total': 0, 'has_more': false}),
            200,
          );
        }

        if (path.endsWith('/api/v1/duplicates')) {
          return http.Response(jsonEncode({'groups': []}), 200);
        }

        if (path.contains('/api/v1/duplicates/detect')) {
          return http.Response(
            jsonEncode({'message': 'ok', 'groups_found': 0}),
            200,
          );
        }

        if (path.endsWith('/api/v1/tags/stats')) {
          return http.Response(jsonEncode({'stats': []}), 200);
        }

        if (path.endsWith('/api/v1/tags')) {
          return http.Response(jsonEncode({'tags': []}), 200);
        }

        return http.Response('{}', 200);
      });

      return MultiProvider(
        providers: [
          ChangeNotifierProvider<NavigationProvider>.value(value: navProvider),
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => ImageListProvider(ApiService(client: mockClient)),
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
          ),
          ChangeNotifierProvider<SearchProvider>.value(value: searchProvider),
          ChangeNotifierProvider<DuplicateProvider>(
            create: (_) => DuplicateProvider(
              service: DuplicateService(client: mockClient),
            ),
          ),
          ChangeNotifierProvider<ThemeProvider>(create: (_) => ThemeProvider()),
          ChangeNotifierProvider<ConfigProvider>(
            create: (_) => ConfigProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: FluentAppShell()),
      );
    }

    setUp(() {
      navProvider = NavigationProvider();
      searchProvider = _RecordingSearchProvider(
        service: SearchService(
          client: MockClient((_) async => http.Response('{}', 200)),
        ),
      );
    });

    testWidgets('renders persistent search box and shell actions', (
      tester,
    ) async {
      await tester.pumpWidget(createShell());
      await tester.pumpAndSettle();

      expect(find.text('Search images and tags'), findsOneWidget);
      expect(find.text('Import Library'), findsOneWidget);
      expect(find.text('Open Settings'), findsOneWidget);
    });

    testWidgets('submitting shell search navigates to search view', (
      tester,
    ) async {
      await tester.pumpWidget(createShell());
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(fluent.TextBox).first, '  rem cat  ');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();

      expect(searchProvider.submittedQuery, 'rem cat');
      expect(navProvider.selectedIndex, NavigationProvider.searchIndex);
    });

    testWidgets('opening settings from top bar navigates to settings page', (
      tester,
    ) async {
      await tester.pumpWidget(createShell());
      await tester.pumpAndSettle();

      await tester.tap(find.text('Open Settings'));
      await tester.pumpAndSettle();

      expect(navProvider.selectedIndex, NavigationProvider.settingsIndex);
    });
  });
}
