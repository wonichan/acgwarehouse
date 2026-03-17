import 'dart:convert';
import 'package:http/http.dart' as http;

/// Service for batch operations on images
class BatchService {
  final http.Client _client;
  final String baseUrl;

  BatchService({
    http.Client? client,
    this.baseUrl = 'http://localhost:8080',
  }) : _client = client ?? http.Client();

  /// Batch adds tags to multiple images
  Future<int> batchAddTags(List<int> imageIds, List<int> tagIds) async {
    final uri = Uri.parse('$baseUrl/api/v1/batch/tags/add');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'image_ids': imageIds,
        'tag_ids': tagIds,
      }),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to add tags: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['images_updated'] as int? ?? 0;
  }

  /// Batch removes tags from multiple images
  Future<int> batchRemoveTags(List<int> imageIds, List<int> tagIds) async {
    final uri = Uri.parse('$baseUrl/api/v1/batch/tags/remove');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'image_ids': imageIds,
        'tag_ids': tagIds,
      }),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to remove tags: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['images_updated'] as int? ?? 0;
  }

  /// Batch moves images to a collection
  Future<int> batchMoveToCollection(List<int> imageIds, int collectionId) async {
    final uri = Uri.parse('$baseUrl/api/v1/batch/collections/move');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'image_ids': imageIds,
        'collection_id': collectionId,
      }),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to move images: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['images_moved'] as int? ?? 0;
  }

  /// Batch removes images from a collection
  Future<int> batchRemoveFromCollection(List<int> imageIds, int collectionId) async {
    final uri = Uri.parse('$baseUrl/api/v1/batch/collections/remove');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'image_ids': imageIds,
        'collection_id': collectionId,
      }),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to remove images from collection: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['images_removed'] as int? ?? 0;
  }

  /// Batch deletes images
  Future<int> batchDeleteImages(List<int> imageIds) async {
    final uri = Uri.parse('$baseUrl/api/v1/batch/images/delete');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'image_ids': imageIds,
      }),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to delete images: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['images_deleted'] as int? ?? 0;
  }

  void dispose() {
    _client.close();
  }
}