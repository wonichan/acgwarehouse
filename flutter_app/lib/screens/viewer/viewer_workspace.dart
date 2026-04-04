import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/screens/viewer/viewer_stage.dart';
import 'package:gallery/screens/viewer/viewer_filmstrip.dart';
import 'package:gallery/screens/viewer/viewer_keyboard_scope.dart';

class ViewerWorkspace extends StatefulWidget {
  final ViewerSession session;
  final ValueChanged<ViewerSessionItem>? onItemChanged;
  final VoidCallback? onEscape;

  const ViewerWorkspace({
    super.key,
    required this.session,
    this.onItemChanged,
    this.onEscape,
  });

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

  void _updateSelection(int newIndex) {
    if (_selectedIndex != newIndex) {
      setState(() => _selectedIndex = newIndex);
      if (widget.onItemChanged != null) {
        widget.onItemChanged!(widget.session.items[_selectedIndex]);
      }
    }
  }

  void _handleNext() {
    if (_selectedIndex < widget.session.items.length - 1) {
      _updateSelection(_selectedIndex + 1);
    }
  }

  void _handlePrevious() {
    if (_selectedIndex > 0) {
      _updateSelection(_selectedIndex - 1);
    }
  }

  void _handleEscape() {
    if (widget.onEscape != null) {
      widget.onEscape!();
    }
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
              _updateSelection(idx);
            },
          ),
        ],
      ),
    );
  }
}
