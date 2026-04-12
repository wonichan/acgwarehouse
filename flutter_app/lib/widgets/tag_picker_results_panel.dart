import 'package:flutter/material.dart';

import '../models/tag.dart';

class TagPickerResultsPanel extends StatelessWidget {
  final List<Tag> tags;
  final bool isLoading;
  final bool isLoadingMore;
  final String emptyMessage;
  final ScrollController? scrollController;
  final ValueChanged<Tag> onTagTap;

  const TagPickerResultsPanel({
    super.key,
    required this.tags,
    required this.isLoading,
    required this.isLoadingMore,
    required this.emptyMessage,
    required this.onTagTap,
    this.scrollController,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    if (isLoading && tags.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (tags.isEmpty) {
      return Center(
        child: Text(
          emptyMessage,
          style: Theme.of(
            context,
          ).textTheme.bodyMedium?.copyWith(color: colorScheme.onSurfaceVariant),
          textAlign: TextAlign.center,
        ),
      );
    }

    return Container(
      margin: const EdgeInsets.only(top: 8),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerHighest,
        border: Border.all(color: colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(12),
      ),
      child: ListView.builder(
        controller: scrollController,
        itemCount: tags.length + (isLoadingMore ? 1 : 0),
        itemBuilder: (context, index) {
          if (index >= tags.length) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(
                child: SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            );
          }

          final tag = tags[index];
          return ListTile(
            dense: true,
            title: Text(tag.preferredLabel),
            subtitle: tag.primaryCategory != null
                ? Text(tag.primaryCategory!)
                : null,
            trailing: Text('${tag.usageCount}'),
            onTap: () => onTagTap(tag),
          );
        },
      ),
    );
  }
}
