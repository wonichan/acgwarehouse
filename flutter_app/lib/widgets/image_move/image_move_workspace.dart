import 'dart:ui' show FontFeature;

import 'package:file_picker/file_picker.dart' as file_picker;
import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../models/image_move.dart';
import '../../models/tag.dart';
import '../../providers/config_provider.dart';
import '../../providers/image_provider.dart';
import '../../providers/tag_provider.dart';
import '../../services/image_move_service.dart';

typedef DirectoryPicker = Future<String?> Function();

class ImageMoveWorkspace extends StatefulWidget {
  final ImageMoveService? imageMoveService;
  final DirectoryPicker? directoryPicker;

  const ImageMoveWorkspace({
    super.key,
    this.imageMoveService,
    this.directoryPicker,
  });

  @override
  State<ImageMoveWorkspace> createState() => _ImageMoveWorkspaceState();
}

class _ImageMoveWorkspaceState extends State<ImageMoveWorkspace> {
  late ImageMoveService _service;
  late final bool _ownsService;
  late final DirectoryPicker _directoryPicker;
  final TextEditingController _tagSearchController = TextEditingController();

  final List<String> _sourceDirs = <String>[];
  Tag? _selectedTag;
  String? _targetDir;
  ImageMovePreview? _preview;
  ImageMoveResult? _result;
  List<ImageMoveBatch> _history = const <ImageMoveBatch>[];
  ImageMoveBatch? _activeJob;
  bool _isPreviewLoading = false;
  bool _isExecuting = false;
  bool _isHistoryLoading = false;
  bool _allowTargetInsideSource = false;
  String? _errorMessage;
  String? _successMessage;
  String _tagQuery = '';
  String _conflict = 'skip';
  int _selectedTabIndex = 0;

  bool get _isBusy => _isPreviewLoading || _isExecuting;

  bool get _canPreview =>
      !_isBusy &&
      _sourceDirs.isNotEmpty &&
      _selectedTag != null &&
      (_targetDir?.trim().isNotEmpty ?? false);

  bool get _canExecute =>
      !_isBusy && _preview != null && (_preview?.movable ?? 0) > 0;

  @override
  void initState() {
    super.initState();
    _ownsService = widget.imageMoveService == null;
    _directoryPicker =
        widget.directoryPicker ??
        () => file_picker.FilePicker.getDirectoryPath();

    _service =
        widget.imageMoveService ??
        ImageMoveService(baseUrl: context.read<ConfigProvider>().baseUrl);

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final tagProvider = context.read<TagProvider>();
      if (tagProvider.allTags.isEmpty && !tagProvider.isLoading) {
        tagProvider.loadTags();
      }
    });
  }

  @override
  void dispose() {
    _tagSearchController.dispose();
    if (_ownsService) {
      _service.dispose();
    }
    super.dispose();
  }

  Future<void> _addSourceDir() async {
    final dir = await _directoryPicker();
    if (!mounted || dir == null || dir.trim().isEmpty) return;
    final normalized = dir.trim();
    if (_sourceDirs.contains(normalized)) return;
    setState(() {
      _sourceDirs.add(normalized);
      _clearResponses();
    });
  }

  Future<void> _chooseTargetDir() async {
    final dir = await _directoryPicker();
    if (!mounted || dir == null || dir.trim().isEmpty) return;
    setState(() {
      _targetDir = dir.trim();
      _clearResponses();
    });
  }

  Future<void> _previewMove() async {
    if (!_canPreview) return;
    setState(() {
      _isPreviewLoading = true;
      _errorMessage = null;
      _successMessage = null;
      _result = null;
    });

    try {
      final preview = await _service.preview(_buildRequest());
      if (!mounted) return;
      setState(() {
        _preview = preview;
        if (preview.totalMatched > 1000) {
          _successMessage = '命中数量较大，可创建后台任务执行';
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = _friendlyError(error);
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isPreviewLoading = false;
      });
    }
  }

  Future<void> _executeMove() async {
    if (!_canExecute) return;
    if (_conflict == 'overwrite') {
      final confirmed = await _confirmOverwrite();
      if (!confirmed) return;
    }
    setState(() {
      _isExecuting = true;
      _errorMessage = null;
      _successMessage = null;
    });

    try {
      final result = await _service.execute(_buildRequest());
      if (!mounted) return;
      setState(() {
        _result = result;
        _successMessage = '移动完成';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = _friendlyError(error);
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isExecuting = false;
      });
    }
  }

  Future<void> _createJob() async {
    if (!_canPreview) return;
    if (_conflict == 'overwrite') {
      final confirmed = await _confirmOverwrite();
      if (!confirmed) return;
    }
    setState(() {
      _isExecuting = true;
      _errorMessage = null;
      _successMessage = null;
    });
    try {
      final job = await _service.createJob(_buildRequest());
      if (!mounted) return;
      setState(() {
        _activeJob = job;
        _selectedTabIndex = 1;
        _successMessage = '后台任务已创建';
      });
      await _loadHistory();
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = _friendlyError(error);
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isExecuting = false;
      });
    }
  }

  Future<void> _loadHistory() async {
    setState(() {
      _isHistoryLoading = true;
    });
    try {
      final history = await _service.history();
      if (!mounted) return;
      setState(() {
        _history = history;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = _friendlyError(error);
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _isHistoryLoading = false;
      });
    }
  }

  Future<bool> _confirmOverwrite() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('确认覆盖'),
        content: const Text('覆盖会替换目标目录中的同名文件。'),
        actions: [
          Button(
            child: const Text('取消'),
            onPressed: () => Navigator.pop(context, false),
          ),
          FilledButton(
            child: const Text('确认覆盖'),
            onPressed: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    return confirmed ?? false;
  }

  ImageMoveRequest _buildRequest() {
    return ImageMoveRequest(
      sourceDirs: List<String>.from(_sourceDirs),
      tagId: _selectedTag!.id,
      targetDir: _targetDir!,
      conflict: _conflict,
      allowTargetInsideSource: _allowTargetInsideSource,
    );
  }

  void _clearResponses() {
    _preview = null;
    _result = null;
    _errorMessage = null;
    _successMessage = null;
  }

  String _friendlyError(Object error) {
    final message = error.toString();
    final serverMessage = RegExp(r'ApiError\([^)]*\): ([^(]+)').firstMatch(
      message,
    );
    if (serverMessage != null) {
      return serverMessage.group(1)?.trim() ?? message;
    }
    return '操作失败：$message';
  }

  @override
  Widget build(BuildContext context) {
    return Consumer2<TagProvider, ImageListProvider>(
      builder: (context, tagProvider, imageProvider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('移动'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                CommandBarButton(
                  icon: const Icon(FluentIcons.refresh),
                  label: const Text('刷新图库'),
                  onPressed: _isBusy
                      ? null
                      : () => imageProvider.loadImages(refresh: true),
                ),
                const CommandBarSeparator(),
                CommandBarButton(
                  icon: const Icon(FluentIcons.view),
                  label: const Text('预览'),
                  onPressed: _canPreview ? _previewMove : null,
                ),
                CommandBarButton(
                  icon: const Icon(FluentIcons.move_to_folder),
                  label: const Text('开始移动'),
                  onPressed: _canExecute ? _executeMove : null,
                ),
                CommandBarButton(
                  icon: const Icon(FluentIcons.cloud_import_export),
                  label: const Text('后台任务'),
                  onPressed: _canPreview ? _createJob : null,
                ),
              ],
            ),
          ),
          content: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (_errorMessage != null) ...[
                  InfoBar(
                    severity: InfoBarSeverity.error,
                    title: Text(_errorMessage!),
                  ),
                  const SizedBox(height: 12),
                ],
                if (_successMessage != null) ...[
                  InfoBar(
                    severity: InfoBarSeverity.success,
                    title: Text(_successMessage!),
                  ),
                  const SizedBox(height: 12),
                ],
                Row(
                  children: [
                    ToggleButton(
                      checked: _selectedTabIndex == 0,
                      onChanged: (_) {
                        setState(() {
                          _selectedTabIndex = 0;
                        });
                      },
                      child: const Text('移动任务'),
                    ),
                    const SizedBox(width: 8),
                    ToggleButton(
                      checked: _selectedTabIndex == 1,
                      onChanged: (_) {
                        setState(() {
                          _selectedTabIndex = 1;
                        });
                        _loadHistory();
                      },
                      child: const Text('历史记录'),
                    ),
                  ],
                ),
                if (_selectedTabIndex == 0)
                  _buildMoveTab(tagProvider)
                else
                  _buildHistoryTab(),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildMoveTab(TagProvider tagProvider) {
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _SelectionPanel(
            title: '来源目录',
            action: Button(
              onPressed: _isBusy ? null : _addSourceDir,
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(FluentIcons.folder_open, size: 16),
                  SizedBox(width: 6),
                  Text('添加来源目录'),
                ],
              ),
            ),
            child: _buildSourceDirs(),
          ),
          const SizedBox(height: 12),
          _SelectionPanel(
            title: '指定标签',
            action: tagProvider.isLoading
                ? const ProgressRing(strokeWidth: 2)
                : Button(
                    onPressed: _isBusy ? null : tagProvider.loadTags,
                    child: const Text('刷新标签'),
                  ),
            child: _buildTagSelector(tagProvider),
          ),
          const SizedBox(height: 12),
          _SelectionPanel(
            title: '目标与策略',
            action: Button(
              onPressed: _isBusy ? null : _chooseTargetDir,
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(FluentIcons.folder_open, size: 16),
                  SizedBox(width: 6),
                  Text('选择目标目录'),
                ],
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _PathLine(path: _targetDir ?? '尚未选择'),
                const SizedBox(height: 10),
                ComboBox<String>(
                  value: _conflict,
                  items: const [
                    ComboBoxItem(value: 'skip', child: Text('跳过同名文件')),
                    ComboBoxItem(value: 'rename', child: Text('自动重命名')),
                    ComboBoxItem(value: 'overwrite', child: Text('覆盖同名文件')),
                  ],
                  onChanged: _isBusy
                      ? null
                      : (value) {
                          if (value == null) return;
                          setState(() {
                            _conflict = value;
                            _clearResponses();
                          });
                        },
                ),
                const SizedBox(height: 8),
                Checkbox(
                  checked: _allowTargetInsideSource,
                  content: const Text('允许目标目录位于来源目录内部'),
                  onChanged: _isBusy
                      ? null
                      : (value) {
                          setState(() {
                            _allowTargetInsideSource = value ?? false;
                            _clearResponses();
                          });
                        },
                ),
              ],
            ),
          ),
          if (_activeJob != null) ...[
            const SizedBox(height: 12),
            _JobPanel(batch: _activeJob!),
          ],
          const SizedBox(height: 16),
          _ResultPanel(
            title: '预览结果',
            summary: _preview == null
                ? null
                : [
                    _Metric('命中', _preview!.totalMatched),
                    _Metric('可移动', _preview!.movable),
                    _Metric('跳过', _preview!.skipped),
                  ],
            loading: _isPreviewLoading,
            emptyText: _preview == null
                ? '完成三项选择后点击预览'
                : _preview!.items.isEmpty
                ? '没有命中图片'
                : null,
            items: _preview?.items ?? const <ImageMoveItem>[],
          ),
          const SizedBox(height: 12),
          _ResultPanel(
            title: '执行结果',
            summary: _result == null
                ? null
                : [
                    _Metric('已移动', _result!.moved),
                    _Metric('跳过', _result!.skipped),
                    _Metric('失败', _result!.failed),
                  ],
            loading: _isExecuting,
            emptyText: _result == null ? '预览后可开始移动' : null,
            items: _result?.items ?? const <ImageMoveItem>[],
          ),
        ],
      ),
    );
  }

  Widget _buildHistoryTab() {
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: _HistoryPanel(
        loading: _isHistoryLoading,
        batches: _history,
        onRefresh: _loadHistory,
      ),
    );
  }

  Widget _buildSourceDirs() {
    if (_sourceDirs.isEmpty) {
      return const _MutedText('尚未选择来源目录');
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final dir in _sourceDirs)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              children: [
                Expanded(child: _PathLine(path: dir)),
                IconButton(
                  icon: const Icon(FluentIcons.delete),
                  onPressed: _isBusy
                      ? null
                      : () {
                          setState(() {
                            _sourceDirs.remove(dir);
                            _clearResponses();
                          });
                        },
                ),
              ],
            ),
          ),
        Button(
          onPressed: _isBusy || _sourceDirs.isEmpty
              ? null
              : () {
                  setState(() {
                    _sourceDirs.clear();
                    _clearResponses();
                  });
                },
          child: const Text('清空全部'),
        ),
      ],
    );
  }

  Widget _buildTagSelector(TagProvider tagProvider) {
    if (tagProvider.error != null && tagProvider.allTags.isEmpty) {
      return Text('标签加载失败：${tagProvider.error}');
    }

    final filtered = tagProvider.allTags
        .where(
          (tag) =>
              _tagQuery.isEmpty ||
              tag.preferredLabel.toLowerCase().contains(
                _tagQuery.toLowerCase(),
              ),
        )
        .toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextBox(
          controller: _tagSearchController,
          placeholder: '搜索标签',
          enabled: !_isBusy,
          prefix: const Padding(
            padding: EdgeInsets.symmetric(horizontal: 8),
            child: Icon(FluentIcons.search, size: 14),
          ),
          onChanged: (value) {
            setState(() {
              _tagQuery = value.trim();
            });
          },
        ),
        const SizedBox(height: 8),
        if (_selectedTag != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              '已选择：${_selectedTag!.preferredLabel}（${_selectedTag!.usageCount}）',
            ),
          ),
        ConstrainedBox(
          constraints: const BoxConstraints(maxHeight: 190),
          child: filtered.isEmpty
              ? const _MutedText('没有可选标签')
              : SingleChildScrollView(
                  child: RadioGroup<int>(
                    groupValue: _selectedTag?.id,
                    onChanged: (tagId) {
                      if (_isBusy || tagId == null) return;
                      setState(() {
                        _selectedTag = filtered.firstWhere(
                          (tag) => tag.id == tagId,
                        );
                        _clearResponses();
                      });
                    },
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        for (final tag in filtered)
                          Padding(
                            padding: const EdgeInsets.only(bottom: 4),
                            child: RadioButton<int>(
                              value: tag.id,
                              enabled: !_isBusy,
                              content: Text(
                                '${tag.preferredLabel} (${tag.usageCount})',
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
        ),
      ],
    );
  }
}

class _SelectionPanel extends StatelessWidget {
  final String title;
  final Widget action;
  final Widget child;

  const _SelectionPanel({
    required this.title,
    required this.action,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: FluentTheme.of(context).resources.cardBackgroundFillColorDefault,
        border: Border.all(
          color: FluentTheme.of(context).resources.cardStrokeColorDefault,
        ),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  style: FluentTheme.of(context).typography.subtitle,
                ),
              ),
              action,
            ],
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _ResultPanel extends StatelessWidget {
  final String title;
  final List<_Metric>? summary;
  final bool loading;
  final String? emptyText;
  final List<ImageMoveItem> items;

  const _ResultPanel({
    required this.title,
    required this.summary,
    required this.loading,
    required this.emptyText,
    required this.items,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: FluentTheme.of(context).resources.cardBackgroundFillColorDefault,
        border: Border.all(
          color: FluentTheme.of(context).resources.cardStrokeColorDefault,
        ),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  style: FluentTheme.of(context).typography.subtitle,
                ),
              ),
              if (loading) const ProgressRing(strokeWidth: 2),
            ],
          ),
          if (summary != null) ...[
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: summary!
                  .map((metric) => _MetricPill(metric: metric))
                  .toList(),
            ),
          ],
          const SizedBox(height: 12),
          if (emptyText != null)
            _MutedText(emptyText!)
          else
            _MoveItemList(items: items.take(80).toList()),
        ],
      ),
    );
  }
}

class _MoveItemList extends StatelessWidget {
  final List<ImageMoveItem> items;

  const _MoveItemList({required this.items});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (final item in items)
          Container(
            margin: const EdgeInsets.only(bottom: 8),
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: FluentTheme.of(
                context,
              ).resources.cardBackgroundFillColorSecondary,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        item.filename,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: FluentTheme.of(context).typography.bodyStrong,
                      ),
                    ),
                    _StatusBadge(status: item.status),
                  ],
                ),
                const SizedBox(height: 6),
                _PathLine(path: item.sourcePath),
                const SizedBox(height: 4),
                _PathLine(path: item.targetPath),
                if (item.reason != null && item.reason!.isNotEmpty) ...[
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          '原因：${_reasonLabel(item.reason!)}${item.retryable ? ' · 可重试' : ''}',
                        ),
                      ),
                      IconButton(
                        icon: const Icon(FluentIcons.copy),
                        onPressed: () => Clipboard.setData(
                          ClipboardData(text: item.sourcePath),
                        ),
                      ),
                    ],
                  ),
                ],
                if (item.overwritten) ...[
                  const SizedBox(height: 4),
                  const Text('将覆盖目标同名文件'),
                ],
              ],
            ),
          ),
      ],
    );
  }
}

class _JobPanel extends StatelessWidget {
  final ImageMoveBatch batch;

  const _JobPanel({required this.batch});

  @override
  Widget build(BuildContext context) {
    final total = batch.progress.total == 0 ? 1 : batch.progress.total;
    final value = batch.progress.processed / total;
    return _SelectionPanel(
      title: '后台任务 #${batch.id}',
      action: _StatusBadge(status: batch.status),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ProgressBar(value: value.clamp(0.0, 1.0) * 100),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _MetricPill(metric: _Metric('已处理', batch.progress.processed)),
              _MetricPill(metric: _Metric('已移动', batch.progress.moved)),
              _MetricPill(metric: _Metric('跳过', batch.progress.skipped)),
              _MetricPill(metric: _Metric('失败', batch.progress.failed)),
            ],
          ),
          if ((batch.progress.currentPath ?? '').isNotEmpty) ...[
            const SizedBox(height: 8),
            _PathLine(path: batch.progress.currentPath!),
          ],
        ],
      ),
    );
  }
}

class _HistoryPanel extends StatelessWidget {
  final bool loading;
  final List<ImageMoveBatch> batches;
  final VoidCallback onRefresh;

  const _HistoryPanel({
    required this.loading,
    required this.batches,
    required this.onRefresh,
  });

  @override
  Widget build(BuildContext context) {
    return _SelectionPanel(
      title: '最近移动记录',
      action: loading
          ? const ProgressRing(strokeWidth: 2)
          : Button(onPressed: onRefresh, child: const Text('刷新')),
      child: batches.isEmpty
          ? const _MutedText('暂无移动记录')
          : Column(
              children: [
                for (final batch in batches)
                  Container(
                    margin: const EdgeInsets.only(bottom: 8),
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: FluentTheme.of(
                        context,
                      ).resources.cardBackgroundFillColorSecondary,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                '#${batch.id} ${batch.targetDir}',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: FluentTheme.of(
                                  context,
                                ).typography.bodyStrong,
                              ),
                            ),
                            _StatusBadge(status: batch.status),
                          ],
                        ),
                        const SizedBox(height: 8),
                        Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          children: [
                            _MetricPill(metric: _Metric('命中', batch.totalMatched)),
                            _MetricPill(metric: _Metric('已移动', batch.moved)),
                            _MetricPill(metric: _Metric('跳过', batch.skipped)),
                            _MetricPill(metric: _Metric('失败', batch.failed)),
                          ],
                        ),
                        if (batch.items.isNotEmpty) ...[
                          const SizedBox(height: 8),
                          _MoveItemList(items: batch.items.take(20).toList()),
                        ],
                      ],
                    ),
                  ),
              ],
            ),
    );
  }
}

class _PathLine extends StatelessWidget {
  final String path;

  const _PathLine({required this.path});

  @override
  Widget build(BuildContext context) {
    return Text(
      path,
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
      style: TextStyle(
        fontFeatures: const [FontFeature.tabularFigures()],
        color: FluentTheme.of(context).resources.textFillColorSecondary,
      ),
    );
  }
}

class _Metric {
  final String label;
  final int value;

  const _Metric(this.label, this.value);
}

class _MetricPill extends StatelessWidget {
  final _Metric metric;

  const _MetricPill({required this.metric});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: FluentTheme.of(context).accentColor.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text('${metric.label} ${metric.value}'),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final String status;

  const _StatusBadge({required this.status});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: _statusColor(context).withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(_statusLabel(status)),
    );
  }

  Color _statusColor(BuildContext context) {
    final resources = FluentTheme.of(context).resources;
    return switch (status) {
      'movable' || 'moved' => resources.systemFillColorSuccess,
      'failed' => resources.systemFillColorCritical,
      _ => resources.systemFillColorCaution,
    };
  }
}

class _MutedText extends StatelessWidget {
  final String text;

  const _MutedText(this.text);

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: TextStyle(
        color: FluentTheme.of(context).resources.textFillColorSecondary,
      ),
    );
  }
}

String _statusLabel(String status) {
  return switch (status) {
    'movable' => '可移动',
    'moved' => '已移动',
    'skipped' => '跳过',
    'failed' => '失败',
    'queued' => '排队中',
    'running' => '执行中',
    'completed' => '已完成',
    'cancelled' => '已取消',
    _ => status,
  };
}

String _reasonLabel(String reason) {
  return switch (reason) {
    'source_missing' => '源文件不存在',
    'target_exists' => '目标已存在',
    'permission_denied' => '权限不足',
    'invalid_source_dir' => '来源目录无效',
    'invalid_target_dir' => '目标目录无效',
    'system_target_dir' => '系统关键目录',
    'db_update_failed' => '数据库更新失败',
    'rollback_failed' => '回滚失败',
    'move_failed' => '移动失败',
    _ => reason,
  };
}
