import 'package:flutter/foundation.dart';
import '../models/viewer_window_context.dart';
import '../services/search_service.dart';
import '../models/image.dart';

/// Provider for search state management
class SearchProvider with ChangeNotifier {
  final SearchService _service;

  List<ImageModel> _results = [];
  List<String> _searchHistory = [];
  bool _isLoading = false;
  String? _error;
  int _totalResults = 0;
  int _currentOffset = 0;
  static const int _pageSize = 20;

  String _currentQuery = '';
  List<int> _currentTagIds = [];
  String _sortBy = 'relevance';
  String _sortOrder = 'desc';

  SearchProvider({required SearchService service}) : _service = service;

  List<ImageModel> get results => _results;
  List<String> get searchHistory => _searchHistory;
  bool get isLoading => _isLoading;
  String? get error => _error;
  int get totalResults => _totalResults;
  bool get hasMore => _results.length < _totalResults;
  String get currentQuery => _currentQuery;
  String get sortBy => _sortBy;
  String get sortOrder => _sortOrder;
  ViewerWindowSearchSnapshot get viewerWindowSnapshot =>
      ViewerWindowSearchSnapshot(
        query: _currentQuery,
        tagIds: List<int>.unmodifiable(_currentTagIds),
        sortBy: _sortBy,
        sortOrder: _sortOrder,
      );

  int indexOfResult(int imageId) =>
      _results.indexWhere((image) => image.id == imageId);

  /// Search images
  Future<void> search({
    String? query,
    List<int>? tagIds,
    String? sortBy,
    String? sortOrder,
    bool refresh = true,
  }) async {
    if (_isLoading) return;

    if (refresh) {
      _currentOffset = 0;
      _results = [];
    }

    // Update filters
    if (query != null) _currentQuery = query;
    if (tagIds != null) _currentTagIds = tagIds;
    if (sortBy != null) _sortBy = sortBy;
    if (sortOrder != null) _sortOrder = sortOrder;

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final result = await _service.search(
        query: _currentQuery.isNotEmpty ? _currentQuery : null,
        tagIds: _currentTagIds.isNotEmpty ? _currentTagIds : null,
        sortBy: _sortBy,
        sortOrder: _sortOrder,
        limit: _pageSize,
        offset: _currentOffset,
      );

      if (refresh) {
        _results = result.images;
      } else {
        _results.addAll(result.images);
      }

      _totalResults = result.total;
      _currentOffset += result.images.length;

      // Add to history
      if (_currentQuery.isNotEmpty && !_searchHistory.contains(_currentQuery)) {
        _searchHistory.insert(0, _currentQuery);
        if (_searchHistory.length > 10) {
          _searchHistory.removeLast();
        }
      }
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Load more results (pagination)
  Future<void> loadMore() async {
    if (!hasMore || _isLoading) return;
    await search(refresh: false);
  }

  /// Search by filename
  Future<void> searchByFilename(String pattern, {bool refresh = true}) async {
    if (_isLoading) return;

    if (refresh) {
      _currentOffset = 0;
      _results = [];
    }

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      final result = await _service.searchByFilename(
        pattern: pattern,
        limit: _pageSize,
        offset: _currentOffset,
      );

      if (refresh) {
        _results = result.images;
      } else {
        _results.addAll(result.images);
      }

      _totalResults = result.total;
      _currentOffset += result.images.length;
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Set sort options
  void setSort(String sortBy, String sortOrder) {
    _sortBy = sortBy;
    _sortOrder = sortOrder;
    if (_currentQuery.isNotEmpty || _currentTagIds.isNotEmpty) {
      search(refresh: true);
    }
  }

  /// Clear search
  void clearSearch() {
    _currentQuery = '';
    _currentTagIds = [];
    _results = [];
    _totalResults = 0;
    _currentOffset = 0;
    _error = null;
    notifyListeners();
  }

  /// Load search history
  Future<void> loadSearchHistory() async {
    try {
      _searchHistory = await _service.getSearchHistory();
      notifyListeners();
    } catch (e) {
      // Ignore history load failures
    }
  }

  /// Clear search history
  Future<void> clearSearchHistory() async {
    await _service.clearSearchHistory();
    _searchHistory = [];
    notifyListeners();
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
