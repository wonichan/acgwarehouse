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
  testWidgets('FluentTagFilterPane keeps tree semantics while searching', (
    tester,
  ) async {
    final client = MockClient((request) async {
      if (request.url.path.contains('tree')) {
        return http.Response(
          '{"tree":[{"id":1,"preferred_label":"characters","primary_category":"c","level":"root","slug":"characters","review_state":"approved","trust_score":1.0,"usage_count":1,"tree_usage_count":40,"created_at":"2023-01-01T00:00:00Z","children":[{"id":2,"preferred_label":"protagonist","primary_category":"c","level":"parent","slug":"protagonist","review_state":"approved","trust_score":1.0,"usage_count":1,"tree_usage_count":20,"created_at":"2023-01-01T00:00:00Z","children":[{"id":3,"preferred_label":"heroine","primary_category":"c","level":"child","slug":"heroine","review_state":"approved","trust_score":1.0,"usage_count":1,"tree_usage_count":7,"created_at":"2023-01-01T00:00:00Z"}]}]}]}',
          200,
        );
      }
      return http.Response(
        '{"tags":[{"id":1,"preferred_label":"characters","primary_category":"c","level":"root","slug":"characters","review_state":"approved","trust_score":1.0,"usage_count":1,"created_at":"2023-01-01T00:00:00Z"},{"id":2,"preferred_label":"protagonist","primary_category":"c","level":"parent","slug":"protagonist","review_state":"approved","trust_score":1.0,"usage_count":1,"created_at":"2023-01-01T00:00:00Z"},{"id":3,"preferred_label":"heroine","primary_category":"c","level":"child","slug":"heroine","review_state":"approved","trust_score":1.0,"usage_count":1,"created_at":"2023-01-01T00:00:00Z"}],"total":3}',
        200,
      );
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

    expect(find.byType(fluent.TreeView), findsOneWidget);
    expect(find.text('R'), findsWidgets);
    expect(find.text('P'), findsWidgets);
    expect(find.text('C'), findsWidgets);

    await tester.enterText(find.byType(fluent.TextBox), 'hero');
    await tester.pumpAndSettle();

    expect(find.byType(fluent.TreeView), findsOneWidget);
    expect(find.text('characters'), findsOneWidget);
    expect(find.text('protagonist'), findsOneWidget);
    expect(find.text('heroine'), findsOneWidget);
    expect(find.text('7'), findsWidgets);
  });

  testWidgets('untagged toggle updates draft and waits for apply', (
    tester,
  ) async {
    final client = MockClient((request) async {
      if (request.url.path.contains('tree')) {
        return http.Response(
          '{"tree":[{"id":1,"preferred_label":"characters","level":"root","tree_usage_count":2,"children":[]}]}',
          200,
        );
      }
      return http.Response(
        '{"tags":[{"id":1,"preferred_label":"characters","level":"root","slug":"characters","review_state":"approved","trust_score":1.0,"usage_count":2,"created_at":"2023-01-01T00:00:00Z"}],"total":1}',
        200,
      );
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
    expect(appliedFilter!.exactTagIds, isEmpty);
    expect(appliedFilter!.hasTags, isFalse);
    expect(appliedFilter!.hasPendingTags, isNull);
  });
}
