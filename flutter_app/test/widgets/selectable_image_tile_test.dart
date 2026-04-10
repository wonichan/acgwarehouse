import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/widgets/selectable_image_tile.dart';

ImageModel _buildImage() {
  return ImageModel(
    id: 1,
    path: 'C:/images/test.png',
    filename: 'test.png',
    sourceRoot: 'C:/images',
    fileSize: 2048,
    width: 800,
    height: 600,
    format: 'png',
    phash: 12345,
    createdAt: DateTime.parse('2026-04-10T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-10T00:00:00.000Z'),
  );
}

void main() {
  testWidgets('SelectableImageTile uses ColorScheme colors for selection UI', (
    tester,
  ) async {
    final theme = ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: Colors.purple),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: theme,
        home: Scaffold(
          body: SelectableImageTile(
            image: _buildImage(),
            isSelectionMode: true,
            isSelected: true,
            imageBuilder: (_) => const ColoredBox(color: Colors.black12),
          ),
        ),
      ),
    );

    final overlayContainer = tester.widget<Container>(
      find.byWidgetPredicate(
        (widget) =>
            widget is Container &&
            widget.decoration is BoxDecoration &&
            (widget.decoration as BoxDecoration).border != null,
      ),
    );
    final overlayDecoration = overlayContainer.decoration! as BoxDecoration;
    final overlayBorder = overlayDecoration.border! as Border;

    expect(overlayBorder.top.color, theme.colorScheme.primary);
    expect(
      overlayDecoration.color,
      theme.colorScheme.primary.withValues(alpha: 0.1),
    );

    final checkIcon = tester.widget<Icon>(find.byIcon(Icons.check));
    expect(checkIcon.color, theme.colorScheme.onPrimary);
  });
}
