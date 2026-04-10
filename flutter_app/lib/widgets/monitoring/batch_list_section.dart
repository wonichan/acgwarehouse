import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../../models/monitoring_models.dart';
import '../../providers/monitoring_provider.dart';
import '../../theme/monitoring_theme.dart';

class BatchListSection extends StatelessWidget {
  const BatchListSection({super.key});

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<MonitoringProvider>();
    final batches = [...provider.batches]
      ..sort((left, right) {
        final leftTime =
            left.createdAt ?? DateTime.fromMillisecondsSinceEpoch(0);
        final rightTime =
            right.createdAt ?? DateTime.fromMillisecondsSinceEpoch(0);
        return rightTime.compareTo(leftTime);
      });

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          '批次任务监控',
          style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 24),
        if (batches.isEmpty)
          _BatchEmptyState()
        else
          Column(
            children: batches
                .map(
                  (batch) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: _BatchRowTile(batch: batch),
                  ),
                )
                .toList(),
          ),
      ],
    );
  }
}

class _BatchRowTile extends StatelessWidget {
  const _BatchRowTile({required this.batch});

  final BatchRow batch;

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<MonitoringProvider>();
    final isSelected = provider.selectedBatchId == batch.id;
    final failedCount = batch.statusCounts['failed'] ?? 0;
    final theme = MonitoringTheme.of(context);

    return Container(
      decoration: BoxDecoration(
        color: isSelected ? theme.selectedBackground : theme.cardBackground,
        border: Border.all(
          color: isSelected ? theme.selectedBorder : theme.cardBorder,
        ),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Button(
              onPressed: () {
                provider.selectBatch(isSelected ? null : batch.id);
              },
              child: SizedBox(
                width: double.infinity,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        _StatusBadge(status: batch.status),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            batch.summaryLabel.isEmpty
                                ? 'Batch #${batch.id}'
                                : batch.summaryLabel,
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                        if (failedCount > 0)
                          _ErrorCountBadge(count: failedCount),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: ProgressBar(
                            value: _progressPercentage(batch) / 100,
                            backgroundColor: theme.progressBackground,
                            activeColor: theme.progressActive,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          '${_progressPercentage(batch)}%',
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Text(
                      _formatTimestamp(batch.createdAt),
                      style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: theme.timestampText,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            if (isSelected) ...[
              const SizedBox(height: 16),
              if (_canRetryBatch(batch)) ...[
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Consumer<MonitoringProvider>(
                    builder: (context, provider, _) {
                      return Row(
                        children: [
                          Button(
                            onPressed: provider.isRetrying
                                ? null
                                : () => _handleRetryBatch(context, batch.id),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                if (provider.isRetrying)
                                  const SizedBox(
                                    width: 14,
                                    height: 14,
                                    child: ProgressRing(strokeWidth: 2),
                                  )
                                else
                                  const Icon(FluentIcons.refresh, size: 14),
                                const SizedBox(width: 6),
                                Text(provider.isRetrying ? '重试中...' : '重试失败任务'),
                              ],
                            ),
                          ),
                        ],
                      );
                    },
                  ),
                ),
              ],
              _TaskDetailList(batch: batch),
            ],
          ],
        ),
      ),
    );
  }
}

bool _canRetryBatch(BatchRow batch) {
  return const ['failed', 'partial_failed'].contains(batch.status);
}

void _handleRetryBatch(BuildContext context, int batchId) {
  context.read<MonitoringProvider>().retryFailedBatch(batchId);
}

void _handleRetryTask(BuildContext context, int taskId) {
  context.read<MonitoringProvider>().retryFailedTask(taskId);
}

class _TaskDetailList extends StatelessWidget {
  const _TaskDetailList({required this.batch});

  final BatchRow batch;

  @override
  Widget build(BuildContext context) {
    final tasks = context.watch<MonitoringProvider>().tasks;
    final theme = MonitoringTheme.of(context);

    if (tasks.isEmpty) {
      return const Text(
        '暂无任务明细',
        style: TextStyle(fontSize: 16, fontWeight: FontWeight.w400),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: tasks.map((task) {
        final retryHint = _retryHintForTask(batch, task);
        final canRetry = task.status == 'failed';
        return Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Container(
            width: double.infinity,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: theme.detailBackground,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        task.imageFilename ?? 'Image #${task.imageId}',
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    _StatusBadge(status: task.status),
                    if (canRetry) ...[
                      const SizedBox(width: 8),
                      Consumer<MonitoringProvider>(
                        builder: (context, provider, _) {
                          return HyperlinkButton(
                            onPressed: provider.isRetrying
                                ? null
                                : () => _handleRetryTask(context, task.id),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                if (provider.isRetrying)
                                  const SizedBox(
                                    width: 12,
                                    height: 12,
                                    child: ProgressRing(strokeWidth: 2),
                                  )
                                else
                                  const Icon(FluentIcons.refresh, size: 12),
                                const SizedBox(width: 4),
                                Text(
                                  provider.isRetrying ? '重试中' : '重试',
                                  style: const TextStyle(fontSize: 12),
                                ),
                              ],
                            ),
                          );
                        },
                      ),
                    ],
                  ],
                ),
                if ((task.errorSummary ?? '').isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    task.errorSummary!,
                    style: const TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w400,
                    ),
                  ),
                ],
                if (retryHint != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    retryHint,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: theme.statusPending,
                    ),
                  ),
                ],
              ],
            ),
          ),
        );
      }).toList(),
    );
  }

  String? _retryHintForTask(BatchRow batch, TaskDetail task) {
    if (task.status != 'failed') {
      return null;
    }

    for (final group in batch.failureGroups) {
      if (group.retryRecommended && (group.retryHint?.isNotEmpty ?? false)) {
        return group.retryHint;
      }
    }
    return null;
  }
}

class _BatchEmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = MonitoringTheme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 48),
      decoration: BoxDecoration(
        color: theme.emptyStateBackground,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.emptyStateBorder),
      ),
      child: const Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '暂无导入任务',
            style: TextStyle(fontSize: 28, fontWeight: FontWeight.w600),
          ),
          SizedBox(height: 16),
          Text(
            '当有图片导入任务提交后，批次和处理状态将在此处展示。',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.w400),
          ),
        ],
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = MonitoringTheme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: _statusColor(status, theme),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        _statusLabel(status),
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: theme.statusBadgeText,
        ),
      ),
    );
  }
}

class _ErrorCountBadge extends StatelessWidget {
  const _ErrorCountBadge({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = MonitoringTheme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: theme.errorBadgeBackground,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        '$count 错误',
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: theme.errorBadgeText,
        ),
      ),
    );
  }
}

Color _statusColor(String status, MonitoringTheme theme) {
  return switch (status) {
    'pending' => theme.statusPending,
    'running' => theme.statusRunning,
    'completed' => theme.statusCompleted,
    'failed' => theme.statusFailed,
    _ => theme.statusUnknown,
  };
}

String _statusLabel(String status) {
  return switch (status) {
    'pending' => '等待中',
    'running' => '进行中',
    'completed' => '已完成',
    'failed' => '失败',
    _ => status,
  };
}

int _progressPercentage(BatchRow batch) {
  final totalFromStatuses = batch.statusCounts.values.fold<int>(
    0,
    (sum, value) => sum + value,
  );
  final total = totalFromStatuses > 0 ? totalFromStatuses : batch.totalImages;
  if (total <= 0) {
    return 0;
  }
  final completed = batch.statusCounts['completed'] ?? 0;
  return ((completed / total) * 100).round().clamp(0, 100);
}

String _formatTimestamp(DateTime? value) {
  if (value == null) {
    return '--';
  }

  String twoDigits(int number) => number.toString().padLeft(2, '0');
  return '${value.year}-${twoDigits(value.month)}-${twoDigits(value.day)} ${twoDigits(value.hour)}:${twoDigits(value.minute)}';
}
