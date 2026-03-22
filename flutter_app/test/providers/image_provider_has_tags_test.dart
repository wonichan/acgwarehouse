import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/models/image.dart';

import 'image_provider_has_tags_test.mocks.dart';

@GenerateMocks([ApiService])
void main() {
  late ImageListProvider provider;
  late MockApiService mockApiService;

  setUp(() {
    mockApiService = MockApiService();
    provider = ImageListProvider(mockApiService);
  });

  group('ImageListProvider - hasTags filter', () {
    group('setHasTagsFilter', () {
      test('setHasTagsFilter(false) should call API with has_tags=false parameter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Act
        await provider.setHasTagsFilter(false);

        // Assert - API should be called with hasTags: false
        verify(mockApiService.fetchImages(
          offset: 0,
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: false,
        )).called(1);
      });

      test('setHasTagsFilter(null) should clear the filter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Act
        await provider.setHasTagsFilter(null);

        // Assert - API should be called without hasTags (or with null)
        expect(provider.hasTagsFilter, isNull);
        verify(mockApiService.fetchImages(
          offset: 0,
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).called(1);
      });

      test('setHasTagsFilter(false) should clear selected tag IDs (mutually exclusive)', () async {
        // Arrange - first set some tags
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set initial tag filter
        await provider.setTagFilter([1, 2, 3]);
        expect(provider.selectedTagIds, [1, 2, 3]);

        // Act - switch to hasTags filter
        await provider.setHasTagsFilter(false);

        // Assert - tag IDs should be cleared
        expect(provider.selectedTagIds, isEmpty);
      });

      test('setHasTagsFilter should reset pagination (offset=0, hasMore=true)', () async {
        // Arrange - set up initial pagination state
        final testImages = List.generate(
          20,
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

        // First load returns 20 items
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: testImages, nextCursor: '20', hasMore: true, total: 100)
        );

        // Load first page
        await provider.loadImages();
        expect(provider.images.length, 20);
        expect(provider.hasMore, true);

        // Act - change hasTags filter
        await provider.setHasTagsFilter(false);

        // Assert - pagination should be reset
        verify(mockApiService.fetchImages(
          offset: 0,  // Reset to 0
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: false,
        )).called(1);
      });

      test('setHasTagsFilter(true) should call API with has_tags=true parameter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Act
        await provider.setHasTagsFilter(true);

        // Assert - API should be called with hasTags: true
        verify(mockApiService.fetchImages(
          offset: 0,
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: true,
        )).called(1);
      });

      test('setHasTagsFilter should preserve sort settings', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set sort
        await provider.setSort(SortField.fileSize, true);

        // Act - change hasTags filter
        await provider.setHasTagsFilter(false);

        // Assert - sort should still be file_size/asc
        expect(provider.sortField, SortField.fileSize);
        expect(provider.sortAsc, true);
        verify(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          sortBy: 'file_size',
          sortDir: 'asc',
          tagIds: anyNamed('tagIds'),
          hasTags: false,
        )).called(1);
      });
    });

    group('setTagFilter mutual exclusivity', () {
      test('setTagFilter should clear hasTagsFilter when setting tag filter', () async {
        // Arrange
        when(mockApiService.fetchImages(
          offset: anyNamed('offset'),
          limit: anyNamed('limit'),
          sortBy: anyNamed('sortBy'),
          sortDir: anyNamed('sortDir'),
          tagIds: anyNamed('tagIds'),
          hasTags: anyNamed('hasTags'),
        )).thenAnswer((_) async =>
          PaginationResponse<ImageModel>(items: [], nextCursor: null, hasMore: false, total: 0)
        );

        // Set initial hasTags filter
        await provider.setHasTagsFilter(false);
        expect(provider.hasTagsFilter, false);

        // Act - switch to tag filter
        await provider.setTagFilter([1, 2]);

        // Assert - hasTagsFilter should be cleared
        expect(provider.hasTagsFilter, isNull);
      });
    });
  });
}
