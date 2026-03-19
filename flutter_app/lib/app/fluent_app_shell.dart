import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/navigation_provider.dart';
import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/duplicate_screen.dart';

/// FluentApp Shell - Windows 桌面端导航框架
/// 包含 NavigationView 侧边导航栏和页面容器
class FluentAppShell extends StatelessWidget {
  const FluentAppShell({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, navProvider, child) {
        return NavigationView(
          titleBar: const TitleBar(
            title: Text('ACGWarehouse'),
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
                body: const GalleryScreen(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.search),
                title: const Text('搜索'),
                body: const SearchScreen(),
              ),
              PaneItem(
                icon: const Icon(FluentIcons.copy),
                title: const Text('重复检测'),
                body: const DuplicateScreen(),
              ),
            ],
          ),
        );
      },
    );
  }
}