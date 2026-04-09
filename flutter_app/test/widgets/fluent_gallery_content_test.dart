import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter/material.dart' show Material;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/widgets/fluent_image_card.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/collection_service.dart';
import 'package:gallery/widgets/justified_image_grid.dart';
import 'package:gallery/widgets/fluent_gallery_content.dart';
import 'package:provider/provider.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class _TrackingImageListProvider extends ImageListProvider {
  _TrackingImageListProvider({
    required List<ImageModel> initialImages,
    required this.initialHasMore,
  }) : _initialImages = initialImages,
       super(ApiService(baseUrl: 'http://localhost:8080'));

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

class _MutableImageListProvider extends ImageListProvider {
  _MutableImageListProvider({required List<ImageModel> initialImages})
    : _images = List<ImageModel>.from(initialImages),
      super(ApiService(baseUrl: 'http://localhost:8080'));

  List<ImageModel> _images;

  @override
  List<ImageModel> get images => _images;

  @override
  bool get isLoading => false;

  @override
  bool get hasMore => false;

  @override
  ViewMode get viewMode => ViewMode.grid;

  @override
  Future<void> loadImages({bool refresh = false}) async {}

  @override
  void removeImageById(int imageId) {
    _images = _images.where((image) => image.id != imageId).toList();
    notifyListeners();
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
    // The UI now uses JustifiedImageGrid with ListView.builder
    expect(find.byType(JustifiedImageGrid), findsOneWidget);
    expect(find.byType(ListView), findsOneWidget);
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

  testWidgets('shows context menu and opens collection dialog on right click', (
    tester,
  ) async {
    final provider = _MutableImageListProvider(initialImages: [buildImage(1)]);
    final apiService = ApiService(
      baseUrl: 'http://localhost:8080',
      client: MockClient((request) async => http.Response('{}', 200)),
    );
    final collectionService = CollectionService(
      baseUrl: 'http://localhost:8080',
      client: MockClient((request) async {
        if (request.method == 'GET' &&
            request.url.path.endsWith('/api/v1/collections')) {
          return http.Response('{"collections":[]}', 200);
        }
        return http.Response('{}', 200);
      }),
    );

    await tester.pumpWidget(
      ChangeNotifierProvider<ImageListProvider>.value(
        value: provider,
        child: fluent.FluentApp(
          home: fluent.ScaffoldPage(
            content: Material(
              child: FluentGalleryContent(
                apiService: apiService,
                collectionService: collectionService,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(FluentImageCard), buttons: kSecondaryButton);
    await tester.pumpAndSettle();

    expect(find.text('打开源文件'), findsOneWidget);
    expect(find.text('收藏'), findsOneWidget);
    expect(find.text('删除源文件及缩略图'), findsOneWidget);

    await tester.tap(find.text('收藏'));
    await tester.pumpAndSettle();

    expect(find.text('收藏到合集'), findsOneWidget);
    expect(find.text('暂无合集'), findsOneWidget);
  });

  testWidgets(
    'permanent delete removes image from provider after confirmation',
    (tester) async {
      final provider = _MutableImageListProvider(
        initialImages: [buildImage(1)],
      );
      Uri? deleteRequest;
      final apiService = ApiService(
        baseUrl: 'http://localhost:8080',
        client: MockClient((request) async {
          if (request.method == 'DELETE' &&
              request.url.path.endsWith('/api/v1/images/1/permanent')) {
            deleteRequest = request.url;
            return http.Response('{"status":"deleted"}', 200);
          }
          return http.Response('{}', 200);
        }),
      );
      final collectionService = CollectionService(
        baseUrl: 'http://localhost:8080',
        client: MockClient(
          (request) async => http.Response('{"collections":[]}', 200),
        ),
      );

      await tester.pumpWidget(
        ChangeNotifierProvider<ImageListProvider>.value(
          value: provider,
          child: fluent.FluentApp(
            home: fluent.ScaffoldPage(
              content: Material(
                child: FluentGalleryContent(
                  apiService: apiService,
                  collectionService: collectionService,
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(FluentImageCard), findsOneWidget);

      await tester.tap(find.byType(FluentImageCard), buttons: kSecondaryButton);
      await tester.pumpAndSettle();
      await tester.tap(find.text('删除源文件及缩略图'));
      await tester.pumpAndSettle();

      expect(find.text('确认彻底删除'), findsOneWidget);
      await tester.tap(find.text('确认删除'));
      await tester.pumpAndSettle();

      expect(deleteRequest?.path, contains('/api/v1/images/1/permanent'));
      expect(find.byType(FluentImageCard), findsNothing);
      expect(find.text('图片已彻底删除'), findsOneWidget);
    },
  );
}
