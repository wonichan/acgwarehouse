import 'package:flutter/foundation.dart';
import '../models/tag.dart';
import '../services/tag_service.dart';

/// Provider for tag management and governance statistics
class TagProvider extends ChangeNotifier {
  final TagService _tagService;

  List<TagStatistics> _statistics = [];
  bool _isLoading = false;
  String? _error;

  TagProvider(this._tagService);

  List<TagStatistics> get statistics => _statistics;
  bool get isLoading => _isLoading;
  String? get error => _error;

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

  /// Loads tag governance statistics from the backend
  Future<void> loadStatistics() async {
    if (_isLoading) return;

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _statistics = await _tagService.getTagStatistics();
      _error = null;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading tag statistics: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Clears the current error
  void clearError() {
    _error = null;
    notifyListeners();
  }
}