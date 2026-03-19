import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

class AddTagDialog extends StatefulWidget {
  final int imageId;

  const AddTagDialog({super.key, required this.imageId});

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
    final tagService = context.read<TagService>();
    try {
      final tags = await tagService.searchTags(query);
      setState(() {
        _suggestions = tags;
        _loading = false;
      });
    } catch (e) {
      setState(() => _loading = false);
    }
  }

  Future<void> _selectTag(int tagId) async {
    final tagService = context.read<TagService>();
    final scaffoldMessenger = ScaffoldMessenger.of(context);
    try {
      await tagService.addImageTag(widget.imageId, tagId: tagId);
      if (mounted) {
        Navigator.pop(context, true);
      }
    } catch (e) {
      if (mounted) {
        scaffoldMessenger.showSnackBar(
          SnackBar(content: Text('添加失败: $e')),
        );
      }
    }
  }

  Future<void> _createNewTag(String label) async {
    if (label.isEmpty) return;
    final tagService = context.read<TagService>();
    final scaffoldMessenger = ScaffoldMessenger.of(context);
    try {
      await tagService.addImageTag(widget.imageId, tagLabel: label);
      if (mounted) {
        Navigator.pop(context, true);
      }
    } catch (e) {
      if (mounted) {
        scaffoldMessenger.showSnackBar(
          SnackBar(content: Text('创建失败: $e')),
        );
      }
    }
  }
}
