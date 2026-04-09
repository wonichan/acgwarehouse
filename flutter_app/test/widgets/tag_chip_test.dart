import 'package:flutter/gestures.dart';
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

  testWidgets('renders tag label in Text widget', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: TagChip(tag: sampleTag)),
      ),
    );

    // Should find the tag label as Text
    expect(find.text('测试标签'), findsOneWidget);
    // Text widget exists
    expect(find.byType(Text), findsWidgets);
  });

  testWidgets('uses MouseRegion for hover detection', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: TagChip(tag: sampleTag)),
      ),
    );

    // Should have MouseRegion for hover functionality
    expect(find.byType(MouseRegion), findsWidgets);
  });

  testWidgets('shows colored dot indicator based on style', (tester) async {
    // Test confirmed style (green dot)
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(tag: sampleTag, style: TagChipStyle.confirmed),
        ),
      ),
    );

    // Should find a colored container (the dot indicator)
    final dotContainer = find.byWidgetPredicate(
      (widget) =>
          widget is Container &&
          widget.decoration is BoxDecoration &&
          (widget.decoration as BoxDecoration).shape == BoxShape.circle,
    );
    expect(dotContainer, findsOneWidget);
  });

  testWidgets('shows action icons immediately when showActions is true', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(tag: sampleTag, onDelete: () {}, showActions: true),
        ),
      ),
    );

    // Should show icons immediately when showActions is true
    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
  });

  testWidgets('shows confirm/reject icons when showActions is true', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(
            tag: sampleTag,
            onConfirm: () {},
            onReject: () {},
            showActions: true,
          ),
        ),
      ),
    );

    // Should show icons immediately when showActions is true
    expect(find.byIcon(Icons.check), findsOneWidget);
    expect(find.byIcon(Icons.close), findsOneWidget);
  });

  testWidgets('shows action icons on hover when callbacks provided', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(tag: sampleTag, onConfirm: () {}, onDelete: () {}),
        ),
      ),
    );

    // Icons should NOT be visible initially (not hovered, showActions=false)
    expect(find.byIcon(Icons.check), findsNothing);
    expect(find.byIcon(Icons.delete_outline), findsNothing);

    // Get the position of the TagChip and send a hover event
    final tagChipFinder = find.byType(TagChip);
    final center = tester.getCenter(tagChipFinder);

    // Send hover enter event
    tester.binding.handlePointerEvent(
      PointerHoverEvent(position: center, kind: PointerDeviceKind.mouse),
    );
    await tester.pump();

    // Icons should now be visible on hover
    expect(find.byIcon(Icons.check), findsOneWidget);
    expect(find.byIcon(Icons.delete_outline), findsOneWidget);

    // Send hover exit event (move far away)
    tester.binding.handlePointerEvent(
      PointerHoverEvent(
        position: const Offset(-1000, -1000),
        kind: PointerDeviceKind.mouse,
      ),
    );
    await tester.pump();

    // Icons should disappear when no longer hovered
    expect(find.byIcon(Icons.check), findsNothing);
    expect(find.byIcon(Icons.delete_outline), findsNothing);
  });

  testWidgets('applies different styles for different review states', (
    tester,
  ) async {
    // Test pending style with callbacks provided
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(
            tag: sampleTag,
            style: TagChipStyle.pending,
            onConfirm: () {},
            onReject: () {},
            showActions: true,
          ),
        ),
      ),
    );
    expect(find.byIcon(Icons.check), findsOneWidget);
    expect(find.byIcon(Icons.close), findsOneWidget);

    // Test rejected style with callbacks provided
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(
            tag: sampleTag,
            style: TagChipStyle.rejected,
            onConfirm: () {},
            onReject: () {},
            showActions: true,
          ),
        ),
      ),
    );
    // Rejected tags should have strike-through text
    final textWidget = tester.widget<Text>(find.text('测试标签'));
    expect(textWidget.style?.decoration, TextDecoration.lineThrough);
  });

  testWidgets('renders all action icons with all callbacks provided', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(
            tag: sampleTag,
            onConfirm: () {},
            onReject: () {},
            onMerge: () {},
            onEdit: () {},
            onDelete: () {},
            showActions: true,
          ),
        ),
      ),
    );

    // All action icons should be visible
    expect(find.byIcon(Icons.check), findsOneWidget);
    expect(find.byIcon(Icons.close), findsOneWidget);
    expect(find.byIcon(Icons.merge_type), findsOneWidget);
    expect(find.byIcon(Icons.edit), findsOneWidget);
    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
  });

  testWidgets('shows merge and edit icons on hover', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TagChip(tag: sampleTag, onMerge: () {}, onEdit: () {}),
        ),
      ),
    );

    // Icons should NOT be visible initially
    expect(find.byIcon(Icons.merge_type), findsNothing);
    expect(find.byIcon(Icons.edit), findsNothing);

    // Get the position of the TagChip and send a hover event
    final tagChipFinder = find.byType(TagChip);
    final center = tester.getCenter(tagChipFinder);

    // Send hover enter event
    tester.binding.handlePointerEvent(
      PointerHoverEvent(position: center, kind: PointerDeviceKind.mouse),
    );
    await tester.pump();

    // Icons should now be visible on hover
    expect(find.byIcon(Icons.merge_type), findsOneWidget);
    expect(find.byIcon(Icons.edit), findsOneWidget);
  });
}
