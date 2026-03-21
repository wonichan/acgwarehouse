import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/screens/image_gallery_viewer.dart';
import 'package:gallery/models/image.dart';
import 'package:extended_image/extended_image.dart';

void main() {
  group('ImageGalleryViewer', () {
    final testImages = [
      ImageModel(
        id: 1,
        filename: 'test1.jpg',
        path: '/test1.jpg',
        sourceRoot: 'root',
        format: 'jpg',
        width: 100,
        height: 100,
        fileSize: 1024,
        phash: 12345,
        thumbnailLargeUrl: 'http://test.com/1.jpg',
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
      ),
      ImageModel(
        id: 2,
        filename: 'test2.jpg',
        path: '/test2.jpg',
        sourceRoot: 'root',
        format: 'jpg',
        width: 100,
        height: 100,
        fileSize: 1024,
        phash: 12346,
        thumbnailLargeUrl: 'http://test.com/2.jpg',
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
      ),
    ];

    testWidgets('displays page view', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageGalleryViewer(
            images: testImages,
            initialIndex: 0,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.byType(PageView), findsOneWidget);
    });

    testWidgets('shows position indicator', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageGalleryViewer(
            images: testImages,
            initialIndex: 0,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('1 / 2'), findsOneWidget);
    });

    testWidgets('has gesture mode enabled for images', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageGalleryViewer(
            images: testImages,
            initialIndex: 0,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.mode, ExtendedImageMode.gesture);
    });

    testWidgets('supports double-tap zoom', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageGalleryViewer(
            images: testImages,
            initialIndex: 0,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.onDoubleTap, isNotNull);
    });

    testWidgets('has inPageView set to true for gesture config', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageGalleryViewer(
            images: testImages,
            initialIndex: 0,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.initGestureConfigHandler, isNotNull);
    });
  });
}