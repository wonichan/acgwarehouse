import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart' show Builder;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/collection.dart';
import 'package:gallery/services/collection_service.dart';
import 'package:gallery/widgets/image_collection_picker_dialog.dart';

class _FakeCollectionService extends CollectionService {
  _FakeCollectionService({this.collections = const [], this.createdCollection});

  final List<Collection> collections;
  final Collection? createdCollection;
  final List<int> addedCollectionIds = <int>[];
  final List<int> addedImageIds = <int>[];
  final List<String> createdNames = <String>[];

  @override
  Future<List<Collection>> fetchCollections({
    int limit = 20,
    int offset = 0,
  }) async {
    return collections;
  }

  @override
  Future<Collection> createCollection(
    String name, {
    String? description,
  }) async {
    createdNames.add(name);
    return createdCollection ??
        Collection(
          id: 5,
          name: name,
          description: description,
          coverImageId: null,
          imageCount: 0,
          createdAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
          updatedAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
        );
  }

  @override
  Future<void> addImageToCollection(int collectionId, int imageId) async {
    addedCollectionIds.add(collectionId);
    addedImageIds.add(imageId);
  }
}

void main() {
  bool? lastDialogResult;

  fluent.Widget buildDialogHost(CollectionService collectionService) {
    return fluent.FluentApp(
      home: fluent.ScaffoldPage(
        content: Builder(
          builder: (context) {
            return fluent.Button(
              onPressed: () async {
                lastDialogResult = await fluent.showDialog<bool>(
                  context: context,
                  builder: (context) {
                    return ImageCollectionPickerDialog(
                      imageId: 42,
                      collectionService: collectionService,
                    );
                  },
                );
              },
              child: const fluent.Text('打开收藏对话框'),
            );
          },
        ),
      ),
    );
  }

  setUp(() {
    lastDialogResult = null;
  });

  testWidgets('adds image to selected collection', (tester) async {
    final service = _FakeCollectionService(
      collections: <Collection>[
        Collection(
          id: 1,
          name: '默认合集',
          description: null,
          coverImageId: null,
          imageCount: 3,
          createdAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
          updatedAt: DateTime.parse('2026-04-07T00:00:00.000Z'),
        ),
      ],
    );

    await tester.pumpWidget(buildDialogHost(service));
    await tester.tap(find.text('打开收藏对话框'));
    await tester.pumpAndSettle();

    expect(find.text('默认合集 (3)'), findsOneWidget);

    await tester.tap(find.text('收藏'));
    await tester.pumpAndSettle();

    expect(service.addedCollectionIds, <int>[1]);
    expect(service.addedImageIds, <int>[42]);
    expect(lastDialogResult, isTrue);
  });

  testWidgets('creates collection and adds image in one flow', (tester) async {
    final service = _FakeCollectionService();

    await tester.pumpWidget(buildDialogHost(service));
    await tester.tap(find.text('打开收藏对话框'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(fluent.TextBox).first, '新收藏夹');
    await tester.tap(find.text('新建并收藏'));
    await tester.pumpAndSettle();

    expect(service.createdNames, <String>['新收藏夹']);
    expect(service.addedCollectionIds, <int>[5]);
    expect(service.addedImageIds, <int>[42]);
    expect(lastDialogResult, isTrue);
  });
}
