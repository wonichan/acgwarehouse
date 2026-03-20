import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/duplicate_screen.dart';
import '../screens/tag_management_screen.dart';
import '../providers/image_provider.dart';
import '../providers/navigation_provider.dart';
import '../widgets/fluent_gallery_content.dart';
import '../models/image.dart';

/// Fluent 风格图库页面
/// 包含 CommandBar 工具栏和图库内容
class FluentGalleryPage extends StatelessWidget {
  const FluentGalleryPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<ImageListProvider>(
      builder: (context, provider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('图库'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                // View toggle
                CommandBarButton(
                  icon: Icon(
                    provider.viewMode == ViewMode.grid
                        ? FluentIcons.bulleted_list_text
                        : FluentIcons.tiles,
                  ),
                  label: Text(provider.viewMode == ViewMode.grid ? '网格' : '瀑布流'),
                  onPressed: () {
                    provider.setViewMode(
                      provider.viewMode == ViewMode.grid
                          ? ViewMode.masonry
                          : ViewMode.grid,
                    );
                  },
                ),
                const CommandBarSeparator(),
                // Refresh button
                CommandBarButton(
                  icon: const Icon(FluentIcons.refresh),
                  label: const Text('刷新'),
                  onPressed: () {
                    provider.loadImages(refresh: true);
                  },
                ),
                // Tag filter
                CommandBarButton(
                  icon: const Icon(FluentIcons.filter),
                  label: const Text('筛选'),
                  onPressed: () {
                    _showTagFilterDialog(context);
                  },
                ),
                // Tag management
                CommandBarButton(
                  icon: const Icon(FluentIcons.tag),
                  label: const Text('标签管理'),
                  onPressed: () {
                    context.read<NavigationProvider>().setSelectedIndex(3);
                  },
                ),
              ],
            ),
          ),
          content: FluentGalleryContent(
            onImageTap: (image) => _showImageDetail(context, image),
          ),
        );
      },
    );
  }

  void _showTagFilterDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: const Text('标签筛选'),
        content: const Text('标签筛选功能将在后续实现'),
        actions: [
          FilledButton(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }

  void _showImageDetail(BuildContext context, ImageModel image) {
    showDialog(
      context: context,
      builder: (context) => ContentDialog(
        title: Text(image.filename),
        content: Text('图片详情功能将在 08-03 中实现'),
        actions: [
          FilledButton(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }
}

/// Fluent 风格搜索页面容器
class FluentSearchPage extends StatelessWidget {
  const FluentSearchPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const SearchScreen();
  }
}

/// Fluent 风格重复检测页面容器
class FluentDuplicatePage extends StatelessWidget {
  const FluentDuplicatePage({super.key});

  @override
  Widget build(BuildContext context) {
    return const DuplicateScreen();
  }
}

/// Fluent 风格标签管理页面容器
/// 包装共享的 TagManagementScreen Widget
class FluentTagManagementPage extends StatelessWidget {
  const FluentTagManagementPage({super.key});

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: const PageHeader(
        title: Text('标签管理'),
      ),
      content: const TagManagementScreen(),
    );
  }
}
