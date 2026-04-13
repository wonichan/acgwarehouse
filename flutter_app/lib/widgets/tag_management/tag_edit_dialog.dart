import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../models/tag.dart';
import '../../models/tag_governance.dart';
import '../../providers/tag_provider.dart';

/// Lightweight Fluent dialog for editing tag label, category, level, and parent, or creating a new tag.
class TagEditDialog extends StatefulWidget {
  final TagGovernanceRow? row; // null means create mode

  const TagEditDialog({super.key, this.row});

  @override
  State<TagEditDialog> createState() => _TagEditDialogState();
}

class _TagEditDialogState extends State<TagEditDialog> {
  late TextEditingController _labelController;
  late TextEditingController _categoryController;
  String? _level;
  int? _parentId;
  List<Tag> _parentCandidates = [];
  bool _isSaving = false;
  bool _isLoadingParents = false;

  bool get _supportsParentSelection => _level == 'parent' || _level == 'child';

  bool get _requiresParentSelection => _level == 'parent';

  @override
  void initState() {
    super.initState();
    _labelController = TextEditingController(
      text: widget.row?.preferredLabel ?? '',
    );
    _categoryController = TextEditingController(
      text: widget.row?.primaryCategory ?? '',
    );
    _level = widget.row?.level;
    _parentId = widget.row?.parentId;

    if (_supportsParentSelection) {
      _loadParentCandidates();
    }
  }

  @override
  void dispose() {
    _labelController.dispose();
    _categoryController.dispose();
    super.dispose();
  }

  Future<void> _loadParentCandidates() async {
    setState(() {
      _isLoadingParents = true;
    });
    try {
      final candidates = await context.read<TagProvider>().getParentCandidates(
        _level!,
      );
      if (mounted) {
        setState(() {
          _parentCandidates = candidates
              .where((t) => t.id != widget.row?.tagId)
              .toList();
          if (!_parentCandidates.any((t) => t.id == _parentId)) {
            _parentId = null;
          }
          _isLoadingParents = false;
        });
      }
    } catch (e) {
      debugPrint('Failed to load parents: $e');
      if (mounted) {
        setState(() => _isLoadingParents = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final isCreate = widget.row == null;
    return ContentDialog(
      title: Text(isCreate ? '新建标签' : '编辑标签: ${widget.row!.preferredLabel}'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            InfoLabel(
              label: '标签名',
              child: TextBox(controller: _labelController, placeholder: '标签名称'),
            ),
            const SizedBox(height: 12),
            InfoLabel(
              label: '分类',
              child: TextBox(
                controller: _categoryController,
                placeholder: '主分类',
              ),
            ),
            const SizedBox(height: 12),
            InfoLabel(
              label: '层级',
              child: ComboBox<String>(
                value: _level,
                placeholder: const Text('请选择层级'),
                items: const [
                  ComboBoxItem(value: 'root', child: Text('Root (祖级)')),
                  ComboBoxItem(value: 'parent', child: Text('Parent (父级)')),
                  ComboBoxItem(value: 'child', child: Text('Child (子级)')),
                ],
                onChanged: (val) {
                  if (val != null && val != _level) {
                    setState(() {
                      _level = val;
                      if (!_supportsParentSelection) {
                        _parentId = null;
                      } else {
                        _parentCandidates = [];
                      }
                    });
                    if (_supportsParentSelection) {
                      _loadParentCandidates();
                    }
                  }
                },
                isExpanded: true,
              ),
            ),
            if (_supportsParentSelection) ...[
              const SizedBox(height: 12),
              InfoLabel(
                label: '父标签',
                child: _isLoadingParents
                    ? const ProgressRing()
                    : ComboBox<int?>(
                        value: _parentCandidates.any((c) => c.id == _parentId)
                            ? _parentId
                            : null,
                        placeholder: const Text('选择父标签'),
                        items: [
                          if (_level == 'child')
                            const ComboBoxItem<int?>(
                              value: null,
                              child: Text('无父标签'),
                            ),
                          ..._parentCandidates.map(
                            (c) => ComboBoxItem<int?>(
                              value: c.id,
                              child: Text(c.preferredLabel),
                            ),
                          ),
                        ],
                        onChanged: (val) {
                          setState(() {
                            _parentId = val;
                          });
                        },
                        isExpanded: true,
                      ),
              ),
            ],
          ],
        ),
      ),
      actions: [
        Button(
          child: const Text('取消'),
          onPressed: () => Navigator.pop(context),
        ),
        FilledButton(
          onPressed:
              _isSaving ||
                  _level == null ||
                  (_requiresParentSelection && _parentId == null)
              ? null
              : _handleSave,
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
      if (widget.row == null) {
        // Create mode
        await provider.createTag(
          preferredLabel: newLabel,
          primaryCategory: newCategory.isEmpty ? null : newCategory,
          level: _level,
          parentId: _parentId,
        );
      } else {
        // Update mode
        final row = widget.row!;
        bool needsLoad = false;
        if (newLabel != row.preferredLabel ||
            newCategory != (row.primaryCategory ?? '')) {
          await provider.updateTag(
            row.tagId,
            preferredLabel: newLabel != row.preferredLabel ? newLabel : null,
            primaryCategory: newCategory != (row.primaryCategory ?? '')
                ? newCategory
                : null,
          );
          needsLoad = true;
        }

        if (_level != (row.level ?? 'root') || _parentId != row.parentId) {
          if (_level != (row.level ?? 'root')) {
            await provider.changeTagLevel(
              row.tagId,
              _level!,
              parentId: _parentId,
            );
          } else if (_parentId != row.parentId) {
            await provider.reparentTag(row.tagId, _parentId);
          }
          needsLoad = true;
        }

        if (needsLoad) {
          await provider.loadGovernanceTags();
        }
      }
      if (mounted) {
        Navigator.pop(context);
        if (widget.row == null) {
          await context.read<TagProvider>().loadGovernanceTags();
        }
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
