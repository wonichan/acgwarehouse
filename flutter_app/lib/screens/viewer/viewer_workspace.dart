import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';

class ViewerWorkspace extends StatelessWidget {
  final ViewerSession session;

  const ViewerWorkspace({super.key, required this.session});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Title/chrome band
        Container(
          height: 40,
          color: Theme.of(context).colorScheme.surface,
          child: const Center(child: Text('Viewer')),
        ),
        Expanded(
          child: Row(
            children: [
              // Main stage region
              Expanded(
                child: Container(
                  color: Theme.of(context).colorScheme.surfaceContainer,
                  child: const Center(child: Text('Stage Placeholder')),
                ),
              ),
              // Fixed right sidebar
              ViewerMetadataSidebar(
                item: session.selectedItem,
                tags: const [], // Placeholder for tags
              ),
            ],
          ),
        ),
        // Bottom filmstrip region
        Container(
          height: 120,
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          child: const Center(child: Text('Filmstrip Placeholder')),
        ),
      ],
    );
  }
}
