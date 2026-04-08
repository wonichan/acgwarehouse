import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/collection.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/services/collection_service.dart';
import 'package:gallery/widgets/fluent_collections_content.dart';
import 'package:gallery/widgets/fluent_image_card.dart';

class _FakeCollectionService extends CollectionService {
  _FakeCollectionService({
    this.collections = const <Collection>[],
    this.imagesByCollectionId = const <int, List<ImageModel>>{},
  });

  final List<Collection> collections;
  final Map<int, List<ImageModel>> imagesByCollectionId;
  final List<int> fetchedCollectionImageIds = <int>[];

  @override
  Future<List<Collection>> fetchCollections({
    int limit = 20,
    int offset = 0,
  }) async {
    return collections;
  }

  @override
  Future<List<ImageModel>> fetchCollectionImages(
    int collectionId, {
    int limit = 20,
    int offset = 0,
  }) async {
    fetchedCollectionImageIds.add(collectionId);
    return imagesByCollectionId[collectionId] ?? const <ImageModel>[];
  }
}

Collection _buildCollection({
  required int id,
  required String name,
  required int imageCount,
}) {
  return Collection(
    id: id,
    name: name,
    description: null,
    coverImageId: null,
    imageCount: imageCount,
    createdAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
  );
}

ImageModel _buildImage(int id, String filename) {
  return ImageModel(
    id: id,
    path: 'C:/images/$filename',
    filename: filename,
    sourceRoot: 'C:/images',
    fileSize: 2048,
    width: 800,
    height: 600,
    format: 'png',
    phash: id,
    createdAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
  );
}

void main() {
  testWidgets('shows a clear empty state when no collections exist', (
    tester,
  ) async {
    await tester.pumpWidget(
      fluent.FluentApp(
        home: FluentCollectionsContent(
          collectionService: _FakeCollectionService(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('暂无合集'), findsOneWidget);
    expect(find.text('请先在图片上右键选择“收藏”'), findsOneWidget);
  });

  testWidgets('switches collections and shows images or an empty state', (
    tester,
  ) async {
    final service = _FakeCollectionService(
      collections: <Collection>[
        _buildCollection(id: 1, name: '角色合集', imageCount: 1),
        _buildCollection(id: 2, name: '空合集', imageCount: 0),
      ],
      imagesByCollectionId: <int, List<ImageModel>>{
        1: <ImageModel>[_buildImage(1, 'alpha.png')],
        2: const <ImageModel>[],
      },
    );

    await tester.pumpWidget(
      fluent.FluentApp(
        home: FluentCollectionsContent(collectionService: service),
      ),
    );
    await tester.pumpAndSettle();

    expect(service.fetchedCollectionImageIds, <int>[1]);
    expect(find.text('角色合集'), findsWidgets);
    expect(find.byType(FluentImageCard), findsOneWidget);

    await tester.tap(find.text('空合集'));
    await tester.pumpAndSettle();

    expect(service.fetchedCollectionImageIds, <int>[1, 2]);
    expect(find.byType(FluentImageCard), findsNothing);
    expect(find.text('该合集暂无图片'), findsOneWidget);
  });
}
