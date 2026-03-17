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
    return DuplicateGroup(
      id: json['id'] as int,
      recommendedImageId: json['recommended_image_id'] as int,
      similarityThreshold: json['similarity_threshold'] as int? ?? 10,
      createdAt: DateTime.parse(json['created_at'] as String),
      relations: (json['relations'] as List?)
              ?.map((r) => DuplicateRelation.fromJson(r as Map<String, dynamic>))
              .toList() ??
          [],
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
    return DuplicateRelation(
      imageId: json['image_id'] as int,
      isRecommended: json['is_recommended'] as bool? ?? false,
      fileHash: json['file_hash'] as String?,
      pHashDistance: json['phash_distance'] as int?,
      image: json['image'] != null
          ? ImageModel.fromJson(json['image'] as Map<String, dynamic>)
          : null,
    );
  }
}

/// Detection result
class DetectionResult {
  final String message;
  final int groupsFound;

  const DetectionResult({
    required this.message,
    required this.groupsFound,
  });

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
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// Get list of duplicate groups
  Future<List<DuplicateGroup>> getDuplicateGroups({
    int limit = 20,
    int offset = 0,
  }) async {
    final uri = Uri.parse(ApiConfig.duplicates).replace(
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
    return DuplicateGroup.fromJson(json['group'] as Map<String, dynamic>);
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