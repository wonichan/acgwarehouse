import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../../providers/monitoring_provider.dart';

class SidecarDiagnosticSection extends StatelessWidget {
  const SidecarDiagnosticSection({super.key});

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<MonitoringProvider>();
    final overview = provider.overview;
    final sidecar = overview?.sidecar;
    final state = sidecar?.state ?? 'not_started';
    final label = _stateLabel(state);
    final statusColor = _stateColor(state);
    final canRestart = state == 'degraded' || state == 'stopped';
    final pendingTasks =
        overview?.tasks['pending'] ?? overview?.tasks['ready'] ?? 0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Sidecar 诊断',
          style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 24),
        Container(
          width: double.infinity,
          constraints: const BoxConstraints(minHeight: 320),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: const Color(0xFFD9E2F2)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                key: Key('sidecar-status-band-$state'),
                height: 3,
                decoration: BoxDecoration(
                  color: statusColor,
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(8),
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                label,
                                style: TextStyle(
                                  fontSize: 28,
                                  fontWeight: FontWeight.w600,
                                  color: statusColor,
                                ),
                              ),
                              const SizedBox(height: 12),
                              Text(
                                '最后探测时间 ${_formatTimestamp(sidecar?.lastProbeAt)}',
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ],
                          ),
                        ),
                        Button(
                          onPressed: canRestart && !provider.isRestarting
                              ? () => _showRestartDialog(context, provider)
                              : null,
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              if (provider.isRestarting) ...[
                                const SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: ProgressRing(strokeWidth: 2),
                                ),
                                const SizedBox(width: 8),
                              ],
                              const Text('重启 Sidecar'),
                            ],
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 24),
                    Row(
                      children: [
                        Expanded(
                          child: _MetricCard(
                            label: '队列深度',
                            value: '${overview?.queue.queueSize ?? 0}',
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: _MetricCard(
                            label: '活跃 Worker',
                            value: '${overview?.queue.workerCount ?? 0}',
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: _MetricCard(
                            label: '待处理任务',
                            value: '$pendingTasks',
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 24),
                    if ((sidecar?.lastErrorSummary ?? '').isNotEmpty)
                      _ErrorSummaryCard(
                        timestamp: _formatTimestamp(sidecar?.lastProbeAt),
                        summary: sidecar!.lastErrorSummary!,
                      )
                    else
                      const Text(
                        '近期无错误记录',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Future<void> _showRestartDialog(
    BuildContext context,
    MonitoringProvider provider,
  ) async {
    final interruptedCount =
        provider.restartImpact?.interruptedTaskCount ??
        provider.overview?.tasks['running'] ??
        0;

    await showDialog<void>(
      context: context,
      builder: (dialogContext) {
        return ContentDialog(
          title: const Text('确认重启 Sidecar'),
          content: Text('重启将中断正在进行的 $interruptedCount 个计算任务。确定要继续吗？'),
          actions: [
            Button(
              onPressed: () => Navigator.pop(dialogContext),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () async {
                await provider.restartSidecar();
                if (dialogContext.mounted) {
                  Navigator.pop(dialogContext);
                }
              },
              child: const Text('确认重启'),
            ),
          ],
        );
      },
    );
  }
}

class _MetricCard extends StatelessWidget {
  const _MetricCard({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFFF5F7FB),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            value,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w400),
          ),
        ],
      ),
    );
  }
}

class _ErrorSummaryCard extends StatelessWidget {
  const _ErrorSummaryCard({required this.timestamp, required this.summary});

  final String timestamp;
  final String summary;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFFF5F7FB),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '最近错误',
            style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            timestamp,
            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            summary,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w400),
          ),
        ],
      ),
    );
  }
}

Color _stateColor(String state) {
  return switch (state) {
    'ready' => const Color(0xFF16A34A),
    'degraded' => const Color(0xFFD97706),
    'stopped' => const Color(0xFFC42B1C),
    'starting' => const Color(0xFF2563EB),
    'not_started' || 'not_configured' => const Color(0xFFD97706),
    _ => const Color(0xFF52637A),
  };
}

String _stateLabel(String state) {
  return switch (state) {
    'ready' => '就绪',
    'degraded' => '降级',
    'stopped' => '已停止',
    'starting' => '启动中',
    'not_started' || 'not_configured' => '未配置',
    _ => state,
  };
}

String _formatTimestamp(DateTime? value) {
  if (value == null) {
    return '--';
  }

  String twoDigits(int number) => number.toString().padLeft(2, '0');
  return '${value.year}-${twoDigits(value.month)}-${twoDigits(value.day)} ${twoDigits(value.hour)}:${twoDigits(value.minute)}';
}
