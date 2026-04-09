import 'package:fluent_ui/fluent_ui.dart';
import '../models/image.dart';
import 'cached_image_widget.dart';

typedef FluentImageTapCallback = void Function(ImageModel image);

/// Fluent 风格图片卡片
/// 简约设计：圆角矩形，悬停时显示阴影。
/// 支持选择模式：选中时显示覆盖层和勾选标记。
class FluentImageCard extends StatefulWidget {
  final ImageModel image;
  final FluentImageTapCallback? onTap;
  final void Function(ImageModel image, TapDownDetails details)?
  onSecondaryTapDown;
  final double borderRadius;

  /// Selection mode support
  final bool isSelected;
  final void Function(ImageModel image, bool selected)? onSelect;

  const FluentImageCard({
    super.key,
    required this.image,
    this.onTap,
    this.onSecondaryTapDown,
    this.borderRadius = 8.0,
    this.isSelected = false,
    this.onSelect,
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
        onTap: widget.onSelect != null
            ? () {
                widget.onSelect!(widget.image, !widget.isSelected);
              }
            : widget.onTap != null
            ? () => widget.onTap!(widget.image)
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
                      color: theme.accentColor.withValues(alpha: 0.3),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ]
                : [],
          ),
          child: Stack(
            children: [
              ClipRRect(
                borderRadius: BorderRadius.circular(widget.borderRadius),
                child: _buildImage(thumbnailUrl, theme),
              ),
              // Selection overlay
              if (widget.isSelected)
                Positioned.fill(
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(widget.borderRadius),
                    child: Container(
                      decoration: BoxDecoration(
                        color: theme.accentColor.withValues(alpha: 0.35),
                        borderRadius: BorderRadius.circular(
                          widget.borderRadius,
                        ),
                        border: Border.all(color: theme.accentColor, width: 2),
                      ),
                      child: Align(
                        alignment: Alignment.topRight,
                        child: Padding(
                          padding: const EdgeInsets.all(4),
                          child: Container(
                            padding: const EdgeInsets.all(2),
                            decoration: const BoxDecoration(
                              color: Colors.white,
                              shape: BoxShape.circle,
                            ),
                            child: Icon(
                              FluentIcons.check_mark,
                              size: 12,
                              color: theme.accentColor,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
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

    return CachedImageWidget(
      imageUrl: thumbnailUrl,
      fit: BoxFit.cover,
      placeholder: Container(
        color: theme.resources.cardBackgroundFillColorSecondary,
        child: Center(
          child: ProgressRing(strokeWidth: 2, activeColor: theme.accentColor),
        ),
      ),
      errorBuilder: Container(
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
