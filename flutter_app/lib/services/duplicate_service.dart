import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/image.dart';

/// Duplicate detection result group
class DuplicateGroup {
  final int id;
  final int recommendedImageId;
  final int similarityThreshold;
  final DateTime createdAt;
  final List<DuplicateRelation> relations;

  const DuplicateGroup({
    required this.id,
    required this.recommendedImageId,
    required this.similarityThreshold,
    required this.createdAt,
    required this.relations,
  });

  factory DuplicateGroup.fromJson(Map<String, dynamic> json) {
    final groupJson = (json['group'] as Map<String, dynamic>?) ?? json;
    final relationItems =
        (json['relations'] as List?) ?? (json['images'] as List?) ?? const [];

    return DuplicateGroup(
      id: groupJson['id'] as int,
      recommendedImageId: groupJson['recommended_image_id'] as int,
      similarityThreshold: groupJson['similarity_threshold'] as int? ?? 10,
      createdAt: DateTime.parse(groupJson['created_at'] as String),
      relations: relationItems
          .map((r) => DuplicateRelation.fromJson(r as Map<String, dynamic>))
          .toList(),
    );
  }
}

/// Duplicate relation within a group
class DuplicateRelation {
  final int imageId;
  final bool isRecommended;
  final String? fileHash;
  final int? pHashDistance;
  final ImageModel? image;

  const DuplicateRelation({
    required this.imageId,
    required this.isRecommended,
    this.fileHash,
    this.pHashDistance,
    this.image,
  });

  factory DuplicateRelation.fromJson(Map<String, dynamic> json) {
    final candidateImageJson = (json['image'] as Map<String, dynamic>?) ?? json;
    final resolvedImageId = json['image_id'] ?? json['id'];

    return DuplicateRelation(
      imageId: resolvedImageId as int,
      isRecommended: json['is_recommended'] as bool? ?? false,
      fileHash: json['file_hash'] as String?,
      pHashDistance: json['phash_distance'] as int?,
      image: _hasCompleteImageModel(candidateImageJson)
          ? ImageModel.fromJson(candidateImageJson)
          : null,
    );
  }

  static bool _hasCompleteImageModel(Map<String, dynamic> json) {
    return json['id'] != null &&
        json['path'] != null &&
        json['filename'] != null &&
        json['source_root'] != null &&
        json['file_size'] != null &&
        json['width'] != null &&
        json['height'] != null &&
        json['format'] != null &&
        json['phash'] != null &&
        json['created_at'] != null &&
        json['updated_at'] != null;
  }
}

/// Detection result
class DetectionResult {
  final String message;
  final int groupsFound;

  const DetectionResult({required this.message, required this.groupsFound});

  factory DetectionResult.fromJson(Map<String, dynamic> json) {
    return DetectionResult(
      message: json['message'] as String? ?? '',
      groupsFound: json['groups_found'] as int? ?? 0,
    );
  }
}

/// Duplicate detection service
class DuplicateService {
  final http.Client _client;

  DuplicateService({http.Client? client}) : _client = client ?? http.Client();

  /// Trigger duplicate detection
  Future<DetectionResult> detectDuplicates({int threshold = 10}) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.detectDuplicates),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'threshold': threshold}),
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to detect duplicates: ${response.statusCode}',
        response.statusCode,
      );
    }

    return DetectionResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// Get list of duplicate groups
  Future<List<DuplicateGroup>> getDuplicateGroups({
    int limit = 20,
    int offset = 0,
  }) async {
    final uri = Uri.parse(ApiConfig.duplicates).replace(
      queryParameters: {'limit': limit.toString(), 'offset': offset.toString()},
    );

    final response = await _client.get(
      uri,
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to get duplicate groups: ${response.statusCode}',
        response.statusCode,
      );
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final groups = (json['groups'] as List)
        .map((g) => DuplicateGroup.fromJson(g as Map<String, dynamic>))
        .toList();

    return groups;
  }

  /// Get single duplicate group detail
  Future<DuplicateGroup> getDuplicateGroup(int id) async {
    final response = await _client.get(
      Uri.parse(ApiConfig.duplicateDetail(id)),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to get duplicate group: ${response.statusCode}',
        response.statusCode,
      );
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return DuplicateGroup.fromJson(json);
  }

  /// Delete duplicate group record (not the images)
  Future<void> deleteDuplicateGroup(int id) async {
    final response = await _client.delete(
      Uri.parse(ApiConfig.duplicateDetail(id)),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to delete duplicate group: ${response.statusCode}',
        response.statusCode,
      );
    }
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
