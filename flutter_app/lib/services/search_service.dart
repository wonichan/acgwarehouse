import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/image.dart';

/// Search result with pagination info
class SearchResult {
  final List<ImageModel> images;
  final int total;
  final bool hasMore;

  const SearchResult({
    required this.images,
    required this.total,
    required this.hasMore,
  });

  factory SearchResult.fromJson(Map<String, dynamic> json) {
    return SearchResult(
      images: (json['images'] as List)
          .map((i) => ImageModel.fromJson(i as Map<String, dynamic>))
          .toList(),
      total: json['total'] as int? ?? 0,
      hasMore: json['has_more'] as bool? ?? false,
    );
  }
}

/// Image search result with similarity score
class ImageSearchResult {
  final ImageModel image;
  final double similarity;

  const ImageSearchResult({required this.image, required this.similarity});

  factory ImageSearchResult.fromJson(Map<String, dynamic> json) {
    return ImageSearchResult(
      image: ImageModel.fromJson(json['image'] as Map<String, dynamic>),
      similarity: (json['similarity'] as num?)?.toDouble() ?? 0.0,
    );
  }
}

/// Search service for keyword and image-based search
class SearchService {
  final http.Client _client;
  final String _baseUrl;

  SearchService({http.Client? client, required String baseUrl})
    : _client = client ?? http.Client(),
      _baseUrl = baseUrl;

  String get baseUrl => _baseUrl;

  /// Search images by query
  Future<SearchResult> search({
    String? query,
    List<int>? tagIds,
    String sortBy = 'relevance',
    String sortOrder = 'desc',
    int limit = 20,
    int offset = 0,
  }) async {
    final queryParams = <String, String>{
      'limit': limit.toString(),
      'offset': offset.toString(),
      'sort_by': sortBy,
      'sort_order': sortOrder,
    };

    if (query != null && query.isNotEmpty) {
      queryParams['q'] = query;
    }

    if (tagIds != null && tagIds.isNotEmpty) {
      queryParams['tag_ids'] = tagIds.join(',');
    }

    final uri = Uri.parse(
      ApiConfig.search(_baseUrl),
    ).replace(queryParameters: queryParams);

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw SearchException(
        'Search failed: ${response.statusCode}',
        response.statusCode,
      );
    }

    return SearchResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// Search images by filename pattern
  Future<SearchResult> searchByFilename({
    required String pattern,
    int limit = 20,
    int offset = 0,
  }) async {
    final uri = Uri.parse(ApiConfig.searchByFilename(_baseUrl)).replace(
      queryParameters: {
        'pattern': pattern,
        'limit': limit.toString(),
        'offset': offset.toString(),
      },
    );

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw SearchException(
        'Filename search failed: ${response.statusCode}',
        response.statusCode,
      );
    }

    return SearchResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// Search images by image similarity (以图搜图)
  /// Note: This is a placeholder for future implementation
  Future<List<ImageSearchResult>> searchByImage({
    required String imagePath,
    int limit = 20,
  }) async {
    // TODO: Implement image-based search when backend supports it
    // For now, return empty list
    return [];
  }

  /// Get search history
  Future<List<String>> getSearchHistory() async {
    final uri = Uri.parse('${ApiConfig.search(_baseUrl)}/history');

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      return []; // Return empty on failure
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return (json['history'] as List?)?.map((h) => h as String).toList() ?? [];
  }

  /// Clear search history
  Future<void> clearSearchHistory() async {
    final uri = Uri.parse('${ApiConfig.search(_baseUrl)}/history');

    await _client.delete(uri, headers: {'Content-Type': 'application/json'});
  }

  void dispose() {
    _client.close();
  }
}

/// Exception thrown when search requests fail
class SearchException implements Exception {
  final String message;
  final int statusCode;

  SearchException(this.message, this.statusCode);

  @override
  String toString() => 'SearchException: $message (status: $statusCode)';
}
