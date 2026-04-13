import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart' as material;

import '../widgets/tag_management/tag_management_workspace.dart';

/// Material-route wrapper for the shared tag-governance workspace.
///
/// This keeps legacy Material navigation paths pointed at the same
/// hierarchy-aware management experience used by the Fluent shell.
class TagManagementScreen extends material.StatelessWidget {
  const TagManagementScreen({super.key});

  @override
  material.Widget build(material.BuildContext context) {
    final brightness = material.Theme.of(context).brightness;

    return fluent.FluentTheme(
      data: fluent.FluentThemeData(
        brightness: brightness == material.Brightness.dark
            ? fluent.Brightness.dark
            : fluent.Brightness.light,
      ),
      child: const TagManagementWorkspace(),
    );
  }
}
