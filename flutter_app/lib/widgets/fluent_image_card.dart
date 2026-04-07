import 'package:fluent_ui/fluent_ui.dart';
import 'package:cached_network_image/cached_network_image.dart';
import '../models/image.dart';

typedef FluentImageTapCallback = void Function(ImageModel image);

/// Fluent 风格图片卡片
/// 简约设计：圆角矩形，悬停时显示阴影
class FluentImageCard extends StatefulWidget {
  final ImageModel image;
  final FluentImageTapCallback? onTap;
  final FluentImageTapCallback? onDoubleClick;
  final void Function(ImageModel image, TapDownDetails details)? onSecondaryTapDown;
  final double borderRadius;

  const FluentImageCard({
    super.key,
    required this.image,
    this.onTap,
    this.onDoubleClick,
    this.onSecondaryTapDown,
    this.borderRadius = 8.0,
  });

  @override
  State<FluentImageCard> createState() => _FluentImageCardState();
}

class _FluentImageCardState extends State<FluentImageCard> {
  bool _isHovered = false;

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final thumbnailUrl = widget.image.thumbnailSmallUrl;

    return MouseRegion(
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      child: GestureDetector(
        onTap: widget.onTap != null ? () => widget.onTap!(widget.image) : null,
        onDoubleTap: widget.onDoubleClick != null
            ? () => widget.onDoubleClick!(widget.image)
            : null,
        onSecondaryTapDown: widget.onSecondaryTapDown != null
            ? (details) => widget.onSecondaryTapDown!(widget.image, details)
            : null,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(widget.borderRadius),
            boxShadow: _isHovered
                ? [
                    BoxShadow(
                      color: theme.accentColor.withOpacity(0.3),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ]
                : [],
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(widget.borderRadius),
            child: _buildImage(thumbnailUrl, theme),
          ),
        ),
      ),
    );
  }

  Widget _buildImage(String? thumbnailUrl, FluentThemeData theme) {
    if (thumbnailUrl == null || thumbnailUrl.isEmpty) {
      return Container(
        color: theme.resources.cardBackgroundFillColorSecondary,
        child: Center(
          child: Icon(
            FluentIcons.photo2,
            size: 48,
            color: theme.resources.textFillColorSecondary,
          ),
        ),
      );
    }

    return CachedNetworkImage(
      imageUrl: thumbnailUrl,
      fit: BoxFit.cover,
      placeholder: (context, url) => Container(
        color: theme.resources.cardBackgroundFillColorSecondary,
        child: Center(
          child: ProgressRing(strokeWidth: 2, activeColor: theme.accentColor),
        ),
      ),
      errorWidget: (context, url, error) => Container(
        color: theme.resources.cardBackgroundFillColorSecondary,
        child: Center(
          child: Icon(
            FluentIcons.error,
            size: 48,
            color: theme.resources.systemFillColorCritical,
          ),
        ),
      ),
    );
  }
}
