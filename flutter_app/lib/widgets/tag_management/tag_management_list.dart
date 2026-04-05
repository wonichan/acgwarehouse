import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../models/tag_governance.dart';
import '../../providers/tag_provider.dart';

/// List-first governance table displaying tag rows with inline actions.
class TagManagementList extends StatelessWidget {
  final void Function(TagGovernanceRow row) onEdit;
  final void Function(TagGovernanceRow row) onMerge;
  final void Function(TagGovernanceRow row) onDelete;
  final void Function(TagGovernanceRow row) onViewAffectedImages;

  const TagManagementList({
    super.key,
    required this.onEdit,
    required this.onMerge,
    required this.onDelete,
    required this.onViewAffectedImages,
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<TagProvider>(
      builder: (context, provider, _) {
        final rows = provider.governanceRows;

        if (rows.isEmpty) {
          return const Center(child: Text('No governance tags found'));
        }

        return ListView.builder(
          itemCount: rows.length,
          itemBuilder: (context, index) {
            final row = rows[index];
            return _GovernanceRowTile(
              row: row,
              isSelected: provider.selectedGovernanceIds.contains(row.tagId),
              onToggleSelect: () =>
                  provider.toggleGovernanceSelection(row.tagId),
              onEdit: () => onEdit(row),
              onMerge: () => onMerge(row),
              onDelete: () => onDelete(row),
              onViewAffectedImages: () => onViewAffectedImages(row),
            );
          },
        );
      },
    );
  }
}

class _GovernanceRowTile extends StatelessWidget {
  final TagGovernanceRow row;
  final bool isSelected;
  final VoidCallback onToggleSelect;
  final VoidCallback onEdit;
  final VoidCallback onMerge;
  final VoidCallback onDelete;
  final VoidCallback onViewAffectedImages;

  const _GovernanceRowTile({
    required this.row,
    required this.isSelected,
    required this.onToggleSelect,
    required this.onEdit,
    required this.onMerge,
    required this.onDelete,
    required this.onViewAffectedImages,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Checkbox(
        checked: isSelected,
        onChanged: (_) => onToggleSelect(),
      ),
      title: Text(row.preferredLabel),
      subtitle: Row(
        children: [
          if (row.primaryCategory != null)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: Text(
                row.primaryCategory!,
                style: TextStyle(fontSize: 11, color: Colors.grey[100]),
              ),
            ),
          Text(
            'Usage: ${row.usageCount} | AI: ${row.aiCount} | Manual: ${row.manualCount}',
            style: const TextStyle(fontSize: 11),
          ),
        ],
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          HyperlinkButton(onPressed: onEdit, child: const Text('Edit')),
          HyperlinkButton(onPressed: onMerge, child: const Text('Merge')),
          HyperlinkButton(onPressed: onDelete, child: const Text('Delete')),
          HyperlinkButton(
            onPressed: onViewAffectedImages,
            child: const Text('View affected images'),
          ),
        ],
      ),
    );
  }
}
