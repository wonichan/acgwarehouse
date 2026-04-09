import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart' show Material;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/widgets/fluent_search_content.dart';
import 'package:gallery/widgets/fluent_image_card.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/services/search_service.dart';
import 'package:provider/provider.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  testWidgets('FluentSearchContent double click routes to viewer', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      return http.Response(
        '{"images":[],"total":0,"has_more":false}',
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    });

    final searchProvider = SearchProvider(
      service: SearchService(
        baseUrl: 'http://localhost:8080',
        client: mockClient,
      ),
    );
    searchProvider.results.add(
      ImageModel(
        id: 1,
        filename: 'image.jpg',
        path: '/path/to/image.jpg',
        sourceRoot: 'http://example.com/',
        fileSize: 1024,
        width: 800,
        height: 600,
        format: 'jpg',
        phash: 12345678,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        thumbnailSmallUrl: 'http://example.com/thumb.jpg',
      ),
    );

    bool tapped = false;
    bool doubleTapped = false;

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<SearchProvider>.value(value: searchProvider),
        ],
        child: fluent.FluentApp(
          home: fluent.ScaffoldPage(
            content: Material(
              child: FluentSearchContent(
                onImageTap: (image) => tapped = true,
                onImageDoubleTap: (image) => doubleTapped = true,
              ),
            ),
          ),
        ),
      ),
    );

    await tester.pump(const Duration(milliseconds: 50));

    // We already seeded it, so it should be length 1.
    // The search provider might be loading if we call search,
    // so let's just use the seeded provider and pump the widget.

    final card = find.byType(FluentImageCard).first;
    // single tap is removed per plan, we shouldn't test it or we can test it does nothing
    // await tester.tap(card);
    // await tester.pumpAndSettle();
    // expect(tapped, isTrue);

    // double tap
    await tester.tap(card);
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(card);
    await tester.pump(const Duration(milliseconds: 500));

    expect(doubleTapped, isTrue);
  });
}
