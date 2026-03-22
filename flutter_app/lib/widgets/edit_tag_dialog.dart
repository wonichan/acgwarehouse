import 'package:flutter/material.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

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
  final _controller = TextEditingController();
  List<Tag> _suggestions = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _controller.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
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
            // 加载指示器
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(8.0),
                child: Center(child: CircularProgressIndicator()),
              ),
            // 建议列表
            if (_suggestions.isNotEmpty)
              Flexible(
                child: Container(
                  constraints: const BoxConstraints(maxHeight: 200),
                  decoration: BoxDecoration(
                    border: Border.all(color: Colors.grey.shade300),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  margin: const EdgeInsets.only(top: 8),
                  child: ListView.builder(
                    shrinkWrap: true,
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
                        onTap: () => _selectTag(tag),
                      );
                    },
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
          onPressed: _controller.text.isNotEmpty
              ? () => _createNewTag(_controller.text)
              : null,
          child: const Text('创建新标签'),
        ),
      ],
    );
  }

  Future<void> _searchTags(String query) async {
    setState(() {
      // 当输入变化时，清空建议并触发重建以更新按钮状态
      if (query.isEmpty) {
        _suggestions = [];
      }
    });

    if (query.isEmpty) {
      return;
    }

    setState(() => _loading = true);
    try {
      final tags = await widget.tagService.searchTags(query);
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
      'label': label,
    });
  }
}
