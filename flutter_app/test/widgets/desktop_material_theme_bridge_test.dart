import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/theme/app_theme.dart';
import 'package:gallery/widgets/desktop_material_theme_bridge.dart';

void main() {
  testWidgets('bridges Material dialog and snackbar themes in Fluent app', (
    tester,
  ) async {
    ThemeData? capturedTheme;

    await tester.pumpWidget(
      fluent.FluentApp(
        home: DesktopMaterialThemeBridge(
          brightness: Brightness.dark,
          child: Builder(
            builder: (context) {
              capturedTheme = Theme.of(context);
              return const SizedBox.shrink();
            },
          ),
        ),
      ),
    );

    final expectedTheme = AppTheme.getMaterialTheme(Brightness.dark);

    expect(capturedTheme, isNotNull);
    expect(
      capturedTheme!.dialogTheme.backgroundColor,
      expectedTheme.dialogTheme.backgroundColor,
    );
    expect(
      capturedTheme!.snackBarTheme.backgroundColor,
      expectedTheme.snackBarTheme.backgroundColor,
    );
    expect(
      capturedTheme!.snackBarTheme.contentTextStyle?.color,
      expectedTheme.snackBarTheme.contentTextStyle?.color,
    );
  });
}
