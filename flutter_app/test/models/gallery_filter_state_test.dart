import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/gallery_filter_state.dart';

void main() {
  group('GalleryFilterState', () {
    test(
      'normalized clears tag selections and pending flag for untagged mode',
      () {
        final state = GalleryFilterState(
          exactTagIds: {1, 2},
          subtreeRootTagIds: {3},
          hasTags: false,
          hasPendingTags: true,
        );

        final normalized = state.normalized();

        expect(normalized.exactTagIds, isEmpty);
        expect(normalized.subtreeRootTagIds, isEmpty);
        expect(normalized.hasTags, isFalse);
        expect(normalized.hasPendingTags, isNull);
      },
    );
  });
}
