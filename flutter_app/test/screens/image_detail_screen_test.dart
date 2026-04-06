import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/screens/image_detail_screen.dart';

void main() {
  bool isForbiddenLegacyMetadataSurface(Widget widget) {
    const legacyPaneColor = Color(0xFFF3F3F3);

    if (widget is Card) {
      final color = widget.color;
      return color == Colors.white || color == legacyPaneColor;
    }

    if (widget is Container) {
      final decoration = widget.decoration;
      if (decoration is BoxDecoration) {
        final color = decoration.color;
        return color == Colors.white || color == legacyPaneColor;
      }
    }

    if (widget is DecoratedBox) {
      final decoration = widget.decoration;
      if (decoration is BoxDecoration) {
        final color = decoration.color;
        return color == Colors.white || color == legacyPaneColor;
      }
    }

    if (widget is ColoredBox) {
      return widget.color == Colors.white || widget.color == legacyPaneColor;
    }

    return false;
  }

  final image = ImageModel(
    id: 1,
    path: '/library/demo/image.jpg',
    filename: 'image.jpg',
    sourceRoot: '/library/demo',
    fileSize: 1024 * 1024,
    width: 1920,
    height: 1080,
    format: 'jpeg',
    phash: 123,
    thumbnailSmallUrl: 'http://small.jpg',
    thumbnailLargeUrl: 'http://large.jpg',
    createdAt: DateTime.utc(2023, 1, 1),
    updatedAt: DateTime.utc(2023, 1, 1),
  );

  testWidgets(
    'renders the image details metadata area as a dark theme-aware pane',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1400, 900));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData.dark(),
          home: ImageDetailScreen(image: image),
        ),
      );
      await tester.pump();

      expect(
        find.byKey(const ValueKey('image-detail-metadata-pane')),
        findsOneWidget,
      );
      expect(find.byKey(const ValueKey('metadata-pane-root')), findsOneWidget);
      expect(
        find.byKey(const ValueKey('metadata-section-basic')),
        findsOneWidget,
      );
      expect(find.text('元数据'), findsOneWidget);

      expect(
        find.descendant(
          of: find.byKey(const ValueKey('metadata-pane-root')),
          matching: find.byType(Card),
        ),
        findsNothing,
        reason: 'Metadata pane content should stop using card-based grouping.',
      );
      expect(
        find.descendant(
          of: find.byKey(const ValueKey('image-detail-metadata-pane')),
          matching: find.byWidgetPredicate(
            isForbiddenLegacyMetadataSurface,
            description: 'legacy light metadata surface',
          ),
        ),
        findsNothing,
        reason:
            'Dark-mode details pane should not reuse legacy light metadata surfaces.',
      );
    },
  );
}
