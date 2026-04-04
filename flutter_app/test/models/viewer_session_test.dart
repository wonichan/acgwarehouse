import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/models/viewer_session.dart';

void main() {
  group('ViewerSession', () {
    test(
      'builds a result-set-scoped session with ordered items and selected index',
      () {
        final images = [
          _image(id: 101, filename: 'alpha.png'),
          _image(id: 202, filename: 'beta.png'),
          _image(id: 303, filename: 'gamma.png'),
        ];

        final session = ViewerSession.fromResultSet(
          source: ViewerSessionSource.gallery,
          images: images,
          selectedImageId: 202,
        );

        expect(session.source, ViewerSessionSource.gallery);
        expect(session.initialSelectedIndex, 1);
        expect(
          session.items.map((item) => item.imageId),
          orderedEquals([101, 202, 303]),
        );
        expect(session.items[1].filename, 'beta.png');
        expect(session.items[1].path, 'C:/images/beta.png');
      },
    );

    test('round-trips JSON safely for spawned viewer windows', () {
      final session = ViewerSession.fromResultSet(
        source: ViewerSessionSource.search,
        images: [
          _image(id: 11, filename: 'search-a.jpg'),
          _image(id: 12, filename: 'search-b.jpg'),
        ],
        selectedImageId: 12,
      );

      final restored = ViewerSession.fromJson(session.toJson());

      expect(restored.source, ViewerSessionSource.search);
      expect(restored.initialSelectedIndex, 1);
      expect(restored.items, hasLength(2));
      expect(restored.items.first.filename, 'search-a.jpg');
      expect(restored.items.last.thumbnailLargeUrl, '/thumbs/search-b.jpg');
    });
  });
}

ImageModel _image({required int id, required String filename}) {
  final stem = filename.split('.').first;
  return ImageModel(
    id: id,
    path: 'C:/images/$filename',
    filename: filename,
    sourceRoot: 'C:/images',
    fileSize: 4096,
    width: 1920,
    height: 1080,
    format: filename.split('.').last,
    phash: id * 10,
    thumbnailSmallUrl: '/thumbs/$stem-small.jpg',
    thumbnailLargeUrl: '/thumbs/$stem.jpg',
    createdAt: DateTime.parse('2026-04-05T00:00:00Z'),
    updatedAt: DateTime.parse('2026-04-05T00:00:00Z'),
  );
}
