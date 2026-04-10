import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/search_provider.dart';
import '../widgets/image_grid.dart';
import '../models/image.dart';

/// Screen for searching images
class SearchScreen extends StatefulWidget {
  const SearchScreen({super.key});

  @override
  State<SearchScreen> createState() => _SearchScreenState();
}

class _SearchScreenState extends State<SearchScreen> {
  final TextEditingController _searchController = TextEditingController();
  final FocusNode _searchFocusNode = FocusNode();
  bool _showHistory = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SearchProvider>().loadSearchHistory();
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    _searchFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('搜索')),
      body: Column(
        children: [
          // Search bar
          _buildSearchBar(context),

          // Sort options
          _buildSortOptions(context),

          // Results or history
          Expanded(
            child: Consumer<SearchProvider>(
              builder: (context, provider, child) {
                if (provider.isLoading && provider.results.isEmpty) {
                  return const Center(child: CircularProgressIndicator());
                }

                if (provider.error != null) {
                  return Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          Icons.error_outline,
                          size: 48,
                          color: Theme.of(context).colorScheme.error,
                        ),
                        const SizedBox(height: 16),
                        Text('错误: ${provider.error}'),
                        const SizedBox(height: 16),
                        ElevatedButton(
                          onPressed: () {
                            provider.clearError();
                            _performSearch(provider);
                          },
                          child: const Text('重试'),
                        ),
                      ],
                    ),
                  );
                }

                // Show history if no search yet
                if (provider.results.isEmpty &&
                    provider.searchHistory.isNotEmpty &&
                    _showHistory) {
                  return _buildSearchHistory(context, provider);
                }

                // Show empty state
                if (provider.results.isEmpty) {
                  return Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          Icons.search,
                          size: 48,
                          color: Theme.of(context).colorScheme.outline,
                        ),
                        const SizedBox(height: 16),
                        const Text('没有找到匹配的图片'),
                        const SizedBox(height: 16),
                        ElevatedButton(
                          onPressed: () => provider.clearSearch(),
                          child: const Text('清除搜索'),
                        ),
                      ],
                    ),
                  );
                }

                // Show results
                return Column(
                  children: [
                    // Results count
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 8,
                      ),
                      child: Row(
                        children: [
                          Text('找到 ${provider.totalResults} 张图片'),
                          const Spacer(),
                          if (provider.currentQuery.isNotEmpty)
                            Chip(
                              label: Text(provider.currentQuery),
                              onDeleted: () {
                                _searchController.clear();
                                provider.clearSearch();
                                setState(() => _showHistory = true);
                              },
                            ),
                        ],
                      ),
                    ),
                    // Results grid
                    Expanded(
                      child: RefreshIndicator(
                        onRefresh: () => provider.search(refresh: true),
                        child: NotificationListener<ScrollNotification>(
                          onNotification: (notification) {
                            if (notification is ScrollEndNotification &&
                                notification.metrics.pixels >=
                                    notification.metrics.maxScrollExtent -
                                        200) {
                              provider.loadMore();
                            }
                            return false;
                          },
                          child: ImageGrid(
                            images: provider.results,
                            onImageTap: (image) =>
                                _showImageDetail(context, image),
                          ),
                        ),
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSearchBar(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: TextField(
        controller: _searchController,
        focusNode: _searchFocusNode,
        decoration: InputDecoration(
          hintText: '搜索图片...',
          prefixIcon: const Icon(Icons.search),
          suffixIcon: _searchController.text.isNotEmpty
              ? IconButton(
                  icon: const Icon(Icons.clear),
                  onPressed: () {
                    _searchController.clear();
                    context.read<SearchProvider>().clearSearch();
                    setState(() => _showHistory = true);
                  },
                )
              : null,
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
          filled: true,
        ),
        onSubmitted: (_) {
          _performSearch(context.read<SearchProvider>());
        },
        onChanged: (_) {
          setState(() {});
        },
      ),
    );
  }

  Widget _buildSortOptions(BuildContext context) {
    return Consumer<SearchProvider>(
      builder: (context, provider, child) {
        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Row(
            children: [
              const Text('排序: '),
              const SizedBox(width: 8),
              DropdownButton<String>(
                value: provider.sortBy,
                items: const [
                  DropdownMenuItem(value: 'relevance', child: Text('相关度')),
                  DropdownMenuItem(value: 'created_at', child: Text('时间')),
                  DropdownMenuItem(value: 'filename', child: Text('文件名')),
                  DropdownMenuItem(value: 'file_size', child: Text('大小')),
                ],
                onChanged: (value) {
                  if (value != null) {
                    provider.setSort(value, provider.sortOrder);
                  }
                },
              ),
              const SizedBox(width: 16),
              IconButton(
                icon: Icon(
                  provider.sortOrder == 'desc'
                      ? Icons.arrow_downward
                      : Icons.arrow_upward,
                ),
                onPressed: () {
                  provider.setSort(
                    provider.sortBy,
                    provider.sortOrder == 'desc' ? 'asc' : 'desc',
                  );
                },
                tooltip: provider.sortOrder == 'desc' ? '降序' : '升序',
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildSearchHistory(BuildContext context, SearchProvider provider) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: provider.searchHistory.length,
      itemBuilder: (context, index) {
        final query = provider.searchHistory[index];
        return ListTile(
          leading: const Icon(Icons.history),
          title: Text(query),
          trailing: IconButton(
            icon: const Icon(Icons.north_west),
            onPressed: () {
              _searchController.text = query;
              _performSearch(provider, query: query);
            },
          ),
          onTap: () {
            _searchController.text = query;
            _performSearch(provider, query: query);
          },
        );
      },
    );
  }

  void _performSearch(SearchProvider provider, {String? query}) {
    final searchText = query ?? _searchController.text.trim();
    if (searchText.isEmpty) return;

    setState(() => _showHistory = false);
    provider.search(query: searchText, refresh: true);
  }

  void _showImageDetail(BuildContext context, ImageModel image) {
    // Navigate to image detail screen
    // Navigator.push(
    //   context,
    //   MaterialPageRoute(
    //     builder: (context) => ImageDetailScreen(image: image),
    //   ),
    // );
  }
}
