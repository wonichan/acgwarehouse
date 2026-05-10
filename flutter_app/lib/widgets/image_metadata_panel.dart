import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../providers/tag_provider.dart';
import '../widgets/tag_chip.dart';
import '../widgets/add_tag_dialog.dart';
import '../widgets/edit_tag_dialog.dart';
import '../widgets/image_metadata_pane_theme.dart';

class ImageMetadataPanel extends StatefulWidget {
  final int imageId;
  final Widget metadataSection;

  const ImageMetadataPanel({
    super.key,
    required this.imageId,
    required this.metadataSection,
  });

  @override
  State<ImageMetadataPanel> createState() => _ImageMetadataPanelState();
}

class _ImageMetadataPanelState extends State<ImageMetadataPanel> {
  Timer? _pollTimer;
  String? _aiStatus;
  bool _isAITriggered = false;

  final TextEditingController _promptController = TextEditingController();
  String _defaultPrompt = '';
  bool _isLoadingPrompt = false;
  bool _useCustomPrompt = false;

  @override
  void initState() {
    super.initState();
    _loadDefaultPrompt();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _loadImageTags();
      }
    });
  }

  @override
  void didUpdateWidget(covariant ImageMetadataPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.imageId != widget.imageId) {
      _resetAIPanel();
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _loadImageTags();
        }
      });
    }
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _promptController.dispose();
    super.dispose();
  }

  void _resetAIPanel() {
    _pollTimer?.cancel();
    _pollTimer = null;
    setState(() {
      _aiStatus = null;
      _isAITriggered = false;
    });
  }

  Future<void> _loadDefaultPrompt() async {
    setState(() => _isLoadingPrompt = true);
    try {
      final tagProvider = context.read<TagProvider>();
      _defaultPrompt = await tagProvider.getDefaultAIPrompt();
      _promptController.text = _defaultPrompt;
    } catch (e) {
      debugPrint('Error loading default prompt: $e');
    } finally {
      if (mounted) {
        setState(() => _isLoadingPrompt = false);
      }
    }
  }

  Future<void> _loadImageTags() async {
    final tagProvider = context.read<TagProvider>();
    await tagProvider.loadImageTags(widget.imageId);
  }

  Future<void> _triggerAITags() async {
    final tagProvider = context.read<TagProvider>();
    try {
      _pollTimer?.cancel();
      final prompt = _useCustomPrompt && _promptController.text.isNotEmpty
          ? _promptController.text
          : null;
      await tagProvider.triggerAITags(widget.imageId, prompt: prompt);
      setState(() {
        _isAITriggered = true;
        _aiStatus = '队列中';
      });
      _startPolling();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('触发 AI 标签失败: $e')));
      }
    }
  }

  void _startPolling() {
    final tagProvider = context.read<TagProvider>();
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      try {
        final status = await tagProvider.getAITagStatus(widget.imageId);
        final statusStr = status['status'] as String? ?? 'unknown';

        if (mounted) {
          setState(() {
            _aiStatus = _translateStatus(statusStr);
          });
        }

        if (statusStr == 'completed' || statusStr == 'failed') {
          timer.cancel();
          if (mounted) {
            setState(() {
              _isAITriggered = false;
            });
          }
          if (statusStr == 'completed') {
            await Future.delayed(const Duration(milliseconds: 1500));
            await _loadImageTagsWithRetry();
          }
        }
      } catch (e) {
        debugPrint('Error polling AI status: $e');
      }
    });
  }

  String _translateStatus(String status) {
    switch (status) {
      case 'queued':
        return '队列中';
      case 'running':
        return '分析中...';
      case 'completed':
        return '已完成';
      case 'failed':
        return '失败';
      default:
        return status;
    }
  }

  Future<void> _loadImageTagsWithRetry() async {
    final tagProvider = context.read<TagProvider>();
    await _loadImageTags();
    final pending = tagProvider.imageTags['pending'] ?? [];
    if (pending.isEmpty) {
      await Future.delayed(const Duration(milliseconds: 1000));
      await _loadImageTags();
    }
  }

  Future<void> _confirmTag(int tagId) async {
    final tagProvider = context.read<TagProvider>();
    try {
      await tagProvider.confirmImageTag(widget.imageId, tagId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('确认标签失败: $e')));
      }
    }
  }

  Future<void> _rejectTag(int tagId) async {
    final tagProvider = context.read<TagProvider>();
    try {
      await tagProvider.rejectImageTag(widget.imageId, tagId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('拒绝标签失败: $e')));
      }
    }
  }

  Future<void> _showMergeDialog(Tag pendingTag) async {
    final tagProvider = context.read<TagProvider>();
    final messenger = ScaffoldMessenger.of(context);
    final targetTag = await showDialog<Tag>(
      context: context,
      builder: (context) =>
          _MergeTagDialog(tagProvider: tagProvider, sourceTag: pendingTag),
    );

    if (targetTag != null && mounted) {
      try {
        await tagProvider.mergeImageTag(
          widget.imageId,
          pendingTag.id,
          targetTagId: targetTag.id,
        );
        await _loadImageTags();
        messenger.showSnackBar(
          SnackBar(content: Text('已合并到 ${targetTag.preferredLabel}')),
        );
      } catch (e) {
        messenger.showSnackBar(SnackBar(content: Text('合并失败: $e')));
      }
    }
  }

  Future<void> _showEditTagDialog(Tag tag) async {
    final tagProvider = context.read<TagProvider>();
    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (context) => EditTagDialog(
        imageId: widget.imageId,
        currentTag: tag,
        tagService: tagProvider.tagService,
      ),
    );

    if (result != null && mounted) {
      try {
        if (result['tagId'] != null) {
          await tagProvider.mergeImageTag(
            widget.imageId,
            tag.id,
            targetTagId: result['tagId'] as int,
          );
        } else if (result['tagLabel'] != null) {
          await tagProvider.mergeImageTag(
            widget.imageId,
            tag.id,
            targetLabel: result['tagLabel'] as String,
            targetLevel: result['tagLevel'] as String?,
            targetParentId: result['tagParentId'] as int?,
          );
        }
        await _loadImageTags();
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('标签已更新为: ${result['label']}')));
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('更新标签失败: $e')));
        }
      }
    }
  }

  Future<void> _addTag() async {
    final tagProvider = context.read<TagProvider>();
    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => AddTagDialog(
        imageId: widget.imageId,
        tagService: tagProvider.tagService,
      ),
    );

    if (result is Map && result['success'] == false && mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('操作失败: ${result['error']}')));
    }

    await _loadImageTags();
  }

  Future<void> _removeTag(int tagId) async {
    final tagProvider = context.read<TagProvider>();
    try {
      await tagProvider.removeImageTag(widget.imageId, tagId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('移除标签失败: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = ImageMetadataPaneTheme.of(context);

    return Container(
      key: const Key('metadata-pane-root'),
      color: theme.panelSurface,
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              key: const Key('metadata-section-basic'),
              child: widget.metadataSection,
            ),
            _buildAITagSection(context, theme),
            _buildTagsSection(context, theme),
          ],
        ),
      ),
    );
  }

  Widget _buildAITagSection(
    BuildContext context,
    ImageMetadataPaneTheme theme,
  ) {
    return Container(
      key: const Key('metadata-section-ai'),
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 4),
      decoration: theme.sectionDecoration,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                Icon(Icons.auto_awesome, color: theme.iconColor, size: 20),
                const SizedBox(width: 8),
                Text(
                  'AI 标签',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color: theme.textForeground,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: Theme.of(context).colorScheme.primary,
                    foregroundColor: Theme.of(context).colorScheme.onPrimary,
                    disabledBackgroundColor: Theme.of(
                      context,
                    ).colorScheme.primary.withValues(alpha: 0.6),
                    disabledForegroundColor: Theme.of(
                      context,
                    ).colorScheme.onPrimary,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(6),
                    ),
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 8,
                    ),
                    minimumSize: const Size(0, 32),
                  ),
                  onPressed: _isAITriggered ? null : _triggerAITags,
                  icon: _isAITriggered
                      ? SizedBox.square(
                          dimension: 14,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            valueColor: AlwaysStoppedAnimation<Color>(
                              Theme.of(context).colorScheme.onPrimary,
                            ),
                          ),
                        )
                      : const Icon(Icons.play_arrow, size: 16),
                  label: const Text(
                    '生成',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
                  ),
                ),
                if (_aiStatus != null) ...[
                  const SizedBox(width: 8),
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 4,
                    ),
                    decoration: BoxDecoration(
                      color: theme.statusBackground,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      _aiStatus!,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.statusForeground,
                      ),
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              decoration: BoxDecoration(
                color: theme.inputFillColor,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: theme.borderColor),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '自定义提示词',
                          style: TextStyle(
                            fontSize: 13,
                            color: theme.textForeground,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          _useCustomPrompt
                              ? '使用自定义提示词控制标签生成风格'
                              : '默认使用系统提示词，可按需展开编辑',
                          style: TextStyle(
                            fontSize: 11,
                            color: theme.textMuted,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 12),
                  Switch(
                    value: _useCustomPrompt,
                    onChanged: (value) {
                      setState(() => _useCustomPrompt = value);
                    },
                    activeThumbColor: Theme.of(context).colorScheme.primary,
                    activeTrackColor: Theme.of(
                      context,
                    ).colorScheme.primary.withValues(alpha: 0.2),
                  ),
                  if (_isLoadingPrompt)
                    const SizedBox(
                      width: 8,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                ],
              ),
            ),
            if (_useCustomPrompt) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _promptController,
                maxLines: 4,
                style: TextStyle(fontSize: 13, color: theme.textForeground),
                decoration: InputDecoration(
                  hintText: '输入自定义提示词...',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(6),
                    borderSide: BorderSide(color: theme.borderColor),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(6),
                    borderSide: BorderSide(color: theme.borderColor),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(6),
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                  filled: true,
                  fillColor: theme.inputFillColor,
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 10,
                  ),
                  suffixIcon: IconButton(
                    icon: const Icon(Icons.refresh, size: 18),
                    tooltip: '恢复默认提示词',
                    color: theme.textMuted,
                    onPressed: () {
                      _promptController.text = _defaultPrompt;
                    },
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Text(
                '提示：编辑后将用于下一次 AI 标签生成。',
                style: TextStyle(fontSize: 12, color: theme.textMuted),
              ),
            ] else
              Padding(
                padding: const EdgeInsets.only(top: 8.0),
                child: Text(
                  '点击“生成”触发 AI 分析，结果会进入待确认分组。',
                  style: TextStyle(fontSize: 12, color: theme.textMuted),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildTagsSection(BuildContext context, ImageMetadataPaneTheme theme) {
    return Consumer<TagProvider>(
      builder: (context, provider, child) {
        final confirmed = provider.imageTags['confirmed'] ?? [];
        final pending = provider.imageTags['pending'] ?? [];
        final rejected = provider.imageTags['rejected'] ?? [];
        final totalCount = confirmed.length + pending.length + rejected.length;
        final isLoadingTags = provider.isLoadingImageTags;

        return Container(
          key: const Key('metadata-section-tags'),
          margin: const EdgeInsets.fromLTRB(12, 8, 12, 12),
          decoration: theme.sectionDecoration,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      '标签${totalCount > 0 ? ' ($totalCount)' : ''}',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: theme.textForeground,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.add, size: 20),
                      onPressed: _addTag,
                      tooltip: '添加标签',
                      color: theme.iconColor,
                      constraints: const BoxConstraints(),
                      padding: EdgeInsets.zero,
                    ),
                  ],
                ),
                const SizedBox(height: 14),
                if (isLoadingTags && totalCount == 0)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 2),
                    child: Row(
                      key: const Key('metadata-tags-loading'),
                      children: [
                        SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: theme.iconColor,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          '加载标签中...',
                          style: TextStyle(
                            color: theme.textMuted,
                            fontSize: 13,
                          ),
                        ),
                      ],
                    ),
                  ),
                if (pending.isNotEmpty) ...[
                  _buildTagGroup(
                    context,
                    theme,
                    label: '待确认',
                    count: pending.length,
                    tone: _TagGroupTone.pending,
                    children: pending
                        .map(
                          (tag) => TagChip(
                            tag: tag,
                            style: TagChipStyle.pending,
                            onConfirm: () => _confirmTag(tag.id),
                            onReject: () => _rejectTag(tag.id),
                            onMerge: () => _showMergeDialog(tag),
                            onEdit: () => _showEditTagDialog(tag),
                          ),
                        )
                        .toList(),
                  ),
                  const SizedBox(height: 12),
                ],
                if (confirmed.isNotEmpty) ...[
                  _buildTagGroup(
                    context,
                    theme,
                    label: '已确认',
                    count: confirmed.length,
                    tone: _TagGroupTone.confirmed,
                    children: confirmed
                        .map(
                          (tag) => TagChip(
                            tag: tag,
                            style: TagChipStyle.confirmed,
                            onDelete: () => _removeTag(tag.id),
                            onEdit: () => _showEditTagDialog(tag),
                            onMerge: () => _showMergeDialog(tag),
                          ),
                        )
                        .toList(),
                  ),
                  const SizedBox(height: 12),
                ],
                if (rejected.isNotEmpty) ...[
                  _buildTagGroup(
                    context,
                    theme,
                    label: '已拒绝',
                    count: rejected.length,
                    tone: _TagGroupTone.rejected,
                    children: rejected
                        .map(
                          (tag) => TagChip(
                            tag: tag,
                            style: TagChipStyle.rejected,
                            onDelete: () => _removeTag(tag.id),
                          ),
                        )
                        .toList(),
                  ),
                ],
                if (!isLoadingTags &&
                    confirmed.isEmpty &&
                    pending.isEmpty &&
                    rejected.isEmpty)
                  Text(
                    '暂无标签',
                    style: TextStyle(color: theme.textMuted, fontSize: 13),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildTagGroup(
    BuildContext context,
    ImageMetadataPaneTheme theme, {
    required String label,
    required int count,
    required _TagGroupTone tone,
    required List<Widget> children,
  }) {
    final isRejected = tone == _TagGroupTone.rejected;
    final backgroundColor = switch (tone) {
      _TagGroupTone.pending => theme.pendingTagBackground,
      _TagGroupTone.confirmed => theme.inputFillColor.withValues(alpha: 0.55),
      _TagGroupTone.rejected => theme.inputFillColor.withValues(alpha: 0.35),
    };
    final badgeColor = switch (tone) {
      _TagGroupTone.pending => theme.pendingBadgeBackground,
      _TagGroupTone.confirmed => theme.confirmedBadgeBackground,
      _TagGroupTone.rejected => theme.statusBackground,
    };
    final badgeTextColor = switch (tone) {
      _TagGroupTone.pending => theme.pendingBadgeForeground,
      _TagGroupTone.confirmed => theme.confirmedBadgeForeground,
      _TagGroupTone.rejected => theme.textMuted,
    };

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: backgroundColor,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: theme.borderColor),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text(
                    label,
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: isRejected
                          ? theme.textMuted
                          : theme.textForeground,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 6,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: badgeColor,
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      '$count',
                      style: TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                        color: badgeTextColor,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              Wrap(spacing: 10, runSpacing: 8, children: children),
            ],
          ),
        ),
      ],
    );
  }
}

enum _TagGroupTone { pending, confirmed, rejected }

class _MergeTagDialog extends StatefulWidget {
  final TagProvider tagProvider;
  final Tag sourceTag;

  const _MergeTagDialog({required this.tagProvider, required this.sourceTag});

  @override
  State<_MergeTagDialog> createState() => _MergeTagDialogState();
}

class _MergeTagDialogState extends State<_MergeTagDialog> {
  final TextEditingController _searchController = TextEditingController();
  List<Tag> _searchResults = [];
  bool _isSearching = false;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_onSearch);
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _onSearch() async {
    final query = _searchController.text;
    if (query.isEmpty) {
      setState(() => _searchResults = []);
      return;
    }

    setState(() => _isSearching = true);
    try {
      await widget.tagProvider.searchTags(query);
      if (mounted) {
        setState(() {
          _searchResults = widget.tagProvider.filteredTags
              .where(
                (t) =>
                    t.id != widget.sourceTag.id &&
                    (widget.sourceTag.level == null ||
                        t.level == widget.sourceTag.level),
              )
              .toList();
        });
      }
    } catch (e) {
      debugPrint('Search error: $e');
    } finally {
      if (mounted) {
        setState(() => _isSearching = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('合并标签'),
      content: SizedBox(
        width: 300,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('将"${widget.sourceTag.preferredLabel}" 合并到:'),
            const SizedBox(height: 16),
            TextField(
              controller: _searchController,
              decoration: const InputDecoration(
                hintText: '搜索标签...',
                prefixIcon: Icon(Icons.search),
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            if (_isSearching)
              const Center(child: CircularProgressIndicator())
            else if (_searchResults.isNotEmpty)
              ConstrainedBox(
                constraints: const BoxConstraints(maxHeight: 200),
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: _searchResults.length,
                  itemBuilder: (context, index) {
                    final tag = _searchResults[index];
                    return ListTile(
                      title: Text(tag.preferredLabel),
                      subtitle: Text('${tag.usageCount} 张图片'),
                      onTap: () => Navigator.pop(context, tag),
                    );
                  },
                ),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('取消'),
        ),
      ],
    );
  }
}
