import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/image_provider.dart';
import '../providers/tag_provider.dart';

class GalleryFilterPanel extends StatefulWidget {
  const GalleryFilterPanel({super.key, this.width = 320});

  final double width;

  @override
  State<GalleryFilterPanel> createState() => _GalleryFilterPanelState();
}

class _GalleryFilterPanelState extends State<GalleryFilterPanel> {
  bool _loaded = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_loaded) {
      return;
    }
    _loaded = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      context.read<TagProvider>().loadTags();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: widget.width,
      color: FluentTheme.of(context).resources.cardBackgroundFillColorSecondary,
      padding: const EdgeInsets.all(16),
      child: Consumer2<ImageListProvider, TagProvider>(
        builder: (context, imageProvider, tagProvider, child) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Filter by Tags',
                style: FluentTheme.of(context).typography.subtitle,
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Expanded(child: Text('Show untagged images only')),
                  ToggleSwitch(
                    checked: imageProvider.hasTagsFilter == false,
                    onChanged: (checked) async {
                      if (checked) {
                        tagProvider.clearSelection();
                        await imageProvider.setHasTagsFilter(false);
                        return;
                      }
                      await imageProvider.setHasTagsFilter(null);
                    },
                  ),
                ],
              ),
              const SizedBox(height: 16),
              Expanded(
                child: _buildTagList(context, imageProvider, tagProvider),
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildTagList(
    BuildContext context,
    ImageListProvider imageProvider,
    TagProvider tagProvider,
  ) {
    if (tagProvider.isLoading) {
      return const Center(child: ProgressRing());
    }

    if (tagProvider.allTags.isEmpty) {
      return Text(
        'No tags available',
        style: FluentTheme.of(context).typography.body,
      );
    }

    return ListView.separated(
      itemCount: tagProvider.allTags.length,
      separatorBuilder: (_, __) => const SizedBox(height: 4),
      itemBuilder: (context, index) {
        final tag = tagProvider.allTags[index];
        final selected = tagProvider.selectedTagIds.contains(tag.id);

        return Checkbox(
          content: Text(tag.preferredLabel),
          checked: selected,
          onChanged: (value) async {
            tagProvider.toggleTag(tag.id);
            await imageProvider.setTagFilter(
              tagProvider.selectedTagIds.toList(),
            );
          },
        );
      },
    );
  }
}
