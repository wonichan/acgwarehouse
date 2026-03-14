import 'package:flutter/foundation.dart';
import '../models/image.dart';
import '../services/api_service.dart';

enum ViewMode { grid, masonry }

enum SortField { createdAt, filename, fileSize }

class ImageListProvider extends ChangeNotifier {
  final ApiService _apiService;
  
  List<ImageModel> _images = [];
  bool _isLoading = false;
  bool _hasMore = true;
  String? _nextCursor;
  ViewMode _viewMode = ViewMode.grid;
  SortField _sortField = SortField.createdAt;
  bool _sortAsc = false;
  
  ImageListProvider(this._apiService);
  
  List<ImageModel> get images => _images;
  bool get isLoading => _isLoading;
  bool get hasMore => _hasMore;
  ViewMode get viewMode => _viewMode;
  SortField get sortField => _sortField;
  bool get sortAsc => _sortAsc;
  
  Future<void> loadImages({bool refresh = false}) async {
    if (_isLoading) return;
    if (!refresh && !_hasMore) return;
    
    _isLoading = true;
    notifyListeners();
    
    try {
      final response = await _apiService.fetchImages(
        cursor: refresh ? null : _nextCursor,
        sortBy: _sortField.name == 'createdAt' ? 'created_at' : 
                _sortField.name == 'fileSize' ? 'file_size' : 'filename',
        sortDir: _sortAsc ? 'asc' : 'desc',
      );
      
      if (refresh) {
        _images = response.items;
      } else {
        _images.addAll(response.items);
      }
      _nextCursor = response.nextCursor;
      _hasMore = response.hasMore;
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
  
  void setSort(SortField field, bool asc) {
    _sortField = field;
    _sortAsc = asc;
    loadImages(refresh: true);
  }
}
