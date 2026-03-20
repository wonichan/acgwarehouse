import 'package:fluent_ui/fluent_ui.dart';

/// Reusable CommandBar components for Fluent UI pages
class FluentCommandBars {
  /// Creates a standard view toggle button
  static CommandBarButton viewToggle({
    required bool isGridView,
    required VoidCallback onToggle,
  }) {
    return CommandBarButton(
      icon: Icon(isGridView ? FluentIcons.tiles : FluentIcons.bulleted_list_text),
      label: Text(isGridView ? '网格' : '列表'),
      onPressed: onToggle,
    );
  }

  /// Creates a refresh button
  static CommandBarButton refresh({
    required VoidCallback onRefresh,
  }) {
    return CommandBarButton(
      icon: const Icon(FluentIcons.refresh),
      label: const Text('刷新'),
      onPressed: onRefresh,
    );
  }

  /// Creates a filter button
  static CommandBarButton filter({
    required VoidCallback onFilter,
  }) {
    return CommandBarButton(
      icon: const Icon(FluentIcons.filter),
      label: const Text('筛选'),
      onPressed: onFilter,
    );
  }

  /// Creates a tag management button
  static CommandBarButton tagManagement({
    required VoidCallback onNavigate,
  }) {
    return CommandBarButton(
      icon: const Icon(FluentIcons.tag),
      label: const Text('标签管理'),
      onPressed: onNavigate,
    );
  }

  /// Creates a sort button
  static CommandBarButton sort({
    required VoidCallback onSort,
  }) {
    return CommandBarButton(
      icon: const Icon(FluentIcons.sort),
      label: const Text('排序'),
      onPressed: onSort,
    );
  }

  /// Creates a search button
  static CommandBarButton search({
    required VoidCallback onSearch,
  }) {
    return CommandBarButton(
      icon: const Icon(FluentIcons.search),
      label: const Text('搜索'),
      onPressed: onSearch,
    );
  }

  /// Creates a CommandBar separator
  static CommandBarItem separator() {
    return const CommandBarSeparator();
  }
}
