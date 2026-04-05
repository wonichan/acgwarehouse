import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../models/tag_governance.dart';
import '../../providers/image_provider.dart';
import '../../providers/navigation_provider.dart';
import '../../providers/tag_provider.dart';
import 'tag_management_list.dart';
import 'tag_merge_panel.dart';
import 'tag_bulk_action_bar.dart';
import 'tag_edit_dialog.dart';

/// Fluent desktop tag-governance workspace.
///
/// List-first layout: the governance table is the primary surface.
/// Merge panel and bulk action bar appear as adjunct tools.
class TagManagementWorkspace extends StatefulWidget {
  const TagManagementWorkspace({super.key});

  @override
  State<TagManagementWorkspace> createState() => _TagManagementWorkspaceState();
}

class _TagManagementWorkspaceState extends State<TagManagementWorkspace> {
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final provider = context.read<TagProvider>();
      if (provider.governanceRows.isEmpty) {
        provider.loadGovernanceTags();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: PageHeader(
        title: const Text('Tag Governance'),
        commandBar: CommandBar(
          mainAxisAlignment: MainAxisAlignment.end,
          primaryItems: [
            CommandBarButton(
              icon: const Icon(FluentIcons.refresh),
              label: const Text('Refresh'),
              onPressed: () {
                context.read<TagProvider>().loadGovernanceTags(
                  search: _searchQuery.isNotEmpty ? _searchQuery : null,
                );
              },
            ),
          ],
        ),
      ),
      content: Consumer<TagProvider>(
        builder: (context, tagProvider, child) {
          if (tagProvider.isRunningGovernanceAction &&
              tagProvider.governanceRows.isEmpty) {
            return const Center(child: ProgressRing());
          }

          if (tagProvider.governanceError != null &&
              tagProvider.governanceRows.isEmpty) {
            return Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    'Error: ${tagProvider.governanceError}',
                    style: TextStyle(color: Colors.red),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: () => tagProvider.loadGovernanceTags(),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            );
          }

          return Column(
            children: [
              // Summary stats row
              _buildSummaryStats(context, tagProvider),
              // Search box
              _buildSearchBox(context),
              // Bulk action bar (visible only when rows are selected)
              if (tagProvider.selectedGovernanceIds.isNotEmpty)
                TagBulkActionBar(
                  onCleanup: _handleBulkCleanup,
                  onMergeInto: _handleBulkMergeInto,
                ),
              // Merge panel (visible when merge source is active)
              if (tagProvider.activeMergeSource != null)
                TagMergePanel(
                  sourceRow: tagProvider.activeMergeSource!,
                  allRows: tagProvider.governanceRows,
                  onConfirm: _handleMergeConfirm,
                  onCancel: () => tagProvider.clearActiveMergeSource(),
                ),
              // Main governance list
              Expanded(
                child: TagManagementList(
                  onEdit: _handleEdit,
                  onMerge: _handleMerge,
                  onDelete: _handleDelete,
                  onViewAffectedImages: _handleViewAffectedImages,
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildSummaryStats(BuildContext context, TagProvider provider) {
    final rows = provider.governanceRows;
    int totalUsage = 0;
    int totalAI = 0;
    int totalManual = 0;
    int totalPending = 0;

    for (final row in rows) {
      totalUsage += row.usageCount;
      totalAI += row.aiCount;
      totalManual += row.manualCount;
      totalPending += row.pendingCount;
    }

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          _buildStatCard(context, 'Usage', totalUsage),
          const SizedBox(width: 12),
          _buildStatCard(context, 'AI', totalAI),
          const SizedBox(width: 12),
          _buildStatCard(context, 'Manual', totalManual),
          const SizedBox(width: 12),
          _buildStatCard(context, 'Pending', totalPending),
        ],
      ),
    );
  }

  Widget _buildStatCard(BuildContext context, String label, int value) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(8),
          child: Column(
            children: [
              Text(
                '$value',
                style: FluentTheme.of(context).typography.subtitle,
              ),
              Text(label),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSearchBox(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: TextBox(
        placeholder: 'Search tags...',
        onChanged: (value) {
          _searchQuery = value;
          context.read<TagProvider>().loadGovernanceTags(
            search: value.isNotEmpty ? value : null,
          );
        },
      ),
    );
  }

  void _handleEdit(TagGovernanceRow row) {
    showDialog(
      context: context,
      builder: (_) => TagEditDialog(row: row),
    );
  }

  void _handleMerge(TagGovernanceRow row) {
    context.read<TagProvider>().setActiveMergeSource(row);
  }

  Future<void> _handleDelete(TagGovernanceRow row) async {
    final provider = context.read<TagProvider>();
    await provider.loadDeletePreview(row.tagId);

    if (!mounted) return;

    final preview = provider.deletePreview;
    final String blockingReason =
        preview?.blockingReason ??
        (row.canDelete ? '' : 'Tag is in use by images');
    final int affectedCount =
        preview?.affectedImageCount ?? row.affectedImageCount;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('Delete Tag'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Tag: ${row.preferredLabel}'),
            const SizedBox(height: 8),
            Text('$affectedCount affected image(s)'),
            if (blockingReason.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(blockingReason, style: const TextStyle(color: Colors.red)),
            ],
          ],
        ),
        actions: [
          Button(
            child: const Text('Cancel'),
            onPressed: () => Navigator.pop(context, false),
          ),
          if (row.canDelete)
            FilledButton(
              child: const Text('Delete'),
              onPressed: () => Navigator.pop(context, true),
            ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      await provider.deleteTag(row.tagId);
      if (mounted) {
        await provider.loadGovernanceTags(
          search: _searchQuery.isNotEmpty ? _searchQuery : null,
        );
      }
    }
  }

  void _handleViewAffectedImages(TagGovernanceRow row) {
    context.read<ImageListProvider>().setTagFilter([row.tagId]);
    context.read<NavigationProvider>().setSelectedIndex(
      NavigationProvider.galleryIndex,
    );
  }

  Future<void> _handleBulkCleanup() async {
    final provider = context.read<TagProvider>();
    await provider.cleanupSelectedUnusedTags();
    if (mounted) {
      await provider.loadGovernanceTags(
        search: _searchQuery.isNotEmpty ? _searchQuery : null,
      );
    }
  }

  Future<void> _handleBulkMergeInto(int targetTagId) async {
    final provider = context.read<TagProvider>();
    await provider.mergeSelectionInto(targetTagId);
    if (mounted) {
      provider.clearGovernanceSelection();
      await provider.loadGovernanceTags(
        search: _searchQuery.isNotEmpty ? _searchQuery : null,
      );
    }
  }

  Future<void> _handleMergeConfirm(int targetTagId) async {
    final provider = context.read<TagProvider>();
    await provider.mergeSelectionInto(targetTagId);
    if (mounted) {
      provider.clearActiveMergeSource();
      provider.clearGovernanceSelection();
      await provider.loadGovernanceTags(
        search: _searchQuery.isNotEmpty ? _searchQuery : null,
      );
    }
  }
}
