import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import '../services/viewer_window_service.dart';
import '../theme/app_theme.dart';
import '../utils/window_manager.dart';
import '../screens/viewer/viewer_workspace.dart';

class ViewerWindowApp extends StatefulWidget {
  final ViewerWindowBootstrapData bootstrapData;

  const ViewerWindowApp({super.key, required this.bootstrapData});

  @override
  State<ViewerWindowApp> createState() => _ViewerWindowAppState();
}

class _ViewerWindowAppState extends State<ViewerWindowApp> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      AppWindowManager.setTitle(widget.bootstrapData.policy.title);
    });
  }

  @override
  Widget build(BuildContext context) {
    return fluent.FluentApp(
      title: widget.bootstrapData.policy.title,
      theme: AppTheme.getFluentTheme(Brightness.dark),
      home: fluent.NavigationView(
        content: ViewerWorkspace(
          session: widget.bootstrapData.session,
          onItemChanged: (item) {
            AppWindowManager.setTitle(
              ViewerWindowService.buildWindowTitle(item.filename),
            );
          },
          onEscape: () {
            // Close the window on escape
            AppWindowManager.close();
          },
        ),
      ),
    );
  }
}
