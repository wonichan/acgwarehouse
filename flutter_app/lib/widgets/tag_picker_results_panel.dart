import 'package:flutter/material.dart';

import '../models/tag.dart';

String _tagLevelLabel(String? level) {
  switch (level) {
    case 'root':
      return '祖级';
    case 'parent':
      return '父级';
    case 'child':
      return '子级';
    default:
      return '未知层级';
  }
}

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
          final hierarchyParts = <String>[];
          if (tag.level != null) {
            hierarchyParts.add(_tagLevelLabel(tag.level));
          }
          if (tag.parentId != null) {
            hierarchyParts.add('父标签 #${tag.parentId}');
          }
          final subtitleParts = <String>[
            if (tag.primaryCategory != null && tag.primaryCategory!.isNotEmpty)
              tag.primaryCategory!,
            ...hierarchyParts,
          ];

          return ListTile(
            dense: true,
            title: Row(
              children: [
                Expanded(child: Text(tag.preferredLabel)),
                if (tag.level != null)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: colorScheme.primary.withValues(alpha: 0.08),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      _tagLevelLabel(tag.level),
                      style: Theme.of(context).textTheme.labelSmall?.copyWith(
                        color: colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
              ],
            ),
            subtitle: subtitleParts.isNotEmpty
                ? Text(subtitleParts.join(' · '))
                : null,
            trailing: Text('${tag.usageCount}'),
            onTap: () => onTagTap(tag),
          );
        },
      ),
    );
  }
}
