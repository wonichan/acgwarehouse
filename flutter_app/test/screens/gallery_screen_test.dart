import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/screens/gallery_screen.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_grid.dart';
import 'package:provider/provider.dart';

class _FakeTagService extends TagService {
  bool called = false;

  @override
  Future<Map<String, dynamic>> batchTriggerAITags(List<int> imageIds) async {
    called = true;
    return {'job_ids': [1, 2], 'image_ids': imageIds};
  }
}

class _FakeImageListProvider extends ImageListProvider {
  _FakeImageListProvider() : super(ApiService());

  @override
  List<ImageModel> get images => [
        ImageModel(
          id: 1,
          path: 'p1',
          filename: 'a.png',
          sourceRoot: 'root',
          fileSize: 100,
          width: 100,
          height: 100,
          format: 'png',
          phash: 1,
          thumbnailSmallUrl: 'https://example.com/1.jpg',
          thumbnailLargeUrl: null,
          createdAt: DateTime(2026),
          updatedAt: DateTime(2026),
        ),
      ];

  @override
  bool get isLoading => false;

  @override
  bool get hasMore => false;

  @override
  int get total => 1;

  @override
  ViewMode get viewMode => ViewMode.grid;

  @override
  SortField get sortField => SortField.createdAt;

  @override
  bool get sortAsc => false;

  @override
  List<int> get selectedTagIds => const [];

  @override
  bool? get hasTagsFilter => null;

  @override
  Future<void> loadImages({bool refresh = false}) async {}

  @override
  void setViewMode(ViewMode mode) {}

  @override
  Future<void> setSort(SortField field, bool asc) async {}

  @override
  Future<void> setTagFilter(List<int> tagIds) async {}

  @override
  Future<void> setHasTagsFilter(bool? hasTags) async {}
}

class _TestTagProvider extends TagProvider {
  _TestTagProvider(this._tagService) : super(_tagService);

  final _FakeTagService _tagService;

  @override
  TagService get tagService => _tagService;
}

void main() {
  group('GalleryScreen', () {
    testWidgets('builds without error', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert - widget should build without throwing
      expect(find.byType(GalleryScreen), findsOneWidget);
    });

    testWidgets('has app bar with title', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert
      expect(find.text('ACGWarehouse'), findsOneWidget);
    });

    testWidgets('shows action buttons in app bar', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert - has filter, sort, and manage tags buttons
      expect(find.byIcon(Icons.filter_list), findsOneWidget);
      expect(find.byIcon(Icons.sort), findsOneWidget);
      expect(find.byIcon(Icons.label_outline), findsOneWidget);
    });

    testWidgets('enters selection mode from grid long press', (tester) async {
      final selectionProvider = SelectionProvider();

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(_FakeTagService()),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();
      expect(find.byType(ImageGrid), findsOneWidget);

      await tester.longPress(find.byKey(const ValueKey('image-1')));
      await tester.pumpAndSettle();

      expect(selectionProvider.isSelectionMode, isTrue);
    });

    testWidgets('shows batch action when selection exists', (tester) async {
      final selectionProvider = SelectionProvider()..enterSelectionMode();
      selectionProvider.toggleSelection(1);

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(_FakeTagService()),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();
      expect(find.textContaining('批量操作'), findsOneWidget);
    });
  });

  group('Async AI Tag Generation', () {
    testWidgets('closes bottom sheet immediately when AI generate tapped', (tester) async {
      // Arrange
      final selectionProvider = SelectionProvider()..enterSelectionMode();
      selectionProvider.toggleSelection(1);
      selectionProvider.toggleSelection(2);

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(_FakeTagService()),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Tap batch action button to open bottom sheet
      await tester.tap(find.textContaining('批量操作'));
      await tester.pumpAndSettle();

      // Verify bottom sheet is showing
      expect(find.text('AI生成标签'), findsOneWidget);

      // Act - tap AI generate button using the icon's parent InkWell
      final aiIcon = find.byIcon(Icons.auto_awesome);
      expect(aiIcon, findsOneWidget);
      await tester.tap(aiIcon);
      await tester.pumpAndSettle();

      // Assert - bottom sheet should be closed
      expect(find.text('AI生成标签'), findsNothing);
    });

    testWidgets('shows snackbar immediately with correct message', (tester) async {
      // Arrange
      final selectionProvider = SelectionProvider()..enterSelectionMode();
      selectionProvider.toggleSelection(1);
      selectionProvider.toggleSelection(2);
      selectionProvider.toggleSelection(3);

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(_FakeTagService()),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Open batch operations
      await tester.tap(find.textContaining('批量操作'));
      await tester.pumpAndSettle();

      // Act - tap AI generate icon
      await tester.tap(find.byIcon(Icons.auto_awesome));
      await tester.pump();

      // Assert - snackbar should show immediately with correct message
      expect(find.textContaining('AI标签生成任务已在后台启动'), findsOneWidget);
      expect(find.textContaining('3张图片'), findsOneWidget);
    });

    testWidgets('exits selection mode immediately', (tester) async {
      // Arrange
      final selectionProvider = SelectionProvider()..enterSelectionMode();
      selectionProvider.toggleSelection(1);

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(_FakeTagService()),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();
      expect(selectionProvider.isSelectionMode, isTrue);

      // Open batch operations
      await tester.tap(find.textContaining('批量操作'));
      await tester.pumpAndSettle();

      // Act - tap AI generate icon
      await tester.tap(find.byIcon(Icons.auto_awesome));
      await tester.pump();

      // Assert - selection mode should be exited immediately
      expect(selectionProvider.isSelectionMode, isFalse);
    });

    testWidgets('fires API call without blocking UI', (tester) async {
      // Arrange - use a fake service that tracks calls
      final fakeTagService = _FakeTagService();
      final selectionProvider = SelectionProvider()..enterSelectionMode();
      selectionProvider.toggleSelection(1);
      selectionProvider.toggleSelection(2);

      await tester.pumpWidget(
        MaterialApp(
          home: GalleryScreen(
            imageListProvider: _FakeImageListProvider(),
            tagProvider: _TestTagProvider(fakeTagService),
            selectionProvider: selectionProvider,
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Open batch operations
      await tester.tap(find.textContaining('批量操作'));
      await tester.pumpAndSettle();

      // Reset the called flag before act
      fakeTagService.called = false;

      // Act - tap AI generate icon
      await tester.tap(find.byIcon(Icons.auto_awesome));
      await tester.pump();

      // Assert - API should have been called
      expect(fakeTagService.called, isTrue);
    });
  });
}
