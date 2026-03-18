import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/models/image.dart';

import 'image_provider_test.mocks.dart';

@GenerateMocks([ApiService])
void main() {
  late ImageListProvider provider;
  late MockApiService mockApiService;

  setUp(() {
    mockApiService = MockApiService();
    provider = ImageListProvider(mockApiService);
  });

  group('ImageListProvider', () {
    group('setTagFilter', () {
      test('stores selected tag IDs', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Act
        await provider.setTagFilter([1, 2, 3]);

        // Assert
        expect(provider.selectedTagIds, [1, 2, 3]);
      });

      test('resets pagination and reloads images with tag filter', () async {
        // Arrange
        final images = [
          ImageModel(
            id: 1,
            path: '/path/to/image.jpg',
            filename: 'image.jpg',
            sourceRoot: '/path/to',
            fileSize: 1024,
            width: 100,
            height: 100,
            format: 'jpg',
            phash: 0,
            createdAt: DateTime(2024),
            updatedAt: DateTime(2024),
          ),
        ];
        
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: images, nextCursor: null, hasMore: false, total: 1)
        );

        // Pre-load some images
        await provider.loadImages();
        
        // Act
        await provider.setTagFilter([5, 10]);

        // Assert
        expect(provider.selectedTagIds, [5, 10]);
        verify(mockApiService.fetchImages(
          offset: 0,  // Reset offset
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: [5, 10],
        )).called(1);
      });

      test('clears tag filter when passed empty list', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set initial filter
        await provider.setTagFilter([1, 2]);
        
        // Act - clear filter
        await provider.setTagFilter([]);

        // Assert
        expect(provider.selectedTagIds, isEmpty);
        verify(mockApiService.fetchImages(
          offset: 0,
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).called(greaterThanOrEqualTo(1));
      });

      test('preserves sort settings when changing tag filter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set sort
        provider.setSort(SortField.filename, true);
        
        // Act
        await provider.setTagFilter([1]);

        // Assert - sort should still be filename/asc
        expect(provider.sortField, SortField.filename);
        expect(provider.sortAsc, true);
      });
    });

    group('loadImages with tag filter', () {
      test('passes current tagIds to API when loading more', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set tag filter
        provider.setTagFilter([7, 8]);
        await provider.loadImages(refresh: true);

        // Assert
        verify(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: [7, 8],
        )).called(1);
      });
    });

    group('pagination state machine', () {
      test('prevents duplicate in-flight loads', () async {
        // Arrange
        final completer = Future.delayed(const Duration(milliseconds: 100), () => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );
        
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) => completer);

        // Act - start loading twice without waiting
        final future1 = provider.loadImages();
        final future2 = provider.loadImages();
        
        await Future.wait([future1, future2]);

        // Assert - API should only be called once
        verify(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).called(1);
      });

      test('stops loading when hasMore is false', () async {
        // Create test images
        final testImages = List.generate(
          10,
          (i) => ImageModel(
            id: i + 1,
            path: '/path/to/image$i.jpg',
            filename: 'image$i.jpg',
            sourceRoot: '/path/to',
            fileSize: 1024,
            width: 100,
            height: 100,
            format: 'jpg',
            phash: 0,
            createdAt: DateTime(2024),
            updatedAt: DateTime(2024),
          ),
        );

        // Arrange - first load returns 10 items with hasMore=true
        when(mockApiService.fetchImages(
          offset: 0,
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: testImages, nextCursor: '10', hasMore: true, total: 15)
        );

        // Load first page
        await provider.loadImages();
        expect(provider.hasMore, true);
        expect(provider.images.length, 10);

        // Arrange - second load returns remaining items with hasMore=false
        when(mockApiService.fetchImages(
          offset: 10,
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: testImages.sublist(0, 5), nextCursor: '', hasMore: false, total: 15)
        );

        // Load second page
        await provider.loadImages();
        expect(provider.hasMore, false);
        expect(provider.images.length, 15);

        // Arrange - reset mock for third call attempt
        reset(mockApiService);
        
        // Try to load again - should not call API
        await provider.loadImages();
        verifyNever(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        ));
      });

      test('resets pagination on refresh', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: '20', hasMore: true, total: 40)
        );

        // Load first page
        await provider.loadImages();
        expect(provider.hasMore, true);

        // Act - refresh
        await provider.loadImages(refresh: true);

        // Assert - offset should reset to 0
        verify(mockApiService.fetchImages(
          offset: 0,
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).called(2); // Once for initial load, once for refresh
      });

      test('resets pagination when sort changes', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Load first
        await provider.loadImages();

        // Act - change sort
        await provider.setSort(SortField.fileSize, true);

        // Assert - offset should be 0
        verify(mockApiService.fetchImages(
          offset: 0,
          sortBy: 'file_size',
          sortDir: 'asc',
          tagIds: anyNamed('tagIds'),
        )).called(1);
      });
    });
  });
}