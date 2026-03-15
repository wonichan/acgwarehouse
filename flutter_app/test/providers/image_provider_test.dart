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
          cursor: anyNamed('cursor'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false)
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
          cursor: anyNamed('cursor'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: images, nextCursor: null, hasMore: false)
        );

        // Pre-load some images
        await provider.loadImages();
        
        // Act
        await provider.setTagFilter([5, 10]);

        // Assert
        expect(provider.selectedTagIds, [5, 10]);
        verify(mockApiService.fetchImages(
          cursor: null,  // Reset cursor
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: [5, 10],
        )).called(1);
      });

      test('clears tag filter when passed empty list', () async {
        // Arrange
        when(mockApiService.fetchImages(
          cursor: anyNamed('cursor'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false)
        );

        // Set initial filter
        await provider.setTagFilter([1, 2]);
        
        // Act - clear filter
        await provider.setTagFilter([]);

        // Assert
        expect(provider.selectedTagIds, isEmpty);
        verify(mockApiService.fetchImages(
          cursor: null,
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).called(greaterThanOrEqualTo(1));
      });

      test('preserves sort settings when changing tag filter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          cursor: anyNamed('cursor'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false)
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
          cursor: anyNamed('cursor'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
        )).thenAnswer((_) async => 
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false)
        );

        // Set tag filter
        provider.setTagFilter([7, 8]);
        await provider.loadImages(refresh: true);

        // Assert
        verify(mockApiService.fetchImages(
          cursor: anyNamed('cursor'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: [7, 8],
        )).called(1);
      });
    });
  });
}