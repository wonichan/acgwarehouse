import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/monitoring_models.dart';
import 'package:gallery/services/monitoring_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  group('MonitoringOverview', () {
    test('fromJson parses nested task platform overview contract', () {
      final overview = MonitoringOverview.fromJson({
        'health': {'status': 'ok', 'message': 'healthy'},
        'queue': {
          'is_running': true,
          'is_paused': false,
          'queue_size': 4,
          'worker_count': 2,
        },
        'batches': {'pending': 3, 'running': 1},
        'tasks': {'queued': 8, 'running': 2},
      });

      expect(overview.health.status, 'ok');
      expect(overview.queue.isRunning, isTrue);
      expect(overview.queue.queueSize, 4);
      expect(overview.batches['pending'], 3);
      expect(overview.tasks['running'], 2);
      expect(overview.toJson()['queue_size'], isNull);
    });
  });

  group('BatchRow', () {
    test('fromJson parses status counts and failure groups', () {
      final batch = BatchRow.fromJson({
        'id': 12,
        'source_type': 'import',
        'summary_label': 'Batch #12',
        'status': 'running',
        'total_images': 20,
        'new_images': 15,
        'status_counts': {'pending': 10, 'completed': 5},
        'task_type_counts': {'tagging': 12, 'dedupe': 8},
        'failure_groups': [
          {
            'reason_key': 'worker_timeout',
            'reason_label': 'Worker timeout',
            'count': 2,
            'retry_recommended': true,
            'retry_hint': 'Retry after the worker recovers',
          },
        ],
      });

      expect(batch.id, 12);
      expect(batch.sourceType, 'import');
      expect(batch.statusCounts['pending'], 10);
      expect(batch.taskTypeCounts['tagging'], 12);
      expect(batch.failureGroups, hasLength(1));
      expect(batch.failureGroups.first.retryRecommended, isTrue);
      expect(
        batch.failureGroups.first.retryHint,
        'Retry after the worker recovers',
      );
    });
  });

  group('TaskDetail', () {
    test('fromJson parses task read model fields', () {
      final task = TaskDetail.fromJson({
        'id': 99,
        'batch_id': 12,
        'image_id': 301,
        'image_path': 'library/images/301.png',
        'image_filename': '301.png',
        'task_type': 'tagging',
        'status': 'failed',
        'error_summary': 'worker unavailable',
      });

      expect(task.id, 99);
      expect(task.batchId, 12);
      expect(task.imageId, 301);
      expect(task.imageFilename, '301.png');
      expect(task.status, 'failed');
      expect(task.errorSummary, 'worker unavailable');
    });
  });

  group('MonitoringService', () {
    late List<http.Request> requests;
    late MonitoringService service;

    setUp(() {
      requests = [];
      service = MonitoringService(
        baseUrl: 'http://localhost:8080',
        client: MockClient((request) async {
          requests.add(request);

          if (request.url.path == '/admin/api/task-platform/overview') {
            return http.Response(
              jsonEncode({
                'health': {'status': 'ok', 'message': 'healthy'},
                'queue': {
                  'is_running': true,
                  'is_paused': false,
                  'queue_size': 4,
                  'worker_count': 2,
                },
                'batches': {'running': 1},
                'tasks': {'queued': 8},
              }),
              200,
            );
          }

          if (request.url.path == '/admin/api/task-batches') {
            return http.Response(
              jsonEncode({
                'task_batches': [
                  {
                    'id': 12,
                    'source_type': 'import',
                    'summary_label': 'Batch #12',
                    'status': 'running',
                    'total_images': 20,
                    'new_images': 15,
                    'created_at': '2026-04-05T10:30:00Z',
                    'status_counts': {'pending': 10, 'completed': 5},
                    'task_type_counts': {'tagging': 12},
                    'failure_groups': [],
                  },
                ],
              }),
              200,
            );
          }

          if (request.url.path == '/admin/api/tasks') {
            return http.Response(
              jsonEncode({
                'tasks': [
                  {
                    'id': 99,
                    'batch_id': 12,
                    'image_id': 301,
                    'image_path': 'library/images/301.png',
                    'image_filename': '301.png',
                    'task_type': 'tagging',
                    'status': 'failed',
                    'error_summary': 'worker unavailable',
                  },
                ],
              }),
              200,
            );
          }

          return http.Response('not found', 404);
        }),
        basicAuthHeader: 'Basic ZGVtbzpkZW1v',
      );
    });

    test(
      'fetchOverview calls admin overview and returns typed model',
      () async {
        final overview = await service.fetchOverview();

        expect(overview, isA<MonitoringOverview>());
        expect(overview.queue.workerCount, 2);
        expect(requests.single.method, 'GET');
        expect(
          requests.single.url.toString(),
          'http://localhost:8080/admin/api/task-platform/overview',
        );
        expect(requests.single.headers['Authorization'], 'Basic ZGVtbzpkZW1v');
      },
    );

    test(
      'fetchBatches calls admin task-batches and returns typed rows',
      () async {
        final batches = await service.fetchBatches(limit: 25);

        expect(batches, hasLength(1));
        expect(batches.first, isA<BatchRow>());
        expect(batches.first.createdAt, DateTime.parse('2026-04-05T10:30:00Z'));
        expect(requests.single.url.path, '/admin/api/task-batches');
        expect(requests.single.url.queryParameters['limit'], '25');
      },
    );

    test('fetchTasks calls admin tasks endpoint with batch filter', () async {
      final tasks = await service.fetchTasks(batchId: 12, limit: 50);

      expect(tasks, hasLength(1));
      expect(tasks.first, isA<TaskDetail>());
      expect(requests.single.url.path, '/admin/api/tasks');
      expect(requests.single.url.queryParameters['batch_id'], '12');
      expect(requests.single.url.queryParameters['limit'], '50');
    });

    test(
      'webSocketHeaders exposes admin auth for desktop websocket bootstrap',
      () {
        expect(service.webSocketHeaders, {
          'Authorization': 'Basic ZGVtbzpkZW1v',
        });
      },
    );
  });
}
