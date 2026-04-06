import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import 'package:window_manager/window_manager.dart';

import '../providers/navigation_provider.dart';
import '../providers/search_provider.dart';
import '../services/import_service.dart';
import '../widgets/fluent_settings_page.dart';
import 'fluent_screens.dart';

class FluentAppShell extends StatelessWidget {
  final VoidCallback? onImportLibrary;
  final ImportService? importService;

  const FluentAppShell({super.key, this.onImportLibrary, this.importService});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationView(
          titleBar: TitleBar(
            title: DragToMoveArea(
              child: Align(
                alignment: AlignmentDirectional.centerStart,
                child: Text('ACGWarehouse - ${navProvider.currentPageTitle}'),
              ),
            ),
          ),
          pane: NavigationPane(
            selected: navProvider.selectedIndex,
            onChanged: navProvider.setSelectedIndex,
            displayMode: PaneDisplayMode.auto,
            items: [
              PaneItem(
                icon: const Icon(FluentIcons.photo2),
                title: const Text('图库'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentGalleryPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.copy),
                title: const Text('重复检测'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentDuplicatePage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.search),
                title: const Text('搜索'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentSearchPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.tag),
                title: const Text('标签管理'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentTagManagementPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.settings),
                title: const Text('设置'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentSettingsPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.diagnostic),
                title: const Text('运营监控'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentOperationsMonitoringPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.command_prompt),
                title: const Text('日志终端'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  importService: importService,
                  child: const FluentLogViewerPage(),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ShellPage extends StatelessWidget {
  final Widget child;
  final VoidCallback? onImportLibrary;
  final ImportService? importService;

  const _ShellPage({
    required this.child,
    this.onImportLibrary,
    this.importService,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        _DesktopShellTopBar(
          onImportLibrary: onImportLibrary,
          importService: importService,
        ),
        Expanded(child: child),
      ],
    );
  }
}

class _DesktopShellTopBar extends StatefulWidget {
  final VoidCallback? onImportLibrary;
  final ImportService? importService;

  const _DesktopShellTopBar({this.onImportLibrary, this.importService});

  @override
  State<_DesktopShellTopBar> createState() => _DesktopShellTopBarState();
}

class _DesktopShellTopBarState extends State<_DesktopShellTopBar> {
  final TextEditingController _searchController = TextEditingController();
  late final ImportService _importService;
  String? _importFeedback;

  @override
  void initState() {
    super.initState();
    _importService = widget.importService ?? ImportService();
  }

  @override
  void dispose() {
    if (widget.importService == null) {
      _importService.dispose();
    }
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _submitSearch() async {
    final query = _searchController.text.trim();
    if (query.isEmpty) return;

    await context.read<SearchProvider>().search(query: query);
    if (!mounted) return;
    context.read<NavigationProvider>().setSelectedIndex(
      NavigationProvider.searchIndex,
    );
  }

  Future<void> _triggerImport() async {
    widget.onImportLibrary?.call();

    try {
      await _importService.triggerImport();
      if (!mounted) return;
      setState(() {
        _importFeedback = 'Library import queued';
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _importFeedback = 'Library import could not start';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              SizedBox(
                width: 260,
                child: TextBox(
                  controller: _searchController,
                  placeholder: 'Search images and tags',
                  onSubmitted: (_) {
                    _submitSearch();
                  },
                ),
              ),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: _triggerImport,
                child: const Text('Import Library'),
              ),
              const SizedBox(width: 8),
              Button(
                onPressed: () {
                  context.read<NavigationProvider>().setSelectedIndex(
                    NavigationProvider.settingsIndex,
                  );
                },
                child: const Text('Open Settings'),
              ),
            ],
          ),
          if (_importFeedback != null) ...[
            const SizedBox(height: 8),
            InfoBar(
              severity: _importFeedback == 'Library import queued'
                  ? InfoBarSeverity.success
                  : InfoBarSeverity.error,
              title: Text(_importFeedback!),
            ),
          ],
        ],
      ),
    );
  }
}
