import 'dart:async';
import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import 'package:provider/provider.dart';
import '../models/image.dart';
import '../providers/config_provider.dart';
import '../providers/tag_provider.dart';
import '../services/tag_service.dart';
import '../widgets/image_lightbox.dart';
import '../widgets/image_metadata_panel.dart';
import '../widgets/image_metadata_pane_theme.dart';

class ImageDetailScreen extends StatefulWidget {
  final ImageModel image;

  const ImageDetailScreen({super.key, required this.image});

  @override
  State<ImageDetailScreen> createState() => _ImageDetailScreenState();
}

class _ImageDetailScreenState extends State<ImageDetailScreen> {
  late TagProvider _tagProvider;

  @override
  void initState() {
    super.initState();
    _tagProvider = TagProvider(
      TagService(baseUrl: context.read<ConfigProvider>().baseUrl),
    );
    _loadImageTags();
  }

  @override
  void dispose() {
    _tagProvider.dispose();
    super.dispose();
  }

  Future<void> _loadImageTags() async {
    await _tagProvider.loadImageTags(widget.image.id);
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
    // Windows Photos style: near-black immersive background for image viewer
    final viewerSurface = isDark ? const Color(0xFF0E0E10) : panelSurface;

    return ChangeNotifierProvider.value(
      value: _tagProvider,
      child: Scaffold(
        backgroundColor: pageSurface,
        appBar: AppBar(
          backgroundColor: pageSurface,
          surfaceTintColor: Colors.transparent,
          title: const Text('图片详情'),
        ),
        body: isDesktopLayout
            ? _buildDesktopLayout(
                context,
                pageSurface,
                panelSurface,
                viewerSurface,
              )
            : _buildCompactLayout(
                context,
                pageSurface,
                panelSurface,
                viewerSurface,
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
                  imageId: widget.image.id,
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
      child: SingleChildScrollView(
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
                  imageId: widget.image.id,
                  metadataSection: _buildMetadataSection(context, paneTheme),
                ),
              ),
            ),
          ],
        ),
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
    // 优先使用缩略图，如果缩略图为空则回退到原始图片路径
    final largeUrl = widget.image.thumbnailLargeUrl;
    final originalPath = widget.image.path;
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
          context,
          imageUrl: displayUrl,
          heroTag: 'image-${widget.image.id}',
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
              tag: 'image-${widget.image.id}',
              child: ExtendedImage.network(
                displayUrl,
                fit: BoxFit.contain,
                // Enable zoom functionality with gesture mode
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
                // Double-tap zoom support (toggle between 1x and 2x)
                onDoubleTap: (state) {
                  final pointerDownPosition = state.pointerDownPosition;
                  final begin = state.gestureDetails?.totalScale ?? 1.0;
                  final end = begin == 1.0 ? 2.0 : 1.0; // Toggle zoom

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
                    return const Center(
                      child: Icon(Icons.error, color: Colors.red),
                    );
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
                widget.image.filename,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '尺寸',
                widget.image.displaySize,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '格式',
                widget.image.format.toUpperCase(),
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '大小',
                widget.image.displayFileSize,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '路径',
                widget.image.path,
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
              _buildMetadataRow(
                context,
                '导入时间',
                widget.image.createdAt.toString(),
                paneTheme.textForeground,
                paneTheme.textMuted,
              ),
            ],
          ),
        ),
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
