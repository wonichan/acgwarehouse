import 'package:flutter/foundation.dart';
import '../models/tag.dart';
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

  TagProvider(this._tagService);

  // Getters
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
      final tag = await _tagService.addImageTag(imageId, tagId: tagId, tagLabel: tagLabel);
      // Add to confirmed list
      _imageTags['confirmed']?.add(tag);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error adding tag: $e');
      rethrow;
    }
  }

  // 触发 AI 标签生成
  Future<int> triggerAITags(int imageId) async {
    try {
      final jobId = await _tagService.triggerAITags(imageId);
      return jobId;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error triggering AI tags: $e');
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
}
