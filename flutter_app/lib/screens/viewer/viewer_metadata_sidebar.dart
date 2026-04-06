import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';

class ViewerMetadataSidebar extends StatelessWidget {
  final ViewerSessionItem item;

  const ViewerMetadataSidebar({super.key, required this.item});

  @override
  Widget build(BuildContext context) {
    // Windows Photos-like background, always light regardless of app theme
    const panelSurface = Color(0xFFF3F3F3);
    final foreground = _foregroundForSurface(panelSurface);
    final mutedForeground = _mutedForegroundForSurface(panelSurface);

    return Container(
      width: 320,
      decoration: const BoxDecoration(
        color: panelSurface,
        border: Border(left: BorderSide(color: Color(0xFFE5E5E5))),
      ),
      child: Material(
        type: MaterialType.transparency,
        child: ImageMetadataPanel(
          imageId: item.imageId,
          metadataSection: _buildMetadataSection(
            context,
            foreground,
            mutedForeground,
          ),
        ),
      ),
    );
  }

  Widget _buildMetadataSection(
    BuildContext context,
    Color foreground,
    Color mutedForeground,
  ) {
    return Card(
      elevation: 0,
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: const BorderSide(color: Color(0xFFE5E5E5)),
      ),
      color: Colors.white,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Image Details',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: foreground,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 12),
            _buildMetadataRow(
              'Filename',
              item.filename,
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              'Format',
              item.format.toUpperCase(),
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              'Resolution',
              '${item.width}x${item.height}',
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              'Size',
              '${(item.fileSize / 1024).toStringAsFixed(1)} KB',
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow('Path', item.path, foreground, mutedForeground),
            _buildMetadataRow(
              'Imported',
              item.createdAtIso8601,
              foreground,
              mutedForeground,
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

  Color _foregroundForSurface(Color surface) {
    return ThemeData.estimateBrightnessForColor(surface) == Brightness.dark
        ? Colors.white
        : Colors.black87;
  }

  Color _mutedForegroundForSurface(Color surface) {
    return ThemeData.estimateBrightnessForColor(surface) == Brightness.dark
        ? Colors.white70
        : Colors.black54;
  }
}
