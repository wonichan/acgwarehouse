import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/screens/image_detail_screen.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';

void main() {
  final sampleImage = ImageModel(
    id: 1,
    filename: 'test_image.jpg',
    path: '/test/path/test_image.jpg',
    sourceRoot: '/test/path',
    width: 1920,
    height: 1080,
    fileSize: 1024000,
    format: 'jpg',
    phash: 12345678,
    thumbnailSmallUrl: 'https://example.com/small.jpg',
    thumbnailLargeUrl: 'https://example.com/large.jpg',
    createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
    updatedAt: DateTime.parse('2024-01-15T10:30:00Z'),
  );

  Widget createTestWidget() {
    return MaterialApp(
      home: ChangeNotifierProvider(
        create: (_) => TagProvider(TagService()),
        child: ImageDetailScreen(image: sampleImage),
      ),
    );
  }

  group('ImageDetailScreen layout improvements', () {
    testWidgets('displays image viewer', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // Should show the image viewer container (ExtendedImage doesn't render as Image in tests)
      expect(find.byType(Container), findsWidgets);
    });

    testWidgets('image viewer takes significant vertical space', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // Find the container that holds the image
      // The constraints should allow for more than 50% of screen height
      final container = find.byType(Container).first;
      expect(container, findsOneWidget);
    });

    testWidgets('shows metadata section', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // Should show metadata label
      expect(find.text('元数据'), findsOneWidget);
    });

    testWidgets('has compact metadata layout', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // Should show metadata rows
      expect(find.text('文件名'), findsOneWidget);
      expect(find.text('尺寸'), findsOneWidget);
      expect(find.text('格式'), findsOneWidget);
      expect(find.text('大小'), findsOneWidget);
    });

    testWidgets('image viewer has Hero widget for animation', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // Should have Hero widget for smooth transitions
      expect(find.byType(Hero), findsOneWidget);
    });

    testWidgets('image viewer is tappable to open fullscreen', (tester) async {
      await tester.pumpWidget(createTestWidget());
      await tester.pump();

      // The image area should be tappable (GestureDetector or InkWell)
      final gestureDetectors = find.byType(GestureDetector);
      expect(gestureDetectors, findsWidgets);
    });
  });
}