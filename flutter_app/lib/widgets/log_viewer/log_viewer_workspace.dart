import 'dart:async';

import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';
import '../../providers/log_viewer_provider.dart';
import '../../models/log_models.dart';
import 'log_terminal.dart';

class LogViewerWorkspace extends StatefulWidget {
  const LogViewerWorkspace({super.key});

  @override
  State<LogViewerWorkspace> createState() => _LogViewerWorkspaceState();
}

class _LogViewerWorkspaceState extends State<LogViewerWorkspace> {
  late final LogViewerProvider _provider;

  @override
  void initState() {
    super.initState();
    _provider = context.read<LogViewerProvider>();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        unawaited(_provider.connect());
      }
    });
  }

  @override
  void dispose() {
    unawaited(_provider.disconnect());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<LogViewerProvider>(
      builder: (context, provider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('日志终端'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [
                // Connection status badge
                _buildConnectionBadge(provider.wsConnected),
                const CommandBarSeparator(),
                // Pause/resume
                CommandBarButton(
                  icon: Icon(
                    provider.isPaused ? FluentIcons.play : FluentIcons.pause,
                  ),
                  label: Text(provider.isPaused ? '继续滚动' : '暂停滚动'),
                  onPressed: () => provider.togglePause(),
                ),
                // Clear
                CommandBarButton(
                  icon: const Icon(FluentIcons.clear),
                  label: const Text('清除'),
                  onPressed: provider.lines.isNotEmpty
                      ? () => provider.clearLines()
                      : null,
                ),
              ],
            ),
          ),
          content: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Source tabs (Go / Python)
              _buildSourceTabs(provider),
              // Paused indicator
              if (provider.isPaused) _buildPausedBanner(),
              // Terminal content
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 32,
                    vertical: 16,
                  ),
                  child: LogTerminal(
                    lines: provider.lines,
                    isPaused: provider.isPaused,
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  CommandBarButton _buildConnectionBadge(bool connected) {
    final color = connected ? const Color(0xFF16A34A) : const Color(0xFFC42B1C);
    return CommandBarButton(
      onPressed: null,
      icon: Icon(
        connected ? FluentIcons.plug_connected : FluentIcons.plug_disconnected,
      ),
      label: Text(
        connected ? '已连接' : '已断开',
        style: TextStyle(
          color: color,
          fontSize: 14,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Widget _buildSourceTabs(LogViewerProvider provider) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Row(
        children: [
          _buildTab(label: 'Go (Core)', isSelected: true, onPressed: () {}),
        ],
      ),
    );
  }

  Widget _buildTab({
    required String label,
    required bool isSelected,
    required VoidCallback onPressed,
  }) {
    return Button(
      onPressed: onPressed,
      style: ButtonStyle(
        backgroundColor: WidgetStateProperty.resolveWith((states) {
          if (isSelected) {
            return FluentTheme.of(context).accentColor.withValues(alpha: 0.1);
          }
          return Colors.transparent;
        }),
        shape: WidgetStateProperty.all(
          const RoundedRectangleBorder(side: BorderSide.none),
        ),
      ),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(
              color: isSelected
                  ? FluentTheme.of(context).accentColor
                  : Colors.transparent,
              width: 2,
            ),
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
            color: isSelected ? FluentTheme.of(context).accentColor : null,
          ),
        ),
      ),
    );
  }

  Widget _buildPausedBanner() {
    return const Padding(
      padding: EdgeInsets.symmetric(horizontal: 32, vertical: 8),
      child: InfoBar(title: Text('自动滚动已暂停'), severity: InfoBarSeverity.warning),
    );
  }
}
