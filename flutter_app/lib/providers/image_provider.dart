import 'package:flutter/foundation.dart';
import '../models/gallery_filter_state.dart';
import '../models/image.dart';
import '../services/api_service.dart';

enum ViewMode { grid, masonry }

enum SortField { createdAt, filename, fileSize }

class ImageListProvider extends ChangeNotifier {
  final ApiService _apiService;

  List<ImageModel> _images = [];
  bool _isLoading = false;
  bool _hasMore = true;
  int _currentOffset = 0;
  int _total = 0;
  ViewMode _viewMode = ViewMode.grid;
  SortField _sortField = SortField.createdAt;
  bool _sortAsc = false;
  GalleryFilterState _filter = GalleryFilterState();

  ImageListProvider(this._apiService);

  List<ImageModel> get images => _images;
  bool get isLoading => _isLoading;
  bool get hasMore => _hasMore;
  int get total => _total;
  ViewMode get viewMode => _viewMode;
  SortField get sortField => _sortField;
  bool get sortAsc => _sortAsc;
  GalleryFilterState get filter => _filter;
  List<int> get selectedTagIds => _filter.exactTagIds.toList();
  bool? get hasTagsFilter => _filter.hasTags;
  bool? get hasPendingTagsFilter => _filter.hasPendingTags;

  int indexOfImage(int imageId) =>
      _images.indexWhere((image) => image.id == imageId);

  Future<void> loadImages({bool refresh = false}) async {
    // Prevent duplicate in-flight loads
    if (_isLoading) return;

    // Stop at last page
    if (!refresh && !_hasMore) return;

    _isLoading = true;
    notifyListeners();

    try {
      // On refresh, reset offset and hasMore
      if (refresh) {
        _currentOffset = 0;
        _hasMore = true;
      }

      debugPrint(
        '加载图片: offset=$_currentOffset, exactTagIds=${_filter.exactTagIds.toList()}, subtreeRootTagIds=${_filter.subtreeRootTagIds.toList()}, hasTags=${_filter.hasTags}, hasPendingTags=${_filter.hasPendingTags}, sortBy=${_sortField.name}, sortDir=${_sortAsc ? 'asc' : 'desc'}',
      );

      final response = await _apiService.fetchImages(
        offset: refresh ? 0 : _currentOffset,
        sortBy: _sortField.name == 'createdAt'
            ? 'created_at'
            : _sortField.name == 'fileSize'
            ? 'file_size'
            : 'filename',
        sortDir: _sortAsc ? 'asc' : 'desc',
        exactTagIds: _filter.exactTagIds.isNotEmpty
            ? _filter.exactTagIds.toList()
            : null,
        subtreeRootTagIds: _filter.subtreeRootTagIds.isNotEmpty
            ? _filter.subtreeRootTagIds.toList()
            : null,
        hasTags: _filter.hasTags,
        hasPendingTags: _filter.hasPendingTags,
      );

      if (refresh) {
        _images = response.items;
      } else {
        _images.addAll(response.items);
      }

      _total = response.total;

      // Update offset for next page based on current loaded count
      _currentOffset = _images.length;

      // Update hasMore state from backend response
      _hasMore = response.hasMore;

      // Safety check: hasMore should also be false if we've loaded all items
      if (_currentOffset >= _total && _total > 0) {
        _hasMore = false;
      }
    } catch (e) {
      debugPrint('Error loading images: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  void setViewMode(ViewMode mode) {
    _viewMode = mode;
    notifyListeners();
  }

  Future<void> setSort(SortField field, bool asc) async {
    debugPrint('setSort 被调用: field=${field.name}, asc=$asc');
    _sortField = field;
    _sortAsc = asc;
    // Reset pagination when sort changes
    _currentOffset = 0;
    _hasMore = true;
    await loadImages(refresh: true);
  }

  Future<void> applyFilter(GalleryFilterState next) async {
    _filter = next.normalized();
    _resetPaginationForFilterChange();
    notifyListeners();
    await loadImages(refresh: true);
  }

  /// Sets the tag filter and reloads images with the new filter
  /// Preserves current sort settings and resets pagination
  Future<void> setTagFilter(List<int> tagIds) async {
    debugPrint('setTagFilter 被调用: tagIds=$tagIds');
    await applyFilter(_filter.copyWith(exactTagIds: tagIds.toSet()));
  }

  /// Sets the hasTags filter and reloads images with the new filter
  /// When hasTags is false, shows only untagged images
  /// When hasTags is null, clears the filter
  /// Preserves current sort settings and resets pagination
  Future<void> setHasTagsFilter(bool? hasTags) async {
    debugPrint('setHasTagsFilter 被调用: hasTags=$hasTags');
    await applyFilter(
      _filter.copyWith(
        hasTags: hasTags,
        hasPendingTags: hasTags == false ? null : _filter.hasPendingTags,
      ),
    );
  }

  Future<void> setHasPendingTagsFilter(bool? hasPendingTags) async {
    debugPrint('setHasPendingTagsFilter 被调用: hasPendingTags=$hasPendingTags');
    await applyFilter(
      _filter.copyWith(
        hasPendingTags: hasPendingTags,
        hasTags: hasPendingTags == true ? null : _filter.hasTags,
      ),
    );
  }

  void _resetPaginationForFilterChange() {
    _currentOffset = 0;
    _hasMore = true;
    _images = [];
  }

  void removeImageById(int imageId) {
    final before = _images.length;
    _images = _images.where((image) => image.id != imageId).toList();
    if (_images.length != before) {
      _total = _total > 0 ? _total - 1 : 0;
      if (_currentOffset > _images.length) {
        _currentOffset = _images.length;
      }
      notifyListeners();
    }
  }
}
