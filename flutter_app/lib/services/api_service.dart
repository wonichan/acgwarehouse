import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/image.dart';

/// Pagination response wrapper for list endpoints
class PaginationResponse<T> {
  final List<T> items;
  final String? nextCursor;
  final bool hasMore;
  final int total;

  PaginationResponse({
    required this.items,
    this.nextCursor,
    required this.hasMore,
    required this.total,
  });
}

/// API service for communicating with the ACGWarehouse backend
class ApiService {
  final http.Client _client;
  final String baseUrl;

  ApiService({
    http.Client? client,
    this.baseUrl = 'http://localhost:8080',
  }) : _client = client ?? http.Client();

  /// Fetches a paginated list of images with optional filtering
  /// 
  /// [offset] - Pagination offset for fetching next page (matches backend's next_cursor)
  /// [limit] - Maximum number of items to return
  /// [sortBy] - Field to sort by (created_at, filename, file_size)
  /// [sortDir] - Sort direction (asc or desc)
  /// [tagIds] - Optional list of tag IDs to filter images by (AND semantics)
  Future<PaginationResponse<ImageModel>> fetchImages({
    int offset = 0,
    int limit = 20,
    String sortBy = 'created_at',
    String sortDir = 'desc',
    List<int>? tagIds,
  }) async {
    final queryParams = <String, String>{
      'offset': offset.toString(),
      'limit': limit.toString(),
      'sort_by': sortBy,
      'sort_dir': sortDir,
    };

    if (tagIds != null && tagIds.isNotEmpty) {
      queryParams['tag_ids'] = tagIds.join(',');
    }

    final uri = Uri.parse('$baseUrl/api/v1/images').replace(
      queryParameters: queryParams,
    );

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to fetch images: ${response.statusCode}',
        response.statusCode,
      );
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    
    // Backend returns 'images' array, not 'items'
    final images = (json['images'] as List)
        .map((item) => ImageModel.fromJson(item as Map<String, dynamic>))
        .toList();

    return PaginationResponse<ImageModel>(
      items: images,
      nextCursor: json['next_cursor'] as String?,
      hasMore: json['has_more'] as bool? ?? false,
      total: json['total'] as int? ?? 0,
    );
  }

  void dispose() {
    _client.close();
  }
}

/// Exception thrown when API requests fail
class ApiException implements Exception {
  final String message;
  final int statusCode;

  ApiException(this.message, this.statusCode);

  @override
  String toString() => 'ApiException: $message (status: $statusCode)';
}