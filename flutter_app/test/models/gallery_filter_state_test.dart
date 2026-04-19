import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/gallery_filter_state.dart';

void main() {
  group('GalleryFilterState', () {
    test(
      'normalized keeps normal tags when hasTags is false',
      () {
        final state = GalleryFilterState(
          exactTagIds: {1, 2},
          subtreeRootTagIds: {3},
          hasTags: false,
        );

        final normalized = state.normalized();

        expect(normalized.exactTagIds, {1, 2});
        expect(normalized.subtreeRootTagIds, {3});
        expect(normalized.hasTags, isFalse);
        expect(normalized.hasPendingTags, isNull);
      },
    );

    test('normalized resolves conflicting special flags deterministically', () {
      final state = GalleryFilterState(
        exactTagIds: {9},
        subtreeRootTagIds: {4},
        hasTags: false,
        hasPendingTags: true,
      );

      final normalized = state.normalized();

      expect(normalized.exactTagIds, {9});
      expect(normalized.subtreeRootTagIds, {4});
      expect(normalized.hasTags, isNull);
      expect(normalized.hasPendingTags, isTrue);
    });
  });
}
