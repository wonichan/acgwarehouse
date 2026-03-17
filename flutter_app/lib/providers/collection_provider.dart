import 'package:flutter/foundation.dart';
import '../models/collection.dart';
import '../services/collection_service.dart';

class CollectionProvider extends ChangeNotifier {
  final CollectionService _collectionService;

  List<Collection> _collections = [];
  int? _selectedCollectionId;
  bool _isLoading = false;
  String? _error;

  CollectionProvider(this._collectionService);

  // Getters
  List<Collection> get collections => _collections;
  int? get selectedCollectionId => _selectedCollectionId;
  Collection? get selectedCollection => _collections.where((c) => c.id == _selectedCollectionId).firstOrNull;
  bool get isLoading => _isLoading;
  String? get error => _error;

  /// Loads all collections from the API
  Future<void> loadCollections() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _collections = await _collectionService.fetchCollections();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error loading collections: $e');
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  /// Creates a new collection
  Future<Collection?> createCollection(String name, {String? description}) async {
    try {
      final collection = await _collectionService.createCollection(name, description: description);
      _collections.insert(0, collection);
      notifyListeners();
      return collection;
    } catch (e) {
      _error = e.toString();
      debugPrint('Error creating collection: $e');
      rethrow;
    }
  }

  /// Updates an existing collection
  Future<void> updateCollection(int id, String name, {String? description}) async {
    try {
      final updated = await _collectionService.updateCollection(id, name, description: description);
      final index = _collections.indexWhere((c) => c.id == id);
      if (index != -1) {
        _collections[index] = updated;
        notifyListeners();
      }
    } catch (e) {
      _error = e.toString();
      debugPrint('Error updating collection: $e');
      rethrow;
    }
  }

  /// Deletes a collection
  Future<void> deleteCollection(int id) async {
    try {
      await _collectionService.deleteCollection(id);
      _collections.removeWhere((c) => c.id == id);
      if (_selectedCollectionId == id) {
        _selectedCollectionId = null;
      }
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      debugPrint('Error deleting collection: $e');
      rethrow;
    }
  }

  /// Adds an image to a collection
  Future<void> addImageToCollection(int collectionId, int imageId) async {
    try {
      await _collectionService.addImageToCollection(collectionId, imageId);
      // Update the image count locally
      final index = _collections.indexWhere((c) => c.id == collectionId);
      if (index != -1) {
        final collection = _collections[index];
        _collections[index] = collection.copyWith(
          imageCount: collection.imageCount + 1,
        );
        notifyListeners();
      }
    } catch (e) {
      _error = e.toString();
      debugPrint('Error adding image to collection: $e');
      rethrow;
    }
  }

  /// Removes an image from a collection
  Future<void> removeImageFromCollection(int collectionId, int imageId) async {
    try {
      await _collectionService.removeImageFromCollection(collectionId, imageId);
      // Update the image count locally
      final index = _collections.indexWhere((c) => c.id == collectionId);
      if (index != -1) {
        final collection = _collections[index];
        _collections[index] = collection.copyWith(
          imageCount: collection.imageCount > 0 ? collection.imageCount - 1 : 0,
        );
        notifyListeners();
      }
    } catch (e) {
      _error = e.toString();
      debugPrint('Error removing image from collection: $e');
      rethrow;
    }
  }

  /// Sets the cover image for a collection
  Future<void> setCoverImage(int collectionId, int imageId) async {
    try {
      await _collectionService.setCoverImage(collectionId, imageId);
      // Update locally
      final index = _collections.indexWhere((c) => c.id == collectionId);
      if (index != -1) {
        final collection = _collections[index];
        _collections[index] = collection.copyWith(coverImageId: imageId);
        notifyListeners();
      }
    } catch (e) {
      _error = e.toString();
      debugPrint('Error setting cover image: $e');
      rethrow;
    }
  }

  /// Selects a collection
  void selectCollection(int? collectionId) {
    _selectedCollectionId = collectionId;
    notifyListeners();
  }

  /// Clears the selection
  void clearSelection() {
    _selectedCollectionId = null;
    notifyListeners();
  }

  void clearError() {
    _error = null;
    notifyListeners();
  }
}