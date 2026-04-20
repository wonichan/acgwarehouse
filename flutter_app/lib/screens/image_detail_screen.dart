import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:extended_image/extended_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../models/image.dart';
import '../providers/config_provider.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';
import '../widgets/image_lightbox.dart';
import '../widgets/image_metadata_panel.dart';
import '../widgets/image_metadata_pane_theme.dart';

/// Image detail screen with optional gallery navigation.
///
/// When [images] and [initialIndex] are provided, the screen allows
/// browsing through the list via arrow keys, buttons, and a bottom
/// filmstrip of thumbnails. Otherwise it displays a single image.
class ImageDetailScreen extends StatefulWidget {
  final ImageModel image;
  final TagService? tagService;

  /// Full list of images for navigation context.
  final List<ImageModel> images;

  /// Index of [image] within [images] (-1 means no navigation).
  final int initialIndex;

  const ImageDetailScreen({
    super.key,
    required this.image,
    this.tagService,
    this.images = const [],
    this.initialIndex = -1,
  });

  bool get _canNavigate =>
      images.isNotEmpty && initialIndex >= 0 && initialIndex < images.length;

  @override
  State<ImageDetailScreen> createState() => _ImageDetailScreenState();
}

class _ImageDetailScreenState extends State<ImageDetailScreen> {
  late TagProvider _tagProvider;
  late int _currentIndex;
  late List<ImageModel> _images;
  final ScrollController _filmstripController = ScrollController();

  ImageModel get _currentImage =>
      _images.isNotEmpty && _currentIndex >= 0 && _currentIndex < _images.length
      ? _images[_currentIndex]
      : widget.image;

  @override
  void initState() {
    super.initState();
    _images = widget._canNavigate ? widget.images : [widget.image];
    _currentIndex = widget._canNavigate ? widget.initialIndex : 0;
    _tagProvider = TagProvider(
      widget.tagService ??
          TagService(baseUrl: context.read<ConfigProvider>().baseUrl),
    );
    _loadImageTags();
  }

  @override
  void didUpdateWidget(covariant ImageDetailScreen oldWidget) {
    super.didUpdateWidget(oldWidget);

    // Resync navigation context if the images list or initial index changed
    if (widget._canNavigate &&
        (widget.images != oldWidget.images ||
            widget.initialIndex != oldWidget.initialIndex)) {
      _images = widget.images;
      _currentIndex = widget.initialIndex;
    }

    if (oldWidget.image != widget.image) {
      _tagProvider.loadImageTags(widget.image.id);
    }
  }

  @override
  void dispose() {
    _filmstripController.dispose();
    _tagProvider.dispose();
    super.dispose();
  }

  Future<void> _loadImageTags() async {
    await _tagProvider.loadImageTags(_currentImage.id);
  }

  void _navigate(int delta) {
    if (!widget._canNavigate) return;

    final newIndex = _currentIndex + delta;
    if (newIndex < 0 || newIndex >= _images.length) return;

    setState(() {
      _currentIndex = newIndex;
    });
    _loadImageTags();
    _scrollFilmstripToCurrent();
  }

  void _selectImage(int index) {
    if (index == _currentIndex) return;
    if (index < 0 || index >= _images.length) return;

    setState(() {
      _currentIndex = index;
    });
    _loadImageTags();
    _scrollFilmstripToCurrent();
  }

  void _scrollFilmstripToCurrent() {
    if (!_filmstripController.hasClients) return;

    final offset =
        (_currentIndex * 84.0) - (MediaQuery.of(context).size.width / 2) + 42;
    _filmstripController.animateTo(
      offset.clamp(0.0, _filmstripController.position.maxScrollExtent),
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeInOut,
    );
  }

  KeyEventResult _onKeyEvent(FocusNode node, KeyEvent event) {
    if (event is KeyDownEvent) {
      if (event.logicalKey == LogicalKeyboardKey.arrowRight) {
        _navigate(1);
        return KeyEventResult.handled;
      }
      if (event.logicalKey == LogicalKeyboardKey.arrowLeft) {
        _navigate(-1);
        return KeyEventResult.handled;
      }
      if (event.logicalKey == LogicalKeyboardKey.escape) {
        Navigator.of(context).pop();
        return KeyEventResult.handled;
      }
    }
    return KeyEventResult.ignored;
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final isDesktopLayout = MediaQuery.of(context).size.width >= 900;
    final isDark = colorScheme.brightness == Brightness.dark;
    final pageSurface = _opaqueColor(
      Color.alphaBlend(
        colorScheme.outlineVariant.withValues(alpha: 0.08),
        colorScheme.surface,
      ),
    );
    final panelSurface = _opaqueColor(colorScheme.surfaceContainerHighest);
    final viewerSurface = isDark ? const Color(0xFF0E0E10) : panelSurface;

    return Focus(
      autofocus: true,
      onKeyEvent: _onKeyEvent,
      child: ChangeNotifierProvider.value(
        value: _tagProvider,
        child: Scaffold(
          backgroundColor: pageSurface,
          appBar: AppBar(
            backgroundColor: pageSurface,
            surfaceTintColor: Colors.transparent,
            title: widget._canNavigate
                ? Row(
                    children: [
                      Text('图片详情'),
                      const SizedBox(width: 12),
                      Text(
                        '${_currentIndex + 1} / ${_images.length}',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: colorScheme.onSurface.withValues(alpha: 0.5),
                        ),
                      ),
                    ],
                  )
                : const Text('图片详情'),
            actions: [
              if (widget._canNavigate) ...[
                TextButton.icon(
                  onPressed: _currentIndex > 0 ? () => _navigate(-1) : null,
                  icon: const Icon(Icons.arrow_back_ios, size: 16),
                  label: const Text('上一张'),
                ),
                TextButton.icon(
                  onPressed: _currentIndex < _images.length - 1
                      ? () => _navigate(1)
                      : null,
                  icon: const Icon(Icons.arrow_forward_ios, size: 16),
                  label: const Text('下一张'),
                  iconAlignment: IconAlignment.end,
                ),
              ],
            ],
          ),
          body: isDesktopLayout
              ? Column(
                  children: [
                    Expanded(
                      child: _buildDesktopLayout(
                        context,
                        pageSurface,
                        panelSurface,
                        viewerSurface,
                      ),
                    ),
                    if (widget._canNavigate) _buildFilmstrip(context),
                  ],
                )
              : SingleChildScrollView(
                  child: _buildCompactLayout(
                    context,
                    pageSurface,
                    panelSurface,
                    viewerSurface,
                  ),
                ),
        ),
      ),
    );
  }

  Widget _buildDesktopLayout(
    BuildContext context,
    Color pageSurface,
    Color panelSurface,
    Color viewerSurface,
  ) {
    final paneTheme = ImageMetadataPaneTheme.of(context);

    return Container(
      color: pageSurface,
      padding: const EdgeInsets.all(24),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 320),
            child: Container(
              key: const ValueKey('image-detail-metadata-pane'),
              decoration: BoxDecoration(
                color: paneTheme.panelSurface,
                borderRadius: BorderRadius.circular(20),
                border: Border.all(color: paneTheme.borderColor),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(20),
                child: ImageMetadataPanel(
                  imageId: _currentImage.id,
                  metadataSection: _buildMetadataSection(context, paneTheme),
                ),
              ),
            ),
          ),
          const SizedBox(width: 24),
          Expanded(
            child: Container(
              decoration: BoxDecoration(
                color: viewerSurface,
                borderRadius: BorderRadius.circular(20),
              ),
              padding: const EdgeInsets.all(16),
              child: Center(child: _buildImageViewer(context, viewerSurface)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCompactLayout(
    BuildContext context,
    Color pageSurface,
    Color panelSurface,
    Color viewerSurface,
  ) {
    final paneTheme = ImageMetadataPaneTheme.of(context);

    return Container(
      color: pageSurface,
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            decoration: BoxDecoration(
              color: viewerSurface,
              borderRadius: BorderRadius.circular(20),
            ),
            padding: const EdgeInsets.all(16),
            child: _buildImageViewer(context, viewerSurface),
          ),
          if (widget._canNavigate) ...[
            const SizedBox(height: 8),
            _buildFilmstrip(context),
          ],
          const SizedBox(height: 16),
          Container(
            key: const ValueKey('image-detail-metadata-pane'),
            decoration: BoxDecoration(
              color: paneTheme.panelSurface,
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: paneTheme.borderColor),
            ),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(20),
              child: ImageMetadataPanel(
                imageId: _currentImage.id,
                metadataSection: _buildMetadataSection(context, paneTheme),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Bottom filmstrip for quick navigation between images.
  Widget _buildFilmstrip(BuildContext context) {
    final theme = Theme.of(context);
    final primaryColor = theme.colorScheme.primary;

    return Container(
      height: 100,
      color: theme.colorScheme.surfaceContainerHighest,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Text(
              '${_currentIndex + 1} / ${_images.length}',
              style: theme.textTheme.labelSmall,
            ),
          ),
          Expanded(
            child: ListView.builder(
              controller: _filmstripController,
              scrollDirection: Axis.horizontal,
              itemCount: _images.length,
              itemBuilder: (context, index) {
                final img = _images[index];
                final isSelected = index == _currentIndex;

                return GestureDetector(
                  onTap: () => _selectImage(index),
                  child: Container(
                    width: 68,
                    margin: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 4,
                    ),
                    decoration: BoxDecoration(
                      border: Border.all(
                        color: isSelected ? primaryColor : Colors.transparent,
                        width: isSelected ? 2.5 : 1,
                      ),
                      borderRadius: BorderRadius.circular(6),
                      boxShadow: isSelected
                          ? [
                              BoxShadow(
                                color: primaryColor.withValues(alpha: 0.3),
                                blurRadius: 4,
                              ),
                            ]
                          : [],
                    ),
                    clipBehavior: Clip.antiAlias,
                    child:
                        (context.watch<ConfigProvider>().resolveThumbnailUrl(
                                  img.thumbnailSmallUrl,
                                ) !=
                                null)
                        ? CachedNetworkImage(
                            imageUrl: context
                                .watch<ConfigProvider>()
                                .resolveThumbnailUrl(img.thumbnailSmallUrl)!,
                            fit: BoxFit.cover,
                            placeholder: (ctx, url) => Container(
                              color: theme.colorScheme.surfaceContainerLowest,
                              child: const Center(
                                child: SizedBox.square(
                                  dimension: 16,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                ),
                              ),
                            ),
                            errorWidget: (ctx, url, err) => Container(
                              color: theme.colorScheme.surfaceContainerLowest,
                              child: const Icon(Icons.error, size: 20),
                            ),
                          )
                        : Container(
                            color: theme.colorScheme.surfaceContainerLowest,
                            child: const Icon(Icons.image, size: 24),
                          ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildImageViewer([BuildContext? buildContext, Color? panelSurface]) {
    final resolvedContext = buildContext ?? context;
    final resolvedPanelSurface =
        panelSurface ??
        _opaqueColor(
          Theme.of(resolvedContext).colorScheme.surfaceContainerHighest,
        );
    final largeUrl = resolvedContext.watch<ConfigProvider>().resolveThumbnailUrl(
      _currentImage.thumbnailLargeUrl,
    );
    final originalPath = _currentImage.path;
    final displayUrl = (largeUrl != null && largeUrl.isNotEmpty)
        ? largeUrl
        : originalPath;

    if (displayUrl.isEmpty) {
      return Container(
        height: 300,
        color: resolvedPanelSurface,
        child: const Center(
          child: Icon(Icons.image, size: 64, color: Colors.grey),
        ),
      );
    }

    return GestureDetector(
      onTap: () {
        ImageLightbox.show(
          resolvedContext,
          imageUrl: displayUrl,
          heroTag: 'image-${_currentImage.id}',
        );
      },
      child: Container(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(resolvedContext).size.height * 0.75,
        ),
        decoration: BoxDecoration(
          color: resolvedPanelSurface,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Stack(
          alignment: Alignment.center,
          children: [
            Hero(
              tag: 'image-${_currentImage.id}',
              child: ExtendedImage.network(
                displayUrl,
                key: ValueKey('viewer-${_currentImage.id}'),
                fit: BoxFit.contain,
                mode: ExtendedImageMode.gesture,
                initGestureConfigHandler: (state) {
                  return GestureConfig(
                    minScale: 1.0,
                    maxScale: 3.0,
                    animationMaxScale: 3.5,
                    initialScale: 1.0,
                    inPageView: false,
                    initialAlignment: InitialAlignment.center,
                    cacheGesture: false,
                  );
                },
                onDoubleTap: (state) {
                  final pointerDownPosition = state.pointerDownPosition;
                  final begin = state.gestureDetails?.totalScale ?? 1.0;
                  final end = begin == 1.0 ? 2.0 : 1.0;

                  state.handleDoubleTap(
                    scale: end,
                    doubleTapPosition: pointerDownPosition,
                  );
                },
                loadStateChanged: (state) {
                  if (state.extendedImageLoadState == LoadState.loading) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  if (state.extendedImageLoadState == LoadState.failed) {
                    return Center(child: Icon(Icons.error, color: Colors.red));
                  }
                  return null;
                },
              ),
            ),
            // Tap hint overlay
            Positioned(
              bottom: 8,
              right: 8,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: Colors.black54,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.fullscreen, color: Colors.white, size: 14),
                    SizedBox(width: 4),
                    Text(
                      '点击全屏',
                      style: TextStyle(color: Colors.white, fontSize: 11),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMetadataSection(
    BuildContext context,
    ImageMetadataPaneTheme paneTheme,
  ) {
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      decoration: paneTheme.sectionDecoration,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: SelectionArea(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '元数据',
                style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: paneTheme.textForeground,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 12),
              _buildMetadataRow(
                context,
                '文件名',
                _currentImage.filename,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '尺寸',
                _currentImage.displaySize,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '格式',
                _currentImage.format.toUpperCase(),
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '大小',
                _currentImage.displayFileSize,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '导入时间',
                _currentImage.createdAt.toString(),
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildPathMetadataRow(
                context,
                _currentImage.path,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildPathMetadataRow(
    BuildContext context,
    String value,
    Color foreground,
    Color mutedForeground,
  ) {
    return Padding(
      key: const Key('metadata-path-row'),
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 70,
            child: Text(
              '路径',
              style: TextStyle(color: mutedForeground, fontSize: 13),
            ),
          ),
          Expanded(
            child: Tooltip(
              message: value,
              child: Text(
                value,
                key: const Key('metadata-path-value'),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: foreground,
                  fontWeight: FontWeight.w500,
                  fontSize: 13,
                ),
              ),
            ),
          ),
          IconButton(
            tooltip: '复制路径',
            onPressed: () => Clipboard.setData(ClipboardData(text: value)),
            icon: const Icon(Icons.copy_outlined, size: 16),
            visualDensity: VisualDensity.compact,
            constraints: const BoxConstraints(),
            padding: const EdgeInsets.only(left: 8),
          ),
        ],
      ),
    );
  }

  Widget _buildMetadataRow(
    BuildContext context,
    String label,
    String value,
    Color foreground,
    Color mutedForeground,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 70,
            child: Text(
              label,
              style: TextStyle(color: mutedForeground, fontSize: 13),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: TextStyle(
                color: foreground,
                fontWeight: FontWeight.w500,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Color _opaqueColor(Color color) {
    return color.withValues(alpha: 1);
  }
}
