/// Unified API error types for the ACGWarehouse Flutter app.
///
/// All services should throw these instead of generic Exception,
/// enabling callers to catch and handle errors by type.
library;

import 'dart:convert';
import 'package:http/http.dart' as http;

/// Base class for API-level errors (HTTP error responses).
/// Contains the failing status code and a description of the failed operation.
class ApiError implements Exception {
  final String message;
  final int statusCode;
  final String? operation;

  const ApiError({
    required this.message,
    required this.statusCode,
    this.operation,
  });

  @override
  String toString() =>
      'ApiError(${operation ?? 'unknown'}): $message (status: $statusCode)';
}

/// Network-level error (connection refused, timeout, etc.).
/// Wraps the underlying cause for debugging.
class NetworkError implements Exception {
  final String message;
  final Object? cause;

  const NetworkError({this.message = 'Network request failed', this.cause});

  @override
  String toString() =>
      'NetworkError: $message${cause != null ? ' ($cause)' : ''}';
}

/// HTTP client configuration error (e.g. missing base URL).
class ConfigurationError implements Exception {
  final String message;

  const ConfigurationError(this.message);

  @override
  String toString() => 'ConfigurationError: $message';
}

/// Checks an HTTP response and throws [ApiError] if not successful (2xx).
///
/// [operation] describes the operation being performed, used in error messages.
void ensureHttpResponse(http.Response response, String operation) {
  if (response.statusCode >= 200 && response.statusCode < 300) return;

  String message = 'Failed to $operation: ${response.statusCode}';

  // Try to extract error message from JSON body
  try {
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final serverMsg = json['message'] ?? json['error'] ?? json['detail'];
    if (serverMsg is String && serverMsg.isNotEmpty) {
      message = serverMsg;
    }
  } catch (_) {
    // Body is not valid JSON, use default message
  }

  throw ApiError(
    message: message,
    statusCode: response.statusCode,
    operation: operation,
  );
}
