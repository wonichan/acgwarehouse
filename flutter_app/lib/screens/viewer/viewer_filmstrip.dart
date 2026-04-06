import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:gallery/models/image.dart';

class ViewerFilmstrip extends StatefulWidget {
  final List<ImageModel> items;
  final int selectedIndexInWindow;
  final int selectedIndex;
  final int total;
  final ValueChanged<int> onIndexChanged;

  const ViewerFilmstrip({
    super.key,
    required this.items,
    required this.selectedIndexInWindow,
    required this.selectedIndex,
    required this.total,
    required this.onIndexChanged,
  });

  @override
  State<ViewerFilmstrip> createState() => _ViewerFilmstripState();
}

class _ViewerFilmstripState extends State<ViewerFilmstrip> {
  final ScrollController _scrollController = ScrollController();

  @override
  void didUpdateWidget(ViewerFilmstrip oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.selectedIndexInWindow != widget.selectedIndexInWindow) {
      _scrollToIndex(widget.selectedIndexInWindow);
    }
  }

  void _scrollToIndex(int index) {
    if (!_scrollController.hasClients) {
      return;
    }

    final offset =
        (index * 120.0) - (MediaQuery.of(context).size.width / 2) + 60;
    _scrollController.animateTo(
      offset.clamp(0.0, _scrollController.position.maxScrollExtent),
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeInOut,
    );
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 120,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Text(
              '${widget.selectedIndex + 1} of ${widget.total}',
              style: Theme.of(context).textTheme.labelSmall,
            ),
          ),
          Expanded(
            child: ListView.builder(
              controller: _scrollController,
              scrollDirection: Axis.horizontal,
              itemCount: widget.items.length,
              itemBuilder: (context, index) {
                final item = widget.items[index];
                final isSelected = index == widget.selectedIndexInWindow;

                return GestureDetector(
                  onTap: () => widget.onIndexChanged(index),
                  child: Container(
                    width: 100,
                    margin: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 8,
                    ),
                    decoration: BoxDecoration(
                      border: Border.all(
                        color: isSelected
                            ? Theme.of(context).colorScheme.primary
                            : Colors.transparent,
                        width: isSelected ? 3 : 1,
                      ),
                      borderRadius: BorderRadius.circular(8),
                      boxShadow: isSelected
                          ? [
                              BoxShadow(
                                color: Theme.of(
                                  context,
                                ).colorScheme.primary.withValues(alpha: 0.4),
                                blurRadius: 4,
                              ),
                            ]
                          : [],
                    ),
                    clipBehavior: Clip.antiAlias,
                    child: item.thumbnailSmallUrl != null
                        ? CachedNetworkImage(
                            imageUrl: item.thumbnailSmallUrl!,
                            fit: BoxFit.cover,
                            placeholder: (context, url) => const Center(
                              child: CircularProgressIndicator(),
                            ),
                            errorWidget: (context, url, error) =>
                                const Icon(Icons.error),
                          )
                        : const Icon(Icons.image),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
