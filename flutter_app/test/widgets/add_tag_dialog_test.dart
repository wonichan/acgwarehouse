import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/add_tag_dialog.dart';
import 'package:mocktail/mocktail.dart';

class MockTagService extends Mock implements TagService {}

void main() {
  late MockTagService mockTagService;

  final defaultTags = List.generate(
    20,
    (index) => Tag(
      id: index + 1,
      preferredLabel: index == 0
          ? 'blue hair'
          : index == 1
          ? 'school uniform'
          : 'default tag ${index + 1}',
      slug: 'default-tag-${index + 1}',
      primaryCategory: index.isEven ? 'hair' : 'clothes',
      reviewState: 'confirmed',
      trustScore: 0.9,
      usageCount: 100 - index,
      createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
      level: index == 0
          ? 'parent'
          : index == 1
          ? 'child'
          : null,
      parentId: index == 1 ? 99 : null,
    ),
  );

  final secondPageTags = [
    Tag(
      id: 3,
      preferredLabel: 'long hair',
      slug: 'long-hair',
      primaryCategory: 'hair',
      reviewState: 'confirmed',
      trustScore: 0.7,
      usageCount: 77,
      createdAt: DateTime.parse('2024-01-03T00:00:00Z'),
    ),
  ];

  final searchTags = [
    Tag(
      id: 4,
      preferredLabel: 'red hair',
      slug: 'red-hair',
      primaryCategory: 'hair',
      reviewState: 'confirmed',
      trustScore: 0.95,
      usageCount: 120,
      createdAt: DateTime.parse('2024-01-04T00:00:00Z'),
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
      usageCount: 50,
      createdAt: DateTime.parse('2024-01-05T00:00:00Z'),
      level: 'root',
    ),
  ];

  Widget createTestWidget() {
    return MaterialApp(
      home: Builder(
        builder: (context) {
          return Scaffold(
            body: Center(
              child: ElevatedButton(
                onPressed: () async {
                  final result = await showDialog<dynamic>(
                    context: context,
                    builder: (context) =>
                        AddTagDialog(imageId: 123, tagService: mockTagService),
                  );
                  AddTagDialogTestResult.lastResult = result;
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
    AddTagDialogTestResult.lastResult = null;
    when(
      () => mockTagService.fetchTags(
        limit: any(named: 'limit'),
        offset: any(named: 'offset'),
      ),
    ).thenAnswer((_) async => defaultTags);
    when(
      () => mockTagService.searchTags(any()),
    ).thenAnswer((_) async => searchTags);
    when(
      () => mockTagService.addImageTag(
        any(),
        tagId: any(named: 'tagId'),
        tagLabel: any(named: 'tagLabel'),
        level: any(named: 'level'),
        parentId: any(named: 'parentId'),
      ),
    ).thenAnswer((_) async => defaultTags.first);
    when(
      () => mockTagService.getParentCandidates(any()),
    ).thenAnswer((_) async => parentCandidates);
  });

  group('AddTagDialog', () {
    testWidgets('shows default tags immediately when dialog opens', (
      tester,
    ) async {
      await tester.pumpWidget(createTestWidget());

      await tester.tap(find.text('Open Dialog'));
      await tester.pump();
      await tester.pumpAndSettle();

      verify(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 0),
      ).called(1);
      expect(find.text('blue hair'), findsOneWidget);
      expect(find.text('school uniform'), findsOneWidget);
      expect(find.text('父级'), findsWidgets);
    });

    testWidgets(
      'typing switches to search results and clearing restores default tags',
      (tester) async {
        await tester.pumpWidget(createTestWidget());

        await tester.tap(find.text('Open Dialog'));
        await tester.pumpAndSettle();

        await tester.enterText(find.byType(TextField), 'red');
        await tester.pump();
        await tester.pumpAndSettle();

        verify(() => mockTagService.searchTags('red')).called(1);
        expect(find.text('red hair'), findsOneWidget);
        expect(find.text('blue hair'), findsNothing);

        await tester.enterText(find.byType(TextField), '');
        await tester.pump();
        await tester.pumpAndSettle();

        expect(find.text('blue hair'), findsOneWidget);
        expect(find.text('school uniform'), findsOneWidget);
      },
    );

    testWidgets('loads more default tags when scrolled near bottom', (
      tester,
    ) async {
      when(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 0),
      ).thenAnswer((_) async => defaultTags);
      when(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 20),
      ).thenAnswer((_) async => secondPageTags);

      await tester.pumpWidget(createTestWidget());

      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      await tester.drag(find.byType(ListView).first, const Offset(0, -1000));
      await tester.pump();
      await tester.pumpAndSettle();

      verify(
        () => mockTagService.fetchTags(limit: any(named: 'limit'), offset: 20),
      ).called(1);
      expect(find.text('long hair', skipOffstage: false), findsOneWidget);
    });

    testWidgets('manual create passes explicit level and parent selection', (
      tester,
    ) async {
      await tester.pumpWidget(createTestWidget());

      await tester.tap(find.text('Open Dialog'));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField).first, 'heroine');

      await tester.tap(find.byType(DropdownButtonFormField<String>).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('父级 (Parent)').last);
      await tester.pumpAndSettle();

      verify(() => mockTagService.getParentCandidates('parent')).called(1);

      await tester.tap(find.byType(DropdownButtonFormField<int?>).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('characters').last);
      await tester.pumpAndSettle();

      await tester.tap(find.text('创建新标签'));
      await tester.pumpAndSettle();

      verify(
        () => mockTagService.addImageTag(
          123,
          tagId: any(named: 'tagId'),
          tagLabel: 'heroine',
          level: 'parent',
          parentId: 99,
        ),
      ).called(1);
    });
  });
}

class AddTagDialogTestResult {
  static dynamic lastResult;
}
