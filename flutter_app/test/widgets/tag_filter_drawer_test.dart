import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/tag_filter_drawer.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('TagFilterDrawer header uses ColorScheme primary colors', (
    tester,
  ) async {
    final theme = ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
    );
    final client = MockClient((request) async {
      return http.Response('{"tags":[]}', 200);
    });

    await tester.pumpWidget(
      MaterialApp(
        theme: theme,
        home: Scaffold(
          body: ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(
              TagService(baseUrl: 'http://localhost:8080', client: client),
            ),
            child: const TagFilterDrawer(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final title = tester.widget<Text>(find.text('标签筛选'));
    expect(title.style?.color, theme.colorScheme.onPrimary);

    final selectionCount = tester.widget<Text>(find.text('已选择 0 个标签'));
    expect(
      selectionCount.style?.color,
      theme.colorScheme.onPrimary.withValues(alpha: 0.7),
    );
  });
}
