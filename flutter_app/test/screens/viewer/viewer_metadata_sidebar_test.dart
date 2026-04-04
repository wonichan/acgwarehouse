import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';

void main() {
  group('ViewerMetadataSidebar', () {
    late ViewerSessionItem testItem;

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
    });

    testWidgets('renders Image Details heading and expected metadata labels', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ViewerMetadataSidebar(item: testItem, tags: const []),
          ),
        ),
      );

      expect(find.text('Image Details'), findsOneWidget);
      expect(find.text('Filename'), findsOneWidget);
      expect(find.text('Format'), findsOneWidget);
      expect(find.text('Resolution'), findsOneWidget);
      expect(find.text('Size'), findsOneWidget);
      expect(find.text('Path'), findsOneWidget);
      expect(find.text('Imported'), findsOneWidget);
      expect(find.text('Tags'), findsOneWidget);
    });

    testWidgets('omits AI-tag triggers and editing workflows', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ViewerMetadataSidebar(item: testItem, tags: const []),
          ),
        ),
      );

      // Assert absence of "生成" / "AI 标签" buttons or anything related to triggering AI.
      expect(find.text('AI 标签'), findsNothing);
      expect(find.text('生成'), findsNothing);
      expect(find.byIcon(Icons.edit), findsNothing);
      expect(find.byIcon(Icons.add), findsNothing);
      expect(find.byIcon(Icons.merge_type), findsNothing);
    });
  });
}
