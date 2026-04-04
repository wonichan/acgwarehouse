import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import 'package:window_manager/window_manager.dart';

import '../providers/navigation_provider.dart';
import '../providers/search_provider.dart';
import '../widgets/fluent_settings_page.dart';
import 'fluent_screens.dart';

class FluentAppShell extends StatelessWidget {
  final VoidCallback? onImportLibrary;

  const FluentAppShell({super.key, this.onImportLibrary});

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
                  child: const FluentGalleryPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.copy),
                title: const Text('重复检测'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  child: const FluentDuplicatePage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.search),
                title: const Text('搜索'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  child: const FluentSearchPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.tag),
                title: const Text('标签管理'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  child: const FluentTagManagementPage(),
                ),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.settings),
                title: const Text('设置'),
                body: _ShellPage(
                  onImportLibrary: onImportLibrary,
                  child: const FluentSettingsPage(),
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

  const _ShellPage({required this.child, this.onImportLibrary});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        _DesktopShellTopBar(onImportLibrary: onImportLibrary),
        Expanded(child: child),
      ],
    );
  }
}

class _DesktopShellTopBar extends StatefulWidget {
  final VoidCallback? onImportLibrary;

  const _DesktopShellTopBar({this.onImportLibrary});

  @override
  State<_DesktopShellTopBar> createState() => _DesktopShellTopBarState();
}

class _DesktopShellTopBarState extends State<_DesktopShellTopBar> {
  final TextEditingController _searchController = TextEditingController();

  @override
  void dispose() {
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

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
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
            onPressed: widget.onImportLibrary,
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
    );
  }
}
