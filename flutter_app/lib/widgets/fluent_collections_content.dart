import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/material.dart' as material;
import 'package:provider/provider.dart';

import '../models/collection.dart';
import '../models/image.dart';
import '../providers/config_provider.dart';
import '../services/collection_service.dart';
import 'fluent_image_card.dart';

class FluentCollectionsContent extends StatefulWidget {
  final CollectionService? collectionService;
  final void Function(ImageModel image)? onImageTap;

  const FluentCollectionsContent({
    super.key,
    this.collectionService,
    this.onImageTap,
  });

  @override
  State<FluentCollectionsContent> createState() =>
      _FluentCollectionsContentState();
}

class _FluentCollectionsContentState extends State<FluentCollectionsContent> {
  late final CollectionService _collectionService;
  late final bool _ownsCollectionService;

  List<Collection> _collections = const <Collection>[];
  List<ImageModel> _images = const <ImageModel>[];
  int? _selectedCollectionId;
  bool _isLoadingCollections = true;
  bool _isLoadingImages = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _ownsCollectionService = widget.collectionService == null;
    _collectionService =
        widget.collectionService ??
        CollectionService(baseUrl: context.read<ConfigProvider>().baseUrl);
    _loadCollections();
  }

  @override
  void dispose() {
    if (_ownsCollectionService) {
      _collectionService.dispose();
    }
    super.dispose();
  }

  Future<void> _loadCollections() async {
    setState(() {
      _isLoadingCollections = true;
      _errorMessage = null;
    });

    try {
      final collections = await _collectionService.fetchCollections(limit: 200);
      if (!mounted) return;

      final selectedCollectionId = collections.isNotEmpty
          ? collections.first.id
          : null;

      setState(() {
        _collections = collections;
        _selectedCollectionId = selectedCollectionId;
        _images = const <ImageModel>[];
        _isLoadingCollections = false;
      });

      if (selectedCollectionId != null) {
        await _loadCollectionImages(selectedCollectionId);
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = '加载合集失败: $error';
        _isLoadingCollections = false;
      });
    }
  }

  Future<void> _loadCollectionImages(int collectionId) async {
    setState(() {
      _selectedCollectionId = collectionId;
      _isLoadingImages = true;
      _errorMessage = null;
      _images = const <ImageModel>[];
    });

    try {
      final images = await _collectionService.fetchCollectionImages(
        collectionId,
        limit: 200,
      );
      if (!mounted) return;
      setState(() {
        _images = images;
        _isLoadingImages = false;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _errorMessage = '加载合集图片失败: $error';
        _isLoadingImages = false;
      });
    }
  }

  Future<void> _showImageContextMenu(
    ImageModel image,
    material.Offset globalPosition,
  ) async {
    final selected = await material.showMenu<String>(
      context: context,
      position: material.RelativeRect.fromLTRB(
        globalPosition.dx,
        globalPosition.dy,
        globalPosition.dx,
        globalPosition.dy,
      ),
      items: const [
        material.PopupMenuItem<String>(
          value: 'remove_favorite',
          child: Text('取消收藏'),
        ),
      ],
    );

    if (!mounted || selected != 'remove_favorite') {
      return;
    }

    await _removeImageFromCollection(image);
  }

  Future<void> _removeImageFromCollection(ImageModel image) async {
    final collectionId = _selectedCollectionId;
    if (collectionId == null) {
      return;
    }

    try {
      await _collectionService.removeImageFromCollection(
        collectionId,
        image.id,
      );
      if (!mounted) {
        return;
      }

      setState(() {
        _collections = _collections
            .map(
              (collection) => collection.id == collectionId
                  ? collection.copyWith(
                      imageCount: collection.imageCount > 0
                          ? collection.imageCount - 1
                          : 0,
                    )
                  : collection,
            )
            .toList(growable: false);
      });

      await _loadCollectionImages(collectionId);
    } catch (error) {
      if (!mounted) {
        return;
      }

      setState(() {
        _errorMessage = '取消收藏失败: $error';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoadingCollections) {
      return const Center(child: ProgressRing());
    }

    if (_errorMessage != null && _collections.isEmpty) {
      return _buildCenteredMessage(
        context,
        title: _errorMessage!,
        icon: FluentIcons.error,
        iconColor: FluentTheme.of(context).resources.systemFillColorCritical,
        actionLabel: '重试',
        onPressed: _loadCollections,
      );
    }

    if (_collections.isEmpty) {
      return _buildCenteredMessage(
        context,
        title: '暂无合集',
        subtitle: '请先在图片上右键选择“收藏”',
      );
    }

    return Row(
      children: [
        SizedBox(width: 280, child: _buildCollectionList(context)),
        SizedBox(
          width: 1,
          child: ColoredBox(
            color: FluentTheme.of(context).resources.dividerStrokeColorDefault,
          ),
        ),
        Expanded(child: _buildImagePanel(context)),
      ],
    );
  }

  Widget _buildCollectionList(BuildContext context) {
    final theme = FluentTheme.of(context);

    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: _collections.length,
      separatorBuilder: (_, _) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        final collection = _collections[index];
        final isSelected = collection.id == _selectedCollectionId;

        return GestureDetector(
          onTap: () => _loadCollectionImages(collection.id),
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: isSelected
                  ? theme.accentColor.withValues(alpha: 0.12)
                  : theme.resources.cardBackgroundFillColorSecondary,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                color: isSelected
                    ? theme.accentColor
                    : theme.resources.cardStrokeColorDefault,
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(collection.name, style: theme.typography.bodyStrong),
                const SizedBox(height: 4),
                Text('${collection.imageCount} 张图片'),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildImagePanel(BuildContext context) {
    final selectedCollection = _collections.where(
      (collection) => collection.id == _selectedCollectionId,
    );
    final collection = selectedCollection.isEmpty
        ? null
        : selectedCollection.first;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Text(
            collection?.name ?? '收藏',
            style: FluentTheme.of(context).typography.subtitle,
          ),
        ),
        Expanded(child: _buildImageBody(context)),
      ],
    );
  }

  Widget _buildImageBody(BuildContext context) {
    if (_isLoadingImages) {
      return const Center(child: ProgressRing());
    }

    if (_errorMessage != null) {
      return _buildCenteredMessage(
        context,
        title: _errorMessage!,
        icon: FluentIcons.error,
        iconColor: FluentTheme.of(context).resources.systemFillColorCritical,
        actionLabel: '重试',
        onPressed: () {
          final collectionId = _selectedCollectionId;
          if (collectionId != null) {
            _loadCollectionImages(collectionId);
          }
        },
      );
    }

    if (_images.isEmpty) {
      return _buildCenteredMessage(context, title: '该合集暂无图片');
    }

    return GridView.builder(
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
        maxCrossAxisExtent: 220,
        childAspectRatio: 1,
        mainAxisSpacing: 8,
        crossAxisSpacing: 8,
      ),
      itemCount: _images.length,
      itemBuilder: (context, index) {
        final image = _images[index];
        return FluentImageCard(
          image: image,
          onTap: widget.onImageTap,
          onSecondaryTapDown: (image, details) {
            _showImageContextMenu(image, details.globalPosition);
          },
        );
      },
    );
  }

  Widget _buildCenteredMessage(
    BuildContext context, {
    required String title,
    String? subtitle,
    String? actionLabel,
    VoidCallback? onPressed,
    IconData? icon,
    Color? iconColor,
  }) {
    final theme = FluentTheme.of(context);
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            icon ?? FluentIcons.favorite_star,
            size: 56,
            color: iconColor ?? theme.resources.textFillColorSecondary,
          ),
          const SizedBox(height: 12),
          Text(title, style: theme.typography.subtitle),
          if (subtitle != null) ...[const SizedBox(height: 8), Text(subtitle)],
          if (actionLabel != null && onPressed != null) ...[
            const SizedBox(height: 12),
            Button(onPressed: onPressed, child: Text(actionLabel)),
          ],
        ],
      ),
    );
  }
}
