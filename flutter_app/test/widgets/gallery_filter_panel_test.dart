import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/gallery_filter_panel.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _TrackingImageListProvider extends ImageListProvider {
  _TrackingImageListProvider() : super(ApiService());

  int setTagFilterCalls = 0;
  int setHasTagsFilterCalls = 0;
  List<int> lastTagFilter = const [];
  bool? lastHasTags;

  @override
  Future<void> setTagFilter(List<int> tagIds) async {
    setTagFilterCalls++;
    lastTagFilter = List<int>.from(tagIds);
  }

  @override
  Future<void> setHasTagsFilter(bool? hasTags) async {
    setHasTagsFilterCalls++;
    lastHasTags = hasTags;
  }
}

class _PanelTagProvider extends TagProvider {
  _PanelTagProvider()
    : _tags = [
        Tag(
          id: 1,
          preferredLabel: 'tag-a',
          slug: 'tag-a',
          reviewState: 'approved',
          trustScore: 0.9,
          usageCount: 10,
          createdAt: DateTime(2026),
        ),
      ],
      super(TagService());

  final List<Tag> _tags;
  final Set<int> _selected = {};

  @override
  List<Tag> get allTags => _tags;

  @override
  Set<int> get selectedTagIds => _selected;

  @override
  bool get isLoading => false;

  @override
  Future<void> loadTags() async {}

  @override
  void toggleTag(int tagId) {
    if (_selected.contains(tagId)) {
      _selected.remove(tagId);
    } else {
      _selected.add(tagId);
    }
    notifyListeners();
  }

  @override
  void clearSelection() {
    _selected.clear();
    notifyListeners();
  }
}

void main() {
  testWidgets('gallery workspace keeps persistent right-side filter panel', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":1,"preferred_label":"tag-a","slug":"a","review_state":"approved","trust_score":0.9,"usage_count":12,"created_at":"2026-01-01T00:00:00Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => ImageListProvider(ApiService(client: mockClient)),
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
          ),
          ChangeNotifierProvider<NavigationProvider>(
            create: (_) => NavigationProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: FluentGalleryPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Filter by Tags'), findsOneWidget);
    expect(find.text('Show untagged images only'), findsOneWidget);
  });

  testWidgets('filter controls are keyboard-reachable Fluent controls', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => ImageListProvider(ApiService(client: mockClient)),
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
          ),
          ChangeNotifierProvider<NavigationProvider>(
            create: (_) => NavigationProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: FluentGalleryPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(fluent.ToggleSwitch), findsOneWidget);
  });

  testWidgets('selecting tag applies filter immediately without apply button', (
    tester,
  ) async {
    final imageProvider = _TrackingImageListProvider();
    final tagProvider = _PanelTagProvider();

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>.value(value: imageProvider),
          ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
        ],
        child: const fluent.FluentApp(
          home: fluent.SizedBox(
            width: 320,
            height: 600,
            child: GalleryFilterPanel(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('tag-a'));
    await tester.pumpAndSettle();

    expect(imageProvider.setTagFilterCalls, 1);
    expect(imageProvider.lastTagFilter, [1]);
  });

  testWidgets('toggling untagged-only applies hasTags false immediately', (
    tester,
  ) async {
    final imageProvider = _TrackingImageListProvider();
    final tagProvider = _PanelTagProvider();

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>.value(value: imageProvider),
          ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
        ],
        child: const fluent.FluentApp(
          home: fluent.SizedBox(
            width: 320,
            height: 600,
            child: GalleryFilterPanel(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ToggleSwitch));
    await tester.pumpAndSettle();

    expect(imageProvider.setHasTagsFilterCalls, 1);
    expect(imageProvider.lastHasTags, false);
  });
}
