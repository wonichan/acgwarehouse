import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';
import '../models/tag.dart';
import '../models/tag_governance.dart';

class TagService {
  final http.Client _client;
  final String _baseUrl;

  TagService({http.Client? client, required String baseUrl})
    : _client = client ?? http.Client(),
      _baseUrl = baseUrl;

  /// 获取标签列表
  Future<List<Tag>> fetchTags({
    String? search,
    int limit = 50,
    int offset = 0,
  }) async {
    final uri = Uri.parse(ApiConfig.tags(_baseUrl)).replace(
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
    final response = await _client.get(
      Uri.parse(ApiConfig.imageTags(_baseUrl, imageId)),
    );
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
  Future<Tag> addImageTag(
    int imageId, {
    int? tagId,
    String? tagLabel,
    String? level,
    int? parentId,
  }) async {
    if (tagId == null && tagLabel == null) {
      throw ArgumentError('Either tagId or tagLabel must be provided');
    }

    final response = await _client.post(
      Uri.parse(ApiConfig.imageTags(_baseUrl, imageId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        if (tagId != null) 'tag_id': tagId,
        if (tagLabel != null) 'tag_label': tagLabel,
        if (level != null) 'level': level,
        if (parentId != null) 'parent_id': parentId,
      }),
    );

    if (response.statusCode != 200 && response.statusCode != 201) {
      throw Exception('Failed to add tag: ${response.statusCode}');
    }

    return Tag.fromImageTagJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// 移除图片标签
  Future<void> removeImageTag(int imageId, int tagId) async {
    final response = await _client.delete(
      Uri.parse(ApiConfig.imageTag(_baseUrl, imageId, tagId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to remove tag: ${response.statusCode}');
    }
  }

  /// 确认标签
  Future<void> confirmTag(int imageId, int tagId) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.tagReview(_baseUrl, imageId, tagId)),
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
      Uri.parse(ApiConfig.tagReview(_baseUrl, imageId, tagId)),
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
      Uri.parse(ApiConfig.batchTagReview(_baseUrl, imageId)),
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
      Uri.parse(ApiConfig.batchTagReview(_baseUrl, imageId)),
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
      Uri.parse(ApiConfig.triggerAITags(_baseUrl, imageId)),
      headers: {'Content-Type': 'application/json'},
      body: prompt != null && prompt.isNotEmpty
          ? jsonEncode({'prompt': prompt})
          : null,
    );
    if (response.statusCode != 200 && response.statusCode != 202) {
      throw Exception('Failed to trigger AI tags: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final status = json['status'] as String?;
    final legacyJobID = json['job_id'];
    if (legacyJobID is int) {
      return legacyJobID;
    }

    final jobIDs = json['job_ids'];
    if (jobIDs is List && jobIDs.isNotEmpty && jobIDs.first is int) {
      return jobIDs.first as int;
    }

    if (status == 'skipped') {
      final existingStatus = await getAITagStatus(imageId);
      final existingJobID = existingStatus['job_id'];
      if (existingJobID is int) {
        return existingJobID;
      }
      throw Exception('AI 标签任务已存在，但未找到可复用的任务状态');
    }

    throw Exception('Failed to parse AI trigger response: missing job id');
  }

  /// 获取默认 AI 提示词
  Future<String> getDefaultAIPrompt() async {
    final response = await _client.get(
      Uri.parse(ApiConfig.defaultAIPrompt(_baseUrl)),
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
      Uri.parse(ApiConfig.aiTagStatus(_baseUrl, imageId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to get AI status: ${response.statusCode}');
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 批量触发 AI 标签生成
  ///
  /// 支持两种触发方式：
  /// 1) 传 imageIds：显式按图片 ID 列表触发（兼容旧调用）
  /// 2) 传 tagIds/hasTags/sortBy/sortDir：按当前筛选条件全量触发
  Future<Map<String, dynamic>> batchTriggerAITags({
    List<int>? imageIds,
    List<int>? tagIds,
    bool? hasTags,
    String? sortBy,
    String? sortDir,
  }) async {
    final body = <String, dynamic>{
      if (imageIds != null && imageIds.isNotEmpty) 'image_ids': imageIds,
      if (tagIds != null && tagIds.isNotEmpty) 'tag_ids': tagIds,
      if (hasTags != null) 'has_tags': hasTags,
      if (sortBy != null && sortBy.isNotEmpty) 'sort_by': sortBy,
      if (sortDir != null && sortDir.isNotEmpty) 'sort_dir': sortDir,
    };

    final response = await _client.post(
      Uri.parse(ApiConfig.batchAITags(_baseUrl)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(body),
    );
    if (response.statusCode != 200 && response.statusCode != 202) {
      throw Exception(
        'Failed to batch trigger AI tags: ${response.statusCode}',
      );
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 批量触发 AI 标签重新生成
  ///
  /// 与 batchTriggerAITags 的区别：会重新生成已有 AI 标签的图片的标签
  Future<Map<String, dynamic>> batchRegenerateAITags({
    List<int>? imageIds,
    List<int>? tagIds,
    bool? hasTags,
    String? sortBy,
    String? sortDir,
  }) async {
    final body = <String, dynamic>{
      if (imageIds != null && imageIds.isNotEmpty) 'image_ids': imageIds,
      if (tagIds != null && tagIds.isNotEmpty) 'tag_ids': tagIds,
      if (hasTags != null) 'has_tags': hasTags,
      if (sortBy != null && sortBy.isNotEmpty) 'sort_by': sortBy,
      if (sortDir != null && sortDir.isNotEmpty) 'sort_dir': sortDir,
    };

    final response = await _client.post(
      Uri.parse(ApiConfig.batchRegenerateAITags(_baseUrl)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(body),
    );
    if (response.statusCode != 200 && response.statusCode != 202) {
      throw Exception(
        'Failed to batch regenerate AI tags: ${response.statusCode}',
      );
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 获取标签统计数据
  Future<List<TagStatistics>> getTagStatistics() async {
    final response = await _client.get(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/stats'),
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

  /// 获取标签治理列表（分页）
  Future<GovernanceTagsPage> fetchGovernanceTags({
    String? search,
    int limit = 50,
    int offset = 0,
  }) async {
    final uri = Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/governance')
        .replace(
          queryParameters: {
            if (search != null && search.isNotEmpty) 'search': search,
            'limit': limit.toString(),
            'offset': offset.toString(),
          },
        );

    final response = await _client.get(uri);
    if (response.statusCode != 200) {
      throw Exception(
        'Failed to fetch governance tags: ${response.statusCode}',
      );
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final rows = (json['rows'] as List? ?? [])
        .map(
          (entry) => TagGovernanceRow.fromJson(entry as Map<String, dynamic>),
        )
        .toList();
    final total = json['total'] as int? ?? 0;
    return GovernanceTagsPage(rows: rows, total: total);
  }

  /// 获取标签删除预览
  Future<TagDeletePreview> fetchDeletePreview(int tagId) async {
    final response = await _client.get(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$tagId/delete-preview'),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch delete preview: ${response.statusCode}');
    }

    return TagDeletePreview.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// 显式合并标签到目标标签
  Future<void> mergeTagInto(int sourceTagId, int targetTagId) async {
    final request = TagMergeRequest(targetTagId: targetTagId);
    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$sourceTagId/merge'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(request.toJson()),
    );
    if (response.statusCode != 200) {
      throw Exception(
        'Failed to merge tag into target: ${response.statusCode}',
      );
    }
  }

  /// 获取标签别名列表
  Future<List<String>> getTagAliases(int tagId) async {
    final response = await _client.get(
      Uri.parse(ApiConfig.tagAliases(_baseUrl, tagId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch tag aliases: ${response.statusCode}');
    }

    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final aliases = json['aliases'] as List? ?? [];
    return aliases
        .map((alias) {
          if (alias is String) {
            return alias;
          }
          return (alias as Map<String, dynamic>)['label'] as String? ?? '';
        })
        .where((alias) => alias.isNotEmpty)
        .toList();
  }

  /// 添加标签别名
  Future<void> addTagAlias(int tagId, String label, String aliasType) async {
    final response = await _client.post(
      Uri.parse(ApiConfig.tagAliases(_baseUrl, tagId)),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'label': label, 'alias_type': aliasType}),
    );
    if (response.statusCode != 200 && response.statusCode != 201) {
      throw Exception('Failed to add tag alias: ${response.statusCode}');
    }
  }

  /// 删除标签别名
  Future<void> deleteTagAlias(int tagId, int aliasId) async {
    final response = await _client.delete(
      Uri.parse(ApiConfig.tagAlias(_baseUrl, tagId, aliasId)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to delete tag alias: ${response.statusCode}');
    }
  }

  /// 按选择批量清理标签
  Future<TagGovernanceBatchResult> batchCleanupTags(List<int> tagIds) async {
    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/batch/cleanup'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'tag_ids': tagIds}),
    );
    if (response.statusCode != 200) {
      throw Exception(
        'Failed to cleanup selected tags: ${response.statusCode}',
      );
    }

    return TagGovernanceBatchResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// 合并图片标签到目标标签
  Future<void> mergeImageTag(
    int imageId,
    int tagId, {
    int? targetTagId,
    String? targetLabel,
    String? targetLevel,
    int? targetParentId,
  }) async {
    if (targetTagId == null && targetLabel == null) {
      throw ArgumentError('Either targetTagId or targetLabel must be provided');
    }

    final response = await _client.post(
      Uri.parse(
        '${ApiConfig.baseUrlOf(_baseUrl)}/images/$imageId/tags/$tagId/merge',
      ),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        if (targetTagId != null) 'target_tag_id': targetTagId,
        if (targetLabel != null) 'target_label': targetLabel,
        if (targetLevel != null) 'target_level': targetLevel,
        if (targetParentId != null) 'target_parent_id': targetParentId,
      }),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to merge tag: ${response.statusCode}');
    }
  }

  /// 更新标签
  Future<Tag> updateTag(
    int tagId, {
    String? preferredLabel,
    String? primaryCategory,
    String? reviewState,
  }) async {
    final body = <String, dynamic>{};
    if (preferredLabel != null) body['preferred_label'] = preferredLabel;
    if (primaryCategory != null) body['primary_category'] = primaryCategory;
    if (reviewState != null) body['review_state'] = reviewState;

    final response = await _client.put(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$tagId'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(body),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to update tag: ${response.statusCode}');
    }
    return Tag.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// 创建标签
  Future<Tag> createTag({
    required String preferredLabel,
    String? primaryCategory,
    String? level,
    int? parentId,
  }) async {
    final body = <String, dynamic>{
      'preferred_label': preferredLabel,
      if (primaryCategory != null) 'primary_category': primaryCategory,
      if (level != null) 'level': level,
      if (parentId != null) 'parent_id': parentId,
    };

    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(body),
    );

    if (response.statusCode != 200 && response.statusCode != 201) {
      throw Exception('Failed to create tag: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final tagJson = json['tag'];
    if (tagJson is Map<String, dynamic>) {
      return Tag.fromJson(tagJson);
    }
    return Tag.fromJson(json);
  }

  /// 获取标签层级树
  Future<Map<String, dynamic>> getTree() async {
    final response = await _client.get(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/tree'),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to get tree: ${response.statusCode}');
    }
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  /// 获取父标签候选列表
  Future<List<Tag>> getParentCandidates(String level) async {
    final uri = Uri.parse(
      '${ApiConfig.baseUrlOf(_baseUrl)}/tags/parent-candidates',
    ).replace(queryParameters: {'level': level});
    final response = await _client.get(uri);
    if (response.statusCode != 200) {
      throw Exception(
        'Failed to get parent candidates: ${response.statusCode}',
      );
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return (json['candidates'] as List? ?? [])
        .map((e) => Tag.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// 更改标签层级
  Future<Tag> changeLevel(int tagId, String level, {int? parentId}) async {
    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$tagId/change-level'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'level': level,
        if (parentId != null) 'parent_id': parentId,
      }),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to change level: ${response.statusCode}');
    }
    return Tag.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// 重新指定父标签
  Future<Tag> reparent(int tagId, int? parentId) async {
    final response = await _client.post(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$tagId/reparent'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({if (parentId != null) 'parent_id': parentId}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to reparent: ${response.statusCode}');
    }
    return Tag.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// 删除标签
  Future<void> deleteTag(int tagId) async {
    final response = await _client.delete(
      Uri.parse('${ApiConfig.baseUrlOf(_baseUrl)}/tags/$tagId'),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to delete tag: ${response.statusCode}');
    }
  }

  Future<List<TagBrowseNode>> fetchTreeRoots() async {
    final response = await _client.get(
      Uri.parse(ApiConfig.tagTreeRoots(_baseUrl)),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch tree roots: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return (json['items'] as List? ?? [])
        .map((e) => TagBrowseNode.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<List<TagBrowseNode>> fetchTreeChildren({required int parentId}) async {
    final uri = Uri.parse(ApiConfig.tagTreeChildren(_baseUrl)).replace(
      queryParameters: {'parent_id': parentId.toString()},
    );
    final response = await _client.get(uri);
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch tree children: ${response.statusCode}');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return (json['items'] as List? ?? [])
        .map((e) => TagBrowseNode.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<OrphanTagsPage> fetchOrphanTags({
    int limit = 20,
    int offset = 0,
    String? search,
  }) async {
    final uri = Uri.parse(ApiConfig.tagOrphans(_baseUrl)).replace(
      queryParameters: {
        'limit': limit.toString(),
        'offset': offset.toString(),
        if (search != null && search.isNotEmpty) 'search': search,
      },
    );
    final response = await _client.get(uri);
    if (response.statusCode != 200) {
      throw Exception('Failed to fetch orphan tags: ${response.statusCode}');
    }
    return OrphanTagsPage.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }
}
