import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/screens/image_detail_screen.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
  bool isForbiddenLegacyMetadataSurface(Widget widget) {
    const legacyPaneColor = Color(0xFFF3F3F3);

    if (widget is Card) {
      final color = widget.color;
      return color == Colors.white || color == legacyPaneColor;
    }

    if (widget is Container) {
      final decoration = widget.decoration;
      if (decoration is BoxDecoration) {
        final color = decoration.color;
        return color == Colors.white || color == legacyPaneColor;
      }
    }

    if (widget is DecoratedBox) {
      final decoration = widget.decoration;
      if (decoration is BoxDecoration) {
        final color = decoration.color;
        return color == Colors.white || color == legacyPaneColor;
      }
    }

    if (widget is ColoredBox) {
      return widget.color == Colors.white || widget.color == legacyPaneColor;
    }

    return false;
  }

  final image = ImageModel(
    id: 1,
    path: '/library/demo/image.jpg',
    filename: 'image.jpg',
    sourceRoot: '/library/demo',
    fileSize: 1024 * 1024,
    width: 1920,
    height: 1080,
    format: 'jpeg',
    phash: 123,
    thumbnailSmallUrl: 'http://small.jpg',
    thumbnailLargeUrl: 'http://large.jpg',
    createdAt: DateTime.utc(2023, 1, 1),
    updatedAt: DateTime.utc(2023, 1, 1),
  );

  Widget buildHarness() {
    return ChangeNotifierProvider<ConfigProvider>(
      create: (_) => ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
      child: MaterialApp(
        theme: ThemeData.dark(),
        home: ImageDetailScreen(image: image),
      ),
    );
  }

  testWidgets(
    'renders the image details metadata area as a dark theme-aware pane',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(buildHarness());
      await tester.pump();

      expect(
        find.byKey(const ValueKey('image-detail-metadata-pane')),
        findsOneWidget,
      );
      expect(find.byKey(const ValueKey('metadata-pane-root')), findsOneWidget);
      expect(
        find.byKey(const ValueKey('metadata-section-basic')),
        findsOneWidget,
      );
      expect(find.text('元数据'), findsOneWidget);
      expect(find.text('文件名'), findsOneWidget);
      expect(find.text('路径'), findsOneWidget);
      expect(find.byKey(const Key('metadata-path-row')), findsOneWidget);
      expect(find.byKey(const Key('metadata-path-value')), findsOneWidget);
      expect(find.byTooltip('复制路径'), findsOneWidget);

      final pathText = tester.widget<Text>(
        find.byKey(const Key('metadata-path-value')),
      );
      expect(pathText.maxLines, 1);
      expect(pathText.overflow, TextOverflow.ellipsis);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is Tooltip && widget.message == image.path,
          description: 'full path tooltip',
        ),
        findsOneWidget,
      );

      expect(
        find.descendant(
          of: find.byKey(const ValueKey('metadata-pane-root')),
          matching: find.byType(Card),
        ),
        findsNothing,
        reason: 'Metadata pane content should stop using card-based grouping.',
      );
      expect(
        find.descendant(
          of: find.byKey(const ValueKey('image-detail-metadata-pane')),
          matching: find.byWidgetPredicate(
            isForbiddenLegacyMetadataSurface,
            description: 'legacy light metadata surface',
          ),
        ),
        findsNothing,
        reason:
            'Dark-mode details pane should not reuse legacy light metadata surfaces.',
      );
    },
  );

  testWidgets(
    'detail screen can render pending and rejected tags with injected tag service',
    (WidgetTester tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/images/1/tags')) {
          return http.Response(
            '{"confirmed":[{"id":2,"preferred_label":"confirmed-tag","review_state":"confirmed"}],"pending":[{"id":1,"preferred_label":"pending-tag","review_state":"pending"}],"rejected":[{"id":3,"preferred_label":"rejected-tag","review_state":"rejected"}]}',
            200,
          );
        }
        if (request.url.path.endsWith('/api/v1/ai-tags/default-prompt')) {
          return http.Response('{"default_prompt":"default prompt"}', 200);
        }
        return http.Response('{}', 200);
      });

      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        ChangeNotifierProvider<ConfigProvider>(
          create: (_) =>
              ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
          child: MaterialApp(
            theme: ThemeData.dark(),
            home: ImageDetailScreen(
              image: image,
              tagService: TagService(
                baseUrl: 'http://localhost:8080',
                client: mockClient,
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));
      await tester.pump(const Duration(milliseconds: 200));

      expect(find.text('待确认'), findsOneWidget);
      expect(find.text('已确认'), findsOneWidget);
      expect(find.text('已拒绝'), findsOneWidget);
    },
  );
}
