import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';
import '../models/monitoring_models.dart';

class MonitoringService {
  final http.Client _client;
  final String? _basicAuthHeader;
  final String _baseUrl;

  MonitoringService({
    http.Client? client,
    String? basicAuthHeader,
    required String baseUrl,
  }) : _client = client ?? http.Client(),
       _basicAuthHeader = basicAuthHeader,
       _baseUrl = baseUrl;

  String get baseUrl => _baseUrl;
  String? get basicAuthHeader => _basicAuthHeader;

  Map<String, dynamic>? get webSocketHeaders {
    if (_basicAuthHeader == null || _basicAuthHeader!.isEmpty) {
      return null;
    }
    return {'Authorization': _basicAuthHeader!};
  }

  Future<MonitoringOverview> fetchOverview() async {
    final response = await _client.get(
      Uri.parse('${_baseUrl}/admin/api/task-platform/overview'),
      headers: _headers(),
    );
    _ensureSuccess(response, 'fetch monitoring overview');

    return MonitoringOverview.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<List<BatchRow>> fetchBatches({int? limit}) async {
    final response = await _client.get(
      Uri.parse('${_baseUrl}/admin/api/task-batches').replace(
        queryParameters: {if (limit != null) 'limit': limit.toString()},
      ),
      headers: _headers(),
    );
    _ensureSuccess(response, 'fetch task batches');

    final payload = jsonDecode(response.body);
    final rows = payload is List
        ? payload
        : ((payload as Map<String, dynamic>)['batches'] as List? ??
              payload['task_batches'] as List? ??
              const []);
    return rows
        .map((entry) => BatchRow.fromJson(entry as Map<String, dynamic>))
        .toList();
  }

  Future<List<TaskDetail>> fetchTasks({int? batchId, int? limit}) async {
    final response = await _client.get(
      Uri.parse('${_baseUrl}/admin/api/tasks').replace(
        queryParameters: {
          if (batchId != null) 'batch_id': batchId.toString(),
          if (limit != null) 'limit': limit.toString(),
        },
      ),
      headers: _headers(),
    );
    _ensureSuccess(response, 'fetch tasks');

    final payload = jsonDecode(response.body);
    final rows = payload is List
        ? payload
        : (payload as Map<String, dynamic>)['tasks'] as List? ?? const [];
    return rows
        .map((entry) => TaskDetail.fromJson(entry as Map<String, dynamic>))
        .toList();
  }

  Future<RetryResult> retryFailedBatchTasks(int batchId) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.retryBatch(_baseUrl, batchId)),
      headers: _headers(),
    );
    _ensureSuccess(response, 'retry failed batch tasks');

    return RetryResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<RetryResult> retryFailedTask(int taskId) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.retryTask(_baseUrl, taskId)),
      headers: _headers(),
    );
    _ensureSuccess(response, 'retry failed task');

    return RetryResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Map<String, String> _headers() {
    return {
      'Content-Type': 'application/json',
      if (_basicAuthHeader != null && _basicAuthHeader!.isNotEmpty)
        'Authorization': _basicAuthHeader!,
    };
  }

  void dispose() {
    _client.close();
  }

  void _ensureSuccess(http.Response response, String action) {
    if (response.statusCode != 200) {
      throw Exception('Failed to $action: ${response.statusCode}');
    }
  }
}
