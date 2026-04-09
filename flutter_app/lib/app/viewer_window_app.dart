import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import '../services/api_service.dart';
import '../providers/tag_provider.dart';
import '../services/viewer_window_service.dart';
import '../theme/app_theme.dart';
import '../utils/window_manager.dart';
import '../screens/viewer/viewer_workspace.dart';
import '../widgets/desktop_material_theme_bridge.dart';

class ViewerWindowApp extends StatefulWidget {
  final ViewerWindowBootstrapData bootstrapData;
  final ApiService? apiService;
  final TagProvider? tagProvider;
  final Future<void> Function(String title)? titleSetter;
  final Future<void> Function()? closeWindow;

  const ViewerWindowApp({
    super.key,
    required this.bootstrapData,
    this.apiService,
    this.tagProvider,
    this.titleSetter,
    this.closeWindow,
  });

  @override
  State<ViewerWindowApp> createState() => _ViewerWindowAppState();
}

class _ViewerWindowAppState extends State<ViewerWindowApp> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _setTitle(widget.bootstrapData.policy.title);
    });
  }

  Future<void> _setTitle(String title) {
    return (widget.titleSetter ?? AppWindowManager.setTitle)(title);
  }

  Future<void> _closeWindow() {
    return (widget.closeWindow ?? AppWindowManager.close)();
  }

  @override
  Widget build(BuildContext context) {
    return fluent.FluentApp(
      title: widget.bootstrapData.policy.title,
      theme: AppTheme.getFluentTheme(Brightness.dark),
      home: fluent.ScaffoldPage(
        content: ViewerWorkspace(
          launchContext: widget.bootstrapData.context,
          apiService:
              widget.apiService ??
              ApiService(baseUrl: widget.bootstrapData.baseUrl),
          tagProvider: widget.tagProvider,
          onItemChanged: (item) {
            _setTitle(ViewerWindowService.buildWindowTitle(item.filename));
          },
          onEscape: () {
            _closeWindow();
          },
        ),
      ),
      builder: (context, child) => DesktopMaterialThemeBridge(
        brightness: Brightness.dark,
        child: child ?? const SizedBox.shrink(),
      ),
    );
  }
}
