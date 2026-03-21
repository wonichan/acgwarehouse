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
            child: ResponsiveImageGrid(
              images: testImages,
              viewMode: viewMode,
            ),
          ),
        ),
      );
    }

    testWidgets('shows 2 columns on compact screen', (tester) async {
      // Set view size to compact (phone)
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      
      final gridView = tester.widget<GridView>(find.byType(GridView));
      final delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.crossAxisCount, 2);
    });

    testWidgets('shows 3 columns on medium screen', (tester) async {
      // Set view size to medium (tablet)
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      
      final gridView = tester.widget<GridView>(find.byType(GridView));
      final delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.crossAxisCount, 3);
    });

    testWidgets('shows 4 columns on expanded screen', (tester) async {
      // Set view size to expanded (large tablet)
      tester.view.physicalSize = const Size(900, 1200);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      
      final gridView = tester.widget<GridView>(find.byType(GridView));
      final delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.crossAxisCount, 4);
    });

    testWidgets('uses correct spacing for compact breakpoint', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      var gridView = tester.widget<GridView>(find.byType(GridView));
      var delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.mainAxisSpacing, 4);
      expect(delegate.crossAxisSpacing, 4);
    });

    testWidgets('uses correct spacing for medium breakpoint', (tester) async {
      tester.view.physicalSize = const Size(700, 1000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      var gridView = tester.widget<GridView>(find.byType(GridView));
      var delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.mainAxisSpacing, 8);
      expect(delegate.crossAxisSpacing, 8);
    });

    testWidgets('uses correct spacing for expanded breakpoint', (tester) async {
      tester.view.physicalSize = const Size(900, 1200);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget());
      var gridView = tester.widget<GridView>(find.byType(GridView));
      var delegate = gridView.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount;
      expect(delegate.mainAxisSpacing, 12);
      expect(delegate.crossAxisSpacing, 12);
    });

    testWidgets('uses MasonryGridView in masonry mode', (tester) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(createTestWidget(viewMode: ViewMode.masonry));
      
      // Should find MasonryGridView, not GridView
      expect(find.byType(GridView), findsNothing);
    });
  });
}