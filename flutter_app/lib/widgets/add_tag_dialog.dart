import 'package:flutter/material.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';
import 'tag_picker_results_panel.dart';

class AddTagDialog extends StatefulWidget {
  final int imageId;
  final TagService tagService;

  const AddTagDialog({
    super.key,
    required this.imageId,
    required this.tagService,
  });

  @override
  State<AddTagDialog> createState() => _AddTagDialogState();
}

class _AddTagDialogState extends State<AddTagDialog> {
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
  bool _isCreating = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_handleScroll);
    _loadDefaultTags();
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
          _parentCandidates = candidates;
          _selectedParentId = null;
        });
      }
    } catch (e) {
      debugPrint('Failed to load parent candidates: $e');
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
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

    final requiresParent = _selectedLevel == 'parent';

    return AlertDialog(
      backgroundColor: colorScheme.surfaceContainerHigh,
      title: const Text('添加标签'),
      content: Container(
        width: 400,
        height: 500,
        color: colorScheme.surfaceContainerHigh,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            TextField(
              controller: _controller,
              decoration: const InputDecoration(
                hintText: '搜索标签或输入新标签名称',
                prefixIcon: Icon(Icons.search),
              ),
              onChanged: _searchTags,
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  flex: 1,
                  child: DropdownButtonFormField<String>(
                    initialValue: _selectedLevel,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: '标签层级',
                      contentPadding: EdgeInsets.symmetric(horizontal: 8),
                    ),
                    hint: const Text('请选择层级'),
                    items: const [
                      DropdownMenuItem(value: 'root', child: Text('祖级 (Root)')),
                      DropdownMenuItem(
                        value: 'parent',
                        child: Text('父级 (Parent)'),
                      ),
                      DropdownMenuItem(
                        value: 'child',
                        child: Text('子级 (Child)'),
                      ),
                    ],
                    onChanged: (value) {
                      if (value != null) {
                        setState(() {
                          _selectedLevel = value;
                          _selectedParentId = null;
                        });
                        _loadParentCandidates();
                      }
                    },
                  ),
                ),
                if (_selectedLevel == 'parent' ||
                    _selectedLevel == 'child') ...[
                  const SizedBox(width: 8),
                  Expanded(
                    flex: 1,
                    child: DropdownButtonFormField<int?>(
                      initialValue: _selectedParentId,
                      isExpanded: true,
                      decoration: const InputDecoration(
                        labelText: '父标签',
                        contentPadding: EdgeInsets.symmetric(horizontal: 8),
                      ),
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
                        setState(() {
                          _selectedParentId = value;
                        });
                      },
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 8),
            Expanded(
              child: TagPickerResultsPanel(
                tags: displayedTags,
                isLoading: _loading,
                isLoadingMore: _loadingMore,
                emptyMessage: emptyMessage,
                scrollController: _isSearchMode ? null : _scrollController,
                onTagTap: (tag) => _selectTag(tag.id),
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          child: const Text('取消'),
          onPressed: () => Navigator.pop(context),
        ),
        ElevatedButton(
          onPressed:
              _isCreating ||
                  _selectedLevel == null ||
                  (requiresParent && _selectedParentId == null)
              ? null
              : () => _createNewTag(_controller.text),
          child: _isCreating
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('创建新标签'),
        ),
      ],
    );
  }

  Future<void> _searchTags(String query) async {
    final trimmed = query.trim();
    if (trimmed.isEmpty) {
      setState(() {
        _isSearchMode = false;
        _searchResults = [];
      });
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
          _searchResults = tags;
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
        if (loadMore) {
          _defaultTags = [..._defaultTags, ...tags];
        } else {
          _defaultTags = tags;
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

  Future<void> _selectTag(int tagId) async {
    try {
      await widget.tagService.addImageTag(widget.imageId, tagId: tagId);
      if (mounted) {
        Navigator.pop(context, true);
      }
    } catch (e) {
      // Return error to caller for display
      if (mounted) {
        Navigator.pop(context, {'success': false, 'error': e.toString()});
      }
    }
  }

  Future<void> _createNewTag(String label) async {
    if (label.trim().isEmpty) {
      if (mounted) {
        Navigator.pop(context, {'success': false, 'error': '标签名称不能为空'});
      }
      return;
    }

    if (_selectedLevel == 'parent' && _selectedParentId == null) {
      if (mounted) {
        Navigator.pop(context, {'success': false, 'error': '父级标签必须选择祖级父标签'});
      }
      return;
    }

    setState(() {
      _isCreating = true;
    });

    try {
      await widget.tagService.addImageTag(
        widget.imageId,
        tagLabel: label.trim(),
        level: _selectedLevel,
        parentId: _selectedParentId,
      );
      if (mounted) {
        Navigator.pop(context, true);
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _isCreating = false;
        });
        Navigator.pop(context, {'success': false, 'error': e.toString()});
      }
    }
  }
}
