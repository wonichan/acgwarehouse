import 'dart:convert';
import 'dart:io';
import 'dart:async';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/image.dart';

/// Response from duplicate groups list API
class DuplicateListResponse {
  final List<DuplicateGroup> groups;
  final int total;
  final bool hasMore;

  const DuplicateListResponse({
    required this.groups,
    required this.total,
    required this.hasMore,
  });

  factory DuplicateListResponse.fromJson(Map<String, dynamic> json) {
    final groups =
        (json['groups'] as List?)
            ?.map((g) => DuplicateGroup.fromJson(g as Map<String, dynamic>))
            .toList() ??
        [];

    return DuplicateListResponse(
      groups: groups,
      total: json['total'] as int? ?? 0,
      hasMore: json['has_more'] as bool? ?? false,
    );
  }
}

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
    // Be more lenient - allow loading if we have at least id and path
    // This fixes the issue where images show as gray icons when any field is missing
    return json['id'] != null && json['path'] != null;
  }
}

/// Detection result
class DetectionResult {
  final String taskId;
  final String status;
  final double progress;
  final int processed;
  final int total;
  final String message;

  const DetectionResult({
    required this.taskId,
    required this.status,
    required this.progress,
    required this.processed,
    required this.total,
    required this.message,
  });

  factory DetectionResult.fromJson(Map<String, dynamic> json) {
    return DetectionResult(
      taskId: json['task_id'] as String? ?? '',
      status: json['status'] as String? ?? 'queued',
      progress: (json['progress'] as num?)?.toDouble() ?? 0,
      processed: json['processed'] as int? ?? 0,
      total: json['total'] as int? ?? 0,
      message: json['message'] as String? ?? '',
    );
  }
}

class DuplicateTaskStatus {
  final String taskId;
  final String status;
  final double progress;
  final int processed;
  final int total;
  final String message;
  final String? error;
  final int groupsFound;

  const DuplicateTaskStatus({
    required this.taskId,
    required this.status,
    required this.progress,
    required this.processed,
    required this.total,
    required this.message,
    required this.error,
    required this.groupsFound,
  });

  bool get isTerminal => status == 'completed' || status == 'failed';

  factory DuplicateTaskStatus.fromJson(Map<String, dynamic> json) {
    return DuplicateTaskStatus(
      taskId: json['task_id'] as String? ?? '',
      status: json['status'] as String? ?? 'queued',
      progress: (json['progress'] as num?)?.toDouble() ?? 0,
      processed: json['processed'] as int? ?? 0,
      total: json['total'] as int? ?? 0,
      message: json['message'] as String? ?? '',
      error: json['error'] as String?,
      groupsFound: json['groups_found'] as int? ?? 0,
    );
  }
}

class DuplicateTaskEvent {
  final String event;
  final DuplicateTaskStatus payload;

  const DuplicateTaskEvent({required this.event, required this.payload});
}

/// Duplicate detection service
class DuplicateService {
  final http.Client _client;
  HttpClient? _streamHttpClient;

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

  Future<DuplicateTaskStatus> getDuplicateTaskStatus(String taskId) async {
    final response = await _client.get(
      Uri.parse(ApiConfig.duplicateTaskStatus(taskId)),
      headers: {'Content-Type': 'application/json'},
    );

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to get duplicate task status: ${response.statusCode}',
        response.statusCode,
      );
    }

    return DuplicateTaskStatus.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  Stream<DuplicateTaskEvent> streamDuplicateTaskEvents(String taskId) async* {
    _streamHttpClient ??= HttpClient();
    final client = _streamHttpClient!;

    final uri = Uri.parse(ApiConfig.duplicateTaskEvents(taskId));
    final request = await client.getUrl(uri);
    request.headers.set(HttpHeaders.acceptHeader, 'text/event-stream');
    final response = await request.close();

    if (response.statusCode != 200) {
      throw ApiException(
        'Failed to stream duplicate task events: ${response.statusCode}',
        response.statusCode,
      );
    }

    final lines = response
        .transform(utf8.decoder)
        .transform(const LineSplitter());

    String currentEvent = 'message';
    final dataLines = <String>[];

    await for (final rawLine in lines) {
      final line = rawLine.trimRight();
      if (line.isEmpty) {
        if (dataLines.isNotEmpty) {
          final data = dataLines.join('\n');
          dataLines.clear();
          if (currentEvent != 'heartbeat') {
            final json = jsonDecode(data) as Map<String, dynamic>;
            yield DuplicateTaskEvent(
              event: currentEvent,
              payload: DuplicateTaskStatus.fromJson(json),
            );
          }
        }
        currentEvent = 'message';
        continue;
      }

      if (line.startsWith('event:')) {
        currentEvent = line.substring(6).trim();
        continue;
      }
      if (line.startsWith('data:')) {
        dataLines.add(line.substring(5).trimLeft());
      }
    }
  }

  /// Get list of duplicate groups
  Future<DuplicateListResponse> getDuplicateGroups({
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
    return DuplicateListResponse.fromJson(json);
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
    _streamHttpClient?.close(force: true);
    _streamHttpClient = null;
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
