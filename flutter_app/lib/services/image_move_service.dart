import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';
import '../models/image_move.dart';
import 'api_errors.dart';

class ImageMoveService {
  final http.Client _client;
  final String _baseUrl;
  final bool _ownsClient;

  ImageMoveService({required String baseUrl, http.Client? client})
    : _client = client ?? http.Client(),
      _baseUrl = baseUrl,
      _ownsClient = client == null;

  String get _apiPrefix => ApiConfig.baseUrlOf(_baseUrl);

  Future<ImageMovePreview> preview(ImageMoveRequest request) async {
    final response = await _post('/image-moves/preview', request);
    return ImageMovePreview.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<ImageMoveResult> execute(ImageMoveRequest request) async {
    final response = await _post('/image-moves/execute', request);
    return ImageMoveResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<ImageMoveBatch> createJob(ImageMoveRequest request) async {
    final response = await _post('/image-moves/jobs', request);
    return ImageMoveBatch.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<ImageMoveBatch> getJob(int id) async {
    final response = await _client.get(
      Uri.parse('$_apiPrefix/image-moves/jobs/$id'),
    );
    ensureHttpResponse(response, '/image-moves/jobs/$id');
    return ImageMoveBatch.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<ImageMoveBatch> cancelJob(int id) async {
    final response = await _client.post(
      Uri.parse('$_apiPrefix/image-moves/jobs/$id/cancel'),
    );
    ensureHttpResponse(response, '/image-moves/jobs/$id/cancel');
    return ImageMoveBatch.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Future<List<ImageMoveBatch>> history({int limit = 20}) async {
    final response = await _client.get(
      Uri.parse('$_apiPrefix/image-moves/history?limit=$limit'),
    );
    ensureHttpResponse(response, '/image-moves/history');
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return (body['items'] as List? ?? const [])
        .map((item) => ImageMoveBatch.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<http.Response> _post(String path, ImageMoveRequest request) async {
    final response = await _client.post(
      Uri.parse('$_apiPrefix$path'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(request.toJson()),
    );
    ensureHttpResponse(response, path);
    return response;
  }

  void dispose() {
    if (_ownsClient) {
      _client.close();
    }
  }
}
