import 'dart:async';

import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../../providers/monitoring_provider.dart';
import 'batch_list_section.dart';
import 'sidecar_diagnostic_section.dart';

class OperationsMonitoringWorkspace extends StatefulWidget {
  const OperationsMonitoringWorkspace({super.key});

  @override
  State<OperationsMonitoringWorkspace> createState() =>
      _OperationsMonitoringWorkspaceState();
}

class _OperationsMonitoringWorkspaceState
    extends State<OperationsMonitoringWorkspace> {
  late final MonitoringProvider _provider;

  @override
  void initState() {
    super.initState();
    _provider = context.read<MonitoringProvider>();
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
    return Consumer<MonitoringProvider>(
      builder: (context, provider, child) {
        return ScaffoldPage(
          header: PageHeader(
            title: const Text('批次任务监控'),
            commandBar: CommandBar(
              mainAxisAlignment: MainAxisAlignment.end,
              primaryItems: [_buildConnectionStatusBadge(provider.wsConnected)],
            ),
          ),
          content: provider.serviceUnavailable
              ? _ServiceUnavailableState(onRetry: provider.retryLoad)
              : SingleChildScrollView(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 32,
                      vertical: 24,
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (!provider.wsConnected) ...[
                          _DisconnectBanner(onReconnect: provider.connect),
                          const SizedBox(height: 24),
                        ],
                        const BatchListSection(),
                        const SizedBox(height: 24),
                        const Divider(direction: Axis.horizontal),
                        const SizedBox(height: 24),
                        const SidecarDiagnosticSection(),
                      ],
                    ),
                  ),
                ),
        );
      },
    );
  }
}

CommandBarButton _buildConnectionStatusBadge(bool connected) {
  final color = connected ? const Color(0xFF16A34A) : const Color(0xFFC42B1C);
  return CommandBarButton(
    onPressed: null,
    icon: Icon(
      connected ? FluentIcons.plug_connected : FluentIcons.plug_disconnected,
    ),
    label: Text(
      connected ? '已连接' : '已断开',
      style: TextStyle(color: color, fontSize: 14, fontWeight: FontWeight.w600),
    ),
  );
}

class _DisconnectBanner extends StatelessWidget {
  const _DisconnectBanner({required this.onReconnect});

  final Future<void> Function() onReconnect;

  @override
  Widget build(BuildContext context) {
    return InfoBar(
      severity: InfoBarSeverity.warning,
      title: const Text('实时连接已断开，数据可能不是最新。'),
      action: Button(
        onPressed: () {
          unawaited(onReconnect());
        },
        child: const Text('重新连接'),
      ),
    );
  }
}

class _ServiceUnavailableState extends StatelessWidget {
  const _ServiceUnavailableState({required this.onRetry});

  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 48),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              '服务暂时不可用。检查后端连接后重试。',
              style: TextStyle(fontSize: 28, fontWeight: FontWeight.w600),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () {
                unawaited(onRetry());
              },
              child: const Text('重试'),
            ),
          ],
        ),
      ),
    );
  }
}
