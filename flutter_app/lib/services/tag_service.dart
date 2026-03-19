import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/tag.dart';

class TagService {
  final http.Client _client;

  TagService({http.Client? client}) : _client = client ?? http.Client();

  /// 获取标签列表
  Future<List<Tag>> fetchTags({String? search, int limit = 50, int offset = 0}) async {
    final uri = Uri.parse(ApiConfig.tags).replace(
      queryParameters: {
        if (search != null && search.isNotEmpty) 'search': search,
        'limit': limit.toString(),
        'offset': offset.toString(),
      },
    );

    final response = await _client.get(uri);
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch tags: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final tags = (json['tags'] as List)
        .map((e) => Tag.fromJson(e as Map<String, dynamic>))
        .toList();
    return tags;
  }

  /// 搜索标签（支持别名匹配）
  Future<List<Tag>> searchTags(String query) async {
    return fetchTags(search: query);
  }

  /// 获取图片的标签
  Future<Map<String, List<Tag>>> getImageTags(int imageId) async {
    final response = await _client.get(Uri.parse(ApiConfig.imageTags(imageId)));
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch image tags: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return {
      'confirmed': (json['confirmed'] as List? ?? [])
          .map((e) => Tag.fromImageTagJson(e as Map<String, dynamic>))
          .toList(),
      'pending': (json['pending'] as List? ?? [])
          .map((e) => Tag.fromImageTagJson(e as Map<String, dynamic>))
          .toList(),
      'rejected': (json['rejected'] as List? ?? [])
          .map((e) => Tag.fromImageTagJson(e as Map<String, dynamic>))
          .toList(),
    };
  }

  /// 为图片添加标签
  Future<Tag> addImageTag(int imageId, {int? tagId, String? tagLabel}) async {
    if (tagId == null && tagLabel == null) {
      throw ArgumentError('Either tagId or tagLabel must be provided');
    }

    final response = await _client.post(
      Uri.parse(ApiConfig.imageTags(imageId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        if (tagId != null) 'tag_id': tagId,
        if (tagLabel != null) 'tag_label': tagLabel,
      }),
    );

    if (response.statusCode != 200 && response.statusCode != 201) {
      throw Exception('Failed to add tag: ${response.statusCode}');
    }

    return Tag.fromImageTagJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// 移除图片标签
  Future<void> removeImageTag(int imageId, int tagId) async {
    final response = await _client.delete(
      Uri.parse(ApiConfig.imageTag(imageId, tagId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to remove tag: ${response.statusCode}');
    }
  }

  /// 确认标签
  Future<void> confirmTag(int imageId, int tagId) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.tagReview(imageId, tagId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'action': 'confirm'}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to confirm tag: ${response.statusCode}');
    }
  }

  /// 拒绝标签
  Future<void> rejectTag(int imageId, int tagId) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.tagReview(imageId, tagId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'action': 'reject'}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to reject tag: ${response.statusCode}');
    }
  }

  /// 批量确认标签
  Future<void> batchConfirmTags(int imageId, List<int> tagIds) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.batchTagReview(imageId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'tag_ids': tagIds, 'action': 'confirm'}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to batch confirm: ${response.statusCode}');
    }
  }

  /// 批量拒绝标签
  Future<void> batchRejectTags(int imageId, List<int> tagIds) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.batchTagReview(imageId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'tag_ids': tagIds, 'action': 'reject'}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to batch reject: ${response.statusCode}');
    }
  }

  /// 触发 AI 标签生成
  Future<int> triggerAITags(int imageId, {String? prompt}) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.triggerAITags(imageId)),
      headers: {'Content-Type': 'application/json'},
      body: prompt != null && prompt.isNotEmpty 
          ? jsonEncode({'prompt': prompt})
          : null,
    );
    if (response.statusCode != 200 && response.statusCode != 202) {
      throw Exception('Failed to trigger AI tags: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['job_id'] as int;
  }

  /// 获取默认 AI 提示词
  Future<String> getDefaultAIPrompt() async {
    final response = await _client.get(
      Uri.parse(ApiConfig.defaultAIPrompt),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to get default prompt: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return json['default_prompt'] as String;
  }

  /// 获取 AI 任务状态
  Future<Map<String, dynamic>> getAITagStatus(int imageId) async {
    final response = await _client.get(
      Uri.parse(ApiConfig.aiTagStatus(imageId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to get AI status: ${response.statusCode}');
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 批量触发 AI 标签生成
  Future<Map<String, dynamic>> batchTriggerAITags(List<int> imageIds) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.batchAITags),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'image_ids': imageIds}),
    );
    if (response.statusCode != 200 && response.statusCode != 202) {
      throw Exception('Failed to batch trigger AI tags: ${response.statusCode}');
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 获取标签统计数据
  Future<List<TagStatistics>> getTagStatistics() async {
    final response = await _client.get(
      Uri.parse('${ApiConfig.baseUrl}/tags/stats'),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to get tag statistics: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final stats = json['stats'] as List;
    return stats
        .map((e) => TagStatistics.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// 合并图片标签到目标标签
  Future<void> mergeImageTag(int imageId, int tagId, int targetTagId) async {
    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrl}/images/$imageId/tags/$tagId/merge'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'target_tag_id': targetTagId}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to merge tag: ${response.statusCode}');
    }
  }
}
