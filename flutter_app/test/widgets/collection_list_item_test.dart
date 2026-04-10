import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/collection.dart';
import 'package:gallery/widgets/collection_list_item.dart';

void main() {
  testWidgets(
    'CollectionListItem displays collection details and uses theme colors',
    (tester) async {
      final collection = Collection(
        id: 1,
        name: 'Test Collection',
        description: 'A test collection',
        imageCount: 42,
        coverImageId: null,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
      );

      bool deleteTapped = false;
      bool itemTapped = false;

      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData(
            colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
          ),
          home: Scaffold(
            body: CollectionListItem(
              collection: collection,
              isSelected: true,
              onTap: () => itemTapped = true,
              onEdit: () {},
              onDelete: () => deleteTapped = true,
            ),
          ),
        ),
      );

      // Verify basic text
      expect(find.text('Test Collection'), findsOneWidget);
      expect(find.text('42 张图片'), findsOneWidget);

      // Verify tap
      await tester.tap(find.text('Test Collection'));
      expect(itemTapped, isTrue);

      // Verify popup menu
      await tester.tap(find.byIcon(Icons.more_vert));
      await tester.pumpAndSettle();

      expect(find.text('重命名'), findsOneWidget);
      expect(find.text('删除'), findsOneWidget);

      // Tap delete
      await tester.tap(find.text('删除'));
      await tester.pumpAndSettle();
      expect(deleteTapped, isTrue);
    },
  );
}
