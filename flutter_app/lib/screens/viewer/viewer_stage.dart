import 'dart:io';

import 'package:extended_image/extended_image.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:gallery/models/viewer_session.dart';

class ViewerStage extends StatefulWidget {
  final ViewerSessionItem item;
  final VoidCallback? onOpenContainingFolder;

  const ViewerStage({
    super.key,
    required this.item,
    this.onOpenContainingFolder,
  });

  @override
  State<ViewerStage> createState() => _ViewerStageState();
}

class _ViewerStageState extends State<ViewerStage> {
  bool _showContextMenu = false;
  Offset _contextMenuPosition = Offset.zero;

  void _onSecondaryTapDown(TapDownDetails details) {
    if (widget.onOpenContainingFolder == null) return;

    setState(() {
      _showContextMenu = true;
      _contextMenuPosition = details.globalPosition;
    });
  }

  @override
  Widget build(BuildContext context) {
    final largeUrl = widget.item.thumbnailLargeUrl;
    final originalPath = widget.item.path;
    final displayUrl = (largeUrl != null && largeUrl.isNotEmpty)
        ? largeUrl
        : originalPath;

    if (displayUrl.isEmpty) {
      return const Center(
        child: Icon(FluentIcons.photo2, size: 64, color: Colors.grey),
      );
    }

    return Stack(
      children: [
        GestureDetector(
          onSecondaryTapDown: _onSecondaryTapDown,
          child: ExtendedImage.network(
            displayUrl,
            key: ValueKey(widget.item.imageId),
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
                return const Center(child: ProgressRing());
              }
              if (state.extendedImageLoadState == LoadState.failed) {
                return Center(
                  child: Icon(FluentIcons.error, color: Colors.red),
                );
              }
              return null;
            },
          ),
        ),
        if (_showContextMenu)
          Positioned.fill(
            child: GestureDetector(
              onTap: () => setState(() => _showContextMenu = false),
              behavior: HitTestBehavior.opaque,
              child: Align(
                alignment: Alignment.topLeft,
                child: Padding(
                  padding: EdgeInsets.only(
                    left: _contextMenuPosition.dx,
                    top: _contextMenuPosition.dy,
                  ),
                  child: Container(
                    decoration: BoxDecoration(
                      color: FluentTheme.of(context).micaBackgroundColor,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                        color: FluentTheme.of(
                          context,
                        ).resources.cardStrokeColorDefault,
                      ),
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withValues(alpha: 0.15),
                          blurRadius: 12,
                          offset: const Offset(0, 4),
                        ),
                      ],
                    ),
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(minWidth: 240),
                      child: Padding(
                        padding: const EdgeInsets.all(4),
                        child: Button(
                          onPressed: () {
                            setState(() => _showContextMenu = false);
                            widget.onOpenContainingFolder?.call();
                          },
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(FluentIcons.folder, size: 16),
                              const SizedBox(width: 8),
                              Flexible(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    const Text('打开所在文件夹'),
                                    Text(
                                      _extractDirectory(widget.item.path),
                                      style: TextStyle(
                                        fontSize: 11,
                                        color: FluentTheme.of(
                                          context,
                                        ).resources.textFillColorSecondary,
                                      ),
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }

  String _extractDirectory(String filePath) {
    final separator = Platform.pathSeparator;
    final lastSep = filePath.lastIndexOf(separator);
    if (lastSep < 0) return filePath;
    // Show only the last two directory segments for brevity
    final dir = filePath.substring(0, lastSep);
    final parts = dir.split(separator);
    if (parts.length <= 2) return dir;
    return '...${separator}${parts.skip(parts.length - 2).join(separator)}';
  }
}

/// Opens the containing folder in the system file manager
Future<void> openContainingFolder(String filePath) async {
  if (filePath.isEmpty) return;

  try {
    if (Platform.isWindows) {
      // /select highlights the file in Explorer
      await Process.run('explorer', ['/select,', filePath]);
    } else if (Platform.isMacOS) {
      // -R reveals the file in Finder
      await Process.run('open', ['-R', filePath]);
    } else if (Platform.isLinux) {
      // Open the directory
      final dir = filePath.substring(
        0,
        filePath.lastIndexOf(Platform.pathSeparator) + 1,
      );
      await Process.run('xdg-open', [dir]);
    }
  } catch (e) {
    debugPrint('Failed to open containing folder: $e');
  }
}
