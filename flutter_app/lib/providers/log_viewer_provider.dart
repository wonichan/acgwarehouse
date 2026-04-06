import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../models/log_models.dart';
import '../services/log_stream_service.dart';
import '../services/monitoring_channel_factory.dart';

typedef LogViewerWsUriFactory =
    Uri Function({required LogSource source, int tail});
typedef LogViewerChannelFactory =
    WebSocketChannel Function(Uri uri, {Map<String, dynamic>? headers});
typedef LogViewerSleep = Future<void> Function(Duration duration);

class LogViewerProvider extends ChangeNotifier {
  LogViewerProvider({
    required LogStreamService service,
    required LogViewerWsUriFactory wsUriFactory,
    LogViewerChannelFactory? channelFactory,
    LogViewerSleep? sleep,
  }) : _service = service,
       _wsUriFactory = wsUriFactory,
       _channelFactory = channelFactory ?? createMonitoringChannel,
       _sleep = sleep ?? Future<void>.delayed;

  static const int _defaultTail = 200;

  final LogStreamService _service;
  final LogViewerWsUriFactory _wsUriFactory;
  final LogViewerChannelFactory _channelFactory;
  final LogViewerSleep _sleep;

  LogSource _selectedSource = LogSource.go;
  List<LogLine> _lines = const [];
  bool _wsConnected = false;
  bool _isPaused = false;
  bool _isDisposed = false;
  int _reconnectAttempt = 0;
  bool _reconnectScheduled = false;
  bool _explicitDisconnect = false;
  int _maxLines = 1000;
  int _reconnectGeneration = 0;

  StreamSubscription<dynamic>? _wsSubscription;
  WebSocketChannel? _channel;

  LogSource get selectedSource => _selectedSource;
  List<LogLine> get lines => _lines;
  bool get wsConnected => _wsConnected;
  bool get isPaused => _isPaused;

  Future<void> connect() async {
    _reconnectGeneration += 1;
    _explicitDisconnect = false;
    await _openWebSocket();
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

  Future<void> switchSource(LogSource source) async {
    if (_selectedSource == source && (_wsConnected || _reconnectScheduled)) {
      return;
    }

    _reconnectGeneration += 1;
    await disconnect();
    _selectedSource = source;
    _lines = const [];
    _notifySafely();

    await connect();
  }

  void togglePause() {
    _isPaused = !_isPaused;
    _notifySafely();
  }

  void clearLines() {
    if (_lines.isEmpty) {
      return;
    }

    _lines = const [];
    _notifySafely();
  }

  void setMaxLines(int max) {
    final normalized = max < 1 ? 1 : max;
    if (_maxLines == normalized) {
      return;
    }

    _maxLines = normalized;
    _trimLines();
    _notifySafely();
  }

  Future<void> _openWebSocket() async {
    await _wsSubscription?.cancel();
    _wsSubscription = null;
    await _channel?.sink.close();
    _channel = null;

    final uri = _wsUriFactory(source: _selectedSource, tail: _defaultTail);
    _channel = _channelFactory(uri, headers: _service.webSocketHeaders);
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
    try {
      final decoded = jsonDecode(message as String) as Map<String, dynamic>;
      final event = LogStreamEvent.fromJson(decoded);
      switch (event.type) {
        case 'snapshot':
          final snapshot = jsonDecode(event.payload) as List<dynamic>;
          final snapshotLines = snapshot
              .map(
                (entry) => _buildLogLine(
                  text: entry?.toString() ?? '',
                  source: event.source,
                  timestamp: event.timestamp,
                  isHistorical: true,
                ),
              )
              .toList();
          _lines = snapshotLines;
          break;
        case 'line':
          _lines = [
            ..._lines,
            _buildLogLine(
              text: event.payload,
              source: event.source,
              timestamp: event.timestamp,
              severity: event.severity,
            ),
          ];
          break;
        case 'status':
          _lines = [
            ..._lines,
            _buildLogLine(
              text: event.payload,
              source: event.source,
              timestamp: event.timestamp,
              severity: 'status',
            ),
          ];
          break;
        default:
          return;
      }

      _trimLines();
      _notifySafely();
    } catch (_) {
      // Ignore malformed messages and keep the existing buffer intact.
    }
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
    final myGeneration = _reconnectGeneration;
    while (!_explicitDisconnect && !_isDisposed) {
      final delay = _nextReconnectDelay();
      await _sleep(delay);
      if (_explicitDisconnect ||
          _isDisposed ||
          _reconnectGeneration != myGeneration) {
        break;
      }

      try {
        if (_reconnectGeneration != myGeneration) {
          break;
        }
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

  void _trimLines() {
    if (_lines.length <= _maxLines) {
      return;
    }

    _lines = _lines.sublist(_lines.length - _maxLines);
  }

  LogLine _buildLogLine({
    required String text,
    required LogSource source,
    required DateTime timestamp,
    String? severity,
    bool isHistorical = false,
  }) {
    return LogLine(
      text: text,
      severity: severity ?? _inferSeverity(text),
      timestamp: timestamp,
      source: source,
      isHistorical: isHistorical,
    );
  }

  String _inferSeverity(String text) {
    final normalized = text.toLowerCase();
    if (normalized.contains('error') || normalized.contains('fatal')) {
      return 'error';
    }
    if (normalized.contains('warn')) {
      return 'warning';
    }
    if (normalized.contains('debug')) {
      return 'debug';
    }
    return 'normal';
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
