import 'package:flutter/material.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';
import 'tag_picker_results_panel.dart';

/// 编辑标签对话框
///
/// 用于将图片上的现有标签更改为其他标签（现有或新建）
/// 返回格式: {'tagId': int?, 'tagLabel': String?, 'label': String}
class EditTagDialog extends StatefulWidget {
  final int imageId;
  final Tag currentTag;
  final TagService tagService;

  const EditTagDialog({
    super.key,
    required this.imageId,
    required this.currentTag,
    required this.tagService,
  });

  @override
  State<EditTagDialog> createState() => _EditTagDialogState();
}

class _EditTagDialogState extends State<EditTagDialog> {
  static const int _pageSize = 20;

  final _controller = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  List<Tag> _defaultTags = [];
  List<Tag> _searchResults = [];
  List<Tag> _parentCandidates = [];
  String? _selectedLevel;
  int? _selectedParentId;
  bool _loading = false;
  bool _loadingMore = false;
  bool _isSearchMode = false;
  bool _hasMoreDefaultTags = true;

  bool get _requiresParent => _selectedLevel == 'parent';

  @override
  void initState() {
    super.initState();
    _controller.addListener(() => setState(() {}));
    _scrollController.addListener(_handleScroll);
    _loadDefaultTags();
    _selectedLevel = widget.currentTag.level ?? 'child';
    _selectedParentId = widget.currentTag.parentId;
    _loadParentCandidates();
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _loadParentCandidates() async {
    if (_selectedLevel == null || _selectedLevel == 'root') {
      if (mounted) {
        setState(() {
          _parentCandidates = [];
          _selectedParentId = null;
        });
      }
      return;
    }
    try {
      final candidates = await widget.tagService.getParentCandidates(
        _selectedLevel!,
      );
      if (mounted) {
        setState(() {
          _parentCandidates = candidates
              .where((t) => t.id != widget.currentTag.id)
              .toList();
          if (!_parentCandidates.any((t) => t.id == _selectedParentId)) {
            _selectedParentId = null;
          }
        });
      }
    } catch (_) {}
  }

  void _handleScroll() {
    if (_isSearchMode || _loading || _loadingMore || !_hasMoreDefaultTags) {
      return;
    }
    if (!_scrollController.hasClients) {
      return;
    }

    final position = _scrollController.position;
    if (position.pixels >= position.maxScrollExtent - 120) {
      _loadDefaultTags(loadMore: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final displayedTags = _isSearchMode ? _searchResults : _defaultTags;
    final emptyMessage = _isSearchMode ? '未找到匹配标签' : '暂无可选标签';

    return AlertDialog(
      backgroundColor: colorScheme.surfaceContainerHigh,
      title: const Text('编辑标签'),
      content: SizedBox(
        width: double.maxFinite,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // 显示当前标签
            Text(
              "将 '${widget.currentTag.preferredLabel}' 更改为：",
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 16),
            // 搜索框
            TextField(
              controller: _controller,
              decoration: const InputDecoration(
                hintText: '搜索标签或输入新标签名称',
                prefixIcon: Icon(Icons.search),
                border: OutlineInputBorder(),
              ),
              onChanged: _searchTags,
            ),
            const SizedBox(height: 8),
            InputDecorator(
              decoration: const InputDecoration(labelText: '新标签层级'),
              child: Text(switch (_selectedLevel) {
                'root' => '祖级',
                'parent' => '父级',
                _ => '子级',
              }),
            ),
            if (_selectedLevel == 'parent' || _selectedLevel == 'child') ...[
              const SizedBox(height: 8),
              DropdownButtonFormField<int?>(
                initialValue: _selectedParentId,
                decoration: const InputDecoration(labelText: '父标签'),
                items: [
                  if (_selectedLevel == 'child')
                    const DropdownMenuItem<int?>(
                      value: null,
                      child: Text('无父标签'),
                    ),
                  ..._parentCandidates.map(
                    (tag) => DropdownMenuItem<int?>(
                      value: tag.id,
                      child: Text(tag.preferredLabel),
                    ),
                  ),
                ],
                onChanged: (value) {
                  setState(() => _selectedParentId = value);
                },
              ),
            ],
            const SizedBox(height: 8),
            Flexible(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxHeight: 220),
                child: TagPickerResultsPanel(
                  tags: displayedTags,
                  isLoading: _loading,
                  isLoadingMore: _loadingMore,
                  emptyMessage: emptyMessage,
                  scrollController: _isSearchMode ? null : _scrollController,
                  onTagTap: _selectTag,
                ),
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
        ElevatedButton(
          onPressed:
              _controller.text.isNotEmpty &&
                  (!_requiresParent || _selectedParentId != null)
              ? () => _createNewTag(_controller.text)
              : null,
          child: const Text('创建新标签'),
        ),
      ],
    );
  }

  Future<void> _searchTags(String query) async {
    final trimmed = query.trim();
    setState(() {
      if (trimmed.isEmpty) {
        _isSearchMode = false;
        _searchResults = [];
      }
    });

    if (trimmed.isEmpty) {
      return;
    }

    setState(() {
      _isSearchMode = true;
      _loading = true;
    });
    try {
      final tags = await widget.tagService.searchTags(trimmed);
      if (mounted) {
        setState(() {
          _searchResults = tags
              .where(
                (tag) =>
                    tag.id != widget.currentTag.id &&
                    (widget.currentTag.level == null ||
                        tag.level == widget.currentTag.level),
              )
              .toList();
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _loadDefaultTags({bool loadMore = false}) async {
    if (_loading || _loadingMore) {
      return;
    }
    if (loadMore && !_hasMoreDefaultTags) {
      return;
    }

    setState(() {
      if (loadMore) {
        _loadingMore = true;
      } else {
        _loading = true;
      }
    });

    try {
      final tags = await widget.tagService.fetchTags(
        limit: _pageSize,
        offset: loadMore ? _defaultTags.length : 0,
      );

      if (!mounted) {
        return;
      }

      setState(() {
        final filtered = tags
            .where(
              (tag) =>
                  tag.id != widget.currentTag.id &&
                  (widget.currentTag.level == null ||
                      tag.level == widget.currentTag.level),
            )
            .toList();
        if (loadMore) {
          _defaultTags = [..._defaultTags, ...filtered];
        } else {
          _defaultTags = filtered;
        }
        _hasMoreDefaultTags = tags.length == _pageSize;
        _loading = false;
        _loadingMore = false;
      });
    } catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _loadingMore = false;
          _hasMoreDefaultTags = false;
        });
      }
    }
  }

  void _selectTag(Tag tag) {
    // 返回选择的现有标签
    Navigator.pop(context, {
      'tagId': tag.id,
      'tagLabel': null,
      'label': tag.preferredLabel,
    });
  }

  void _createNewTag(String label) {
    if (label.isEmpty) return;
    // 返回创建新标签的信息
    Navigator.pop(context, {
      'tagId': null,
      'tagLabel': label,
      'tagLevel': _selectedLevel,
      'tagParentId': _selectedParentId,
      'label': label,
    });
  }
}
