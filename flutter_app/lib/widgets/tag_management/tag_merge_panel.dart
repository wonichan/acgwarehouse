import 'package:fluent_ui/fluent_ui.dart';
import '../../models/tag_governance.dart';

/// Merge target-selection and confirmation panel.
///
/// Renders source-tag context, searchable target choices, confirm copy,
/// and a disabled confirm action until a valid target is selected.
class TagMergePanel extends StatefulWidget {
  final TagGovernanceRow sourceRow;
  final List<TagGovernanceRow> allRows;
  final Future<void> Function(int targetTagId) onConfirm;
  final VoidCallback onCancel;

  const TagMergePanel({
    super.key,
    required this.sourceRow,
    required this.allRows,
    required this.onConfirm,
    required this.onCancel,
  });

  @override
  State<TagMergePanel> createState() => _TagMergePanelState();
}

class _TagMergePanelState extends State<TagMergePanel> {
  int? _selectedTargetTagId;
  String _searchQuery = '';
  bool _isMerging = false;

  List<TagGovernanceRow> get _availableTargets {
    return widget.allRows.where((r) => r.tagId != widget.sourceRow.tagId).where(
      (r) {
        if (_searchQuery.isEmpty) return true;
        return r.preferredLabel.toLowerCase().contains(
          _searchQuery.toLowerCase(),
        );
      },
    ).toList();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      color: FluentTheme.of(context).resources.cardBackgroundFillColorSecondary,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Text(
                'Merge "${widget.sourceRow.preferredLabel}" into:',
                style: FluentTheme.of(context).typography.bodyStrong,
              ),
              const Spacer(),
              IconButton(
                icon: const Icon(FluentIcons.cancel),
                onPressed: widget.onCancel,
              ),
            ],
          ),
          const SizedBox(height: 8),
          TextBox(
            placeholder: 'Search target tag...',
            onChanged: (value) => setState(() => _searchQuery = value),
          ),
          const SizedBox(height: 8),
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 120),
            child: ListView(
              shrinkWrap: true,
              children: _availableTargets.map((row) {
                final isSelected = _selectedTargetTagId == row.tagId;
                return ListTile(
                  title: Text(row.preferredLabel),
                  subtitle: Text(
                    'targetTagId: ${row.tagId}',
                    style: const TextStyle(fontSize: 10),
                  ),
                  trailing: isSelected
                      ? const Icon(FluentIcons.check_mark)
                      : null,
                  onPressed: () =>
                      setState(() => _selectedTargetTagId = row.tagId),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 8),
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              Button(child: const Text('Cancel'), onPressed: widget.onCancel),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: (_selectedTargetTagId != null && !_isMerging)
                    ? _handleConfirm
                    : null,
                child: _isMerging
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: ProgressRing(strokeWidth: 2),
                      )
                    : const Text('Confirm Merge'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _handleConfirm() async {
    if (_selectedTargetTagId == null) return;
    setState(() => _isMerging = true);
    try {
      await widget.onConfirm(_selectedTargetTagId!);
    } finally {
      if (mounted) {
        setState(() => _isMerging = false);
      }
    }
  }
}
