import 'package:flutter/material.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

/// Callback type for tag filter changes
typedef TagFilterCallback = void Function(List<int> selectedTagIds);

/// A drawer widget for filtering images by tags
class TagFilterDrawer extends StatefulWidget {
  final TagFilterCallback? onFilterChanged;
  final List<int> initialSelectedIds;
  final TagService? tagService;

  const TagFilterDrawer({
    super.key,
    this.onFilterChanged,
    this.initialSelectedIds = const [],
    this.tagService,
  });

  @override
  State<TagFilterDrawer> createState() => _TagFilterDrawerState();
}

class _TagFilterDrawerState extends State<TagFilterDrawer> {
  final TextEditingController _searchController = TextEditingController();
  List<Tag> _allTags = [];
  List<Tag> _filteredTags = [];
  Set<int> _selectedTagIds = {};
  bool _isLoading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _selectedTagIds = Set<int>.from(widget.initialSelectedIds);
    _loadTags();
    _searchController.addListener(_onSearchChanged);
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _loadTags() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final tagService = widget.tagService ?? TagService();
      final tags = await tagService.fetchTags();
      setState(() {
        _allTags = tags;
        _filteredTags = tags;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _error = 'Failed to load tags: $e';
        _isLoading = false;
      });
    }
  }

  void _onSearchChanged() {
    final query = _searchController.text.toLowerCase();
    setState(() {
      _filteredTags = _allTags
          .where((tag) => tag.label.toLowerCase().contains(query))
          .toList();
    });
  }

  void _toggleTag(int tagId) {
    setState(() {
      if (_selectedTagIds.contains(tagId)) {
        _selectedTagIds.remove(tagId);
      } else {
        _selectedTagIds.add(tagId);
      }
    });
  }

  void _applyFilter() {
    widget.onFilterChanged?.call(_selectedTagIds.toList());
    Navigator.pop(context);
  }

  void _clearFilter() {
    setState(() {
      _selectedTagIds.clear();
    });
    widget.onFilterChanged?.call([]);
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: Column(
        children: [
          AppBar(
            title: const Text('标签筛选'),
            automaticallyImplyLeading: false,
            actions: [
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () => Navigator.pop(context),
              ),
            ],
          ),
          
          // Search field
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                hintText: '搜索标签...',
                prefixIcon: const Icon(Icons.search),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                ),
                contentPadding: const EdgeInsets.symmetric(vertical: 0),
              ),
            ),
          ),
          
          // Selected count chip
          if (_selectedTagIds.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Row(
                children: [
                  Chip(
                    label: Text('已选 ${_selectedTagIds.length} 个标签'),
                    deleteIcon: const Icon(Icons.clear, size: 18),
                    onDeleted: () {
                      setState(() {
                        _selectedTagIds.clear();
                      });
                    },
                  ),
                ],
              ),
            ),
          
          const Divider(),
          
          // Tag list
          Expanded(
            child: _buildTagList(),
          ),
          
          // Action buttons
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _selectedTagIds.isEmpty ? null : _clearFilter,
                    child: const Text('清除筛选'),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FilledButton(
                    onPressed: _applyFilter,
                    child: const Text('应用'),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTagList() {
    if (_isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(_error!, style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: _loadTags,
              child: const Text('重试'),
            ),
          ],
        ),
      );
    }

    if (_filteredTags.isEmpty) {
      return const Center(child: Text('没有找到标签'));
    }

    return ListView.builder(
      itemCount: _filteredTags.length,
      itemBuilder: (context, index) {
        final tag = _filteredTags[index];
        final isSelected = _selectedTagIds.contains(tag.id);
        
        return ListTile(
          leading: Checkbox(
            value: isSelected,
            onChanged: (_) => _toggleTag(tag.id),
          ),
          title: Text(tag.label),
          subtitle: tag.category != null ? Text(tag.category!) : null,
          trailing: Text(
            '${tag.usageCount ?? 0}',
            style: TextStyle(color: Colors.grey[600]),
          ),
          onTap: () => _toggleTag(tag.id),
        );
      },
    );
  }
}