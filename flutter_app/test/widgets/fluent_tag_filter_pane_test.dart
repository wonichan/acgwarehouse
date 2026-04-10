import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/fluent_tag_filter_pane.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('FluentTagFilterPane count badge uses accent theme colors', (
    tester,
  ) async {
    final client = MockClient((request) async {
      return http.Response('{"tags":[]}', 200);
    });
    final provider = TagProvider(
      TagService(baseUrl: 'http://localhost:8080', client: client),
    );
    provider.toggleTag(1);

    await tester.pumpWidget(
      fluent.FluentApp(
        theme: fluent.FluentThemeData(accentColor: fluent.Colors.purple),
        home: ChangeNotifierProvider<TagProvider>.value(
          value: provider,
          child: const fluent.ScaffoldPage(content: FluentTagFilterPane()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final countText = tester.widget<fluent.Text>(find.text('1 个'));
    expect(countText.style?.color, fluent.Colors.purple);

    final badgeContainer = tester.widget<fluent.Container>(
      find.byWidgetPredicate(
        (widget) =>
            widget is fluent.Container &&
            widget.child is fluent.Text &&
            (widget.child as fluent.Text).data == '1 个',
      ),
    );
    final decoration = badgeContainer.decoration! as fluent.BoxDecoration;
    expect(decoration.color, fluent.Colors.purple.withValues(alpha: 0.2));
  });

  testWidgets('FluentTagFilterPane uses systemFillColorCritical for errors', (
    tester,
  ) async {
    final client = MockClient((request) async {
      return http.Response('{}', 500);
    });

    await tester.pumpWidget(
      fluent.FluentApp(
        theme: fluent.FluentThemeData(
          resources: const fluent.ResourceDictionary.light(
            systemFillColorCritical: fluent.Color(0xFFCC0000),
          ),
        ),
        home: ChangeNotifierProvider<TagProvider>(
          create: (_) => TagProvider(
            TagService(baseUrl: 'http://localhost:8080', client: client),
          ),
          child: const fluent.ScaffoldPage(content: FluentTagFilterPane()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final errorIcon = tester.widget<fluent.Icon>(
      find.byIcon(fluent.FluentIcons.error),
    );
    expect(errorIcon.color, const fluent.Color(0xFFCC0000));
  });
}
