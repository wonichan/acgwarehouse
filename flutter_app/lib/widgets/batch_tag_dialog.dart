import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

/// 批量添加标签对话框
class BatchAddTagDialog extends StatefulWidget {
  final List<int> imageIds;

  const BatchAddTagDialog({super.key, required this.imageIds});

  @override
  State<BatchAddTagDialog> createState() => _BatchAddTagDialogState();
}

class _BatchAddTagDialogState extends State<BatchAddTagDialog> {
  final _controller = TextEditingController();
  List<Tag> _suggestions = [];
  List<Tag> _parentCandidates = [];
  String? _selectedLevel;
  int? _selectedParentId;
  bool _loading = false;
  bool _processing = false;

  bool get _requiresParent => _selectedLevel == 'parent';

  @override
  void initState() {
    super.initState();
  }

  @override
  void dispose() {
    _controller.dispose();
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
    final tagService = context.read<TagService>();
    try {
      final candidates = await tagService.getParentCandidates(_selectedLevel!);
      if (mounted) {
        setState(() {
          _parentCandidates = candidates;
          _selectedParentId = null;
        });
      }
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('添加标签 (${widget.imageIds.length} 张图片)'),
      content: SizedBox(
        width: 400,
        height: 300,
        child: Column(
          children: [
            TextField(
              controller: _controller,
              decoration: const InputDecoration(
                hintText: '输入标签名称搜索',
                prefixIcon: Icon(Icons.search),
              ),
              onChanged: _searchTags,
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: DropdownButtonFormField<String>(
                    initialValue: _selectedLevel,
                    decoration: const InputDecoration(labelText: '标签层级'),
                    hint: const Text('请选择层级'),
                    items: const [
                      DropdownMenuItem(value: 'root', child: Text('祖级')),
                      DropdownMenuItem(value: 'parent', child: Text('父级')),
                      DropdownMenuItem(value: 'child', child: Text('子级')),
                    ],
                    onChanged: (value) {
                      if (value == null) return;
                      setState(() {
                        _selectedLevel = value;
                        _selectedParentId = null;
                      });
                      _loadParentCandidates();
                    },
                  ),
                ),
                if (_selectedLevel == 'parent' ||
                    _selectedLevel == 'child') ...[
                  const SizedBox(width: 8),
                  Expanded(
                    child: DropdownButtonFormField<int?>(
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
                  ),
                ],
              ],
            ),
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(16.0),
                child: CircularProgressIndicator(),
              )
            else if (_suggestions.isNotEmpty)
              Expanded(
                child: ListView.builder(
                  itemCount: _suggestions.length,
                  itemBuilder: (context, index) {
                    final tag = _suggestions[index];
                    return ListTile(
                      dense: true,
                      title: Text(tag.preferredLabel),
                      subtitle: tag.primaryCategory != null
                          ? Text(tag.primaryCategory!)
                          : null,
                      trailing: Text('${tag.usageCount}'),
                      onTap: _processing ? null : () => _selectTag(tag.id),
                    );
                  },
                ),
              ),
            if (_processing)
              const Padding(
                padding: EdgeInsets.all(8.0),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    SizedBox(width: 8),
                    Text('处理中...'),
                  ],
                ),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _processing ? null : () => Navigator.pop(context),
          child: const Text('取消'),
        ),
        ElevatedButton(
          onPressed:
              _processing ||
                  _selectedLevel == null ||
                  (_requiresParent && _selectedParentId == null)
              ? null
              : () => _createNewTag(_controller.text),
          child: const Text('创建新标签'),
        ),
      ],
    );
  }

  Future<void> _searchTags(String query) async {
    if (query.isEmpty) {
      setState(() => _suggestions = []);
      return;
    }
    setState(() => _loading = true);
    final tagService = context.read<TagService>();
    try {
      final tags = await tagService.searchTags(query);
      if (mounted) {
        setState(() {
          _suggestions = tags;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _selectTag(int tagId) async {
    setState(() => _processing = true);
    final tagService = context.read<TagService>();
    int successCount = 0;
    int failCount = 0;

    for (final imageId in widget.imageIds) {
      try {
        await tagService.addImageTag(imageId, tagId: tagId);
        successCount++;
      } catch (e) {
        failCount++;
      }
    }

    if (mounted) {
      Navigator.pop(context, {
        'success': true,
        'successCount': successCount,
        'failCount': failCount,
      });
    }
  }

  Future<void> _createNewTag(String label) async {
    if (label.trim().isEmpty) {
      if (mounted) {
        Navigator.pop(context, {'success': false, 'error': '标签名称不能为空'});
      }
      return;
    }

    setState(() => _processing = true);
    final tagService = context.read<TagService>();
    int successCount = 0;
    int failCount = 0;

    for (final imageId in widget.imageIds) {
      try {
        await tagService.addImageTag(
          imageId,
          tagLabel: label.trim(),
          level: _selectedLevel,
          parentId: _selectedParentId,
        );
        successCount++;
      } catch (e) {
        failCount++;
      }
    }

    if (mounted) {
      Navigator.pop(context, {
        'success': true,
        'successCount': successCount,
        'failCount': failCount,
      });
    }
  }
}

/// 批量移除标签对话框
class BatchRemoveTagDialog extends StatefulWidget {
  final List<int> imageIds;

  const BatchRemoveTagDialog({super.key, required this.imageIds});

  @override
  State<BatchRemoveTagDialog> createState() => _BatchRemoveTagDialogState();
}

class _BatchRemoveTagDialogState extends State<BatchRemoveTagDialog> {
  final _controller = TextEditingController();
  List<Tag> _suggestions = [];
  bool _loading = false;
  bool _processing = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('移除标签 (${widget.imageIds.length} 张图片)'),
      content: SizedBox(
        width: 400,
        height: 300,
        child: Column(
          children: [
            TextField(
              controller: _controller,
              decoration: const InputDecoration(
                hintText: '输入标签名称搜索',
                prefixIcon: Icon(Icons.search),
              ),
              onChanged: _searchTags,
            ),
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(16.0),
                child: CircularProgressIndicator(),
              )
            else if (_suggestions.isNotEmpty)
              Expanded(
                child: ListView.builder(
                  itemCount: _suggestions.length,
                  itemBuilder: (context, index) {
                    final tag = _suggestions[index];
                    return ListTile(
                      dense: true,
                      title: Text(tag.preferredLabel),
                      subtitle: tag.primaryCategory != null
                          ? Text(tag.primaryCategory!)
                          : null,
                      trailing: const Icon(Icons.remove_circle_outline),
                      onTap: _processing ? null : () => _removeTag(tag.id),
                    );
                  },
                ),
              ),
            if (_processing)
              const Padding(
                padding: EdgeInsets.all(8.0),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    SizedBox(width: 8),
                    Text('处理中...'),
                  ],
                ),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _processing ? null : () => Navigator.pop(context),
          child: const Text('取消'),
        ),
      ],
    );
  }

  Future<void> _searchTags(String query) async {
    if (query.isEmpty) {
      setState(() => _suggestions = []);
      return;
    }
    setState(() => _loading = true);
    final tagService = context.read<TagService>();
    try {
      final tags = await tagService.searchTags(query);
      if (mounted) {
        setState(() {
          _suggestions = tags;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _removeTag(int tagId) async {
    setState(() => _processing = true);
    final tagService = context.read<TagService>();
    int successCount = 0;
    int failCount = 0;

    for (final imageId in widget.imageIds) {
      try {
        await tagService.removeImageTag(imageId, tagId);
        successCount++;
      } catch (e) {
        failCount++;
      }
    }

    if (mounted) {
      Navigator.pop(context, {
        'success': true,
        'successCount': successCount,
        'failCount': failCount,
      });
    }
  }
}
