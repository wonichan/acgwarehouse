import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart' show StatefulBuilder;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/gallery_filter_state.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/fluent_tag_filter_pane.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('FluentTagFilterPane loads roots and orphans lazily', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":true},{"id":2,"preferred_label":"props","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('tree/children')) {
        return http.Response(
          '{"items":[{"id":3,"preferred_label":"heroine","level":"child","parent_id":1,"has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response(
          '{"items":[],"total":0,"has_more":false}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: fluent.ScaffoldPage(
            content: FluentTagFilterPane(
              initialFilter: GalleryFilterState(),
              onApplyFilter: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Should show root tags with level badges
    expect(find.text('characters'), findsOneWidget);
    expect(find.text('props'), findsOneWidget);
    expect(find.text('R'), findsWidgets);
  });

  testWidgets('untagged toggle updates draft and waits for apply', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response(
          '{"items":[],"total":0,"has_more":false}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );
    GalleryFilterState? appliedFilter;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: StatefulBuilder(
            builder: (context, setState) {
              return fluent.ScaffoldPage(
                content: FluentTagFilterPane(
                  initialFilter: GalleryFilterState(exactTagIds: {1}),
                  onApplyFilter: (value) {
                    setState(() {
                      appliedFilter = value;
                    });
                  },
                ),
              );
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ToggleSwitch).first);
    await tester.pumpAndSettle();

    expect(appliedFilter, isNull);

    await tester.tap(find.text('应用筛选'));
    await tester.pumpAndSettle();

    expect(appliedFilter, isNotNull);
    expect(appliedFilter!.exactTagIds, {1});
    expect(appliedFilter!.hasTags, isFalse);
    expect(appliedFilter!.hasPendingTags, isNull);
  });

  testWidgets('pending then untagged keeps normal tags and clears pending', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response('{"items":[],"total":0,"has_more":false}', 200);
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );
    GalleryFilterState? appliedFilter;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: StatefulBuilder(
            builder: (context, setState) {
              return fluent.ScaffoldPage(
                content: FluentTagFilterPane(
                  initialFilter: GalleryFilterState(exactTagIds: {1}),
                  onApplyFilter: (value) {
                    setState(() {
                      appliedFilter = value;
                    });
                  },
                ),
              );
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final toggles = find.byType(fluent.ToggleSwitch);
    await tester.tap(toggles.at(1)); // pending on
    await tester.pumpAndSettle();
    await tester.tap(toggles.at(0)); // untagged on, pending off
    await tester.pumpAndSettle();

    await tester.tap(find.text('应用筛选'));
    await tester.pumpAndSettle();

    expect(appliedFilter, isNotNull);
    expect(appliedFilter!.exactTagIds, {1});
    expect(appliedFilter!.hasTags, isFalse);
    expect(appliedFilter!.hasPendingTags, isNull);
  });

  testWidgets('untagged then pending keeps normal tags and clears hasTags', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response('{"items":[],"total":0,"has_more":false}', 200);
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );
    GalleryFilterState? appliedFilter;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: StatefulBuilder(
            builder: (context, setState) {
              return fluent.ScaffoldPage(
                content: FluentTagFilterPane(
                  initialFilter: GalleryFilterState(exactTagIds: {1}),
                  onApplyFilter: (value) {
                    setState(() {
                      appliedFilter = value;
                    });
                  },
                ),
              );
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final toggles = find.byType(fluent.ToggleSwitch);
    await tester.tap(toggles.at(0)); // untagged on
    await tester.pumpAndSettle();
    await tester.tap(toggles.at(1)); // pending on, untagged off
    await tester.pumpAndSettle();

    await tester.tap(find.text('应用筛选'));
    await tester.pumpAndSettle();

    expect(appliedFilter, isNotNull);
    expect(appliedFilter!.exactTagIds, {1});
    expect(appliedFilter!.hasTags, isNull);
    expect(appliedFilter!.hasPendingTags, isTrue);
  });

  testWidgets('changing normal tags does not clear active pending filter', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response('{"items":[],"total":0,"has_more":false}', 200);
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );
    GalleryFilterState? appliedFilter;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: StatefulBuilder(
            builder: (context, setState) {
              return fluent.ScaffoldPage(
                content: FluentTagFilterPane(
                  initialFilter: GalleryFilterState(hasPendingTags: true),
                  onApplyFilter: (value) {
                    setState(() {
                      appliedFilter = value;
                    });
                  },
                ),
              );
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.Checkbox).first);
    await tester.pumpAndSettle();

    await tester.tap(find.text('应用筛选'));
    await tester.pumpAndSettle();

    expect(appliedFilter, isNotNull);
    expect(appliedFilter!.subtreeRootTagIds, {1});
    expect(appliedFilter!.hasPendingTags, isTrue);
    expect(appliedFilter!.hasTags, isNull);
  });

  testWidgets('clear filter clears normal and special states together', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":false}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response('{"items":[],"total":0,"has_more":false}', 200);
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: fluent.ScaffoldPage(
            content: FluentTagFilterPane(
              initialFilter: GalleryFilterState(
                exactTagIds: {1},
                hasPendingTags: true,
              ),
              onApplyFilter: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('清空筛选'), findsOneWidget);
    await tester.tap(find.text('清空筛选'));
    await tester.pumpAndSettle();

    expect(find.text('清空筛选'), findsNothing);
    expect(find.text('应用筛选'), findsNothing);
    expect(find.byType(fluent.ToggleSwitch).first, findsOneWidget);
    expect(find.byType(fluent.ToggleSwitch).at(1), findsOneWidget);
  });

  testWidgets('search triggers backend API and shows results', (
    tester,
  ) async {
    final client = MockClient((request) async {
      final path = request.url.path;
      final query = request.url.queryParameters;

      if (path.contains('tree/roots')) {
        return http.Response(
          '{"items":[{"id":1,"preferred_label":"characters","level":"root","has_children":true}]}',
          200,
        );
      }
      if (path.contains('orphans')) {
        return http.Response(
          '{"items":[],"total":0,"has_more":false}',
          200,
        );
      }
      // Search endpoint: GET /api/v1/tags?search=hero
      if (path.endsWith('/tags') && query.containsKey('search')) {
        return http.Response(
          '{"tags":[{"id":3,"preferred_label":"heroine","slug":"heroine","primary_category":"c","review_state":"approved","trust_score":1.0,"usage_count":5,"created_at":"2023-01-01T00:00:00Z","level":"child","parent_id":1}],"total":1}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );

    await tester.pumpWidget(
      fluent.FluentApp(
        home: MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: provider),
          ],
          child: fluent.ScaffoldPage(
            content: FluentTagFilterPane(
              initialFilter: GalleryFilterState(),
              onApplyFilter: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Enter search text
    await tester.enterText(find.byType(fluent.TextBox), 'hero');
    // Wait for debounce (300ms) + API call
    await tester.pumpAndSettle(const Duration(milliseconds: 500));

    // Should show search results
    expect(find.text('heroine'), findsOneWidget);
    expect(find.text('搜索结果 (1)'), findsOneWidget);
  });
}
