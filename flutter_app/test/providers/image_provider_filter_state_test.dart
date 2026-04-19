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
  List<int>? lastExactTagIds;
  List<int>? lastSubtreeRootTagIds;
  bool? lastHasTags;
  bool? lastHasPendingTags;

  @override
  Future<PaginationResponse<ImageModel>> fetchImages({
    int offset = 0,
    int limit = 20,
    String sortBy = 'created_at',
    String sortDir = 'desc',
    List<int>? tagIds,
    List<int>? exactTagIds,
    List<int>? subtreeRootTagIds,
    bool? hasTags,
    bool? hasPendingTags,
  }) async {
    lastOffset = offset;
    lastLimit = limit;
    lastSortBy = sortBy;
    lastSortDir = sortDir;
    lastTagIds = tagIds;
    lastExactTagIds = exactTagIds;
    lastSubtreeRootTagIds = subtreeRootTagIds;
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
        expect(api.lastExactTagIds, [2, 7]);
        expect(api.lastHasTags, isNull);
        expect(api.lastHasPendingTags, isTrue);
      },
    );

    test(
      'applyFilter resolves conflicting special flags and keeps normal tags',
      () async {
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

      expect(provider.filter.exactTagIds, {1, 9});
      expect(provider.filter.subtreeRootTagIds, {20});
      expect(provider.filter.hasTags, isNull);
      expect(provider.filter.hasPendingTags, isTrue);
      expect(api.lastExactTagIds, [1, 9]);
      expect(api.lastSubtreeRootTagIds, [20]);
      expect(api.lastHasTags, isNull);
      expect(api.lastHasPendingTags, isTrue);
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
        expect(api.lastExactTagIds, [5]);
        expect(api.lastHasTags, isNull);
        expect(api.lastHasPendingTags, isTrue);
      },
    );

    test('setHasTagsFilter keeps exact tags and clears pending flag', () async {
      final api = RecordingApiService();
      final provider = ImageListProvider(api);

      await provider.applyFilter(
        GalleryFilterState(exactTagIds: {3}, hasPendingTags: true),
      );
      await provider.setHasTagsFilter(false);

      expect(provider.filter.exactTagIds, {3});
      expect(provider.filter.hasTags, isFalse);
      expect(provider.filter.hasPendingTags, isNull);
      expect(api.lastExactTagIds, [3]);
      expect(api.lastHasTags, isFalse);
      expect(api.lastHasPendingTags, isNull);
    });

    test(
      'setHasPendingTagsFilter(true) keeps exact tags and clears hasTags',
      () async {
        final api = RecordingApiService();
        final provider = ImageListProvider(api);

        await provider.applyFilter(
          GalleryFilterState(exactTagIds: {8}, hasTags: false),
        );
        await provider.setHasPendingTagsFilter(true);

        expect(provider.filter.exactTagIds, {8});
        expect(provider.filter.hasTags, isNull);
        expect(provider.filter.hasPendingTags, isTrue);
        expect(api.lastExactTagIds, [8]);
        expect(api.lastHasTags, isNull);
        expect(api.lastHasPendingTags, isTrue);
      },
    );

    test('applyFilter forwards subtree root tag ids to API', () async {
      final api = RecordingApiService();
      final provider = ImageListProvider(api);

      await provider.applyFilter(
        GalleryFilterState(
          exactTagIds: {1},
          subtreeRootTagIds: {5, 10},
        ),
      );

      expect(provider.filter.exactTagIds, {1});
      expect(provider.filter.subtreeRootTagIds, {5, 10});
      expect(api.lastExactTagIds, [1]);
      expect(api.lastSubtreeRootTagIds, [5, 10]);
    });

    test('applyFilter with empty state clears normal and special filters', () async {
      final api = RecordingApiService();
      final provider = ImageListProvider(api);

      await provider.applyFilter(
        GalleryFilterState(
          exactTagIds: {1},
          subtreeRootTagIds: {2},
          hasTags: false,
          hasPendingTags: true,
        ),
      );

      await provider.applyFilter(GalleryFilterState());

      expect(provider.filter.exactTagIds, isEmpty);
      expect(provider.filter.subtreeRootTagIds, isEmpty);
      expect(provider.filter.hasTags, isNull);
      expect(provider.filter.hasPendingTags, isNull);
      expect(api.lastExactTagIds, isNull);
      expect(api.lastSubtreeRootTagIds, isNull);
      expect(api.lastHasTags, isNull);
      expect(api.lastHasPendingTags, isNull);
    });
  });
}
