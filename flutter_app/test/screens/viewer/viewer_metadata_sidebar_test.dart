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
      'implements Windows Photos-inspired visual structure (cards, white sections, #f3f3f3 background)',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          MaterialApp(
            theme: ThemeData.light(),
            home: Scaffold(
              body: ChangeNotifierProvider<TagProvider>.value(
                value: tagProvider,
                child: ViewerMetadataSidebar(item: testItem),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();

        // Ensure the root container has the light gray/f3f3f3-ish background
        final sidebarContainerFinder = find.byType(Container).first;
        final Container sidebarContainer = tester.widget(
          sidebarContainerFinder,
        );
        final BoxDecoration decoration =
            sidebarContainer.decoration as BoxDecoration;
        expect(
          decoration.color?.value,
          equals(const Color(0xFFF3F3F3).value),
          reason: 'Sidebar should have #f3f3f3 background',
        );

        // The metadata panel should use Cards or grouped white containers for sections
        expect(
          find.byType(Card),
          findsWidgets,
          reason: 'Should use Card widgets for grouping',
        );

        // The AI trigger button should be styled as a blue primary button (often a FilledButton with blue)
        final generateButtonFinder = find.byWidgetPredicate(
          (widget) => widget is FilledButton && widget.onPressed != null,
          description: 'Generate FilledButton',
        );
        expect(generateButtonFinder, findsOneWidget);
        final FilledButton generateBtn = tester.widget(generateButtonFinder);
        expect(
          generateBtn.style?.backgroundColor?.resolve({}),
          equals(Colors.blue),
          reason: 'Generate button should be blue',
        );
        expect(
          generateBtn.style?.shape?.resolve({}),
          isA<RoundedRectangleBorder>(),
          reason: 'Generate button should be rectangular',
        );
      },
    );
    testWidgets(
      'implements Windows Photos-inspired visual structure even under dark theme (cards, white sections, #f3f3f3 background)',
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

        // Ensure the root container has the light gray/f3f3f3-ish background regardless of theme
        final sidebarContainerFinder = find.byType(Container).first;
        final Container sidebarContainer = tester.widget(
          sidebarContainerFinder,
        );
        final BoxDecoration decoration =
            sidebarContainer.decoration as BoxDecoration;
        expect(
          decoration.color?.value,
          equals(const Color(0xFFF3F3F3).value),
          reason: 'Sidebar should have #f3f3f3 background even in dark mode',
        );

        // The metadata panel should use Cards or grouped white containers for sections
        final cardFinder = find.byType(Card);
        expect(
          cardFinder,
          findsWidgets,
          reason: 'Should use Card widgets for grouping',
        );

        final Card firstCard = tester.widget(cardFinder.first);
        expect(
          firstCard.color?.value,
          equals(Colors.white.value),
          reason: 'Cards should be white even in dark mode',
        );
      },
    );
  });
}
