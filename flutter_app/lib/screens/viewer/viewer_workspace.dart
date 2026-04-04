import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/screens/viewer/viewer_stage.dart';
import 'package:gallery/screens/viewer/viewer_filmstrip.dart';
import 'package:gallery/screens/viewer/viewer_keyboard_scope.dart';
import 'package:gallery/services/tag_service.dart';

class ViewerWorkspace extends StatefulWidget {
  final ViewerSession session;
  final ValueChanged<ViewerSessionItem>? onItemChanged;
  final VoidCallback? onEscape;
  final TagProvider? tagProvider;

  const ViewerWorkspace({
    super.key,
    required this.session,
    this.onItemChanged,
    this.onEscape,
    this.tagProvider,
  });

  @override
  State<ViewerWorkspace> createState() => _ViewerWorkspaceState();
}

class _ViewerWorkspaceState extends State<ViewerWorkspace> {
  late int _selectedIndex;
  late TagProvider _tagProvider;
  late bool _ownsTagProvider;

  @override
  void initState() {
    super.initState();
    _selectedIndex = widget.session.initialSelectedIndex;
    _ownsTagProvider = widget.tagProvider == null;
    _tagProvider = widget.tagProvider ?? TagProvider(TagService());
    _loadCurrentItemTags();
  }

  @override
  void didUpdateWidget(covariant ViewerWorkspace oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (oldWidget.tagProvider != widget.tagProvider) {
      if (_ownsTagProvider) {
        _tagProvider.dispose();
      }
      _ownsTagProvider = widget.tagProvider == null;
      _tagProvider = widget.tagProvider ?? TagProvider(TagService());
      _loadCurrentItemTags();
    }

    if (oldWidget.session != widget.session) {
      _selectedIndex = widget.session.initialSelectedIndex;
      _loadCurrentItemTags();
    }
  }

  @override
  void dispose() {
    if (_ownsTagProvider) {
      _tagProvider.dispose();
    }
    super.dispose();
  }

  Future<void> _loadCurrentItemTags() async {
    await _tagProvider.loadImageTags(
      widget.session.items[_selectedIndex].imageId,
    );
  }

  void _updateSelection(int newIndex) {
    if (_selectedIndex != newIndex) {
      setState(() => _selectedIndex = newIndex);
      unawaited(_loadCurrentItemTags());
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

    return ChangeNotifierProvider<TagProvider>.value(
      value: _tagProvider,
      child: Consumer<TagProvider>(
        builder: (context, tagProvider, child) {
          final confirmedTags =
              tagProvider.imageTags['confirmed'] ?? const <Tag>[];

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
                  child: Center(
                    child: Text('Viewer - ${currentItem.filename}'),
                  ),
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
                        tags: confirmedTags,
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
        },
      ),
    );
  }
}
