// test/widgets/responsive_image_grid_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/responsive_image_grid.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/providers/image_provider.dart' show ViewMode;
import 'package:provider/provider.dart';

void main() {
  group('ResponsiveImageGrid', () {
    final testImages = [
      ImageModel(
        id: 1,
        filename: 'test1.jpg',
        path: '/test1.jpg',
        sourceRoot: '/library',
        format: 'jpg',
        width: 100,
        height: 100,
        fileSize: 1024,
        phash: 12345,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        thumbnailSmallUrl: 'http://test.com/1.jpg',
      ),
      ImageModel(
        id: 2,
        filename: 'test2.jpg',
        path: '/test2.jpg',
        sourceRoot: '/library',
        format: 'jpg',
        width: 100,
        height: 100,
        fileSize: 1024,
        phash: 12345,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        thumbnailSmallUrl: 'http://test.com/2.jpg',
      ),
    ];

    Widget createTestWidget({ViewMode viewMode = ViewMode.grid}) {
      return MaterialApp(
        home: Scaffold(
          body: ChangeNotifierProvider(
            create: (_) => SelectionProvider(),
            child: ResponsiveImageGrid(images: testImages, viewMode: viewMode),
          ),
        ),
      );
    }

    testWidgets('uses ListView.builder with row layout', (tester) async {
      await tester.pumpWidget(createTestWidget());

      // Should find JustifiedImageGrid (the renamed widget)
      expect(find.byType(ResponsiveImageGrid), findsOneWidget);
      // Should use ListView.builder for row-based layout
      expect(find.byType(ListView), findsWidgets);
      // Should NOT use GridView (removed)
      expect(find.byType(GridView), findsNothing);
    });

    testWidgets('shows image tiles in rows', (tester) async {
      await tester.pumpWidget(createTestWidget());

      // Find image tiles by their key pattern
      expect(find.byKey(const ValueKey('image-1')), findsOneWidget);
      expect(find.byKey(const ValueKey('image-2')), findsOneWidget);
    });

    testWidgets('renders empty state when no images', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider(
              create: (_) => SelectionProvider(),
              child: ResponsiveImageGrid(
                images: const [],
                viewMode: ViewMode.grid,
              ),
            ),
          ),
        ),
      );

      expect(find.text('No images to display.'), findsOneWidget);
    });

    testWidgets('supports selection mode', (tester) async {
      final selectionProvider = SelectionProvider();
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider.value(
              value: selectionProvider,
              child: ResponsiveImageGrid(
                images: testImages,
                viewMode: ViewMode.grid,
                selectionProvider: selectionProvider,
              ),
            ),
          ),
        ),
      );

      // Enter selection mode by long pressing an image
      await tester.longPress(find.byKey(const ValueKey('image-1')));
      await tester.pumpAndSettle();

      // Should now be in selection mode
      expect(selectionProvider.isSelectionMode, isTrue);
    });
  });
}
