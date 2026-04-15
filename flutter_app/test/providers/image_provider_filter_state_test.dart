import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/gallery_filter_state.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/services/api_service.dart';

class RecordingApiService extends ApiService {
  RecordingApiService() : super(baseUrl: 'http://localhost:8080');

  int? lastOffset;
  int? lastLimit;
  String? lastSortBy;
  String? lastSortDir;
  List<int>? lastTagIds;
  bool? lastHasTags;
  bool? lastHasPendingTags;

  @override
  Future<PaginationResponse<ImageModel>> fetchImages({
    int offset = 0,
    int limit = 20,
    String sortBy = 'created_at',
    String sortDir = 'desc',
    List<int>? tagIds,
    bool? hasTags,
    bool? hasPendingTags,
  }) async {
    lastOffset = offset;
    lastLimit = limit;
    lastSortBy = sortBy;
    lastSortDir = sortDir;
    lastTagIds = tagIds;
    lastHasTags = hasTags;
    lastHasPendingTags = hasPendingTags;

    return PaginationResponse<ImageModel>(
      items: const [],
      nextCursor: null,
      hasMore: false,
      total: 0,
    );
  }
}

void main() {
  group('ImageListProvider filter state', () {
    test(
      'applyFilter stores unified filter and forwards exact tag ids',
      () async {
        final api = RecordingApiService();
        final provider = ImageListProvider(api);

        await provider.applyFilter(
          GalleryFilterState(exactTagIds: {2, 7}, hasPendingTags: true),
        );

        expect(provider.filter.exactTagIds, {2, 7});
        expect(provider.filter.hasPendingTags, isTrue);
        expect(provider.selectedTagIds, [2, 7]);
        expect(api.lastTagIds, [2, 7]);
        expect(api.lastHasTags, isNull);
        expect(api.lastHasPendingTags, isTrue);
      },
    );

    test('applyFilter normalizes untagged mode before calling API', () async {
      final api = RecordingApiService();
      final provider = ImageListProvider(api);

      await provider.applyFilter(
        GalleryFilterState(
          exactTagIds: {1, 9},
          subtreeRootTagIds: {20},
          hasTags: false,
          hasPendingTags: true,
        ),
      );

      expect(provider.filter.exactTagIds, isEmpty);
      expect(provider.filter.subtreeRootTagIds, isEmpty);
      expect(provider.filter.hasTags, isFalse);
      expect(provider.filter.hasPendingTags, isNull);
      expect(api.lastTagIds, isNull);
      expect(api.lastHasTags, isFalse);
      expect(api.lastHasPendingTags, isNull);
    });

    test(
      'setTagFilter preserves pending flag while clearing hasTags mode',
      () async {
        final api = RecordingApiService();
        final provider = ImageListProvider(api);

        await provider.setHasPendingTagsFilter(true);
        await provider.setTagFilter([5]);

        expect(provider.filter.exactTagIds, {5});
        expect(provider.filter.hasTags, isNull);
        expect(provider.filter.hasPendingTags, isTrue);
        expect(api.lastTagIds, [5]);
        expect(api.lastHasTags, isNull);
        expect(api.lastHasPendingTags, isTrue);
      },
    );

    test('setHasTagsFilter clears exact tags and pending flag', () async {
      final api = RecordingApiService();
      final provider = ImageListProvider(api);

      await provider.applyFilter(
        GalleryFilterState(exactTagIds: {3}, hasPendingTags: true),
      );
      await provider.setHasTagsFilter(false);

      expect(provider.filter.exactTagIds, isEmpty);
      expect(provider.filter.hasTags, isFalse);
      expect(provider.filter.hasPendingTags, isNull);
      expect(api.lastTagIds, isNull);
      expect(api.lastHasTags, isFalse);
      expect(api.lastHasPendingTags, isNull);
    });
  });
}
