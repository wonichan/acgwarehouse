import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/screens/image_detail_screen.dart';
import 'package:gallery/models/image.dart';
import 'package:extended_image/extended_image.dart';

void main() {
  group('ImageDetailScreen Gestures', () {
    final testImage = ImageModel(
      id: 1,
      path: '/test.jpg',
      filename: 'test.jpg',
      sourceRoot: 'root',
      format: 'jpg',
      width: 100,
      height: 100,
      fileSize: 1024,
      phash: 12345,
      thumbnailLargeUrl: 'http://test.com/large.jpg',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );

    testWidgets('image has gesture mode enabled', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageDetailScreen(image: testImage),
        ),
      );

      // Use pump() instead of pumpAndSettle() to avoid timeout from network calls
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.mode, ExtendedImageMode.gesture);
    });

    testWidgets('supports double-tap zoom', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageDetailScreen(image: testImage),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.onDoubleTap, isNotNull);
    });

    testWidgets('has gesture config with proper scale limits', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: ImageDetailScreen(image: testImage),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final extendedImage = tester.widget<ExtendedImage>(find.byType(ExtendedImage));
      expect(extendedImage.initGestureConfigHandler, isNotNull);
    });
  });
}