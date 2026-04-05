import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/monitoring_models.dart';
import 'package:gallery/providers/monitoring_provider.dart';
import 'package:gallery/services/monitoring_service.dart';
import 'package:mocktail/mocktail.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class MockMonitoringService extends Mock implements MonitoringService {}

class FakeWebSocketSink implements WebSocketSink {
  Future<void> Function([int? closeCode, String? closeReason])? onClose;
  bool closeCalled = false;

  @override
  Future<void> addStream(Stream stream) async {
    await for (final _ in stream) {}
  }

  @override
  Future<void> close([int? closeCode, String? closeReason]) async {
    closeCalled = true;
    await onClose?.call(closeCode, closeReason);
  }

  @override
  Future<void> get done => Future<void>.value();

  @override
  void add(message) {}

  @override
  void addError(Object error, [StackTrace? stackTrace]) {}
}

class FakeWebSocketChannel implements WebSocketChannel {
  FakeWebSocketChannel()
    : _controller = StreamController<String>.broadcast(),
      sink = FakeWebSocketSink() {
    sink.onClose = ([_, __]) async {
      if (!_controller.isClosed) {
        await _controller.close();
      }
    };
  }

  final StreamController<String> _controller;

  @override
  final FakeWebSocketSink sink;

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => null;

  @override
  Future<void> get ready => Future<void>.value();

  @override
  Stream get stream => _controller.stream;

  @override
  dynamic noSuchMethod(Invocation invocation) {
    return super.noSuchMethod(invocation);
  }

  void addJson(Map<String, dynamic> event) {
    _controller.add(jsonEncode(event));
  }

  void addErrorEvent(Object error) {
    _controller.addError(error);
  }

  Future<void> closeFromServer() async {
    if (!_controller.isClosed) {
      await _controller.close();
    }
  }
}

void main() {
  late MockMonitoringService mockService;
  late MonitoringOverview readyOverview;
  late MonitoringOverview stoppedOverview;
  late List<BatchRow> batches;
  late List<TaskDetail> tasks;

  setUp(() {
    mockService = MockMonitoringService();
    readyOverview = MonitoringOverview.fromJson({
      'health': {'status': 'ok', 'message': 'healthy'},
      'queue': {
        'is_running': true,
        'is_paused': false,
        'queue_size': 2,
        'worker_count': 4,
      },
      'sidecar': {
        'state': 'ready',
        'last_probe_at': '2026-04-05T10:00:00Z',
        'last_probe_result': 'ok',
        'last_error_summary': '',
      },
      'batches': {'running': 1},
      'tasks': {'queued': 3},
    });
    stoppedOverview = MonitoringOverview.fromJson({
      'health': {'status': 'degraded', 'message': 'sidecar stopped'},
      'queue': {
        'is_running': true,
        'is_paused': false,
        'queue_size': 1,
        'worker_count': 2,
      },
      'sidecar': {
        'state': 'stopped',
        'last_probe_at': '2026-04-05T10:05:00Z',
        'last_probe_result': 'error',
        'last_error_summary': 'sidecar unavailable',
      },
      'batches': {'pending': 2},
      'tasks': {'queued': 5},
    });
    batches = [
      BatchRow.fromJson({
        'id': 12,
        'source_type': 'import',
        'summary_label': 'Batch #12',
        'status': 'running',
        'total_images': 20,
        'new_images': 15,
        'status_counts': {'running': 4},
        'task_type_counts': {'tagging': 4},
        'failure_groups': [],
      }),
    ];
    tasks = [
      TaskDetail.fromJson({
        'id': 99,
        'batch_id': 12,
        'image_id': 301,
        'image_path': 'library/images/301.png',
        'image_filename': '301.png',
        'task_type': 'tagging',
        'status': 'failed',
        'error_summary': 'sidecar unavailable',
      }),
    ];
  });

  group('MonitoringProvider', () {
    test(
      'connect loads initial overview and batches with loading cleared',
      () async {
        final channel = FakeWebSocketChannel();
        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);
        when(
          () => mockService.fetchTasks(batchId: 12, limit: any(named: 'limit')),
        ).thenAnswer((_) async => tasks);

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) => channel,
        );

        await provider.connect();

        expect(provider.isLoading, isFalse);
        expect(provider.serviceUnavailable, isFalse);
        expect(provider.overview?.sidecar.state, 'ready');
        expect(provider.batches, hasLength(1));
        expect(provider.wsConnected, isTrue);
      },
    );

    test(
      'connect opens websocket and updates state on incoming events',
      () async {
        final channel = FakeWebSocketChannel();
        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);
        when(
          () => mockService.fetchTasks(batchId: 12, limit: any(named: 'limit')),
        ).thenAnswer((_) async => tasks);

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) => channel,
        );

        await provider.connect();
        await provider.selectBatch(12);

        clearInteractions(mockService);
        channel.addJson({
          'type': 'overview',
          'payload': {
            'health': {'status': 'ok', 'message': 'updated'},
            'queue': {
              'is_running': true,
              'is_paused': false,
              'queue_size': 9,
              'worker_count': 5,
            },
            'sidecar': {
              'state': 'ready',
              'last_probe_at': '2026-04-05T11:00:00Z',
              'last_probe_result': 'ok',
              'last_error_summary': '',
            },
            'batches': {'running': 2},
            'tasks': {'queued': 6},
          },
          'timestamp': '2026-04-05T11:00:00Z',
        });
        await pumpEventQueue();

        expect(provider.overview?.queue.queueSize, 9);
        expect(provider.overview?.tasks['queued'], 6);
        verify(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).called(1);
        verify(
          () => mockService.fetchTasks(batchId: 12, limit: any(named: 'limit')),
        ).called(1);
      },
    );

    test('disconnect closes websocket and cleans up resources', () async {
      final channel = FakeWebSocketChannel();
      when(
        () => mockService.fetchOverview(),
      ).thenAnswer((_) async => readyOverview);
      when(
        () => mockService.fetchBatches(limit: any(named: 'limit')),
      ).thenAnswer((_) async => batches);

      final provider = MonitoringProvider(
        service: mockService,
        wsUriFactory: () =>
            Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
        channelFactory: (_, {headers}) => channel,
      );

      await provider.connect();
      await provider.disconnect();

      expect(channel.sink.closeCalled, isTrue);
      expect(provider.wsConnected, isFalse);
    });

    test(
      'websocket failure marks disconnected and retries with backoff',
      () async {
        final firstChannel = FakeWebSocketChannel();
        final secondChannel = FakeWebSocketChannel();
        final channels = [firstChannel, secondChannel];
        final delays = <Duration>[];
        var wsConnections = 0;

        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () {
            wsConnections += 1;
            return Uri.parse('ws://localhost:8080/admin/api/monitoring/ws');
          },
          channelFactory: (_, {headers}) => channels.removeAt(0),
          sleep: (duration) async {
            delays.add(duration);
          },
        );

        await provider.connect();
        firstChannel.addErrorEvent(Exception('ws down'));
        await firstChannel.closeFromServer();
        await pumpEventQueue();

        expect(delays, [const Duration(seconds: 1)]);
        expect(wsConnections, 2);
        expect(provider.wsConnected, isTrue);
      },
    );

    test(
      'restartSidecar keeps websocket connected and stores impact count',
      () async {
        final channel = FakeWebSocketChannel();
        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);
        when(
          () => mockService.restartSidecar(),
        ).thenAnswer((_) async => const RestartImpact(interruptedTaskCount: 3));

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) => channel,
        );

        await provider.connect();
        await provider.restartSidecar();

        expect(provider.restartImpact?.interruptedTaskCount, 3);
        expect(provider.isRestarting, isFalse);
        expect(provider.wsConnected, isTrue);
      },
    );

    test(
      'selectBatch stores selection and loads tasks for drilldown',
      () async {
        final channel = FakeWebSocketChannel();
        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);
        when(
          () => mockService.fetchTasks(batchId: 12, limit: any(named: 'limit')),
        ).thenAnswer((_) async => tasks);

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) => channel,
        );

        await provider.connect();
        await provider.selectBatch(12);

        expect(provider.selectedBatchId, 12);
        expect(provider.tasks, hasLength(1));
        verify(
          () => mockService.fetchTasks(batchId: 12, limit: any(named: 'limit')),
        ).called(1);
      },
    );

    test(
      'retryLoad clears serviceUnavailable after a failed initial fetch',
      () async {
        final channel = FakeWebSocketChannel();
        var attempts = 0;
        when(() => mockService.fetchOverview()).thenAnswer((_) async {
          attempts += 1;
          if (attempts == 1) {
            throw Exception('backend down');
          }
          return readyOverview;
        });
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) => channel,
        );

        await provider.connect();
        expect(provider.serviceUnavailable, isTrue);
        expect(provider.isLoading, isFalse);

        await provider.retryLoad();

        expect(provider.serviceUnavailable, isFalse);
        expect(provider.overview?.health.status, 'ok');
      },
    );

    test('loads batches even when sidecar state is stopped', () async {
      final channel = FakeWebSocketChannel();
      when(
        () => mockService.fetchOverview(),
      ).thenAnswer((_) async => stoppedOverview);
      when(
        () => mockService.fetchBatches(limit: any(named: 'limit')),
      ).thenAnswer((_) async => batches);

      final provider = MonitoringProvider(
        service: mockService,
        wsUriFactory: () =>
            Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
        channelFactory: (_, {headers}) => channel,
      );

      await provider.connect();

      expect(provider.overview?.sidecar.state, 'stopped');
      expect(provider.batches, hasLength(1));
      expect(provider.serviceUnavailable, isFalse);
    });

    test(
      'connect passes admin auth headers into websocket bootstrap',
      () async {
        final channel = FakeWebSocketChannel();
        Map<String, dynamic>? capturedHeaders;
        when(
          () => mockService.fetchOverview(),
        ).thenAnswer((_) async => readyOverview);
        when(
          () => mockService.fetchBatches(limit: any(named: 'limit')),
        ).thenAnswer((_) async => batches);
        when(
          () => mockService.webSocketHeaders,
        ).thenReturn({'Authorization': 'Basic ZGVtbzpkZW1v'});

        final provider = MonitoringProvider(
          service: mockService,
          wsUriFactory: () =>
              Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
          channelFactory: (_, {headers}) {
            capturedHeaders = headers;
            return channel;
          },
        );

        await provider.connect();

        expect(capturedHeaders, {'Authorization': 'Basic ZGVtbzpkZW1v'});
      },
    );
  });
}
