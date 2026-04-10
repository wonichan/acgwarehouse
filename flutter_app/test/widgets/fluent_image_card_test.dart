import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart' show Material;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/widgets/fluent_image_card.dart';

void main() {
  final testImage = ImageModel(
    id: 1,
    filename: 'image.jpg',
    path: '/path/to/image.jpg',
    sourceRoot: 'http://example.com/',
    fileSize: 1024,
    width: 800,
    height: 600,
    format: 'jpg',
    phash: 12345678,
    createdAt: DateTime.now(),
    updatedAt: DateTime.now(),
    thumbnailSmallUrl: 'http://example.com/thumb.jpg',
  );

  testWidgets('FluentImageCard triggers onTap on single click', (tester) async {
    bool tapped = false;
    await tester.pumpWidget(
      fluent.FluentApp(
        home: fluent.ScaffoldPage(
          content: Material(
            child: FluentImageCard(
              image: testImage,
              onTap: (image) => tapped = true,
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(fluent.GestureDetector));
    await tester.pump();

    expect(tapped, isTrue);
  });

  testWidgets('shows unselected selection affordance in selection mode', (
    tester,
  ) async {
    await tester.pumpWidget(
      fluent.FluentApp(
        home: fluent.ScaffoldPage(
          content: Material(
            child: FluentImageCard(
              image: testImage,
              isSelectionMode: true,
              isSelected: false,
            ),
          ),
        ),
      ),
    );

    // Weak checkmark should be present
    expect(find.byIcon(fluent.FluentIcons.check_mark), findsOneWidget);
  });

  testWidgets(
    'tapping card in selection mode calls onSelect instead of onTap',
    (tester) async {
      bool selected = false;
      bool tapped = false;

      await tester.pumpWidget(
        fluent.FluentApp(
          home: fluent.ScaffoldPage(
            content: Material(
              child: FluentImageCard(
                image: testImage,
                isSelectionMode: true,
                onTap: (img) => tapped = true,
                onSelect: (img, sel) => selected = true,
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.byType(fluent.GestureDetector));
      await tester.pump();

      expect(tapped, isFalse);
      expect(selected, isTrue);
    },
  );

  testWidgets('shows selected overlay and checkmark when selected', (
    tester,
  ) async {
    await tester.pumpWidget(
      fluent.FluentApp(
        home: fluent.ScaffoldPage(
          content: Material(
            child: FluentImageCard(
              image: testImage,
              isSelectionMode: true,
              isSelected: true,
            ),
          ),
        ),
      ),
    );

    expect(find.byIcon(fluent.FluentIcons.check_mark), findsOneWidget);
  });

  testWidgets('FluentImageCard triggers onSecondaryTapDown on right click', (
    tester,
  ) async {
    ImageModel? tappedImage;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: fluent.ScaffoldPage(
          content: Material(
            child: FluentImageCard(
              image: testImage,
              onSecondaryTapDown: (image, details) => tappedImage = image,
            ),
          ),
        ),
      ),
    );

    await tester.tap(
      find.byType(fluent.GestureDetector),
      buttons: kSecondaryButton,
    );
    await tester.pump();

    expect(tappedImage?.id, testImage.id);
  });
}
