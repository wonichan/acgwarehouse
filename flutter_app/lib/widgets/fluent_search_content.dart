import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/search_provider.dart';
import '../models/image.dart';
import 'fluent_image_card.dart';

/// Fluent-styled search content widget
/// Shows search results in grid layout with infinite scroll
class FluentSearchContent extends StatefulWidget {
  final void Function(ImageModel)? onImageTap;
  final ScrollController? scrollController;

  const FluentSearchContent({
    super.key,
    this.onImageTap,
    this.scrollController,
  });

  @override
  State<FluentSearchContent> createState() => _FluentSearchContentState();
}

class _FluentSearchContentState extends State<FluentSearchContent> {
  @override
  Widget build(BuildContext context) {
    return Consumer<SearchProvider>(
      builder: (context, provider, child) {
        // Loading state (initial)
        if (provider.isLoading && provider.results.isEmpty) {
          return const Center(child: ProgressRing());
        }

        // Error state
        if (provider.error != null) {
          return _buildErrorState(context, provider);
        }

        // No results state
        if (provider.results.isEmpty && provider.currentQuery.isNotEmpty) {
          return _buildNoResultsState(context, provider);
        }

        // Results
        if (provider.results.isNotEmpty) {
          return _buildResultsGrid(context, provider);
        }

        // Initial state (no search yet)
        return _buildInitialState(context);
      },
    );
  }

  Widget _buildErrorState(BuildContext context, SearchProvider provider) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            FluentIcons.error,
            size: 64,
            color: FluentTheme.of(context).resources.systemFillColorCritical,
          ),
          const SizedBox(height: 16),
          Text('错误: ${provider.error}'),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: () {
              provider.clearError();
              provider.search();
            },
            child: const Text('重试'),
          ),
        ],
      ),
    );
  }

  Widget _buildNoResultsState(BuildContext context, SearchProvider provider) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(FluentIcons.search, size: 64),
          const SizedBox(height: 16),
          const Text('没有找到匹配的图片'),
          const SizedBox(height: 8),
          Text('搜索: "${provider.currentQuery}"'),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: () => provider.clearSearch(),
            child: const Text('清除搜索'),
          ),
        ],
      ),
    );
  }

  Widget _buildResultsGrid(BuildContext context, SearchProvider provider) {
    return Column(
      children: [
        // Results header
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(
            children: [
              Text('找到 ${provider.totalResults} 张图片'),
              const Spacer(),
              if (provider.currentQuery.isNotEmpty)
                Button(
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(provider.currentQuery),
                      const SizedBox(width: 8),
                      const Icon(FluentIcons.clear, size: 12),
                    ],
                  ),
                  onPressed: () => provider.clearSearch(),
                ),
            ],
          ),
        ),
        // Results grid
        Expanded(
          child: NotificationListener<ScrollNotification>(
            onNotification: (notification) {
              if (notification is ScrollEndNotification &&
                  notification.metrics.pixels >=
                      notification.metrics.maxScrollExtent - 200) {
                provider.loadMore();
              }
              return false;
            },
            child: GridView.builder(
              controller: widget.scrollController,
              padding: const EdgeInsets.all(8),
              gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                maxCrossAxisExtent: 200,
                mainAxisSpacing: 8,
                crossAxisSpacing: 8,
              ),
              itemCount: provider.results.length,
              itemBuilder: (context, index) {
                return FluentImageCard(
                  image: provider.results[index],
                  onTap: widget.onImageTap,
                );
              },
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildInitialState(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(FluentIcons.search, size: 64),
          const SizedBox(height: 16),
          Text('输入关键词搜索图片', style: FluentTheme.of(context).typography.subtitle),
        ],
      ),
    );
  }
}
