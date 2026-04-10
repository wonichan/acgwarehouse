import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/image_metadata_pane_theme.dart';

void main() {
  testWidgets('ImageMetadataPaneTheme exposes theme-driven tag badge colors', (
    tester,
  ) async {
    late ImageMetadataPaneTheme paneTheme;
    late ColorScheme colorScheme;

    await tester.pumpWidget(
      MaterialApp(
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
        ),
        home: Builder(
          builder: (context) {
            paneTheme = ImageMetadataPaneTheme.of(context);
            colorScheme = Theme.of(context).colorScheme;
            return const SizedBox.shrink();
          },
        ),
      ),
    );

    expect(
      paneTheme.pendingBadgeBackground,
      colorScheme.secondaryContainer.withValues(alpha: 0.7),
    );
    expect(paneTheme.pendingBadgeForeground, colorScheme.onSecondaryContainer);
    expect(
      paneTheme.confirmedBadgeBackground,
      colorScheme.primaryContainer.withValues(alpha: 0.7),
    );
    expect(paneTheme.confirmedBadgeForeground, colorScheme.onPrimaryContainer);
  });
}
