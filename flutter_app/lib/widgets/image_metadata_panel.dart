import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../providers/tag_provider.dart';
import '../widgets/tag_chip.dart';
import '../widgets/add_tag_dialog.dart';
import '../widgets/edit_tag_dialog.dart';

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
  }

  @override
  void didUpdateWidget(covariant ImageMetadataPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.imageId != widget.imageId) {
      _resetAIPanel();
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
    await _loadImageTags();
    final tagProvider = context.read<TagProvider>();
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
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('已合并到 ${targetTag.preferredLabel}')),
        );
      } catch (e) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('合并失败: $e')));
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
    final colorScheme = Theme.of(context).colorScheme;
    final panelSurface = _opaqueColor(colorScheme.surfaceContainerHighest);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          widget.metadataSection,
          _buildAITagSection(context, panelSurface),
          _buildTagsSection(context, panelSurface),
        ],
      ),
    );
  }

  Widget _buildAITagSection(BuildContext context, Color panelSurface) {
    final foreground = _foregroundForSurface(panelSurface);
    final mutedForeground = _mutedForegroundForSurface(panelSurface);
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: panelSurface,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.auto_awesome, color: mutedForeground),
              const SizedBox(width: 8),
              Text(
                'AI 标签',
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(color: foreground),
              ),
              const Spacer(),
              if (!_isAITriggered)
                FilledButton.icon(
                  onPressed: _triggerAITags,
                  icon: const Icon(Icons.play_arrow, size: 18),
                  label: const Text('生成'),
                ),
              if (_aiStatus != null) ...[
                const SizedBox(width: 8),
                Chip(label: Text(_aiStatus!)),
              ],
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Text(
                '自定义提示词',
                style: TextStyle(fontSize: 13, color: mutedForeground),
              ),
              const SizedBox(width: 8),
              Switch(
                value: _useCustomPrompt,
                onChanged: (value) {
                  setState(() => _useCustomPrompt = value);
                },
              ),
              if (_isLoadingPrompt)
                const SizedBox(
                  width: 8,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
            ],
          ),
          if (_useCustomPrompt) ...[
            const SizedBox(height: 8),
            TextField(
              controller: _promptController,
              maxLines: 6,
              decoration: InputDecoration(
                hintText: '输入自定义提示词...',
                border: const OutlineInputBorder(),
                filled: true,
                fillColor: Colors.white,
                suffixIcon: IconButton(
                  icon: const Icon(Icons.refresh, size: 20),
                  tooltip: '恢复默认提示词',
                  onPressed: () {
                    _promptController.text = _defaultPrompt;
                  },
                ),
              ),
            ),
            const SizedBox(height: 4),
            Text(
              '提示：可编辑提示词以自定义 AI 生成的标签类型和风格',
              style: TextStyle(fontSize: 11, color: mutedForeground),
            ),
          ] else
            Text(
              '点击"生成"触发 AI 分析，标签将自动添加到待确认列表中',
              style: TextStyle(fontSize: 12, color: mutedForeground),
            ),
        ],
      ),
    );
  }

  Widget _buildTagsSection(BuildContext context, Color panelSurface) {
    final foreground = _foregroundForSurface(panelSurface);
    final mutedForeground = _mutedForegroundForSurface(panelSurface);
    return Consumer<TagProvider>(
      builder: (context, provider, child) {
        final confirmed = provider.imageTags['confirmed'] ?? [];
        final pending = provider.imageTags['pending'] ?? [];
        final rejected = provider.imageTags['rejected'] ?? [];

        return Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    '标签',
                    style: Theme.of(
                      context,
                    ).textTheme.titleLarge?.copyWith(color: foreground),
                  ),
                  IconButton(
                    icon: const Icon(Icons.add),
                    onPressed: _addTag,
                    tooltip: '添加标签',
                    color: foreground,
                  ),
                ],
              ),
              const SizedBox(height: 16),
              if (confirmed.isNotEmpty) ...[
                Text(
                  '已确认',
                  style: Theme.of(
                    context,
                  ).textTheme.titleMedium?.copyWith(color: foreground),
                ),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: confirmed
                      .map(
                        (tag) => TagChip(
                          tag: tag,
                          style: TagChipStyle.confirmed,
                          onDelete: () => _removeTag(tag.id),
                        ),
                      )
                      .toList(),
                ),
                const SizedBox(height: 12),
              ],
              if (pending.isNotEmpty) ...[
                Text(
                  '待确认',
                  style: Theme.of(
                    context,
                  ).textTheme.titleMedium?.copyWith(color: foreground),
                ),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: pending
                      .map((tag) => _buildPendingTagChip(tag))
                      .toList(),
                ),
                const SizedBox(height: 12),
              ],
              if (rejected.isNotEmpty) ...[
                Text(
                  '已拒绝',
                  style: Theme.of(
                    context,
                  ).textTheme.titleMedium?.copyWith(color: mutedForeground),
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: rejected
                      .map(
                        (tag) =>
                            TagChip(tag: tag, style: TagChipStyle.rejected),
                      )
                      .toList(),
                ),
              ],
              if (confirmed.isEmpty && pending.isEmpty && rejected.isEmpty)
                Text('暂无标签', style: TextStyle(color: foreground)),
            ],
          ),
        );
      },
    );
  }

  Widget _buildPendingTagChip(Tag tag) {
    return Card(
      margin: const EdgeInsets.only(right: 8, bottom: 4),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              tag.preferredLabel,
              style: TextStyle(color: Theme.of(context).colorScheme.onSurface),
            ),
            const SizedBox(width: 8),
            InkWell(
              onTap: () => _confirmTag(tag.id),
              child: const Icon(Icons.check, size: 18, color: Colors.green),
            ),
            const SizedBox(width: 4),
            InkWell(
              onTap: () => _rejectTag(tag.id),
              child: const Icon(Icons.close, size: 18, color: Colors.red),
            ),
            const SizedBox(width: 4),
            InkWell(
              onTap: () => _showMergeDialog(tag),
              child: const Icon(Icons.merge_type, size: 18, color: Colors.blue),
            ),
            const SizedBox(width: 4),
            InkWell(
              onTap: () => _showEditTagDialog(tag),
              child: const Icon(Icons.edit, size: 18, color: Colors.orange),
            ),
          ],
        ),
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
              .where((t) => t.id != widget.sourceTag.id)
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
