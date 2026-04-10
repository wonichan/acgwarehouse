import 'package:fluent_ui/fluent_ui.dart';

class MonitoringTheme {
  final BuildContext context;

  MonitoringTheme.of(this.context);

  FluentThemeData get theme => FluentTheme.of(context);
  ResourceDictionary get resources => theme.resources;

  Color get cardBackground => resources.cardBackgroundFillColorDefault;
  Color get cardBorder => resources.cardStrokeColorDefault;

  Color get selectedBackground => theme.accentColor.withValues(alpha: 0.1);
  Color get selectedBorder => theme.accentColor;

  Color get detailBackground => resources.cardBackgroundFillColorSecondary;

  Color get progressBackground => resources.controlStrongFillColorDefault;
  Color get progressActive => theme.accentColor;

  Color get timestampText => resources.textFillColorSecondary;

  Color get emptyStateBackground => resources.cardBackgroundFillColorDefault;
  Color get emptyStateBorder => resources.cardStrokeColorDefault;

  Color get statusPending => resources.systemFillColorCaution;
  Color get statusRunning => theme.accentColor;
  Color get statusCompleted => resources.systemFillColorSuccess;
  Color get statusFailed => resources.systemFillColorCritical;
  Color get statusUnknown => resources.textFillColorSecondary;

  Color get errorBadgeBackground => resources.systemFillColorCritical;
  Color get errorBadgeText => resources.textOnAccentFillColorPrimary;

  Color get statusBadgeText => resources.textOnAccentFillColorPrimary;
}
