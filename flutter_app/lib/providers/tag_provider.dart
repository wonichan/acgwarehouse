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
  int _imageTagsRequestVersion = 0;

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

  // Governance pagination state
  int _governanceOffset = 0;
  int _governanceTotal = 0;
  bool _hasMoreGovernance = true;
  bool _isLoadingMoreGovernance = false;
  String? _governanceSearch;

  // Lazy tree browse state (gallery filter)
  List<TagBrowseNode> _treeRoots = [];
  Map<int, List<TagBrowseNode>> _treeChildrenByParent = {};
  List<TagBrowseNode> _orphanTags = [];
  int _orphanTotal = 0;
  bool _hasMoreOrphans = false;
  bool _isLoadingTreeRoots = false;
  bool _isLoadingOrphans = false;
  String? _treeBrowseError;

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

  // Tag tree getters
  Map<String, dynamic>? _tagTree;
  Map<String, dynamic>? get tagTree => _tagTree;

  // 加载标签树
  Future<void> loadTagTree() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _tagTree = await _tagService.getTree();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading tag tree: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

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
  int get governanceTotal => _governanceTotal;
  bool get hasMoreGovernance => _hasMoreGovernance;
  bool get isLoadingMoreGovernance => _isLoadingMoreGovernance;

  // Lazy tree browse getters (gallery filter)
  List<TagBrowseNode> get treeRoots => _treeRoots;
  Map<int, List<TagBrowseNode>> get treeChildrenByParent =>
      _treeChildrenByParent;
  List<TagBrowseNode> get orphanTags => _orphanTags;
  int get orphanTotal => _orphanTotal;
  bool get hasMoreOrphans => _hasMoreOrphans;
  bool get isLoadingTreeRoots => _isLoadingTreeRoots;
  bool get isLoadingOrphans => _isLoadingOrphans;
  String? get treeBrowseError => _treeBrowseError;
  List<TagBrowseNode> childrenOf(int parentId) =>
      _treeChildrenByParent[parentId] ?? const [];

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

  void setSelection(Iterable<int> tagIds) {
    _selectedTagIds
      ..clear()
      ..addAll(tagIds);
    notifyListeners();
  }

  // 加载图片的标签
  Future<void> loadImageTags(int imageId) async {
    final requestVersion = ++_imageTagsRequestVersion;
    _isLoadingImageTags = true;
    _error = null;
    _imageTags = _emptyImageTags();
    notifyListeners();

    try {
      final imageTags = await _tagService.getImageTags(imageId);
      if (requestVersion != _imageTagsRequestVersion) {
        return;
      }
      _imageTags = {
        'confirmed': List<Tag>.from(imageTags['confirmed'] ?? const <Tag>[]),
        'pending': List<Tag>.from(imageTags['pending'] ?? const <Tag>[]),
        'rejected': List<Tag>.from(imageTags['rejected'] ?? const <Tag>[]),
      };
    } catch (e) {
      if (requestVersion != _imageTagsRequestVersion) {
        return;
      }
      _error = e.toString();
      debugPrint('Error loading image tags: $e');
    } finally {
      if (requestVersion == _imageTagsRequestVersion) {
        _isLoadingImageTags = false;
        notifyListeners();
      }
    }
  }

  Map<String, List<Tag>> _emptyImageTags() {
    return {'confirmed': <Tag>[], 'pending': <Tag>[], 'rejected': <Tag>[]};
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
  Future<void> addImageTag(
    int imageId, {
    int? tagId,
    String? tagLabel,
    String? level,
    int? parentId,
  }) async {
    try {
      final tag = await _tagService.addImageTag(
        imageId,
        tagId: tagId,
        tagLabel: tagLabel,
        level: level,
        parentId: parentId,
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
    String? targetLevel,
    int? targetParentId,
  }) async {
    try {
      await _tagService.mergeImageTag(
        imageId,
        tagId,
        targetTagId: targetTagId,
        targetLabel: targetLabel,
        targetLevel: targetLevel,
        targetParentId: targetParentId,
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

  Future<void> loadTreeRoots() async {
    _isLoadingTreeRoots = true;
    _treeBrowseError = null;
    notifyListeners();

    try {
      _treeRoots = await _tagService.fetchTreeRoots();
    } catch (e) {
      _treeBrowseError = e.toString();
      debugPrint('Error loading tree roots: $e');
    } finally {
      _isLoadingTreeRoots = false;
      notifyListeners();
    }
  }

  Future<void> loadTreeChildren(int parentId) async {
    _treeBrowseError = null;
    notifyListeners();

    try {
      final children =
          await _tagService.fetchTreeChildren(parentId: parentId);
      _treeChildrenByParent[parentId] = children;
    } catch (e) {
      _treeBrowseError = e.toString();
      debugPrint('Error loading tree children for $parentId: $e');
    } finally {
      notifyListeners();
    }
  }

  Future<void> loadOrphanTags({int limit = 20, int offset = 0}) async {
    _isLoadingOrphans = true;
    _treeBrowseError = null;
    notifyListeners();

    try {
      final page =
          await _tagService.fetchOrphanTags(limit: limit, offset: offset);
      if (offset == 0) {
        _orphanTags = page.items;
      } else {
        _orphanTags = [..._orphanTags, ...page.items];
      }
      _orphanTotal = page.total;
      _hasMoreOrphans = page.hasMore;
    } catch (e) {
      _treeBrowseError = e.toString();
      debugPrint('Error loading orphan tags: $e');
    } finally {
      _isLoadingOrphans = false;
      notifyListeners();
    }
  }

  Future<void> searchOrphanTags(String query) async {
    _isLoadingOrphans = true;
    _treeBrowseError = null;
    notifyListeners();

    try {
      final page = await _tagService.fetchOrphanTags(
        search: query,
        limit: 50,
        offset: 0,
      );
      _orphanTags = page.items;
      _orphanTotal = page.total;
      _hasMoreOrphans = page.hasMore;
    } catch (e) {
      _treeBrowseError = e.toString();
      debugPrint('Error searching orphan tags: $e');
    } finally {
      _isLoadingOrphans = false;
      notifyListeners();
    }
  }

  /// 加载治理标签列表（首页）
  Future<void> loadGovernanceTags({String? search}) async {
    _governanceSearch = search;
    _governanceOffset = 0;
    _governanceRows = [];
    _hasMoreGovernance = true;
    _isRunningGovernanceAction = true;
    _governanceError = null;
    notifyListeners();

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        search: search,
        limit: pageSize,
        offset: 0,
      );
      _governanceRows = page.rows;
      _governanceTotal = page.total;
      _governanceOffset = page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error loading governance tags: $e');
    } finally {
      _isRunningGovernanceAction = false;
      notifyListeners();
    }
  }

  /// 加载更多治理标签（无限滚动触发）
  Future<void> loadMoreGovernanceTags() async {
    if (_isLoadingMoreGovernance || !_hasMoreGovernance) return;
    _isLoadingMoreGovernance = true;
    notifyListeners();

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        search: _governanceSearch,
        limit: pageSize,
        offset: _governanceOffset,
      );
      _governanceRows = [..._governanceRows, ...page.rows];
      _governanceTotal = page.total;
      _governanceOffset += page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error loading more governance tags: $e');
    } finally {
      _isLoadingMoreGovernance = false;
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

      await _refreshGovernanceRows();
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

      await _refreshGovernanceRows();
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
      await _refreshGovernanceRows();
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

      await _refreshGovernanceRows();
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

  Future<void> _refreshGovernanceRows({
    String? search,
    bool asPrimaryAction = false,
  }) async {
    // Reset to first page and reload
    _governanceSearch = search ?? _governanceSearch;
    _governanceOffset = 0;
    _hasMoreGovernance = true;

    if (asPrimaryAction) {
      _isRunningGovernanceAction = true;
      _governanceError = null;
      notifyListeners();
    }

    try {
      const pageSize = 50;
      final page = await _tagService.fetchGovernanceTags(
        search: _governanceSearch,
        limit: pageSize,
        offset: 0,
      );
      _governanceRows = page.rows;
      _governanceTotal = page.total;
      _governanceOffset = page.rows.length;
      _hasMoreGovernance = _governanceOffset < _governanceTotal;
    } catch (e) {
      _governanceError = e.toString();
      debugPrint('Error loading governance tags: $e');
    } finally {
      if (asPrimaryAction) {
        _isRunningGovernanceAction = false;
        notifyListeners();
      } else {
        notifyListeners();
      }
    }
  }

  Future<void> _refreshStatisticsState() async {
    _isLoadingStatistics = true;
    _error = null;
    notifyListeners();

    try {
      _statistics = await _tagService.getTagStatistics();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error refreshing tag statistics: $e');
    } finally {
      _isLoadingStatistics = false;
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
      await _refreshStatisticsState();
      await _refreshGovernanceRows();
      await loadTagTree();
      await loadTags();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error updating tag: $e');
      rethrow;
    }
  }

  // 创建标签
  Future<void> createTag({
    required String preferredLabel,
    String? primaryCategory,
    String? level,
    int? parentId,
  }) async {
    try {
      await _tagService.createTag(
        preferredLabel: preferredLabel,
        primaryCategory: primaryCategory,
        level: level,
        parentId: parentId,
      );
      await _refreshStatisticsState();
      await _refreshGovernanceRows();
      await loadTagTree();
      await loadTags();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error creating tag: $e');
      rethrow;
    }
  }

  // 更改标签层级
  Future<void> changeTagLevel(int tagId, String level, {int? parentId}) async {
    try {
      await _tagService.changeLevel(tagId, level, parentId: parentId);
      await _refreshStatisticsState();
      await _refreshGovernanceRows();
      await loadTagTree();
      await loadTags(); // Refresh all tags to get updated hierarchy
    } catch (e) {
      _error = e.toString();
      debugPrint('Error changing tag level: $e');
      rethrow;
    }
  }

  // 重新指定父标签
  Future<void> reparentTag(int tagId, int? parentId) async {
    try {
      await _tagService.reparent(tagId, parentId);
      await _refreshStatisticsState();
      await _refreshGovernanceRows();
      await loadTagTree();
      await loadTags(); // Refresh all tags
    } catch (e) {
      _error = e.toString();
      debugPrint('Error reparenting tag: $e');
      rethrow;
    }
  }

  // 获取可以作为父标签的候选列表
  Future<List<Tag>> getParentCandidates(String level) async {
    try {
      return await _tagService.getParentCandidates(level);
    } catch (e) {
      _error = e.toString();
      debugPrint('Error getting parent candidates: $e');
      rethrow;
    }
  }

  // 删除标签
  Future<void> deleteTag(int tagId) async {
    try {
      await _tagService.deleteTag(tagId);
      await _refreshStatisticsState();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error deleting tag: $e');
      rethrow;
    }
  }
}
