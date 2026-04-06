import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';
import 'package:gallery/widgets/image_metadata_pane_theme.dart';

class ViewerMetadataSidebar extends StatelessWidget {
  final ViewerSessionItem item;

  const ViewerMetadataSidebar({super.key, required this.item});

  @override
  Widget build(BuildContext context) {
    final paneTheme = ImageMetadataPaneTheme.of(context);

    return Container(
      key: const ValueKey('viewer-metadata-sidebar'),
      width: 320,
      decoration: BoxDecoration(
        color: paneTheme.panelSurface,
        border: Border(left: BorderSide(color: paneTheme.borderColor)),
      ),
      child: Material(
        type: MaterialType.transparency,
        child: ImageMetadataPanel(
          imageId: item.imageId,
          metadataSection: _buildMetadataSection(context, paneTheme),
        ),
      ),
    );
  }

  Widget _buildMetadataSection(
    BuildContext context,
    ImageMetadataPaneTheme paneTheme,
  ) {
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      decoration: paneTheme.sectionDecoration,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Image Details',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: paneTheme.textForeground,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 12),
            _buildMetadataRow(
              'Filename',
              item.filename,
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
            _buildMetadataRow(
              'Format',
              item.format.toUpperCase(),
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
            _buildMetadataRow(
              'Resolution',
              '${item.width}x${item.height}',
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
            _buildMetadataRow(
              'Size',
              '${(item.fileSize / 1024).toStringAsFixed(1)} KB',
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
            _buildMetadataRow(
              'Path',
              item.path,
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
            _buildMetadataRow(
              'Imported',
              item.createdAtIso8601,
              paneTheme.textForeground,
              paneTheme.textMuted,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataRow(
    String label,
    String value,
    Color foreground,
    Color mutedForeground,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 70,
            child: Text(
              label,
              style: TextStyle(color: mutedForeground, fontSize: 13),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: TextStyle(
                color: foreground,
                fontWeight: FontWeight.w500,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
