import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/widgets/tag_chip.dart';

void main() {
  final sampleTag = Tag(
    id: 1,
    preferredLabel: '测试标签',
    slug: 'test-tag',
    reviewState: 'confirmed',
    trustScore: 0.8,
    usageCount: 3,
    createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
  );

  testWidgets('shows delete action when onDelete is provided', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(
            tag: sampleTag,
            onDelete: () {},
          ),
        ),
      ),
    );

    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
  });
}
