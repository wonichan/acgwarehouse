import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/screens/viewer/viewer_stage.dart';
import 'package:gallery/screens/viewer/viewer_filmstrip.dart';
import 'package:gallery/screens/viewer/viewer_keyboard_scope.dart';

class ViewerWorkspace extends StatefulWidget {
  final ViewerSession session;

  const ViewerWorkspace({super.key, required this.session});

  @override
  State<ViewerWorkspace> createState() => _ViewerWorkspaceState();
}

class _ViewerWorkspaceState extends State<ViewerWorkspace> {
  late int _selectedIndex;

  @override
  void initState() {
    super.initState();
    _selectedIndex = widget.session.initialSelectedIndex;
  }

  void _handleNext() {
    if (_selectedIndex < widget.session.items.length - 1) {
      setState(() => _selectedIndex++);
    }
  }

  void _handlePrevious() {
    if (_selectedIndex > 0) {
      setState(() => _selectedIndex--);
    }
  }

  void _handleEscape() {
    // In a real app, this would close the window or modal.
    // For now, let's assume parent handles it if needed or it's a no-op.
  }

  @override
  Widget build(BuildContext context) {
    final currentItem = widget.session.items[_selectedIndex];

    return ViewerKeyboardScope(
      onNext: _handleNext,
      onPrevious: _handlePrevious,
      onEscape: _handleEscape,
      child: Column(
        children: [
          // Title/chrome band
          Container(
            height: 40,
            color: Theme.of(context).colorScheme.surface,
            child: Center(child: Text('Viewer - ${currentItem.filename}')),
          ),
          Expanded(
            child: Row(
              children: [
                // Main stage region
                Expanded(
                  child: Container(
                    color: Theme.of(context).colorScheme.surfaceContainer,
                    child: ViewerStage(item: currentItem),
                  ),
                ),
                // Fixed right sidebar
                ViewerMetadataSidebar(
                  item: currentItem,
                  tags: const [], // Placeholder for tags
                ),
              ],
            ),
          ),
          // Bottom filmstrip region
          ViewerFilmstrip(
            session: widget.session,
            selectedIndex: _selectedIndex,
            onIndexChanged: (idx) {
              setState(() => _selectedIndex = idx);
            },
          ),
        ],
      ),
    );
  }
}
