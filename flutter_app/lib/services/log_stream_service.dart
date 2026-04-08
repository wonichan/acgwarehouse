import 'package:http/http.dart' as http;

import '../config/api_config.dart';
import '../models/log_models.dart';

class LogStreamService {
  final http.Client _client;
  final String? _basicAuthHeader;

  LogStreamService({http.Client? client, String? basicAuthHeader})
    : _client = client ?? http.Client(),
      _basicAuthHeader = basicAuthHeader;

  Map<String, dynamic>? get webSocketHeaders {
    final basicAuthHeader = _basicAuthHeader;
    if (basicAuthHeader == null || basicAuthHeader.isEmpty) {
      return null;
    }
    return {'Authorization': basicAuthHeader};
  }

  Uri streamUri({required LogSource source, int tail = 200}) {
    return Uri.parse(
      ApiConfig.logStreamWs(source: _sourceToQueryValue(source), tail: tail),
    );
  }

  void dispose() {
    _client.close();
  }

  String _sourceToQueryValue(LogSource source) {
    return 'go';
  }
}
