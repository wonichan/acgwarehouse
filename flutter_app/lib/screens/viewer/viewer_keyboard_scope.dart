import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class ViewerKeyboardScope extends StatefulWidget {
  final Widget child;
  final VoidCallback onNext;
  final VoidCallback onPrevious;
  final VoidCallback onEscape;

  const ViewerKeyboardScope({
    super.key,
    required this.child,
    required this.onNext,
    required this.onPrevious,
    required this.onEscape,
  });

  @override
  State<ViewerKeyboardScope> createState() => _ViewerKeyboardScopeState();
}

class _ViewerKeyboardScopeState extends State<ViewerKeyboardScope> {
  final FocusNode _focusNode = FocusNode();

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Focus(
      autofocus: true,
      focusNode: _focusNode,
      onKeyEvent: (node, event) {
        if (event is KeyDownEvent) {
          if (event.logicalKey == LogicalKeyboardKey.arrowRight) {
            widget.onNext();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.arrowLeft) {
            widget.onPrevious();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.escape) {
            widget.onEscape();
            return KeyEventResult.handled;
          }
        }
        return KeyEventResult.ignored;
      },
      child: widget.child,
    );
  }
}
