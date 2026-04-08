import 'package:flutter/foundation.dart';
import 'dart:async';
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
  String? _taskId;
  String _taskStatus = 'idle';
  double _taskProgress = 0;
  int _taskProcessed = 0;
  int _taskTotal = 0;
  int _taskGroupsFound = 0;
  StreamSubscription<DuplicateTaskEvent>? _taskEventSubscription;

  DuplicateProvider({required DuplicateService service}) : _service = service;

  List<DuplicateGroup> get groups => _groups;
  bool get isLoading => _isLoading;
  bool get isDetecting => _isDetecting;
  String? get error => _error;
  int get totalGroups => _totalGroups;
  bool get hasMore => _groups.length < _totalGroups;
  String? get taskId => _taskId;
  String get taskStatus => _taskStatus;
  double get taskProgress => _taskProgress;
  int get taskProcessed => _taskProcessed;
  int get taskTotal => _taskTotal;
  int get taskGroupsFound => _taskGroupsFound;

  bool get hasActiveTask => _taskId != null && _isDetecting;

  /// Trigger duplicate detection
  Future<DetectionResult?> detectDuplicates({int threshold = 10}) async {
    _isDetecting = true;
    _error = null;
    _taskStatus = 'queued';
    _taskProgress = 0;
    _taskProcessed = 0;
    _taskTotal = 0;
    _taskGroupsFound = 0;
    notifyListeners();

    try {
      final result = await _service.detectDuplicates(threshold: threshold);
      _taskId = result.taskId;
      _taskStatus = result.status;
      _taskProgress = result.progress;
      _taskProcessed = result.processed;
      _taskTotal = result.total;

      await _subscribeTaskEvents(result.taskId);
      await _refreshTaskStatus(result.taskId);

      return result;
    } catch (e) {
      _error = e.toString();
      return null;
    }
  }

  Future<void> _subscribeTaskEvents(String taskId) async {
    await _taskEventSubscription?.cancel();
    _taskEventSubscription = _service
        .streamDuplicateTaskEvents(taskId)
        .listen(
          (event) async {
            _applyTaskStatus(event.payload);
            if (event.payload.isTerminal) {
              await _onTaskTerminal(event.payload);
            }
          },
          onError: (_) async {
            // fallback to polling snapshot for reconnect-safe state
            await _refreshTaskStatus(taskId);
          },
        );
  }

  Future<void> _refreshTaskStatus(String taskId) async {
    try {
      final status = await _service.getDuplicateTaskStatus(taskId);
      _applyTaskStatus(status);
      if (status.isTerminal) {
        await _onTaskTerminal(status);
      }
    } catch (_) {
      // ignore refresh failure, keep stream path
    }
  }

  void _applyTaskStatus(DuplicateTaskStatus status) {
    _taskId = status.taskId;
    _taskStatus = status.status;
    _taskProgress = status.progress;
    _taskProcessed = status.processed;
    _taskTotal = status.total;
    _taskGroupsFound = status.groupsFound;
    _error = status.error ?? _error;
    notifyListeners();
  }

  Future<void> _onTaskTerminal(DuplicateTaskStatus status) async {
    _isDetecting = false;
    await _taskEventSubscription?.cancel();
    _taskEventSubscription = null;

    if (status.status == 'completed') {
      await loadGroups(refresh: true);
    } else if (status.error != null && status.error!.isNotEmpty) {
      _error = status.error;
    }

    notifyListeners();
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
      final response = await _service.getDuplicateGroups(
        limit: _pageSize,
        offset: _currentOffset,
      );

      if (refresh) {
        _groups = response.groups;
      } else {
        _groups.addAll(response.groups);
      }

      _currentOffset += response.groups.length;
      _totalGroups = response.total;
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
    _taskEventSubscription?.cancel();
    _service.dispose();
    super.dispose();
  }
}
