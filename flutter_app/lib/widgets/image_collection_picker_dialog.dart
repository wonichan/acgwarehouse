import 'package:fluent_ui/fluent_ui.dart';

import '../models/collection.dart';
import '../services/collection_service.dart';

class ImageCollectionPickerDialog extends StatefulWidget {
  final int imageId;
  final CollectionService collectionService;

  const ImageCollectionPickerDialog({
    super.key,
    required this.imageId,
    required this.collectionService,
  });

  @override
  State<ImageCollectionPickerDialog> createState() =>
      _ImageCollectionPickerDialogState();
}

class _ImageCollectionPickerDialogState extends State<ImageCollectionPickerDialog> {
  List<Collection> _collections = const [];
  bool _loading = true;
  bool _adding = false;
  bool _creating = false;
  int? _selectedCollectionId;
  String? _error;

  final TextEditingController _nameController = TextEditingController();
  final TextEditingController _descController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _loadCollections();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _descController.dispose();
    super.dispose();
  }

  Future<void> _loadCollections() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final collections = await widget.collectionService.fetchCollections(limit: 200);
      if (!mounted) return;
      setState(() {
        _collections = collections;
        _selectedCollectionId = collections.isNotEmpty ? collections.first.id : null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '加载合集失败: $e';
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _loading = false;
      });
    }
  }

  Future<void> _addToSelectedCollection() async {
    final collectionId = _selectedCollectionId;
    if (collectionId == null) {
      setState(() {
        _error = '请选择一个合集';
      });
      return;
    }

    setState(() {
      _adding = true;
      _error = null;
    });

    try {
      await widget.collectionService.addImageToCollection(collectionId, widget.imageId);
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '收藏失败: $e';
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _adding = false;
      });
    }
  }

  Future<void> _createAndAdd() async {
    final name = _nameController.text.trim();
    final desc = _descController.text.trim();
    if (name.isEmpty) {
      setState(() {
        _error = '请输入合集名称';
      });
      return;
    }

    setState(() {
      _creating = true;
      _error = null;
    });

    try {
      final created = await widget.collectionService.createCollection(
        name,
        description: desc.isEmpty ? null : desc,
      );

      try {
        await widget.collectionService.addImageToCollection(created.id, widget.imageId);
      } catch (e) {
        if (!mounted) return;
        setState(() {
          _error = '合集已创建，但收藏失败: $e';
        });
        return;
      }

      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '创建合集失败: $e';
      });
    } finally {
      if (!mounted) return;
      setState(() {
        _creating = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);

    return ContentDialog(
      title: const Text('收藏到合集'),
      constraints: const BoxConstraints(maxWidth: 480),
      content: SizedBox(
        width: 440,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (_loading)
              const Center(child: Padding(padding: EdgeInsets.all(16), child: ProgressRing()))
            else ...[
              if (_error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Text(
                    _error!,
                    style: TextStyle(color: theme.resources.systemFillColorCritical),
                  ),
                ),
              const Text('选择已有合集'),
              const SizedBox(height: 8),
              Container(
                constraints: const BoxConstraints(maxHeight: 220),
                decoration: BoxDecoration(
                  border: Border.all(color: theme.resources.cardStrokeColorDefault),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: _collections.isEmpty
                    ? const Center(child: Padding(padding: EdgeInsets.all(12), child: Text('暂无合集')))
                    : ListView.builder(
                        shrinkWrap: true,
                        itemCount: _collections.length,
                        itemBuilder: (context, index) {
                          final item = _collections[index];
                          return RadioButton(
                            checked: _selectedCollectionId == item.id,
                            onChanged: (checked) {
                              if (checked == true) {
                                setState(() {
                                  _selectedCollectionId = item.id;
                                });
                              }
                            },
                            content: Text('${item.name} (${item.imageCount})'),
                          );
                        },
                      ),
              ),
              const SizedBox(height: 16),
              const Text('或新建合集并收藏'),
              const SizedBox(height: 8),
              TextBox(
                controller: _nameController,
                placeholder: '合集名称',
              ),
              const SizedBox(height: 8),
              TextBox(
                controller: _descController,
                placeholder: '描述（可选）',
              ),
              const SizedBox(height: 10),
              Align(
                alignment: Alignment.centerLeft,
                child: Button(
                  onPressed: _creating ? null : _createAndAdd,
                  child: _creating ? const Text('创建中...') : const Text('新建并收藏'),
                ),
              ),
            ],
          ],
        ),
      ),
      actions: [
        Button(
          onPressed: _adding || _creating ? null : () => Navigator.of(context).pop(false),
          child: const Text('取消'),
        ),
        FilledButton(
          onPressed: (_loading || _adding || _creating || _collections.isEmpty)
              ? null
              : _addToSelectedCollection,
          child: _adding ? const Text('收藏中...') : const Text('收藏'),
        ),
      ],
    );
  }
}
