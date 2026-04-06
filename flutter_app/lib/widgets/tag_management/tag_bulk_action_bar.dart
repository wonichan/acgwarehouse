import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../models/tag_governance.dart';
import '../../providers/tag_provider.dart';

/// Multi-select governance action surface.
///
/// Visible only when `selectedGovernanceIds` is non-empty.
/// Provides controls for selected cleanup, bulk category assignment,
/// alias organization, and merge-candidate processing.
class TagBulkActionBar extends StatelessWidget {
  final Future<void> Function() onCleanup;
  final Future<void> Function(int targetTagId) onMergeInto;

  const TagBulkActionBar({
    super.key,
    required this.onCleanup,
    required this.onMergeInto,
  });

  @override
  Widget build(BuildContext context) {
    return Consumer<TagProvider>(
      builder: (context, provider, _) {
        final selectedCount = provider.selectedGovernanceIds.length;
        if (selectedCount == 0) return const SizedBox.shrink();

        final lastResult = provider.lastBatchResult;

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          color: FluentTheme.of(
            context,
          ).resources.cardBackgroundFillColorSecondary,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 8,
                runSpacing: 8,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  Text(
                    '$selectedCount 已选中',
                    style: FluentTheme.of(context).typography.bodyStrong,
                  ),
                  if (provider.isRunningGovernanceAction)
                    const SizedBox(
                      width: 16,
                      height: 16,
                      child: ProgressRing(strokeWidth: 2),
                    ),
                  Button(
                    onPressed: provider.isRunningGovernanceAction
                        ? null
                        : () => _handleCleanup(context, provider),
                    child: const Text('清理已选中'),
                  ),
                  Button(
                    onPressed: provider.isRunningGovernanceAction
                        ? null
                        : () => _showMergeTargetDialog(context, provider),
                    child: const Text('合并到...'),
                  ),
                  HyperlinkButton(
                    onPressed: () => provider.clearGovernanceSelection(),
                    child: const Text('清除选择'),
                  ),
                ],
              ),
              if (lastResult != null) ...[
                const SizedBox(height: 4),
                _buildResultSummary(lastResult),
              ],
            ],
          ),
        );
      },
    );
  }

  Widget _buildResultSummary(TagGovernanceBatchResult result) {
    final succeeded =
        result.deletedTagIds.length +
        (result.failures.isEmpty ? result.deletedTagIds.length : 0);
    final failed = result.failures.length;
    // Use the actual batch result fields for display
    final int successCount = result.deletedTagIds.isNotEmpty
        ? result.deletedTagIds.length
        : (result.failures.isEmpty ? 1 : 0);

    return InfoBar(
      title: Text('$successCount 成功${failed > 0 ? '，$failed 失败' : ''}'),
      severity: failed > 0 ? InfoBarSeverity.warning : InfoBarSeverity.success,
    );
  }

  Future<void> _handleCleanup(
    BuildContext context,
    TagProvider provider,
  ) async {
    await onCleanup();
  }

  void _showMergeTargetDialog(BuildContext context, TagProvider provider) {
    final rows = provider.governanceRows
        .where((r) => !provider.selectedGovernanceIds.contains(r.tagId))
        .toList();

    showDialog(
      context: context,
      builder: (dialogContext) => ContentDialog(
        title: const Text('将已选中合并到目标'),
        content: SizedBox(
          width: 300,
          height: 200,
          child: ListView(
            children: rows.map((row) {
              return ListTile(
                title: Text(row.preferredLabel),
                subtitle: Text(
                  'selectedGovernanceIds: ${provider.selectedGovernanceIds.join(", ")}',
                  style: const TextStyle(fontSize: 10),
                ),
                onPressed: () {
                  Navigator.pop(dialogContext);
                  onMergeInto(row.tagId);
                },
              );
            }).toList(),
          ),
        ),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(dialogContext),
          ),
        ],
      ),
    );
  }
}
