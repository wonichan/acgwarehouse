import 'package:fluent_ui/fluent_ui.dart';
import '../../models/log_models.dart';

class LogTerminal extends StatefulWidget {
  const LogTerminal({
    super.key,
    required this.lines,
    required this.isPaused,
    this.onScrollToBottom,
  });

  final List<LogLine> lines;
  final bool isPaused;
  final VoidCallback? onScrollToBottom;

  @override
  State<LogTerminal> createState() => _LogTerminalState();
}

class _LogTerminalState extends State<LogTerminal> {
  final ScrollController _scrollController = ScrollController();
  bool _userScrolledUp = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void didUpdateWidget(LogTerminal oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!widget.isPaused &&
        !_userScrolledUp &&
        widget.lines.length > oldWidget.lines.length) {
      _scrollToBottom();
    }
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (!_scrollController.hasClients) return;

    // Check if user has scrolled up from the bottom
    final isAtBottom =
        _scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 10.0;

    if (isAtBottom && _userScrolledUp) {
      setState(() {
        _userScrolledUp = false;
      });
      widget.onScrollToBottom?.call();
    } else if (!isAtBottom && !_userScrolledUp) {
      setState(() {
        _userScrolledUp = true;
      });
    }
  }

  void _scrollToBottom() {
    if (!_scrollController.hasClients) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: Color(0xFF1E1E1E),
        borderRadius: BorderRadius.all(Radius.circular(4)),
      ),
      padding: const EdgeInsets.all(8),
      child: ListView.builder(
        controller: _scrollController,
        itemCount: widget.lines.length,
        itemBuilder: (context, index) => _buildLogLine(widget.lines[index]),
      ),
    );
  }

  Widget _buildLogLine(LogLine line) {
    final color = _severityColor(line.severity);
    final sourceTag = _sourceTag(line.source);
    final timestampText = line.isHistorical
        ? '--:--:--.---'
        : _formatTimestamp(line.timestamp);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 1),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            timestampText,
            style: const TextStyle(
              color: Color(0xFF6A9955),
              fontFamily: 'Consolas',
              fontSize: 12,
            ),
          ),
          const SizedBox(width: 8),
          if (sourceTag != null) ...[sourceTag, const SizedBox(width: 8)],
          Expanded(
            child: Text(
              line.text,
              style: TextStyle(
                color: color,
                fontFamily: 'Consolas',
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }

  String _formatTimestamp(DateTime timestamp) {
    final t = timestamp.toLocal();
    final h = t.hour.toString().padLeft(2, '0');
    final m = t.minute.toString().padLeft(2, '0');
    final s = t.second.toString().padLeft(2, '0');
    final ms = t.millisecond.toString().padLeft(3, '0');
    return '$h:$m:$s.$ms';
  }

  Color _severityColor(String severity) {
    switch (severity.toLowerCase()) {
      case 'error':
        return const Color(0xFFE06C75);
      case 'warning':
        return const Color(0xFFE5C07B);
      case 'status':
        return const Color(0xFF56B6C2);
      default:
        return const Color(0xFFABB2BF);
    }
  }

  Widget? _sourceTag(LogSource source) {
    final (label, color) = switch (source) {
      LogSource.go => ('GO', const Color(0xFF56B6C2)),
      LogSource.python => ('PY', const Color(0xFFE5C07B)),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.2),
        borderRadius: BorderRadius.circular(2),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontFamily: 'Consolas',
          fontSize: 10,
          fontWeight: FontWeight.bold,
        ),
      ),
    );
  }
}
