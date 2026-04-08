import 'dart:convert';

class MonitoringHealth {
  final String status;
  final String? message;

  const MonitoringHealth({required this.status, this.message});

  factory MonitoringHealth.fromJson(Map<String, dynamic> json) {
    return MonitoringHealth(
      status: json['status'] as String? ?? 'unknown',
      message: json['message'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {'status': status, 'message': message};
  }
}

class MonitoringQueue {
  final bool isRunning;
  final bool isPaused;
  final int queueSize;
  final int workerCount;

  const MonitoringQueue({
    required this.isRunning,
    required this.isPaused,
    required this.queueSize,
    required this.workerCount,
  });

  factory MonitoringQueue.fromJson(Map<String, dynamic> json) {
    return MonitoringQueue(
      isRunning: json['is_running'] as bool? ?? false,
      isPaused: json['is_paused'] as bool? ?? false,
      queueSize: (json['queue_size'] as num?)?.toInt() ?? 0,
      workerCount: (json['worker_count'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'is_running': isRunning,
      'is_paused': isPaused,
      'queue_size': queueSize,
      'worker_count': workerCount,
    };
  }
}

class MonitoringOverview {
  final MonitoringHealth health;
  final MonitoringQueue queue;
  final Map<String, int> batches;
  final Map<String, int> tasks;

  const MonitoringOverview({
    required this.health,
    required this.queue,
    required this.batches,
    required this.tasks,
  });

  factory MonitoringOverview.fromJson(Map<String, dynamic> json) {
    return MonitoringOverview(
      health: MonitoringHealth.fromJson(
        (json['health'] as Map<String, dynamic>?) ?? const {},
      ),
      queue: MonitoringQueue.fromJson(
        (json['queue'] as Map<String, dynamic>?) ?? const {},
      ),
      batches: _parseCountMap(json['batches']),
      tasks: _parseCountMap(json['tasks']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'health': health.toJson(),
      'queue': queue.toJson(),
      'batches': batches,
      'tasks': tasks,
    };
  }
}

class FailureGroup {
  final String reasonKey;
  final String reasonLabel;
  final int count;
  final bool retryRecommended;
  final String? retryHint;

  const FailureGroup({
    required this.reasonKey,
    required this.reasonLabel,
    required this.count,
    required this.retryRecommended,
    this.retryHint,
  });

  factory FailureGroup.fromJson(Map<String, dynamic> json) {
    return FailureGroup(
      reasonKey: json['reason_key'] as String? ?? '',
      reasonLabel: json['reason_label'] as String? ?? '',
      count: (json['count'] as num?)?.toInt() ?? 0,
      retryRecommended: json['retry_recommended'] as bool? ?? false,
      retryHint: json['retry_hint'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'reason_key': reasonKey,
      'reason_label': reasonLabel,
      'count': count,
      'retry_recommended': retryRecommended,
      'retry_hint': retryHint,
    };
  }
}

class BatchRow {
  final int id;
  final String sourceType;
  final String summaryLabel;
  final String status;
  final int totalImages;
  final int newImages;
  final DateTime? createdAt;
  final DateTime? finishedAt;
  final Map<String, int> statusCounts;
  final Map<String, int> taskTypeCounts;
  final List<FailureGroup> failureGroups;

  const BatchRow({
    required this.id,
    required this.sourceType,
    required this.summaryLabel,
    required this.status,
    required this.totalImages,
    required this.newImages,
    this.createdAt,
    this.finishedAt,
    required this.statusCounts,
    required this.taskTypeCounts,
    required this.failureGroups,
  });

  factory BatchRow.fromJson(Map<String, dynamic> json) {
    return BatchRow(
      id: (json['id'] as num?)?.toInt() ?? 0,
      sourceType: json['source_type'] as String? ?? '',
      summaryLabel: json['summary_label'] as String? ?? '',
      status: json['status'] as String? ?? 'unknown',
      totalImages: (json['total_images'] as num?)?.toInt() ?? 0,
      newImages: (json['new_images'] as num?)?.toInt() ?? 0,
      createdAt: _parseDateTime(json['created_at']),
      finishedAt: _parseDateTime(json['finished_at']),
      statusCounts: _parseCountMap(json['status_counts']),
      taskTypeCounts: _parseCountMap(json['task_type_counts']),
      failureGroups: ((json['failure_groups'] as List?) ?? const [])
          .map((entry) => FailureGroup.fromJson(entry as Map<String, dynamic>))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'source_type': sourceType,
      'summary_label': summaryLabel,
      'status': status,
      'total_images': totalImages,
      'new_images': newImages,
      'created_at': createdAt?.toIso8601String(),
      'finished_at': finishedAt?.toIso8601String(),
      'status_counts': statusCounts,
      'task_type_counts': taskTypeCounts,
      'failure_groups': failureGroups.map((group) => group.toJson()).toList(),
    };
  }
}

class TaskDetail {
  final int id;
  final int batchId;
  final int imageId;
  final String? imagePath;
  final String? imageFilename;
  final String taskType;
  final String status;
  final String? errorSummary;

  const TaskDetail({
    required this.id,
    required this.batchId,
    required this.imageId,
    this.imagePath,
    this.imageFilename,
    required this.taskType,
    required this.status,
    this.errorSummary,
  });

  factory TaskDetail.fromJson(Map<String, dynamic> json) {
    return TaskDetail(
      id: (json['id'] as num?)?.toInt() ?? 0,
      batchId: (json['batch_id'] as num?)?.toInt() ?? 0,
      imageId: (json['image_id'] as num?)?.toInt() ?? 0,
      imagePath: json['image_path'] as String?,
      imageFilename: json['image_filename'] as String?,
      taskType: json['task_type'] as String? ?? '',
      status: json['status'] as String? ?? 'unknown',
      errorSummary: json['error_summary'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'batch_id': batchId,
      'image_id': imageId,
      'image_path': imagePath,
      'image_filename': imageFilename,
      'task_type': taskType,
      'status': status,
      'error_summary': errorSummary,
    };
  }
}

class GoRuntimeMetrics {
  final int queueDepth;
  final int activeWorkers;
  final int pendingTasks;
  final bool isRunning;
  final bool isPaused;

  const GoRuntimeMetrics({
    required this.queueDepth,
    required this.activeWorkers,
    required this.pendingTasks,
    required this.isRunning,
    required this.isPaused,
  });

  factory GoRuntimeMetrics.fromJson(Map<String, dynamic> json) {
    return GoRuntimeMetrics(
      queueDepth: (json['queue_depth'] as num?)?.toInt() ?? 0,
      activeWorkers: (json['active_workers'] as num?)?.toInt() ?? 0,
      pendingTasks: (json['pending_tasks'] as num?)?.toInt() ?? 0,
      isRunning: json['is_running'] as bool? ?? false,
      isPaused: json['is_paused'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'queue_depth': queueDepth,
      'active_workers': activeWorkers,
      'pending_tasks': pendingTasks,
      'is_running': isRunning,
      'is_paused': isPaused,
    };
  }
}

class MonitoringWsEvent {
  final String type;
  final String payload;
  final DateTime timestamp;

  const MonitoringWsEvent({
    required this.type,
    required this.payload,
    required this.timestamp,
  });

  factory MonitoringWsEvent.fromJson(Map<String, dynamic> json) {
    final payload = json['payload'];
    return MonitoringWsEvent(
      type: json['type'] as String? ?? '',
      payload: payload is String ? payload : jsonEncode(payload),
      timestamp:
          _parseDateTime(json['timestamp']) ??
          DateTime.fromMillisecondsSinceEpoch(0),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'type': type,
      'payload': payload,
      'timestamp': timestamp.toIso8601String(),
    };
  }
}

class RetryResult {
  final int retryCount;
  final int batchId;

  const RetryResult({required this.retryCount, required this.batchId});

  factory RetryResult.fromJson(Map<String, dynamic> json) {
    final payload = (json['data'] as Map<String, dynamic>?) ?? json;
    return RetryResult(
      retryCount: (payload['retry_count'] as num?)?.toInt() ?? 0,
      batchId: (payload['batch_id'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {'retry_count': retryCount, 'batch_id': batchId};
  }
}

Map<String, int> _parseCountMap(Object? value) {
  final input = value as Map<String, dynamic>?;
  if (input == null) {
    return const {};
  }

  return input.map(
    (key, count) => MapEntry(key, (count as num?)?.toInt() ?? 0),
  );
}

DateTime? _parseDateTime(Object? value) {
  final text = value as String?;
  if (text == null || text.isEmpty) {
    return null;
  }
  return DateTime.parse(text);
}
