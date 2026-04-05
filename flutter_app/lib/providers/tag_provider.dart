import 'package:flutter/foundation.dart';
import '../models/tag.dart';
import '../models/tag_governance.dart';
import '../services/tag_service.dart';

class TagProvider extends ChangeNotifier {
  final TagService _tagService;

  List<Tag> _allTags = [];
  List<Tag> _filteredTags = [];
  final Set<int> _selectedTagIds = {};
  bool _isLoading = false;
  String? _error;

  // Image-specific tags (confirmed/pending/rejected)
  Map<String, List<Tag>> _imageTags = {
    'confirmed': [],
    'pending': [],
    'rejected': [],
  };
  bool _isLoadingImageTags = false;

  // Tag governance statistics
  List<TagStatistics> _statistics = [];
  bool _isLoadingStatistics = false;

  // Phase 19 governance workspace state
  List<TagGovernanceRow> _governanceRows = [];
  final Set<int> _selectedGovernanceIds = {};
  TagGovernanceRow? _activeMergeSource;
  TagDeletePreview? _deletePreview;
  TagGovernanceBatchResult? _lastBatchResult;
  bool _isRunningGovernanceAction = false;
  String? _governanceError;

  TagProvider(this._tagService);

  // Getters
  TagService get tagService => _tagService;
  List<Tag> get allTags => _allTags;
  List<Tag> get filteredTags => _filteredTags;
  Set<int> get selectedTagIds => _selectedTagIds;
  bool get isLoading => _isLoading;
  String? get error => _error;
  List<Tag> get selectedTags =>
      _allTags.where((t) => _selectedTagIds.contains(t.id)).toList();

  Map<String, List<Tag>> get imageTags => _imageTags;
  bool get isLoadingImageTags => _isLoadingImageTags;

  // Tag statistics getters
  List<TagStatistics> get statistics => _statistics;
  bool get isLoadingStatistics => _isLoadingStatistics;

  // Governance workspace getters
  List<TagGovernanceRow> get governanceRows => _governanceRows;
  Set<int> get selectedGovernanceIds => _selectedGovernanceIds;
  TagGovernanceRow? get activeMergeSource => _activeMergeSource;
  TagDeletePreview? get deletePreview => _deletePreview;
  TagGovernanceBatchResult? get lastBatchResult => _lastBatchResult;
  bool get isRunningGovernanceAction => _isRunningGovernanceAction;
  String? get governanceError => _governanceError;

  /// Calculates totals from current statistics
  Map<String, int> get totals {
    int usageCount = 0;
    int pendingCount = 0;
    int confirmedCount = 0;
    int aiCount = 0;
    int manualCount = 0;

    for (final stat in _statistics) {
      usageCount += stat.usageCount;
      pendingCount += stat.pendingCount;
      confirmedCount += stat.confirmedCount;
      aiCount += stat.aiCount;
      manualCount += stat.manualCount;
    }

    return {
      'usageCount': usageCount,
      'pendingCount': pendingCount,
      'confirmedCount': confirmedCount,
      'aiCount': aiCount,
      'manualCount': manualCount,
    };
  }

  // 加载所有标签
  Future<void> loadTags() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _allTags = await _tagService.fetchTags();
      _filteredTags = _allTags;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading tags: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  // 搜索标签
  Future<void> searchTags(String query) async {
    if (query.isEmpty) {
      _filteredTags = _allTags;
      notifyListeners();
      return;
    }

    _isLoading = true;
    notifyListeners();

    try {
      _filteredTags = await _tagService.searchTags(query);
    } catch (e) {
      _error = e.toString();
      debugPrint('Error searching tags: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  // 选择/取消选择标签
  void toggleTag(int tagId) {
    if (_selectedTagIds.contains(tagId)) {
      _selectedTagIds.remove(tagId);
    } else {
      _selectedTagIds.add(tagId);
    }
    notifyListeners();
  }

  // 清空选择
  void clearSelection() {
    _selectedTagIds.clear();
    notifyListeners();
  }

  // 加载图片的标签
  Future<void> loadImageTags(int imageId) async {
    _isLoadingImageTags = true;
    _error = null;
    notifyListeners();

    try {
      _imageTags = await _tagService.getImageTags(imageId);
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading image tags: $e');
    } finally {
      _isLoadingImageTags = false;
      notifyListeners();
    }
  }

  // 确认图片标签
  Future<void> confirmImageTag(int imageId, int tagId) async {
    try {
      await _tagService.confirmTag(imageId, tagId);
      // Move tag from pending to confirmed
      final tag = _imageTags['pending']?.firstWhere(
        (t) => t.id == tagId,
        orElse: () => Tag(
          id: tagId,
          preferredLabel: '',
          slug: '',
          reviewState: 'confirmed',
          trustScore: 0,
          usageCount: 0,
          createdAt: DateTime.now(),
        ),
      );
      if (tag != null && tag.preferredLabel.isNotEmpty) {
        _imageTags['pending']?.removeWhere((t) => t.id == tagId);
        _imageTags['confirmed']?.add(tag.copyWith(reviewState: 'confirmed'));
      }
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error confirming tag: $e');
      rethrow;
    }
  }

  // 拒绝图片标签
  Future<void> rejectImageTag(int imageId, int tagId) async {
    try {
      await _tagService.rejectTag(imageId, tagId);
      // Move tag from pending to rejected
      final tag = _imageTags['pending']?.firstWhere(
        (t) => t.id == tagId,
        orElse: () => Tag(
          id: tagId,
          preferredLabel: '',
          slug: '',
          reviewState: 'rejected',
          trustScore: 0,
          usageCount: 0,
          createdAt: DateTime.now(),
        ),
      );
      if (tag != null && tag.preferredLabel.isNotEmpty) {
        _imageTags['pending']?.removeWhere((t) => t.id == tagId);
        _imageTags['rejected']?.add(tag.copyWith(reviewState: 'rejected'));
      }
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error rejecting tag: $e');
      rethrow;
    }
  }

  // 移除图片标签
  Future<void> removeImageTag(int imageId, int tagId) async {
    try {
      await _tagService.removeImageTag(imageId, tagId);
      // Remove from all lists
      _imageTags['confirmed']?.removeWhere((t) => t.id == tagId);
      _imageTags['pending']?.removeWhere((t) => t.id == tagId);
      _imageTags['rejected']?.removeWhere((t) => t.id == tagId);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error removing tag: $e');
      rethrow;
    }
  }

  // 添加图片标签
  Future<void> addImageTag(int imageId, {int? tagId, String? tagLabel}) async {
    try {
      final tag = await _tagService.addImageTag(
        imageId,
        tagId: tagId,
        tagLabel: tagLabel,
      );
      // Add to confirmed list
      _imageTags['confirmed']?.add(tag);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error adding tag: $e');
      rethrow;
    }
  }

  // 合并图片标签
  Future<void> mergeImageTag(
    int imageId,
    int tagId, {
    int? targetTagId,
    String? targetLabel,
  }) async {
    try {
      await _tagService.mergeImageTag(
        imageId,
        tagId,
        targetTagId: targetTagId,
        targetLabel: targetLabel,
      );
      // Remove from pending list
      _imageTags['pending']?.removeWhere((t) => t.id == tagId);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error merging tag: $e');
      rethrow;
    }
  }

  // 触发 AI 标签生成
  Future<int> triggerAITags(int imageId, {String? prompt}) async {
    try {
      final jobId = await _tagService.triggerAITags(imageId, prompt: prompt);
      return jobId;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error triggering AI tags: $e');
      rethrow;
    }
  }

  // 获取默认 AI 提示词
  Future<String> getDefaultAIPrompt() async {
    try {
      return await _tagService.getDefaultAIPrompt();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error getting default AI prompt: $e');
      rethrow;
    }
  }

  // 获取 AI 任务状态
  Future<Map<String, dynamic>> getAITagStatus(int imageId) async {
    try {
      return await _tagService.getAITagStatus(imageId);
    } catch (e) {
      _error = e.toString();
      debugPrint('Error getting AI tag status: $e');
      rethrow;
    }
  }

  void clearError() {
    _error = null;
    notifyListeners();
  }

  /// 加载治理标签列表
  Future<void> loadGovernanceTags({String? search}) async {
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      _governanceRows = await _tagService.fetchGovernanceTags(search: search);
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error loading governance tags: $e');
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }

  /// 选择/取消选择治理标签
  void toggleGovernanceSelection(int tagId) {
    if (_selectedGovernanceIds.contains(tagId)) {
      _selectedGovernanceIds.remove(tagId);
    } else {
      _selectedGovernanceIds.add(tagId);
    }
    notifyListeners();
  }

  /// 清空治理多选
  void clearGovernanceSelection() {
    _selectedGovernanceIds.clear();
    notifyListeners();
  }

  /// 设置当前合并源标签
  void setActiveMergeSource(TagGovernanceRow row) {
    _activeMergeSource = row;
    notifyListeners();
  }

  /// 清空当前合并源标签
  void clearActiveMergeSource() {
    _activeMergeSource = null;
    notifyListeners();
  }

  /// 加载删除预览
  Future<void> loadDeletePreview(int tagId) async {
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      _deletePreview = await _tagService.fetchDeletePreview(tagId);
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error loading delete preview: $e');
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }

  /// 对已选标签应用主分类
  Future<TagGovernanceBatchResult> applyPrimaryCategoryToSelection(
    String primaryCategory,
  ) async {
    final failures = <TagGovernanceFailure>[];
    final selectedIds = _selectedGovernanceIds.toList(growable: false);

    await _runGovernanceAction(() async {
      for (final tagId in selectedIds) {
        try {
          await _tagService.updateTag(tagId, primaryCategory: primaryCategory);
        } catch (e) {
          failures.add(_buildFailure(tagId, e));
        }
      }

      await loadGovernanceTags();
    });

    final result = TagGovernanceBatchResult(
      deletedTagIds: const [],
      failures: failures,
    );
    _lastBatchResult = result;
    notifyListeners();
    return result;
  }

  /// 对已选标签添加别名
  Future<TagGovernanceBatchResult> addAliasToSelection(
    String aliasLabel, {
    String aliasType = 'synonym',
  }) async {
    final failures = <TagGovernanceFailure>[];
    final selectedIds = _selectedGovernanceIds.toList(growable: false);

    await _runGovernanceAction(() async {
      for (final tagId in selectedIds) {
        try {
          await _tagService.addTagAlias(tagId, aliasLabel, aliasType);
        } catch (e) {
          failures.add(_buildFailure(tagId, e));
        }
      }

      await loadGovernanceTags();
    });

    final result = TagGovernanceBatchResult(
      deletedTagIds: const [],
      failures: failures,
    );
    _lastBatchResult = result;
    notifyListeners();
    return result;
  }

  /// 清理已选未使用标签
  Future<TagGovernanceBatchResult> cleanupSelectedUnusedTags() async {
    final selectedIds = _selectedGovernanceIds.toList(growable: false);
    late final TagGovernanceBatchResult result;

    await _runGovernanceAction(() async {
      result = await _tagService.batchCleanupTags(selectedIds);
      _lastBatchResult = result;
      await loadGovernanceTags();
    });

    notifyListeners();
    return result;
  }

  /// 合并已选标签到目标标签
  Future<TagGovernanceBatchResult> mergeSelectionInto(int targetTagId) async {
    final failures = <TagGovernanceFailure>[];
    final sourceIds = _selectedGovernanceIds.toSet();
    if (_activeMergeSource != null) {
      sourceIds.add(_activeMergeSource!.tagId);
    }

    await _runGovernanceAction(() async {
      for (final sourceTagId in sourceIds) {
        if (sourceTagId == targetTagId) {
          failures.add(
            _buildFailure(
              sourceTagId,
              Exception('source and target must differ'),
            ),
          );
          continue;
        }

        try {
          await _tagService.mergeTagInto(sourceTagId, targetTagId);
        } catch (e) {
          failures.add(_buildFailure(sourceTagId, e));
        }
      }

      await loadGovernanceTags();
    });

    final result = TagGovernanceBatchResult(
      deletedTagIds: const [],
      failures: failures,
    );
    _lastBatchResult = result;
    notifyListeners();
    return result;
  }

  TagGovernanceFailure _buildFailure(int tagId, Object error) {
    final row = _governanceRows
        .where((item) => item.tagId == tagId)
        .cast<TagGovernanceRow?>()
        .firstWhere(
          (item) => item != null,
          orElse: () =>
              _activeMergeSource?.tagId == tagId ? _activeMergeSource : null,
        );

    return TagGovernanceFailure(
      tagId: tagId,
      preferredLabel: row?.preferredLabel ?? '',
      message: error.toString(),
    );
  }

  Future<void> _runGovernanceAction(Future<void> Function() action) async {
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      await action();
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error running governance action: $e');
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }

  // 加载标签统计数据
  Future<void> loadStatistics() async {
    _isLoadingStatistics = true;
    _error = null;
    notifyListeners();

    try {
      _statistics = await _tagService.getTagStatistics();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading tag statistics: $e');
    } finally {
      _isLoadingStatistics = false;
      notifyListeners();
    }
  }

  // 清理无用标签
  Future<Map<String, dynamic>> cleanUnusedTags() async {
    try {
      final result = await _tagService.cleanUnusedTags();
      // 刷新统计数据
      await loadStatistics();
      return result;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error cleaning unused tags: $e');
      rethrow;
    }
  }

  // 更新标签
  Future<void> updateTag(
    int tagId, {
    String? preferredLabel,
    String? primaryCategory,
    String? reviewState,
  }) async {
    try {
      await _tagService.updateTag(
        tagId,
        preferredLabel: preferredLabel,
        primaryCategory: primaryCategory,
        reviewState: reviewState,
      );
      // 刷新统计数据
      await loadStatistics();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error updating tag: $e');
      rethrow;
    }
  }

  // 删除标签
  Future<void> deleteTag(int tagId) async {
    try {
      await _tagService.deleteTag(tagId);
      // 刷新统计数据
      await loadStatistics();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error deleting tag: $e');
      rethrow;
    }
  }
}
