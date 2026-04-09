import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/collection.dart';
import '../models/image.dart';

/// Service for collection-related API operations
class CollectionService {
  final http.Client _client;
  final String _baseUrl;

  /// Creates a CollectionService. [baseUrl] is the root URL (e.g. 'http://localhost:8080').
  CollectionService({http.Client? client, required String baseUrl})
    : _client = client ?? http.Client(),
      _baseUrl = baseUrl;

  /// Fetches all collections
  Future<List<Collection>> fetchCollections({
    int limit = 20,
    int offset = 0,
  }) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/collections')
        .replace(
          queryParameters: {
            'limit': limit.toString(),
            'offset': offset.toString(),
          },
        );

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to fetch collections: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final collections = (json['collections'] as List)
        .map((item) => Collection.fromJson(item as Map<String, dynamic>))
        .toList();

    return collections;
  }

  /// Fetches a single collection by ID
  Future<Collection> fetchCollection(int id) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/collections/$id');

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to fetch collection: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return Collection.fromJson(json);
  }

  /// Creates a new collection
  Future<Collection> createCollection(
    String name, {
    String? description,
  }) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/collections');

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'name': name, 'description': description}),
    );

    if (response.statusCode != 201) {
      throw Exception('Failed to create collection: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return Collection.fromJson(json);
  }

  /// Updates an existing collection
  Future<Collection> updateCollection(
    int id,
    String name, {
    String? description,
  }) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/collections/$id');

    final response = await _client.put(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'name': name, 'description': description}),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to update collection: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return Collection.fromJson(json);
  }

  /// Deletes a collection
  Future<void> deleteCollection(int id) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/collections/$id');

    final response = await _client.delete(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to delete collection: ${response.statusCode}');
    }
  }

  /// Adds an image to a collection
  Future<void> addImageToCollection(int collectionId, int imageId) async {
    final uri = Uri.parse(
      '${ApiConfig.baseUrlOf(_baseUrl)}/collections/$collectionId/images',
    );

    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'image_id': imageId}),
    );

    if (response.statusCode != 200) {
      throw Exception(
        'Failed to add image to collection: ${response.statusCode}',
      );
    }
  }

  /// Removes an image from a collection
  Future<void> removeImageFromCollection(int collectionId, int imageId) async {
    final uri = Uri.parse(
      '${ApiConfig.baseUrlOf(_baseUrl)}/collections/$collectionId/images/$imageId',
    );

    final response = await _client.delete(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception(
        'Failed to remove image from collection: ${response.statusCode}',
      );
    }
  }

  /// Sets the cover image for a collection
  Future<void> setCoverImage(int collectionId, int imageId) async {
    final uri = Uri.parse(
      '${ApiConfig.baseUrlOf(_baseUrl)}/collections/$collectionId/cover',
    );

    final response = await _client.put(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'image_id': imageId}),
    );

    if (response.statusCode != 200) {
      throw Exception('Failed to set cover image: ${response.statusCode}');
    }
  }

  /// Fetches images in a collection
  Future<List<ImageModel>> fetchCollectionImages(
    int collectionId, {
    int limit = 20,
    int offset = 0,
  }) async {
    final uri =
        Uri.parse(
          '${ApiConfig.baseUrlOf(_baseUrl)}/collections/$collectionId/images',
        ).replace(
          queryParameters: {
            'limit': limit.toString(),
            'offset': offset.toString(),
          },
        );

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw Exception(
        'Failed to fetch collection images: ${response.statusCode}',
      );
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final images = (json['images'] as List)
        .map((item) => ImageModel.fromJson(item as Map<String, dynamic>))
        .toList();

    return images;
  }

  void dispose() {
    _client.close();
  }
}
