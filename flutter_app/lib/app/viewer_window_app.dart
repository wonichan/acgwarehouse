import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import '../services/viewer_window_service.dart';
import '../theme/app_theme.dart';
import '../utils/window_manager.dart';

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
        content: fluent.ScaffoldPage(
          content: Container(
            color: const Color(0xFF0F1115),
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  widget.bootstrapData.session.selectedItem.filename,
                  style: const TextStyle(
                    fontSize: 24,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFFFFFFFF),
                  ),
                ),
                const SizedBox(height: 12),
                const Text(
                  'Viewer host ready',
                  style: TextStyle(color: Color(0xFFC9D1D9)),
                ),
                const SizedBox(height: 16),
                Text(
                  'Window ${widget.bootstrapData.windowId} · ${widget.bootstrapData.session.items.length} item(s)',
                  style: const TextStyle(color: Color(0xFFC9D1D9)),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
