import 'package:flutter/material.dart';

import '../theme/app_theme.dart';

class DesktopMaterialThemeBridge extends StatelessWidget {
  final Brightness brightness;
  final Widget child;

  const DesktopMaterialThemeBridge({
    super.key,
    required this.brightness,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: AppTheme.getMaterialTheme(brightness),
      child: ScaffoldMessenger(child: child),
    );
  }
}
