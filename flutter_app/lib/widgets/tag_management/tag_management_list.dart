import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../../models/tag_governance.dart';
import '../../providers/tag_provider.dart';

/// Tree-aware governance list displaying hierarchy rows with inline actions.
class TagManagementList extends StatelessWidget {
  final void Function(TagGovernanceRow row) onEdit;
  final void Function(TagGovernanceRow row) onMerge;
  final void Function(TagGovernanceRow row) onDelete;
  final void Function(TagGovernanceRow row) onViewAffectedImages;
  final String searchQuery;

  const TagManagementList({
    super.key,
    required this.onEdit,
    required this.onMerge,
    required this.onDelete,
    required this.onViewAffectedImages,
    this.searchQuery = '',
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<TagProvider>(
      builder: (context, provider, _) {
        final rows = provider.governanceRows;
        if (rows.isEmpty) {
          return const Center(child: Text('暂无治理标签'));
        }

        final rowById = {for (final row in rows) row.tagId: row};
        final tree = provider.tagTree?['tree'] as List<dynamic>?;
        final filteredTree = _filterTree(tree ?? const [], searchQuery);
        final items = (tree != null && tree.isNotEmpty)
            ? _buildTreeItems(filteredTree, rowById, provider, 0)
            : rows
                  .where(
                    (row) =>
                        searchQuery.trim().isEmpty ||
                        row.preferredLabel.toLowerCase().contains(
                          searchQuery.trim().toLowerCase(),
                        ),
                  )
                  .map(
                    (row) => _GovernanceRowTile(
                      row: row,
                      depth: 0,
                      isSelected: provider.selectedGovernanceIds.contains(
                        row.tagId,
                      ),
                      onToggleSelect: () =>
                          provider.toggleGovernanceSelection(row.tagId),
                      onEdit: () => onEdit(row),
                      onMerge: () => onMerge(row),
                      onDelete: () => onDelete(row),
                      onViewAffectedImages: () => onViewAffectedImages(row),
                    ),
                  )
                  .toList();

        return ListView(children: items);
      },
    );
  }

  List<dynamic> _filterTree(List<dynamic> nodes, String query) {
    final normalized = query.trim().toLowerCase();
    if (normalized.isEmpty) {
      return nodes;
    }

    final filtered = <dynamic>[];
    for (final rawNode in nodes) {
      final node = Map<String, dynamic>.from(rawNode as Map);
      final label = (node['preferred_label'] as String? ?? '').toLowerCase();
      final children = _filterTree(
        node['children'] as List<dynamic>? ?? const [],
        query,
      );
      if (label.contains(normalized) || children.isNotEmpty) {
        node['children'] = children;
        filtered.add(node);
      }
    }
    return filtered;
  }

  List<Widget> _buildTreeItems(
    List<dynamic> nodes,
    Map<int, TagGovernanceRow> rowById,
    TagProvider provider,
    int depth,
  ) {
    final widgets = <Widget>[];
    for (final node in nodes) {
      final tagId = ((node as Map)['tag_id'] ?? node['id']) as int;
      final childWidgets = _buildTreeItems(
        node['children'] as List<dynamic>? ?? const [],
        rowById,
        provider,
        depth + 1,
      );
      final matchedRow = rowById[tagId];
      if (matchedRow == null && childWidgets.isEmpty) {
        continue;
      }
      final row = matchedRow ?? _rowFromNode(node, rowById);
      widgets.add(
        _GovernanceRowTile(
          row: row,
          depth: depth,
          isSelected: provider.selectedGovernanceIds.contains(row.tagId),
          onToggleSelect: () => provider.toggleGovernanceSelection(row.tagId),
          onEdit: () => onEdit(row),
          onMerge: () => onMerge(row),
          onDelete: () => onDelete(row),
          onViewAffectedImages: () => onViewAffectedImages(row),
        ),
      );

      widgets.addAll(childWidgets);
    }
    return widgets;
  }

  TagGovernanceRow _rowFromNode(
    dynamic rawNode,
    Map<int, TagGovernanceRow> rowById,
  ) {
    final node = Map<String, dynamic>.from(rawNode as Map);
    final tagId = (node['tag_id'] ?? node['id']) as int;
    return rowById[tagId]!;
  }
}

class _GovernanceRowTile extends StatelessWidget {
  final TagGovernanceRow row;
  final int depth;
  final bool isSelected;
  final VoidCallback onToggleSelect;
  final VoidCallback onEdit;
  final VoidCallback onMerge;
  final VoidCallback onDelete;
  final VoidCallback onViewAffectedImages;

  const _GovernanceRowTile({
    required this.row,
    required this.depth,
    required this.isSelected,
    required this.onToggleSelect,
    required this.onEdit,
    required this.onMerge,
    required this.onDelete,
    required this.onViewAffectedImages,
  });

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final level = row.level ?? 'child';
    final levelColor = switch (level) {
      'root' => theme.accentColor,
      'parent' => Colors.orange,
      _ => theme.resources.textFillColorSecondary,
    };

    return Padding(
      padding: EdgeInsets.only(left: 12.0 + (depth * 24), right: 12, top: 4),
      child: Card(
        child: ListTile(
          leading: Checkbox(
            checked: isSelected,
            onChanged: (_) => onToggleSelect(),
          ),
          title: Row(
            children: [
              if (depth > 0)
                Container(
                  width: 14,
                  height: 1,
                  margin: const EdgeInsets.only(right: 8),
                  color: theme.resources.controlStrokeColorDefault,
                ),
              Flexible(child: Text(row.preferredLabel)),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: levelColor.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  level,
                  style: TextStyle(fontSize: 10, color: levelColor),
                ),
              ),
            ],
          ),
          subtitle: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: [
                  if (row.primaryCategory != null)
                    Text(
                      row.primaryCategory!,
                      style: const TextStyle(fontSize: 11),
                    ),
                  Text(
                    '直接使用量: ${row.directUsageCount} | 树总使用量: ${row.treeUsageCount}',
                    style: const TextStyle(fontSize: 11),
                  ),
                  Text(
                    'AI(直/树): ${row.directAiCount}/${row.treeAiCount} | 手动(直/树): ${row.directManualCount}/${row.treeManualCount}',
                    style: const TextStyle(fontSize: 11),
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: [
                  HyperlinkButton(onPressed: onEdit, child: const Text('编辑')),
                  HyperlinkButton(onPressed: onMerge, child: const Text('合并')),
                  HyperlinkButton(onPressed: onDelete, child: const Text('删除')),
                  HyperlinkButton(
                    onPressed: onViewAffectedImages,
                    child: const Text('查看受影响图片'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
