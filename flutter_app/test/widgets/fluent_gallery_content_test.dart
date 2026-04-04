import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart';
import 'package:flutter/material.dart' show Material;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/widgets/fluent_image_card.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/widgets/fluent_gallery_content.dart';
import 'package:provider/provider.dart';
import 'package:flutter_staggered_grid_view/flutter_staggered_grid_view.dart';

class _TrackingImageListProvider extends ImageListProvider {
  _TrackingImageListProvider({
    required List<ImageModel> initialImages,
    required this.initialHasMore,
  }) : _initialImages = initialImages,
       super(ApiService());

  final List<ImageModel> _initialImages;
  final bool initialHasMore;
  int loadImagesCallCount = 0;
  ViewMode forcedViewMode = ViewMode.grid;

  @override
  List<ImageModel> get images => _initialImages;

  @override
  bool get isLoading => false;

  @override
  bool get hasMore => initialHasMore;

  @override
  ViewMode get viewMode => forcedViewMode;

  @override
  Future<void> loadImages({bool refresh = false}) async {
    loadImagesCallCount++;
  }
}

void main() {
  ImageModel buildImage(int id) {
    return ImageModel(
      id: id,
      path: '/test/$id.jpg',
      filename: '$id.jpg',
      sourceRoot: '/test',
      fileSize: 1024,
      width: 100,
      height: 100,
      format: 'jpg',
      phash: id,
      createdAt: DateTime(2026),
      updatedAt: DateTime(2026),
    );
  }

  testWidgets(
    'loads next page when initial gallery does not overflow viewport',
    (tester) async {
      tester.view.physicalSize = const Size(2400, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      final provider = _TrackingImageListProvider(
        initialImages: List.generate(20, (index) => buildImage(index + 1)),
        initialHasMore: true,
      );

      await tester.pumpWidget(
        ChangeNotifierProvider<ImageListProvider>.value(
          value: provider,
          child: const fluent.FluentApp(
            home: SizedBox.expand(child: FluentGalleryContent()),
          ),
        ),
      );

      await tester.pump();

      expect(provider.loadImagesCallCount, 1);
    },
  );

  testWidgets('uses grid as default rendering path', (tester) async {
    final provider = _TrackingImageListProvider(
      initialImages: List.generate(8, (index) => buildImage(index + 1)),
      initialHasMore: false,
    );

    await tester.pumpWidget(
      ChangeNotifierProvider<ImageListProvider>.value(
        value: provider,
        child: const fluent.FluentApp(
          home: SizedBox.expand(child: FluentGalleryContent()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(provider.viewMode, ViewMode.grid);
    expect(find.byType(GridView), findsOneWidget);
    expect(find.byType(MasonryGridView), findsNothing);

    final gridView = tester.widget<GridView>(find.byType(GridView));
    final gridDelegate =
        gridView.gridDelegate as SliverGridDelegateWithMaxCrossAxisExtent;
    expect(gridDelegate.maxCrossAxisExtent, 220);
    expect(gridDelegate.childAspectRatio, 1);
  });

  testWidgets('triggers onImageDoubleTap on double click', (tester) async {
    final provider = _TrackingImageListProvider(
      initialImages: [buildImage(1)],
      initialHasMore: false,
    );

    bool doubleTapped = false;

    await tester.pumpWidget(
      ChangeNotifierProvider<ImageListProvider>.value(
        value: provider,
        child: fluent.FluentApp(
          home: fluent.ScaffoldPage(
            content: Material(
              child: FluentGalleryContent(
                onImageDoubleTap: (image) => doubleTapped = true,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final card = find.byType(FluentImageCard).first;
    // double tap
    await tester.tap(card);
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(card);
    await tester.pump(const Duration(milliseconds: 500));

    expect(doubleTapped, isTrue);
  });
}
