import 'dart:convert';

/// Log source enum
enum LogSource { go }

/// Single log line from the backend
class LogLine {
  final String text;
  final String severity;
  final DateTime timestamp;
  final LogSource source;
  final bool isHistorical;

  const LogLine({
    required this.text,
    required this.severity,
    required this.timestamp,
    required this.source,
    this.isHistorical = false,
  });

  factory LogLine.fromJson(Map<String, dynamic> json) {
    return LogLine(
      text: json['text'] as String? ?? '',
      severity: json['severity'] as String? ?? 'normal',
      timestamp:
          _parseDateTime(json['timestamp']) ??
          DateTime.fromMillisecondsSinceEpoch(0),
      source: _parseLogSource(json['source']),
      isHistorical: json['isHistorical'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'text': text,
      'severity': severity,
      'timestamp': timestamp.toIso8601String(),
      'source': _logSourceToJson(source),
      'isHistorical': isHistorical,
    };
  }
}

/// WebSocket event from the log stream backend
class LogStreamEvent {
  final String type;
  final LogSource source;
  final String payload;
  final String? severity;
  final DateTime timestamp;

  const LogStreamEvent({
    required this.type,
    required this.source,
    required this.payload,
    this.severity,
    required this.timestamp,
  });

  factory LogStreamEvent.fromJson(Map<String, dynamic> json) {
    final payload = json['payload'];
    return LogStreamEvent(
      type: json['type'] as String? ?? '',
      source: _parseLogSource(json['source']),
      payload: payload is String ? payload : jsonEncode(payload),
      severity: json['severity'] as String?,
      timestamp:
          _parseDateTime(json['timestamp']) ??
          DateTime.fromMillisecondsSinceEpoch(0),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'type': type,
      'source': _logSourceToJson(source),
      'payload': payload,
      'severity': severity,
      'timestamp': timestamp.toIso8601String(),
    };
  }
}

/// Connection state for log viewer
enum LogConnectionState { disconnected, connecting, connected, error }

DateTime? _parseDateTime(dynamic value) {
  if (value is! String || value.isEmpty) {
    return null;
  }
  return DateTime.tryParse(value);
}

LogSource _parseLogSource(dynamic value) {
  return LogSource.go;
}

String _logSourceToJson(LogSource value) {
  return 'go';
}
