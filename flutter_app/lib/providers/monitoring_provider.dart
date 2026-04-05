import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../models/monitoring_models.dart';
import '../services/monitoring_service.dart';

typedef MonitoringWsUriFactory = Uri Function();
typedef MonitoringChannelFactory = WebSocketChannel Function(Uri uri);
typedef MonitoringSleep = Future<void> Function(Duration duration);

class MonitoringProvider extends ChangeNotifier {
  MonitoringProvider({
    required MonitoringService service,
    required MonitoringWsUriFactory wsUriFactory,
    MonitoringChannelFactory? channelFactory,
    MonitoringSleep? sleep,
  }) : _service = service,
       _wsUriFactory = wsUriFactory,
       _channelFactory = channelFactory ?? WebSocketChannel.connect,
       _sleep = sleep ?? Future<void>.delayed;

  final MonitoringService _service;
  final MonitoringWsUriFactory _wsUriFactory;
  final MonitoringChannelFactory _channelFactory;
  final MonitoringSleep _sleep;

  MonitoringOverview? _overview;
  List<BatchRow> _batches = const [];
  List<TaskDetail> _tasks = const [];
  bool _wsConnected = false;
  bool _isLoading = false;
  bool _isRestarting = false;
  bool _serviceUnavailable = false;
  RestartImpact? _restartImpact;
  int? _selectedBatchId;

  StreamSubscription<dynamic>? _wsSubscription;
  WebSocketChannel? _channel;
  bool _explicitDisconnect = false;
  bool _isDisposed = false;
  bool _reconnectScheduled = false;
  int _reconnectAttempt = 0;

  MonitoringOverview? get overview => _overview;
  List<BatchRow> get batches => _batches;
  List<TaskDetail> get tasks => _tasks;
  bool get wsConnected => _wsConnected;
  bool get isLoading => _isLoading;
  bool get isRestarting => _isRestarting;
  bool get serviceUnavailable => _serviceUnavailable;
  RestartImpact? get restartImpact => _restartImpact;
  int? get selectedBatchId => _selectedBatchId;

  Future<void> connect() async {
    _explicitDisconnect = false;
    await _loadInitialData();
    if (!_serviceUnavailable) {
      await _openWebSocket();
    }
  }

  Future<void> disconnect() async {
    _explicitDisconnect = true;
    _reconnectScheduled = false;
    _reconnectAttempt = 0;
    _wsConnected = false;
    await _wsSubscription?.cancel();
    _wsSubscription = null;
    await _channel?.sink.close();
    _channel = null;
    _notifySafely();
  }

  Future<void> retryLoad() async {
    await _loadInitialData();
    if (!_serviceUnavailable && !_wsConnected) {
      await _openWebSocket();
    }
  }

  Future<void> restartSidecar() async {
    _isRestarting = true;
    _notifySafely();
    try {
      _restartImpact = await _service.restartSidecar();
      await _refreshData();
    } finally {
      _isRestarting = false;
      _notifySafely();
    }
  }

  Future<void> selectBatch(int? id) async {
    _selectedBatchId = id;
    if (id == null) {
      _tasks = const [];
      _notifySafely();
      return;
    }

    _tasks = await _service.fetchTasks(batchId: id);
    _notifySafely();
  }

  Future<void> _loadInitialData() async {
    _isLoading = true;
    _serviceUnavailable = false;
    _notifySafely();

    try {
      await _refreshData();
      _serviceUnavailable = false;
    } catch (_) {
      _serviceUnavailable = true;
    } finally {
      _isLoading = false;
      _notifySafely();
    }
  }

  Future<void> _refreshData() async {
    final nextOverview = await _service.fetchOverview();
    final nextBatches = await _service.fetchBatches();
    List<TaskDetail> nextTasks = _tasks;
    if (_selectedBatchId != null) {
      nextTasks = await _service.fetchTasks(batchId: _selectedBatchId);
    }

    _overview = nextOverview;
    _batches = nextBatches;
    _tasks = nextTasks;
    _serviceUnavailable = false;
    _notifySafely();
  }

  Future<void> _openWebSocket() async {
    await _wsSubscription?.cancel();
    _channel = _channelFactory(_wsUriFactory());
    _wsSubscription = _channel!.stream.listen(
      _handleSocketMessage,
      onError: (_) => _handleSocketInterrupted(),
      onDone: _handleSocketInterrupted,
      cancelOnError: false,
    );

    try {
      await _channel!.ready;
    } catch (_) {
      _handleSocketInterrupted();
      return;
    }

    _wsConnected = true;
    _reconnectAttempt = 0;
    _reconnectScheduled = false;
    _notifySafely();
  }

  void _handleSocketMessage(dynamic message) {
    final decoded = jsonDecode(message as String) as Map<String, dynamic>;
    final event = MonitoringWsEvent.fromJson(decoded);
    final payload = _decodePayload(event.payload);

    switch (event.type) {
      case 'overview':
        _overview = MonitoringOverview.fromJson(payload);
        break;
      case 'batches':
        final rows =
            payload['batches'] as List? ?? payload['items'] as List? ?? [];
        _batches = rows
            .map((entry) => BatchRow.fromJson(entry as Map<String, dynamic>))
            .toList();
        break;
      case 'sidecar':
        if (_overview != null) {
          _overview = MonitoringOverview(
            health: _overview!.health,
            queue: _overview!.queue,
            sidecar: MonitoringSidecarDiagnostics.fromJson(payload),
            batches: _overview!.batches,
            tasks: _overview!.tasks,
          );
        }
        break;
    }

    _notifySafely();
  }

  void _handleSocketInterrupted() {
    if (_explicitDisconnect || _isDisposed || _reconnectScheduled) {
      return;
    }

    _wsConnected = false;
    _reconnectScheduled = true;
    _notifySafely();
    unawaited(_reconnectLoop());
  }

  Future<void> _reconnectLoop() async {
    while (!_explicitDisconnect && !_isDisposed) {
      final delay = _nextReconnectDelay();
      await _sleep(delay);
      if (_explicitDisconnect || _isDisposed) {
        break;
      }

      try {
        await _openWebSocket();
        return;
      } catch (_) {
        continue;
      }
    }

    _reconnectScheduled = false;
  }

  Duration _nextReconnectDelay() {
    final seconds = switch (_reconnectAttempt) {
      0 => 1,
      1 => 2,
      2 => 4,
      3 => 8,
      4 => 16,
      _ => 30,
    };
    _reconnectAttempt += 1;
    return Duration(seconds: seconds);
  }

  Map<String, dynamic> _decodePayload(String payload) {
    final decoded = jsonDecode(payload);
    if (decoded is Map<String, dynamic>) {
      return decoded;
    }
    if (decoded is List) {
      return {'batches': decoded};
    }
    return const {};
  }

  void _notifySafely() {
    if (!_isDisposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _isDisposed = true;
    _wsSubscription?.cancel();
    _channel?.sink.close();
    super.dispose();
  }
}
