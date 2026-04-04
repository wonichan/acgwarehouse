import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/widgets/tag_chip.dart';

class ViewerMetadataSidebar extends StatelessWidget {
  final ViewerSessionItem item;
  final List<Tag> tags;

  const ViewerMetadataSidebar({
    super.key,
    required this.item,
    required this.tags,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final panelSurface = _opaqueColor(colorScheme.surfaceContainerHighest);
    final foreground = _foregroundForSurface(panelSurface);
    final mutedForeground = _mutedForegroundForSurface(panelSurface);

    return Container(
      width: 320,
      decoration: BoxDecoration(
        color: panelSurface,
        border: Border(
          left: BorderSide(color: colorScheme.outlineVariant.withOpacity(0.5)),
        ),
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildMetadataSection(context, foreground, mutedForeground),
            const SizedBox(height: 16),
            _buildTagsSection(context, foreground),
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataSection(
    BuildContext context,
    Color foreground,
    Color mutedForeground,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Image Details',
          style: Theme.of(
            context,
          ).textTheme.titleMedium?.copyWith(color: foreground),
        ),
        const SizedBox(height: 6),
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
    );
  }

  Widget _buildTagsSection(BuildContext context, Color foreground) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Tags',
          style: Theme.of(
            context,
          ).textTheme.titleMedium?.copyWith(color: foreground),
        ),
        const SizedBox(height: 6),
        if (tags.isNotEmpty)
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: tags
                .map((tag) => TagChip(tag: tag, style: TagChipStyle.confirmed))
                .toList(),
          )
        else
          Text('No Tags', style: TextStyle(color: foreground)),
      ],
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

  Color _opaqueColor(Color color) {
    return Color.fromARGB(255, color.red, color.green, color.blue);
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
