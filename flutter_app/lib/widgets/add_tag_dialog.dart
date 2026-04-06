import 'package:flutter/material.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

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
  final _controller = TextEditingController();
  List<Tag> _suggestions = [];
  bool _loading = false;

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('添加标签'),
      content: SizedBox(
        width: 400,
        height: 300,
        child: Column(
          children: [
            TextField(
              controller: _controller,
              decoration: const InputDecoration(hintText: '输入标签名称'),
              onChanged: _searchTags,
            ),
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(8.0),
                child: CircularProgressIndicator(),
              ),
            if (_suggestions.isNotEmpty)
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
                      onTap: () => _selectTag(tag.id),
                    );
                  },
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
          child: const Text('创建新标签'),
          onPressed: () => _createNewTag(_controller.text),
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
    try {
      final tags = await widget.tagService.searchTags(query);
      setState(() {
        _suggestions = tags;
        _loading = false;
      });
    } catch (e) {
      setState(() => _loading = false);
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
      // Return error for empty input
      if (mounted) {
        Navigator.pop(context, {'success': false, 'error': '标签名称不能为空'});
      }
      return;
    }
    try {
      await widget.tagService.addImageTag(
        widget.imageId,
        tagLabel: label.trim(),
      );
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
}
