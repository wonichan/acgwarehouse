import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/batch_tag_dialog.dart';
import 'package:mocktail/mocktail.dart';
import 'package:provider/provider.dart';

class MockBatchTagService extends Mock implements TagService {}

void main() {
  late MockBatchTagService mockTagService;

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

  Widget createTestWidget() {
    return Provider<TagService>.value(
      value: mockTagService,
      child: const MaterialApp(
        home: Scaffold(body: BatchAddTagDialog(imageIds: [1, 2])),
      ),
    );
  }

  setUp(() {
    mockTagService = MockBatchTagService();
    when(() => mockTagService.searchTags(any())).thenAnswer((_) async => []);
    when(() => mockTagService.getParentCandidates(any())).thenAnswer((_) async {
      return parentCandidates;
    });
    when(
      () => mockTagService.addImageTag(
        any(),
        tagId: any(named: 'tagId'),
        tagLabel: any(named: 'tagLabel'),
        level: any(named: 'level'),
        parentId: any(named: 'parentId'),
      ),
    ).thenAnswer(
      (_) async => Tag(
        id: 5,
        preferredLabel: 'new parent tag',
        slug: 'new-parent-tag',
        reviewState: 'confirmed',
        trustScore: 1,
        usageCount: 0,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'parent',
        parentId: 99,
      ),
    );
  });

  testWidgets('batch add create requires explicit hierarchy fields', (
    tester,
  ) async {
    await tester.pumpWidget(createTestWidget());
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField).first, 'new parent tag');
    await tester.tap(find.byType(DropdownButtonFormField<String>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('父级').last);
    await tester.pumpAndSettle();

    await tester.tap(find.byType(DropdownButtonFormField<int?>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('characters').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('创建新标签'));
    await tester.pumpAndSettle();

    verify(
      () => mockTagService.addImageTag(
        1,
        tagId: any(named: 'tagId'),
        tagLabel: 'new parent tag',
        level: 'parent',
        parentId: 99,
      ),
    ).called(1);
    verify(
      () => mockTagService.addImageTag(
        2,
        tagId: any(named: 'tagId'),
        tagLabel: 'new parent tag',
        level: 'parent',
        parentId: 99,
      ),
    ).called(1);
  });
}
