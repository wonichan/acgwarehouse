import 'package:flutter/material.dart';
import '../models/collection.dart';

class CollectionListItem extends StatelessWidget {
  final Collection collection;
  final VoidCallback? onTap;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;
  final bool isSelected;

  const CollectionListItem({
    super.key,
    required this.collection,
    this.onTap,
    this.onEdit,
    this.onDelete,
    this.isSelected = false,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: _buildCoverThumbnail(),
      title: Text(
        collection.name,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text('${collection.imageCount} 张图片'),
      trailing: PopupMenuButton<String>(
        icon: const Icon(Icons.more_vert),
        onSelected: (value) {
          switch (value) {
            case 'edit':
              onEdit?.call();
              break;
            case 'delete':
              onDelete?.call();
              break;
          }
        },
        itemBuilder: (context) => [
          const PopupMenuItem(
            value: 'edit',
            child: ListTile(
              leading: Icon(Icons.edit),
              title: Text('重命名'),
              contentPadding: EdgeInsets.zero,
            ),
          ),
          const PopupMenuItem(
            value: 'delete',
            child: ListTile(
              leading: Icon(Icons.delete, color: Colors.red),
              title: Text('删除', style: TextStyle(color: Colors.red)),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
      ),
      selected: isSelected,
      selectedTileColor: Theme.of(context).primaryColor.withOpacity(0.1),
      onTap: onTap,
    );
  }

  Widget _buildCoverThumbnail() {
    return Container(
      width: 48,
      height: 48,
      decoration: BoxDecoration(
        color: Colors.grey[300],
        borderRadius: BorderRadius.circular(8),
      ),
      child: collection.coverImageId != null
          ? const Icon(Icons.image, color: Colors.grey)
          : const Icon(Icons.folder, color: Colors.grey),
    );
  }
}