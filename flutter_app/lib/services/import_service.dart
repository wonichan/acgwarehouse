import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';

class ImportTriggerResult {
  final String status;
  final int jobId;

  const ImportTriggerResult({required this.status, required this.jobId});
}

class ImportTriggerException implements Exception {
  final String message;
  final int statusCode;

  ImportTriggerException(this.message, this.statusCode);

  @override
  String toString() => 'ImportTriggerException: $message (status: $statusCode)';
}

class ImportService {
  final http.Client _client;

  ImportService({http.Client? client}) : _client = client ?? http.Client();

  Future<ImportTriggerResult> triggerImport() async {
    final response = await _client.post(
      Uri.parse(ApiConfig.imageScan),
      headers: {'Content-Type': 'application/json'},
    );

    final body = _tryDecodeJson(response.body);

    if (response.statusCode != 202) {
      final error =
          body['error'] as String? ?? 'Library import could not start';
      throw ImportTriggerException(error, response.statusCode);
    }

    final status = body['status'] as String? ?? 'queued';
    final jobID = body['job_id'] as int? ?? 0;
    return ImportTriggerResult(status: status, jobId: jobID);
  }

  Map<String, dynamic> _tryDecodeJson(String raw) {
    final decoded = jsonDecode(raw);
    if (decoded is Map<String, dynamic>) {
      return decoded;
    }
    return <String, dynamic>{};
  }

  void dispose() {
    _client.close();
  }
}
