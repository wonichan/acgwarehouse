import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/edit_tag_dialog.dart';
import 'package:mocktail/mocktail.dart';

class MockTagService extends Mock implements TagService {}

void main() {
  late MockTagService mockTagService;

  final sampleCurrentTag = Tag(
    id: 1,
    preferredLabel: 'blue hair',
    slug: 'blue-hair',
    reviewState: 'pending',
    trustScore: 0.7,
    usageCount: 5,
    createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
    level: 'child',
  );

  final sampleSearchResults = [
    Tag(
      id: 2,
      preferredLabel: 'red hair',
      slug: 'red-hair',
      primaryCategory: 'hair',
      reviewState: 'confirmed',
      trustScore: 0.9,
      usageCount: 50,
      createdAt: DateTime.parse('2024-01-10T08:00:00Z'),
      level: 'child',
    ),
    Tag(
      id: 3,
      preferredLabel: 'black hair',
      slug: 'black-hair',
      primaryCategory: 'hair',
      reviewState: 'confirmed',
      trustScore: 0.85,
      usageCount: 100,
      createdAt: DateTime.parse('2024-01-12T09:00:00Z'),
      level: 'root',
    ),
  ];

  final parentCandidates = [
    Tag(
      id: 99,
      preferredLabel: 'characters',
      slug: 'characters',
      primaryCategory: 'meta',
      reviewState: 'confirmed',
      trustScore: 1,
      usageCount: 40,
      createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
      level: 'root',
    ),
  ];

  final defaultTagResults = List.generate(
    20,
    (index) => Tag(
      id: index + 10,
      preferredLabel: index == 0
          ? 'blonde hair'
          : index == 1
          ? 'smile'
          : 'default edit tag ${index + 1}',
      slug: 'default-edit-tag-${index + 1}',
      primaryCategory: index.isEven ? 'hair' : 'expression',
      reviewState: 'confirmed',
      trustScore: 0.9,
      usageCount: 60 - index,
      createdAt: DateTime.parse('2024-01-11T08:00:00Z'),
      level: index.isEven ? 'child' : 'root',
    ),
  );

  final nextPageResults = [
    Tag(
      id: 12,
      preferredLabel: 'green eyes',
      slug: 'green-eyes',
      primaryCategory: 'eyes',
      reviewState: 'confirmed',
      trustScore: 0.88,
      usageCount: 38,
      createdAt: DateTime.parse('2024-01-14T08:00:00Z'),
      level: 'child',
    ),
  ];

  Widget createTestWidget({required int imageId, required Tag currentTag}) {
    return MaterialApp(
      home: Builder(
        builder: (context) {
          return Scaffold(
            body: Center(
              child: ElevatedButton(
                onPressed: () async {
                  final result = await showDialog<Map<String, dynamic>?>(
                    context: context,
                    builder: (context) => EditTagDialog(
                      imageId: imageId,
                      currentTag: currentTag,
                      tagService: mockTagService,
                    ),
                  );
                  // Store result for verification
                  EditTagDialogTestResult.lastResult = result;
                },
                child: const Text('Open Dialog'),
              ),
            ),
          );
        },
      ),
    );
  }

  setUp(() {
    mockTagService = MockTagService();
    EditTagDialogTestResult.lastResult = null;
    when(
      () => mockTagService.fetchTags(
        limit: any(named: 'limit'),
        offset: any(named: 'offset'),
      ),
    ).thenAnswer((_) async => defaultTagResults);
    when(
      () => mockTagService.getParentCandidates(any()),
    ).thenAnswer((_) async => parentCandidates);
  });

  group('EditTagDialog', () {
    testWidgets('displays search box and current tag text', (tester) async {
      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Verify dialog title
      expect(find.text('编辑标签'), findsOneWidget);

      // Verify current tag text is displayed
      expect(find.textContaining("将 'blue hair' 更改为"), findsOneWidget);

      // Verify search text field exists
      expect(find.byType(TextField), findsOneWidget);
      expect(find.byIcon(Icons.search), findsOneWidget);

      // Verify buttons exist
      expect(find.text('取消'), findsOneWidget);
      expect(find.text('创建新标签'), findsOneWidget);
    });

    testWidgets('filters tag list when searching', (tester) async {
      // Setup mock to return search results
      when(
        () => mockTagService.searchTags('hair'),
      ).thenAnswer((_) async => sampleSearchResults);

      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Enter search text
      await tester.enterText(find.byType(TextField), 'hair');
      await tester.pump();

      // Wait for debounce/async search
      await tester.pump(const Duration(milliseconds: 100));
      await tester.pumpAndSettle();

      // Verify search was called
      verify(() => mockTagService.searchTags('hair')).called(1);

      // Verify only same-level search results are displayed
      expect(find.text('red hair'), findsOneWidget);
      expect(find.text('black hair'), findsNothing);
      // Verify category labels are shown
      expect(find.text('hair'), findsWidgets);
    });

    testWidgets('shows default tags immediately when dialog opens', (
      tester,
    ) async {
      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      await tester.tap(find.text('Open Dialog'));
      await tester.pump();
      await tester.pumpAndSettle();

      verify(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 0),
      ).called(1);
      expect(find.text('blonde hair'), findsOneWidget);
      expect(find.text('smile'), findsNothing);
    });

    testWidgets(
      'typing uses search results and clearing restores default tags',
      (tester) async {
        when(
          () => mockTagService.searchTags('hair'),
        ).thenAnswer((_) async => sampleSearchResults);

        await tester.pumpWidget(
          createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
        );

        await tester.tap(find.text('Open Dialog'));
        await tester.pumpAndSettle();

        await tester.enterText(find.byType(TextField), 'hair');
        await tester.pump();
        await tester.pumpAndSettle();

        expect(find.text('red hair'), findsOneWidget);
        expect(find.text('blonde hair'), findsNothing);
        expect(find.text('black hair'), findsNothing);

        await tester.enterText(find.byType(TextField), '');
        await tester.pump();
        await tester.pumpAndSettle();

        expect(find.text('blonde hair'), findsOneWidget);
        expect(find.text('smile'), findsNothing);
      },
    );

    testWidgets('loads more default tags when scrolled near bottom', (
      tester,
    ) async {
      when(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 0),
      ).thenAnswer((_) async => defaultTagResults);
      when(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 10),
      ).thenAnswer((_) async => nextPageResults);

      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      await tester.drag(find.byType(ListView).first, const Offset(0, -1000));
      await tester.pump();
      await tester.pumpAndSettle();

      verify(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 10),
      ).called(1);
      expect(find.text('green eyes', skipOffstage: false), findsOneWidget);
    });

    testWidgets('returns selected existing tag data on tap', (tester) async {
      // Setup mock
      when(
        () => mockTagService.searchTags('hair'),
      ).thenAnswer((_) async => sampleSearchResults);

      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Enter search text
      await tester.enterText(find.byType(TextField), 'hair');
      await tester.pump(const Duration(milliseconds: 100));
      await tester.pumpAndSettle();

      // Tap on first search result
      await tester.tap(find.text('red hair'));
      await tester.pumpAndSettle();

      // Verify dialog is closed and result is returned
      expect(find.text('编辑标签'), findsNothing);
      expect(EditTagDialogTestResult.lastResult, isNotNull);
      expect(EditTagDialogTestResult.lastResult!['tagId'], 2);
      expect(EditTagDialogTestResult.lastResult!['tagLabel'], isNull);
      expect(EditTagDialogTestResult.lastResult!['label'], 'red hair');
    });

    testWidgets(
      'edit dialog hides cross-level options in default and search lists',
      (tester) async {
        when(
          () => mockTagService.searchTags('hair'),
        ).thenAnswer((_) async => sampleSearchResults);

        await tester.pumpWidget(
          createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
        );

        await tester.tap(find.text('Open Dialog'));
        await tester.pumpAndSettle();

        expect(find.text('blonde hair'), findsOneWidget);
        expect(find.text('smile'), findsNothing);

        await tester.enterText(find.byType(TextField), 'hair');
        await tester.pump(const Duration(milliseconds: 100));
        await tester.pumpAndSettle();

        expect(find.text('red hair'), findsOneWidget);
        expect(find.text('black hair'), findsNothing);
      },
    );

    testWidgets(
      'returns new tag data using current tag level when creating new tag',
      (tester) async {
        await tester.pumpWidget(
          createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
        );

        // Open the dialog
        await tester.tap(find.text('Open Dialog'));
        await tester.pumpAndSettle();

        // Enter new tag name
        await tester.enterText(find.byType(TextField), 'new custom tag');
        await tester.pumpAndSettle();

        // Tap create new tag button
        await tester.tap(find.text('创建新标签'));
        await tester.pumpAndSettle();

        // Verify dialog is closed and result is returned
        expect(find.text('编辑标签'), findsNothing);
        expect(EditTagDialogTestResult.lastResult, isNotNull);
        expect(EditTagDialogTestResult.lastResult!['tagId'], isNull);
        expect(
          EditTagDialogTestResult.lastResult!['tagLabel'],
          'new custom tag',
        );
        expect(EditTagDialogTestResult.lastResult!['tagLevel'], 'child');
        expect(EditTagDialogTestResult.lastResult!['label'], 'new custom tag');
      },
    );

    testWidgets('creating replacement tag keeps current tag level fixed', (
      tester,
    ) async {
      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField).first, 'new child tag');
      await tester.pumpAndSettle();

      await tester.tap(find.text('创建新标签'));
      await tester.pumpAndSettle();

      expect(EditTagDialogTestResult.lastResult, isNotNull);
      expect(EditTagDialogTestResult.lastResult!['tagLabel'], 'new child tag');
      expect(EditTagDialogTestResult.lastResult!['tagLevel'], 'child');
    });

    testWidgets('cancel button closes dialog without returning data', (
      tester,
    ) async {
      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Tap cancel button
      await tester.tap(find.text('取消'));
      await tester.pumpAndSettle();

      // Verify dialog is closed with null result
      expect(find.text('编辑标签'), findsNothing);
      expect(EditTagDialogTestResult.lastResult, isNull);
    });

    testWidgets('clears suggestions when search text is empty', (tester) async {
      // Setup mock
      when(
        () => mockTagService.searchTags('hair'),
      ).thenAnswer((_) async => sampleSearchResults);

      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Enter search text
      await tester.enterText(find.byType(TextField), 'hair');
      await tester.pump(const Duration(milliseconds: 100));
      await tester.pumpAndSettle();

      // Verify results are shown
      expect(find.text('red hair'), findsOneWidget);

      // Clear search text
      await tester.enterText(find.byType(TextField), '');
      await tester.pumpAndSettle();

      // Verify results are cleared
      expect(find.text('red hair'), findsNothing);
      expect(find.text('black hair'), findsNothing);
    });

    testWidgets('create button is disabled when text field is empty', (
      tester,
    ) async {
      await tester.pumpWidget(
        createTestWidget(imageId: 123, currentTag: sampleCurrentTag),
      );

      // Open the dialog
      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      // Find the ElevatedButton (create new tag button)
      final createButton = find.widgetWithText(ElevatedButton, '创建新标签');
      expect(createButton, findsOneWidget);

      // Button should be disabled initially (empty text)
      final buttonWidget = tester.widget<ElevatedButton>(createButton);
      expect(buttonWidget.onPressed, isNull);

      // Entering text is enough because replacement level is now fixed to current tag level
      await tester.enterText(find.byType(TextField), 'some text');
      await tester.pumpAndSettle();

      // Button should now be enabled because the dialog is locked to the current tag level
      final enabledButton = tester.widget<ElevatedButton>(createButton);
      expect(enabledButton.onPressed, isNotNull);
    });
  });
}

// Helper class to store test results
class EditTagDialogTestResult {
  static Map<String, dynamic>? lastResult;
}
