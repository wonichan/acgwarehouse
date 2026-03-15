import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/tag.dart';
import 'api_service.dart';

/// Service for tag-related API operations
class TagService {
  final http.Client _client;
  final String baseUrl;

  TagService({
    http.Client? client,
    this.baseUrl = 'http://localhost:8080',
  }) : _client = client ?? http.Client();

  /// Fetches all tags
  Future<List<Tag>> fetchTags() async {
    final uri = Uri.parse('$baseUrl/api/v1/tags');
    
    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to fetch tags: ${response.statusCode}', response.statusCode);
    }

    final json = jsonDecode(response.body) as List;
    return json.map((t) => Tag.fromJson(t as Map<String, dynamic>)).toList();
  }

  /// Searches tags by query string
  Future<List<Tag>> searchTags(String query) async {
    final uri = Uri.parse('$baseUrl/api/v1/tags/search').replace(
      queryParameters: {'q': query},
    );
    
    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to search tags: ${response.statusCode}', response.statusCode);
    }

    final json = jsonDecode(response.body) as List;
    return json.map((t) => Tag.fromJson(t as Map<String, dynamic>)).toList();
  }

  /// Gets all tags for an image grouped by review state
  Future<Map<String, List<ImageTag>>> getImageTags(int imageId) async {
    final uri = Uri.parse('$baseUrl/api/v1/images/$imageId/tags');
    
    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to fetch image tags: ${response.statusCode}', response.statusCode);
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final result = <String, List<ImageTag>>{
      'pending': [],
      'confirmed': [],
      'rejected': [],
    };

    for (final state in ['pending', 'confirmed', 'rejected']) {
      if (json[state] != null) {
        result[state] = (json[state] as List)
            .map((t) => ImageTag.fromJson(t as Map<String, dynamic>))
            .toList();
      }
    }

    return result;
  }

  /// Triggers AI tag generation for an image
  /// Returns the observation/job ID
  Future<int> triggerAITags(int imageId) async {
    final uri = Uri.parse('$baseUrl/api/v1/images/$imageId/ai-tags');
    
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200 && response.statusCode != 201) {
      throw ApiException('Failed to trigger AI tags: ${response.statusCode}', response.statusCode);
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['observation_id'] as int;
  }

  /// Gets the status of an AI tag generation job
  Future<AITagStatus> getAITagStatus(int imageId) async {
    final uri = Uri.parse('$baseUrl/api/v1/images/$imageId/ai-tags/status');
    
    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to get AI tag status: ${response.statusCode}', response.statusCode);
    }

    return AITagStatus.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// Merges a pending AI tag into an existing governed tag
  Future<void> mergeImageTag(int imageTagId, int targetTagId) async {
    final uri = Uri.parse('$baseUrl/api/v1/image-tags/$imageTagId/merge');
    
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'target_tag_id': targetTagId}),
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to merge tag: ${response.statusCode}', response.statusCode);
    }
  }

  /// Confirms a pending image tag
  Future<void> confirmImageTag(int imageTagId) async {
    final uri = Uri.parse('$baseUrl/api/v1/image-tags/$imageTagId/confirm');
    
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to confirm tag: ${response.statusCode}', response.statusCode);
    }
  }

  /// Rejects a pending image tag
  Future<void> rejectImageTag(int imageTagId) async {
    final uri = Uri.parse('$baseUrl/api/v1/image-tags/$imageTagId/reject');
    
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to reject tag: ${response.statusCode}', response.statusCode);
    }
  }

  /// Adds a manual tag to an image
  Future<ImageTag> addImageTag(int imageId, int tagId) async {
    final uri = Uri.parse('$baseUrl/api/v1/images/$imageId/tags');
    
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'tag_id': tagId}),
    );

    if (response.statusCode != 200 && response.statusCode != 201) {
      throw ApiException('Failed to add tag: ${response.statusCode}', response.statusCode);
    }

    return ImageTag.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// Removes a tag from an image
  Future<void> removeImageTag(int imageTagId) async {
    final uri = Uri.parse('$baseUrl/api/v1/image-tags/$imageTagId');
    
    final response = await _client.delete(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200 && response.statusCode != 204) {
      throw ApiException('Failed to remove tag: ${response.statusCode}', response.statusCode);
    }
  }

  /// Gets tag governance statistics
  Future<List<TagStatistics>> getTagStatistics() async {
    final uri = Uri.parse('$baseUrl/api/v1/tags/statistics');
    
    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException('Failed to get tag statistics: ${response.statusCode}', response.statusCode);
    }

    final json = jsonDecode(response.body) as List;
    return json.map((s) => TagStatistics.fromJson(s as Map<String, dynamic>)).toList();
  }

  void dispose() {
    _client.close();
  }
}