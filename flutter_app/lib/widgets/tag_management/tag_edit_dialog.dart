import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../models/tag_governance.dart';
import '../../providers/tag_provider.dart';

/// Lightweight Fluent dialog for editing tag label and category.
class TagEditDialog extends StatefulWidget {
  final TagGovernanceRow row;

  const TagEditDialog({super.key, required this.row});

  @override
  State<TagEditDialog> createState() => _TagEditDialogState();
}

class _TagEditDialogState extends State<TagEditDialog> {
  late TextEditingController _labelController;
  late TextEditingController _categoryController;
  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _labelController = TextEditingController(text: widget.row.preferredLabel);
    _categoryController = TextEditingController(
      text: widget.row.primaryCategory ?? '',
    );
  }

  @override
  void dispose() {
    _labelController.dispose();
    _categoryController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ContentDialog(
      title: Text('编辑标签: ${widget.row.preferredLabel}'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          InfoLabel(
            label: '标签名',
            child: TextBox(controller: _labelController, placeholder: '标签名称'),
          ),
          const SizedBox(height: 12),
          InfoLabel(
            label: '分类',
            child: TextBox(controller: _categoryController, placeholder: '主分类'),
          ),
        ],
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.pop(context),
        ),
        FilledButton(
          onPressed: _isSaving ? null : _handleSave,
          child: _isSaving
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: ProgressRing(strokeWidth: 2),
                )
              : const Text('保存'),
        ),
      ],
    );
  }

  Future<void> _handleSave() async {
    final newLabel = _labelController.text.trim();
    final newCategory = _categoryController.text.trim();

    if (newLabel.isEmpty) return;

    setState(() => _isSaving = true);
    try {
      final provider = context.read<TagProvider>();
      await provider.updateTag(
        widget.row.tagId,
        preferredLabel: newLabel != widget.row.preferredLabel ? newLabel : null,
        primaryCategory: newCategory != (widget.row.primaryCategory ?? '')
            ? newCategory
            : null,
      );
      if (mounted) {
        Navigator.pop(context);
        await provider.loadGovernanceTags();
      }
    } catch (e) {
      if (mounted) {
        await showDialog(
          context: context,
          builder: (_) => ContentDialog(
            title: const Text('错误'),
            content: Text('保存失败: $e'),
            actions: [
              Button(
                child: const Text('确定'),
                onPressed: () => Navigator.pop(context),
              ),
            ],
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _isSaving = false);
      }
    }
  }
}
