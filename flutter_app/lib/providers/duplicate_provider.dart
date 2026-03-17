import 'package:flutter/foundation.dart';
import '../services/duplicate_service.dart';

/// Provider for duplicate detection state management
class DuplicateProvider with ChangeNotifier {
  final DuplicateService _service;

  List<DuplicateGroup> _groups = [];
  bool _isLoading = false;
  bool _isDetecting = false;
  String? _error;
  int _totalGroups = 0;
  int _currentOffset = 0;
  static const int _pageSize = 20;

  DuplicateProvider({DuplicateService? service})
      : _service = service ?? DuplicateService();

  List<DuplicateGroup> get groups => _groups;
  bool get isLoading => _isLoading;
  bool get isDetecting => _isDetecting;
  String? get error => _error;
  int get totalGroups => _totalGroups;
  bool get hasMore => _groups.length < _totalGroups;

  /// Trigger duplicate detection
  Future<DetectionResult?> detectDuplicates({int threshold = 10}) async {
    _isDetecting = true;
    _error = null;
    notifyListeners();

    try {
      final result = await _service.detectDuplicates(threshold: threshold);
      
      // Refresh groups after detection
      await loadGroups(refresh: true);
      
      return result;
    } catch (e) {
      _error = e.toString();
      return null;
    } finally {
      _isDetecting = false;
      notifyListeners();
    }
  }

  /// Load duplicate groups
  Future<void> loadGroups({bool refresh = false}) async {
    if (_isLoading) return;

    if (refresh) {
      _currentOffset = 0;
      _groups = [];
    }

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final groups = await _service.getDuplicateGroups(
        limit: _pageSize,
        offset: _currentOffset,
      );

      if (refresh) {
        _groups = groups;
      } else {
        _groups.addAll(groups);
      }

      _currentOffset += groups.length;
      _totalGroups = _groups.length + (groups.length < _pageSize ? 0 : _pageSize);
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Load more groups (pagination)
  Future<void> loadMore() async {
    if (!hasMore || _isLoading) return;
    await loadGroups();
  }

  /// Delete a duplicate group
  Future<bool> deleteGroup(int groupId) async {
    try {
      await _service.deleteDuplicateGroup(groupId);
      _groups.removeWhere((g) => g.id == groupId);
      notifyListeners();
      return true;
    } catch (e) {
      _error = e.toString();
      notifyListeners();
      return false;
    }
  }

  /// Clear error
  void clearError() {
    _error = null;
    notifyListeners();
  }

  @override
  void dispose() {
    _service.dispose();
    super.dispose();
  }
}