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
      if (provider.tagTree == null) {
        provider.loadTagTree();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: PageHeader(
        title: const Text('标签治理'),
        commandBar: CommandBar(
          mainAxisAlignment: MainAxisAlignment.end,
          primaryItems: [
            CommandBarButton(
              icon: const Icon(FluentIcons.add),
              label: const Text('新建标签'),
              onPressed: () async {
                final provider = context.read<TagProvider>();
                await showDialog(
                  context: context,
                  builder: (_) => const TagEditDialog(),
                );
                if (!mounted) return;
                await provider.loadGovernanceTags();
                await provider.loadTagTree();
              },
            ),
            CommandBarButton(
              icon: const Icon(FluentIcons.refresh),
              label: const Text('刷新'),
              onPressed: () {
                context.read<TagProvider>().loadGovernanceTags();
                context.read<TagProvider>().loadTagTree();
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
                    '错误: ${tagProvider.governanceError}',
                    style: TextStyle(color: Colors.red),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: () => tagProvider.loadGovernanceTags(),
                    child: const Text('重试'),
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
                  searchQuery: _searchQuery,
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
      totalUsage += row
          .usageCount; // Using usageCount to match existing logic, could use treeUsageCount instead
      totalAI += row.aiCount;
      totalManual += row.manualCount;
      totalPending += row.pendingCount;
    }

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          _buildStatCard(
            context,
            '已加载',
            '${rows.length}/${provider.governanceTotal}',
          ),
          const SizedBox(width: 12),
          _buildStatCard(context, '总计使用量', '$totalUsage'),
          const SizedBox(width: 12),
          _buildStatCard(context, 'AI 生成', '$totalAI'),
          const SizedBox(width: 12),
          _buildStatCard(context, '手动', '$totalManual'),
          const SizedBox(width: 12),
          _buildStatCard(context, '待处理', '$totalPending'),
        ],
      ),
    );
  }

  Widget _buildStatCard(BuildContext context, String label, String value) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(8),
          child: Column(
            children: [
              Text(value, style: FluentTheme.of(context).typography.subtitle),
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
        placeholder: '搜索标签...',
        onChanged: (value) {
          setState(() {
            _searchQuery = value;
          });
          // Trigger paginated reload with search
          context.read<TagProvider>().loadGovernanceTags(search: value);
        },
      ),
    );
  }

  void _handleEdit(TagGovernanceRow row) async {
    final provider = context.read<TagProvider>();
    await showDialog(
      context: context,
      builder: (_) => TagEditDialog(row: row),
    );
    if (!mounted) return;
    await provider.loadGovernanceTags();
    await provider.loadTagTree();
  }

  void _handleMerge(TagGovernanceRow row) {
    context.read<TagProvider>().setActiveMergeSource(row);
  }

  Future<void> _handleDelete(TagGovernanceRow row) async {
    final provider = context.read<TagProvider>();
    await provider.loadDeletePreview(row.tagId);

    if (!mounted) return;

    final preview = provider.deletePreview;
    final bool canDelete = preview?.canDelete ?? row.canDelete;
    final String blockingReason =
        preview?.blockingReason ?? (row.canDelete ? '' : '标签正在被图片使用');
    final int affectedCount =
        preview?.affectedImageCount ?? row.affectedImageCount;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('删除标签'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('标签: ${row.preferredLabel}'),
            const SizedBox(height: 8),
            Text('$affectedCount 张受影响的图片'),
            if (blockingReason.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(blockingReason, style: TextStyle(color: Colors.red)),
            ],
          ],
        ),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(context, false),
          ),
          if (canDelete)
            FilledButton(
              child: const Text('删除'),
              onPressed: () => Navigator.pop(context, true),
            ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      await provider.deleteTag(row.tagId);
      if (mounted) {
        await provider.loadGovernanceTags();
        await provider.loadTagTree();
      }
    }
  }

  void _handleViewAffectedImages(TagGovernanceRow row) {
    context.read<TagProvider>().setSelection([row.tagId]);
    context.read<ImageListProvider>().setTagFilter([row.tagId]);
    context.read<NavigationProvider>().setSelectedIndex(
      NavigationProvider.galleryIndex,
    );
  }

  Future<void> _handleBulkCleanup() async {
    final provider = context.read<TagProvider>();
    await provider.cleanupSelectedUnusedTags();
    if (mounted) {
      await provider.loadGovernanceTags();
      await provider.loadTagTree();
    }
  }

  Future<void> _handleBulkMergeInto(int targetTagId) async {
    final provider = context.read<TagProvider>();
    await provider.mergeSelectionInto(targetTagId);
    if (mounted) {
      provider.clearGovernanceSelection();
      await provider.loadGovernanceTags();
      await provider.loadTagTree();
    }
  }

  Future<void> _handleMergeConfirm(int targetTagId) async {
    final provider = context.read<TagProvider>();
    await provider.mergeSelectionInto(targetTagId);
    if (mounted) {
      provider.clearActiveMergeSource();
      provider.clearGovernanceSelection();
      await provider.loadGovernanceTags();
      await provider.loadTagTree();
    }
  }
}
