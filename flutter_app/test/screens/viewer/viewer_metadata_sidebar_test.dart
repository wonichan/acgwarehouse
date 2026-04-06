import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:provider/provider.dart';

class MockTagService implements TagService {
  @override
  Future<String> getDefaultAIPrompt() async => 'default prompt';

  @override
  Future<int> triggerAITags(int imageId, {String? prompt}) async => 0;

  @override
  Future<Map<String, dynamic>> getAITagStatus(int imageId) async => {
    'status': 'completed',
  };

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

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

  group('ViewerMetadataSidebar', () {
    late ViewerSessionItem testItem;
    late MockTagService mockTagService;
    late TagProvider tagProvider;

    setUp(() {
      testItem = ViewerSessionItem(
        imageId: 1,
        path: '/some/path/image.jpg',
        filename: 'image.jpg',
        sourceRoot: '/some',
        fileSize: 1024 * 1024,
        width: 1920,
        height: 1080,
        format: 'jpeg',
        thumbnailSmallUrl: 'http://small.jpg',
        thumbnailLargeUrl: 'http://large.jpg',
        createdAtIso8601: DateTime(2023, 1, 1).toUtc().toIso8601String(),
        updatedAtIso8601: DateTime(2023, 1, 1).toUtc().toIso8601String(),
      );
      mockTagService = MockTagService();
      tagProvider = TagProvider(mockTagService);
    });

    testWidgets('renders Image Details heading and expected metadata labels', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider<TagProvider>.value(
              value: tagProvider,
              child: ViewerMetadataSidebar(item: testItem),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Image Details'), findsOneWidget);
      expect(find.text('Filename'), findsOneWidget);
      expect(find.text('Format'), findsOneWidget);
      expect(find.text('Resolution'), findsOneWidget);
      expect(find.text('Size'), findsOneWidget);
      expect(find.text('Path'), findsOneWidget);
      expect(find.text('Imported'), findsOneWidget);
    });

    testWidgets('includes AI-tag triggers and editing workflows', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider<TagProvider>.value(
              value: tagProvider,
              child: ViewerMetadataSidebar(item: testItem),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('AI 标签'), findsOneWidget);
      expect(find.text('生成'), findsOneWidget);
      expect(find.byIcon(Icons.add), findsOneWidget);
    });

    testWidgets('opens add tag dialog from viewer sidebar', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider<TagProvider>.value(
              value: tagProvider,
              child: ViewerMetadataSidebar(item: testItem),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.add));
      await tester.pumpAndSettle();

      expect(find.text('添加标签'), findsOneWidget);
    });

    testWidgets(
      'uses a dark theme-aware metadata pane shell instead of the old light treatment',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          MaterialApp(
            theme: ThemeData.dark(),
            home: Scaffold(
              body: ChangeNotifierProvider<TagProvider>.value(
                value: tagProvider,
                child: ViewerMetadataSidebar(item: testItem),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();

        expect(
          find.byKey(const ValueKey('viewer-metadata-sidebar')),
          findsOneWidget,
        );
        expect(
          find.byKey(const ValueKey('metadata-pane-root')),
          findsOneWidget,
        );
        expect(
          find.byKey(const ValueKey('metadata-section-basic')),
          findsOneWidget,
        );

        expect(
          find.descendant(
            of: find.byKey(const ValueKey('metadata-pane-root')),
            matching: find.byType(Card),
          ),
          findsNothing,
          reason:
              'Dark-mode metadata pane should not keep card-based grouping anymore.',
        );
        expect(
          find.descendant(
            of: find.byKey(const ValueKey('viewer-metadata-sidebar')),
            matching: find.byWidgetPredicate(
              isForbiddenLegacyMetadataSurface,
              description: 'legacy light metadata surface',
            ),
          ),
          findsNothing,
          reason:
              'Dark-mode sidebar host should stop forcing legacy light metadata surfaces.',
        );
      },
    );
  });
}
