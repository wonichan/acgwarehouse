import 'package:fluent_ui/fluent_ui.dart';
import '../screens/gallery_screen.dart';
import '../screens/search_screen.dart';
import '../screens/duplicate_screen.dart';
import '../screens/tag_management_screen.dart';

/// Fluent 风格图库页面容器
class FluentGalleryPage extends StatelessWidget {
  const FluentGalleryPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const GalleryScreen();
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
