import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/theme/monitoring_theme.dart';

void main() {
  testWidgets('MonitoringTheme uses Fluent semantic status colors', (
    tester,
  ) async {
    late MonitoringTheme monitoringTheme;
    late ResourceDictionary resources;

    await tester.pumpWidget(
      FluentApp(
        home: Builder(
          builder: (context) {
            monitoringTheme = MonitoringTheme.of(context);
            resources = FluentTheme.of(context).resources;
            return const SizedBox.shrink();
          },
        ),
      ),
    );

    expect(monitoringTheme.statusPending, resources.systemFillColorCaution);
    expect(monitoringTheme.statusCompleted, resources.systemFillColorSuccess);
    expect(monitoringTheme.statusFailed, resources.systemFillColorCritical);
    expect(
      monitoringTheme.errorBadgeBackground,
      resources.systemFillColorCritical,
    );
    expect(
      monitoringTheme.errorBadgeText,
      resources.textOnAccentFillColorPrimary,
    );
  });
}
