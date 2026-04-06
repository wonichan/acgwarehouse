import 'dart:async';

import 'package:flutter/material.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/models/viewer_window_context.dart';
import 'package:gallery/models/viewer_window_result.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/screens/viewer/viewer_filmstrip.dart';
import 'package:gallery/screens/viewer/viewer_keyboard_scope.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/screens/viewer/viewer_stage.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:provider/provider.dart';

class ViewerWorkspace extends StatefulWidget {
  final ViewerWindowContext launchContext;
  final ApiService apiService;
  final ValueChanged<ViewerSessionItem>? onItemChanged;
  final VoidCallback? onEscape;
  final TagProvider? tagProvider;

  const ViewerWorkspace({
    super.key,
    required this.launchContext,
    required this.apiService,
    this.onItemChanged,
    this.onEscape,
    this.tagProvider,
  });

  @override
  State<ViewerWorkspace> createState() => _ViewerWorkspaceState();
}

class _ViewerWorkspaceState extends State<ViewerWorkspace> {
  late TagProvider _tagProvider;
  late bool _ownsTagProvider;
  ViewerWindowContext? _currentContext;
  ViewerWindowResult? _currentWindow;
  bool _isLoadingWindow = true;
  String? _windowError;

  @override
  void initState() {
    super.initState();
    _ownsTagProvider = widget.tagProvider == null;
    _tagProvider = widget.tagProvider ?? TagProvider(TagService());
    _currentContext = widget.launchContext;
    unawaited(_loadWindow(widget.launchContext));
  }

  @override
  void didUpdateWidget(covariant ViewerWorkspace oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (oldWidget.tagProvider != widget.tagProvider) {
      if (_ownsTagProvider) {
        _tagProvider.dispose();
      }
      _ownsTagProvider = widget.tagProvider == null;
      _tagProvider = widget.tagProvider ?? TagProvider(TagService());
      if (_currentItem != null) {
        unawaited(_tagProvider.loadImageTags(_currentItem!.imageId));
      }
    }

    if (oldWidget.launchContext != widget.launchContext ||
        oldWidget.apiService != widget.apiService) {
      _currentContext = widget.launchContext;
      _currentWindow = null;
      _windowError = null;
      _isLoadingWindow = true;
      unawaited(_loadWindow(widget.launchContext));
    }
  }

  @override
  void dispose() {
    if (_ownsTagProvider) {
      _tagProvider.dispose();
    }
    super.dispose();
  }

  ViewerSessionItem? get _currentItem {
    final window = _currentWindow;
    if (window == null || window.items.isEmpty) {
      return null;
    }
    return ViewerSessionItem.fromImage(
      window.items[window.selectedIndexInWindow],
    );
  }

  Future<void> _loadWindow(ViewerWindowContext context) async {
    setState(() {
      _isLoadingWindow = true;
      _windowError = null;
    });

    try {
      final window = await widget.apiService.fetchViewerWindow(
        ViewerWindowRequest(context: context),
      );
      final currentItem = window.items[window.selectedIndexInWindow];

      if (!mounted) {
        return;
      }

      setState(() {
        _currentContext = context;
        _currentWindow = window;
        _windowError = null;
        _isLoadingWindow = false;
      });

      await _tagProvider.loadImageTags(currentItem.id);
      widget.onItemChanged?.call(ViewerSessionItem.fromImage(currentItem));
    } on ViewerWindowApiException catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _isLoadingWindow = false;
        _windowError = error.message;
      });
    } catch (_) {
      if (!mounted) {
        return;
      }
      setState(() {
        _isLoadingWindow = false;
        _windowError = '无法加载查看器窗口';
      });
    }
  }

  void _selectItemInCurrentWindow(int indexInWindow) {
    final window = _currentWindow;
    final context = _currentContext;
    if (window == null || context == null) {
      return;
    }
    if (indexInWindow == window.selectedIndexInWindow) {
      return;
    }

    final item = window.items[indexInWindow];
    final selectedIndex = window.windowStartIndex + indexInWindow;
    final nextContext = _copyContext(
      context,
      selectedIndex: selectedIndex,
      selectedImageId: item.id,
    );

    setState(() {
      _currentContext = nextContext;
      _currentWindow = ViewerWindowResult(
        items: window.items,
        windowStartIndex: window.windowStartIndex,
        selectedIndex: selectedIndex,
        selectedIndexInWindow: indexInWindow,
        total: window.total,
        hasPrevious: selectedIndex > 0,
        hasNext: selectedIndex < window.total - 1,
      );
    });

    unawaited(_tagProvider.loadImageTags(item.id));
    widget.onItemChanged?.call(ViewerSessionItem.fromImage(item));
  }

  void _handleNext() {
    final window = _currentWindow;
    final currentItem = _currentItem;
    final context = _currentContext;
    if (_isLoadingWindow ||
        window == null ||
        currentItem == null ||
        context == null ||
        !window.hasNext) {
      return;
    }

    final nextIndexInWindow = window.selectedIndexInWindow + 1;
    final nextItem = nextIndexInWindow < window.items.length
        ? window.items[nextIndexInWindow]
        : null;
    final nextContext = _copyContext(
      context,
      selectedIndex: window.selectedIndex + 1,
      selectedImageId: nextItem?.id ?? currentItem.imageId,
    );
    unawaited(_loadWindow(nextContext));
  }

  void _handlePrevious() {
    final window = _currentWindow;
    final currentItem = _currentItem;
    final context = _currentContext;
    if (_isLoadingWindow ||
        window == null ||
        currentItem == null ||
        context == null ||
        !window.hasPrevious) {
      return;
    }

    final previousIndexInWindow = window.selectedIndexInWindow - 1;
    final previousItem = previousIndexInWindow >= 0
        ? window.items[previousIndexInWindow]
        : null;
    final previousContext = _copyContext(
      context,
      selectedIndex: window.selectedIndex - 1,
      selectedImageId: previousItem?.id ?? currentItem.imageId,
    );
    unawaited(_loadWindow(previousContext));
  }

  void _handleEscape() {
    widget.onEscape?.call();
  }

  @override
  Widget build(BuildContext context) {
    if (_currentWindow == null) {
      if (_isLoadingWindow) {
        return const Center(child: CircularProgressIndicator());
      }
      return _ViewerWindowErrorState(
        message: _windowError ?? '加载查看器窗口失败',
        onRetry: () => _loadWindow(_currentContext ?? widget.launchContext),
      );
    }

    final currentItem = _currentItem!;
    final canGoPrevious =
        !_isLoadingWindow && (_currentWindow?.hasPrevious ?? false);
    final canGoNext = !_isLoadingWindow && (_currentWindow?.hasNext ?? false);

    return ChangeNotifierProvider<TagProvider>.value(
      value: _tagProvider,
      child: Consumer<TagProvider>(
        builder: (context, tagProvider, child) {
          return ViewerKeyboardScope(
            onNext: _handleNext,
            onPrevious: _handlePrevious,
            onEscape: _handleEscape,
            child: Column(
              children: [
                Container(
                  height: 48,
                  color: Theme.of(context).colorScheme.surface,
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  child: Row(
                    children: [
                      Expanded(child: Text('Viewer - ${currentItem.filename}')),
                      Text(
                        '快捷键：←/→ 切换，Esc 关闭',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                      const SizedBox(width: 12),
                      TextButton(
                        onPressed: canGoPrevious ? _handlePrevious : null,
                        child: const Text('上一张'),
                      ),
                      const SizedBox(width: 8),
                      TextButton(
                        onPressed: canGoNext ? _handleNext : null,
                        child: const Text('下一张'),
                      ),
                    ],
                  ),
                ),
                if (_windowError != null)
                  Material(
                    color: Theme.of(context).colorScheme.errorContainer,
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 8,
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              '加载查看器窗口失败: $_windowError',
                              style: TextStyle(
                                color: Theme.of(
                                  context,
                                ).colorScheme.onErrorContainer,
                              ),
                            ),
                          ),
                          TextButton(
                            onPressed: () =>
                                setState(() => _windowError = null),
                            child: const Text('关闭'),
                          ),
                        ],
                      ),
                    ),
                  ),
                if (_isLoadingWindow)
                  const LinearProgressIndicator(minHeight: 2),
                Expanded(
                  child: Row(
                    children: [
                      Expanded(
                        child: Container(
                          color: Theme.of(context).colorScheme.surfaceContainer,
                          child: ViewerStage(item: currentItem),
                        ),
                      ),
                      ViewerMetadataSidebar(item: currentItem),
                    ],
                  ),
                ),
                ViewerFilmstrip(
                  items: _currentWindow!.items,
                  selectedIndexInWindow: _currentWindow!.selectedIndexInWindow,
                  selectedIndex: _currentWindow!.selectedIndex,
                  total: _currentWindow!.total,
                  onIndexChanged: _selectItemInCurrentWindow,
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  ViewerWindowContext _copyContext(
    ViewerWindowContext context, {
    required int selectedIndex,
    required int selectedImageId,
  }) {
    switch (context.source) {
      case ViewerWindowSource.search:
        return ViewerWindowContext.search(
          selectedIndex: selectedIndex,
          selectedImageId: selectedImageId,
          snapshot: context.snapshot as ViewerWindowSearchSnapshot,
        );
      case ViewerWindowSource.gallery:
        return ViewerWindowContext.gallery(
          selectedIndex: selectedIndex,
          selectedImageId: selectedImageId,
          snapshot: context.snapshot as ViewerWindowGallerySnapshot,
        );
    }
  }
}

class _ViewerWindowErrorState extends StatelessWidget {
  const _ViewerWindowErrorState({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text('加载查看器窗口失败', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          Text(message),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: const Text('重试')),
        ],
      ),
    );
  }
}
