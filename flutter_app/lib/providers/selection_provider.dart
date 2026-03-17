import 'package:flutter/foundation.dart';

/// Provider for managing batch selection state in the gallery
class SelectionProvider extends ChangeNotifier {
  final Set<int> _selectedImageIds = {};
  bool _isSelectionMode = false;
  int? _lastSelectedIndex;

  // Getters
  Set<int> get selectedImageIds => Set.unmodifiable(_selectedImageIds);
  bool get isSelectionMode => _isSelectionMode;
  int get selectedCount => _selectedImageIds.length;
  int? get lastSelectedIndex => _lastSelectedIndex;
  bool get hasSelection => _selectedImageIds.isNotEmpty;

  /// Checks if an image is selected
  bool isSelected(int imageId) => _selectedImageIds.contains(imageId);

  /// Toggles the selection state of an image
  void toggleSelection(int imageId) {
    if (_selectedImageIds.contains(imageId)) {
      _selectedImageIds.remove(imageId);
    } else {
      _selectedImageIds.add(imageId);
    }
    notifyListeners();
  }

  /// Selects a range of images (for shift+click selection)
  void selectRange(List<int> imageIds, int startIndex, int endIndex) {
    if (startIndex > endIndex) {
      final temp = startIndex;
      startIndex = endIndex;
      endIndex = temp;
    }

    for (int i = startIndex; i <= endIndex && i < imageIds.length; i++) {
      _selectedImageIds.add(imageIds[i]);
    }
    notifyListeners();
  }

  /// Selects all images
  void selectAll(List<int> imageIds) {
    _selectedImageIds.addAll(imageIds);
    notifyListeners();
  }

  /// Clears all selections
  void clearSelection() {
    _selectedImageIds.clear();
    _lastSelectedIndex = null;
    notifyListeners();
  }

  /// Enters selection mode
  void enterSelectionMode() {
    _isSelectionMode = true;
    notifyListeners();
  }

  /// Exits selection mode and clears selection
  void exitSelectionMode() {
    _isSelectionMode = false;
    clearSelection();
  }

  /// Sets the last selected index (for range selection)
  void setLastSelectedIndex(int? index) {
    _lastSelectedIndex = index;
  }

  /// Handles image tap based on current mode
  /// Returns true if the tap was handled as selection
  bool handleImageTap(int imageId, {bool longPress = false, int? index}) {
    if (longPress) {
      // Long press always enters selection mode and selects
      if (!_isSelectionMode) {
        enterSelectionMode();
      }
      toggleSelection(imageId);
      _lastSelectedIndex = index;
      return true;
    }

    if (_isSelectionMode) {
      toggleSelection(imageId);
      _lastSelectedIndex = index;
      return true;
    }

    return false;
  }
}