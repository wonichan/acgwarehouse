import 'dart:async';
import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import 'package:provider/provider.dart';
import '../models/image.dart';
import '../models/tag.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';
import '../widgets/tag_chip.dart';
import '../widgets/add_tag_dialog.dart';
import '../widgets/edit_tag_dialog.dart';
import '../widgets/image_lightbox.dart';

class ImageDetailScreen extends StatefulWidget {
  final ImageModel image;

  const ImageDetailScreen({super.key, required this.image});

  @override
  State<ImageDetailScreen> createState() => _ImageDetailScreenState();
}

class _ImageDetailScreenState extends State<ImageDetailScreen> {
  late TagProvider _tagProvider;
  Timer? _pollTimer;
  String? _aiStatus;
  bool _isAITriggered = false;

  // 自定义提示词相关
  final TextEditingController _promptController = TextEditingController();
  String _defaultPrompt = '';
  bool _isLoadingPrompt = false;
  bool _useCustomPrompt = false;

  @override
  void initState() {
    super.initState();
    _tagProvider = TagProvider(TagService());
    _loadImageTags();
    _loadDefaultPrompt();
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _promptController.dispose();
    _tagProvider.dispose();
    super.dispose();
  }

  Future<void> _loadDefaultPrompt() async {
    setState(() => _isLoadingPrompt = true);
    try {
      _defaultPrompt = await _tagProvider.getDefaultAIPrompt();
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
    await _tagProvider.loadImageTags(widget.image.id);
  }

  Future<void> _triggerAITags() async {
    try {
      final prompt = _useCustomPrompt && _promptController.text.isNotEmpty
          ? _promptController.text
          : null;
      await _tagProvider.triggerAITags(widget.image.id, prompt: prompt);
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
    _pollTimer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      try {
        final status = await _tagProvider.getAITagStatus(widget.image.id);
        final statusStr = status['status'] as String? ?? 'unknown';

        if (mounted) {
          setState(() {
            _aiStatus = _translateStatus(statusStr);
          });
        }

        if (statusStr == 'completed' || statusStr == 'failed') {
          timer.cancel();
          if (statusStr == 'completed') {
            // 增加延迟时间到1500ms，确保后端标签数据完全写入数据库
            await Future.delayed(const Duration(milliseconds: 1500));
            // 尝试加载标签，如果pending列表为空则重试一次
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

  /// 加载图片标签，如果pending为空则重试一次
  Future<void> _loadImageTagsWithRetry() async {
    await _loadImageTags();

    // 检查pending标签是否为空，如果是则等待后重试一次
    final pending = _tagProvider.imageTags['pending'] ?? [];
    if (pending.isEmpty) {
      debugPrint('No pending tags found after initial load, retrying...');
      await Future.delayed(const Duration(milliseconds: 1000));
      await _loadImageTags();
    }
  }

  Future<void> _confirmTag(int tagId) async {
    try {
      await _tagProvider.confirmImageTag(widget.image.id, tagId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('确认标签失败: $e')));
      }
    }
  }

  Future<void> _rejectTag(int tagId) async {
    try {
      await _tagProvider.rejectImageTag(widget.image.id, tagId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('拒绝标签失败: $e')));
      }
    }
  }

  Future<void> _showMergeDialog(Tag pendingTag) async {
    final targetTag = await showDialog<Tag>(
      context: context,
      builder: (context) =>
          _MergeTagDialog(tagProvider: _tagProvider, sourceTag: pendingTag),
    );

    if (targetTag != null && mounted) {
      try {
        await _tagProvider.removeImageTag(widget.image.id, pendingTag.id);
        await _tagProvider.addImageTag(widget.image.id, tagId: targetTag.id);
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
    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (context) => EditTagDialog(
        imageId: widget.image.id,
        currentTag: tag,
        tagService: _tagProvider.tagService,
      ),
    );

    if (result != null && mounted) {
      try {
        if (result['tagId'] != null) {
          // 选择现有标签
          await _tagProvider.mergeImageTag(
            widget.image.id,
            tag.id,
            targetTagId: result['tagId'] as int,
          );
        } else if (result['tagLabel'] != null) {
          // 创建新标签
          await _tagProvider.mergeImageTag(
            widget.image.id,
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
    final result = await showDialog<dynamic>(
      context: context,
      builder: (context) => AddTagDialog(imageId: widget.image.id),
    );

    // Handle error result from dialog
    if (result is Map && result['success'] == false && mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('操作失败: ${result['error']}')));
    }

    // Reload tags after dialog closes
    await _loadImageTags();
  }

  Future<void> _removeTag(int tagId) async {
    try {
      await _tagProvider.removeImageTag(widget.image.id, tagId);
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
    final isDesktopLayout = MediaQuery.of(context).size.width >= 900;
    final pageSurface = _opaqueColor(
      Color.alphaBlend(
        colorScheme.outlineVariant.withOpacity(0.08),
        colorScheme.surface,
      ),
    );
    final panelSurface = _opaqueColor(colorScheme.surfaceContainerHighest);

    return ChangeNotifierProvider.value(
      value: _tagProvider,
      child: Scaffold(
        backgroundColor: pageSurface,
        appBar: AppBar(
          backgroundColor: pageSurface,
          surfaceTintColor: Colors.transparent,
          title: const Text('图片详情'),
          actions: [
            IconButton(
              icon: const Icon(Icons.add),
              onPressed: _addTag,
              tooltip: '添加标签',
            ),
          ],
        ),
        body: isDesktopLayout
            ? _buildDesktopLayout(context, pageSurface, panelSurface)
            : _buildCompactLayout(context, pageSurface, panelSurface),
      ),
    );
  }

  Widget _buildDesktopLayout(
    BuildContext context,
    Color pageSurface,
    Color panelSurface,
  ) {
    return Container(
      color: pageSurface,
      padding: const EdgeInsets.all(24),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 320),
            child: Container(
              decoration: BoxDecoration(
                color: panelSurface,
                borderRadius: BorderRadius.circular(20),
              ),
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildMetadataSection(context, panelSurface),
                    _buildAITagSection(context, panelSurface),
                    _buildTagsSection(context, panelSurface),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(width: 24),
          Expanded(
            child: Container(
              decoration: BoxDecoration(
                color: panelSurface,
                borderRadius: BorderRadius.circular(20),
              ),
              padding: const EdgeInsets.all(16),
              child: Center(child: _buildImageViewer(context, panelSurface)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCompactLayout(
    BuildContext context,
    Color pageSurface,
    Color panelSurface,
  ) {
    return Container(
      color: pageSurface,
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              decoration: BoxDecoration(
                color: panelSurface,
                borderRadius: BorderRadius.circular(20),
              ),
              padding: const EdgeInsets.all(16),
              child: _buildImageViewer(context, panelSurface),
            ),
            const SizedBox(height: 16),
            Container(
              decoration: BoxDecoration(
                color: panelSurface,
                borderRadius: BorderRadius.circular(20),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildMetadataSection(context, panelSurface),
                  _buildAITagSection(context, panelSurface),
                  _buildTagsSection(context, panelSurface),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildImageViewer([BuildContext? buildContext, Color? panelSurface]) {
    final resolvedContext = buildContext ?? context;
    final resolvedPanelSurface =
        panelSurface ??
        _opaqueColor(
          Theme.of(resolvedContext).colorScheme.surfaceContainerHighest,
        );
    // 优先使用缩略图，如果缩略图为空则回退到原始图片路径
    final largeUrl = widget.image.thumbnailLargeUrl;
    final originalPath = widget.image.path;
    final displayUrl = (largeUrl != null && largeUrl.isNotEmpty)
        ? largeUrl
        : originalPath;

    if (displayUrl.isEmpty) {
      return Container(
        height: 300,
        color: resolvedPanelSurface,
        child: const Center(
          child: Icon(Icons.image, size: 64, color: Colors.grey),
        ),
      );
    }

    return GestureDetector(
      onTap: () {
        ImageLightbox.show(
          context,
          imageUrl: displayUrl,
          heroTag: 'image-${widget.image.id}',
        );
      },
      child: Container(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(resolvedContext).size.height * 0.75,
        ),
        decoration: BoxDecoration(
          color: resolvedPanelSurface,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Stack(
          alignment: Alignment.center,
          children: [
            Hero(
              tag: 'image-${widget.image.id}',
              child: ExtendedImage.network(
                displayUrl,
                fit: BoxFit.contain,
                // Enable zoom functionality with gesture mode
                mode: ExtendedImageMode.gesture,
                initGestureConfigHandler: (state) {
                  return GestureConfig(
                    minScale: 1.0,
                    maxScale: 3.0,
                    animationMaxScale: 3.5,
                    initialScale: 1.0,
                    inPageView: false,
                    initialAlignment: InitialAlignment.center,
                    cacheGesture: false,
                  );
                },
                // Double-tap zoom support (toggle between 1x and 2x)
                onDoubleTap: (state) {
                  final pointerDownPosition = state.pointerDownPosition;
                  final begin = state.gestureDetails?.totalScale ?? 1.0;
                  final end = begin == 1.0 ? 2.0 : 1.0; // Toggle zoom

                  state.handleDoubleTap(
                    scale: end,
                    doubleTapPosition: pointerDownPosition,
                  );
                },
                loadStateChanged: (state) {
                  if (state.extendedImageLoadState == LoadState.loading) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  if (state.extendedImageLoadState == LoadState.failed) {
                    return const Center(
                      child: Icon(Icons.error, color: Colors.red),
                    );
                  }
                  return null;
                },
              ),
            ),
            // Tap hint overlay
            Positioned(
              bottom: 8,
              right: 8,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Colors.black54,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.fullscreen, color: Colors.white, size: 14),
                    SizedBox(width: 4),
                    Text(
                      '点击全屏',
                      style: TextStyle(color: Colors.white, fontSize: 11),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataSection(BuildContext context, Color panelSurface) {
    final foreground = _foregroundForSurface(panelSurface);
    final mutedForeground = _mutedForegroundForSurface(panelSurface);
    return Padding(
      padding: const EdgeInsets.all(12),
      child: SelectionArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '元数据',
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(color: foreground),
            ),
            const SizedBox(height: 6),
            _buildMetadataRow(
              context,
              '文件名',
              widget.image.filename,
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              context,
              '尺寸',
              widget.image.displaySize,
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              context,
              '格式',
              widget.image.format.toUpperCase(),
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              context,
              '大小',
              widget.image.displayFileSize,
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              context,
              '路径',
              widget.image.path,
              foreground,
              mutedForeground,
            ),
            _buildMetadataRow(
              context,
              '导入时间',
              widget.image.createdAt.toString(),
              foreground,
              mutedForeground,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataRow(
    BuildContext context,
    String label,
    String value,
    Color foreground,
    Color mutedForeground,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 70,
            child: Text(
              label,
              style: TextStyle(color: mutedForeground, fontSize: 13),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: TextStyle(
                color: foreground,
                fontWeight: FontWeight.w500,
                fontSize: 13,
              ),
            ),
          ),
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
          // 提示词开关
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
          // 提示词输入框
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
              '点击"生成"触发 AI 分析，标签将自动添加到待确认列表。',
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
              // Confirmed tags
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

              // Pending tags
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

              // Rejected tags
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

/// Dialog for selecting a target tag to merge into
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
      // Use TagProvider's search which updates filteredTags
      await widget.tagProvider.searchTags(query);
      // Get results from filteredTags, filtering out the source tag
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
            Text('将 "${widget.sourceTag.preferredLabel}" 合并到:'),
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
