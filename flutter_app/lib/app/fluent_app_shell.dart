import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/navigation_provider.dart';
import 'fluent_screens.dart';
import '../widgets/fluent_settings_page.dart';

/// FluentApp Shell - Windows 桌面端导航框架
/// 包含 NavigationView 侧边导航栏和页面容器
class FluentAppShell extends StatelessWidget {
  const FluentAppShell({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationView(
          titleBar: TitleBar(
            title: Text('ACGWarehouse - ${navProvider.currentPageTitle}'),
          ),
          pane: NavigationPane(
            selected: navProvider.selectedIndex,
            onChanged: (index) {
              navProvider.setSelectedIndex(index);
            },
            displayMode: PaneDisplayMode.auto,
            items: [
              PaneItem(
                icon: const Icon(FluentIcons.photo2),
                title: const Text('图库'),
                body: const FluentGalleryPage(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.copy),
                title: const Text('重复检测'),
                body: const FluentDuplicatePage(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.search),
                title: const Text('搜索'),
                body: const FluentSearchPage(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.tag),
                title: const Text('标签管理'),
                body: const FluentTagManagementPage(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.settings),
                title: const Text('设置'),
                body: const FluentSettingsPage(),
              ),
            ],
          ),
        );
      },
    );
  }
}
