import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/viewer_window_context.dart';
import 'package:gallery/models/viewer_window_result.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';
import 'package:gallery/screens/viewer/viewer_workspace.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';

void main() {
  group('ViewerWorkspace', () {
    late TagProvider tagProvider;
    late _FakeViewerApiService apiService;
    late List<int> tagLoads;

    setUp(() {
      tagLoads = [];
      tagProvider = TagProvider(
        _RecordingTagService(tagLoads, {
          1: {
            'confirmed': [_tag(id: 101, label: 'landscape')],
            'pending': const [],
            'rejected': const [],
          },
          2: {
            'confirmed': [_tag(id: 202, label: 'portrait')],
            'pending': const [],
            'rejected': const [],
          },
        }),
      );
      apiService = _FakeViewerApiService([
        ViewerWindowResult(
          items: [
            _image(id: 1, filename: '1.jpg'),
            _image(id: 2, filename: '2.jpg'),
          ],
          windowStartIndex: 0,
          selectedIndex: 0,
          selectedIndexInWindow: 0,
          total: 12,
          hasPrevious: false,
          hasNext: true,
        ),
        ViewerWindowResult(
          items: [
            _image(id: 1, filename: '1.jpg'),
            _image(id: 2, filename: '2.jpg'),
          ],
          windowStartIndex: 0,
          selectedIndex: 1,
          selectedIndexInWindow: 1,
          total: 12,
          hasPrevious: true,
          hasNext: true,
        ),
      ]);
    });

    tearDown(() {
      tagProvider.dispose();
    });

    testWidgets(
      'loads initial viewer window from backend and renders workspace layout',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 0, selectedImageId: 1),
                apiService: apiService,
                tagProvider: tagProvider,
              ),
            ),
          ),
        );

        expect(find.byType(CircularProgressIndicator), findsOneWidget);

        await tester.pump();

        expect(find.byType(ViewerWorkspace), findsOneWidget);
        expect(find.byType(ViewerMetadataSidebar), findsOneWidget);
        expect(find.text('Viewer - 1.jpg'), findsOneWidget);
        expect(find.text('1 of 12'), findsOneWidget);
        expect(apiService.requests.single.context.selectedImageId, 1);
        expect(tagLoads, [1]);
      },
    );

    testWidgets(
      'refreshes title, metadata, and tags when keyboard selection changes',
      (WidgetTester tester) async {
        final changedFilenames = <String>[];

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 0, selectedImageId: 1),
                apiService: apiService,
                tagProvider: tagProvider,
                onItemChanged: (item) {
                  changedFilenames.add(item.filename);
                },
              ),
            ),
          ),
        );

        await tester.pump();
        expect(find.text('landscape'), findsOneWidget);

        await tester.sendKeyEvent(LogicalKeyboardKey.arrowRight);
        await tester.pump();

        expect(apiService.requests, hasLength(2));
        expect(apiService.requests.last.context.selectedIndex, 1);
        expect(apiService.requests.last.context.selectedImageId, 2);
        expect(find.text('Viewer - 2.jpg'), findsOneWidget);
        expect(find.text('portrait'), findsOneWidget);
        expect(find.text('landscape'), findsNothing);
        expect(find.text('2 of 12'), findsOneWidget);
        expect(changedFilenames.last, '2.jpg');
        expect(tagLoads, [1, 2]);
      },
    );

    testWidgets(
      're-queries backend when moving next from the end of the local window',
      (WidgetTester tester) async {
        apiService = _FakeViewerApiService.sequence([
          ViewerWindowResult(
            items: [
              _image(id: 10, filename: '10.jpg'),
              _image(id: 11, filename: '11.jpg'),
            ],
            windowStartIndex: 10,
            selectedIndex: 11,
            selectedIndexInWindow: 1,
            total: 20,
            hasPrevious: true,
            hasNext: true,
          ),
          ViewerWindowResult(
            items: [
              _image(id: 12, filename: '12.jpg'),
              _image(id: 13, filename: '13.jpg'),
            ],
            windowStartIndex: 12,
            selectedIndex: 12,
            selectedIndexInWindow: 0,
            total: 20,
            hasPrevious: true,
            hasNext: true,
          ),
        ]);

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 11, selectedImageId: 11),
                apiService: apiService,
                tagProvider: tagProvider,
              ),
            ),
          ),
        );

        await tester.pump();
        await tester.sendKeyEvent(LogicalKeyboardKey.arrowRight);
        await tester.pump();

        expect(apiService.requests, hasLength(2));
        expect(apiService.requests.last.context.selectedIndex, 12);
        expect(find.text('Viewer - 12.jpg'), findsOneWidget);
      },
    );

    testWidgets(
      're-queries backend when moving previous from the start of the local window',
      (WidgetTester tester) async {
        apiService = _FakeViewerApiService.sequence([
          ViewerWindowResult(
            items: [
              _image(id: 10, filename: '10.jpg'),
              _image(id: 11, filename: '11.jpg'),
            ],
            windowStartIndex: 10,
            selectedIndex: 10,
            selectedIndexInWindow: 0,
            total: 20,
            hasPrevious: true,
            hasNext: true,
          ),
          ViewerWindowResult(
            items: [
              _image(id: 8, filename: '8.jpg'),
              _image(id: 9, filename: '9.jpg'),
            ],
            windowStartIndex: 8,
            selectedIndex: 9,
            selectedIndexInWindow: 1,
            total: 20,
            hasPrevious: true,
            hasNext: true,
          ),
        ]);

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 10, selectedImageId: 10),
                apiService: apiService,
                tagProvider: tagProvider,
              ),
            ),
          ),
        );

        await tester.pump();
        await tester.sendKeyEvent(LogicalKeyboardKey.arrowLeft);
        await tester.pump();

        expect(apiService.requests, hasLength(2));
        expect(apiService.requests.last.context.selectedIndex, 9);
        expect(find.text('Viewer - 9.jpg'), findsOneWidget);
      },
    );

    testWidgets(
      'shows explicit error state when a later backend reload fails',
      (WidgetTester tester) async {
        apiService = _FakeViewerApiService.steps([
          _ViewerApiStep.result(
            ViewerWindowResult(
              items: [
                _image(id: 1, filename: '1.jpg'),
                _image(id: 2, filename: '2.jpg'),
              ],
              windowStartIndex: 0,
              selectedIndex: 0,
              selectedIndexInWindow: 0,
              total: 12,
              hasPrevious: false,
              hasNext: true,
            ),
          ),
          _ViewerApiStep.error(
            ViewerWindowApiException(
              'viewer_window_failed',
              'reload failed',
              500,
            ),
          ),
        ]);

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 0, selectedImageId: 1),
                apiService: apiService,
                tagProvider: tagProvider,
              ),
            ),
          ),
        );

        await tester.pump();
        expect(find.text('Viewer - 1.jpg'), findsOneWidget);

        await tester.sendKeyEvent(LogicalKeyboardKey.arrowRight);
        await tester.pump();

        expect(find.text('Viewer - 1.jpg'), findsOneWidget);
        expect(
          find.text('Failed to load viewer window: reload failed'),
          findsOneWidget,
        );
        expect(find.text('Dismiss'), findsOneWidget);
      },
    );

    testWidgets(
      'shows retryable error state when initial viewer window load fails',
      (WidgetTester tester) async {
        apiService = _FakeViewerApiService(
          [],
          error: ViewerWindowApiException(
            'viewer_window_failed',
            'viewer failed',
            500,
          ),
        );

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(
                launchContext: _context(selectedIndex: 0, selectedImageId: 1),
                apiService: apiService,
                tagProvider: tagProvider,
              ),
            ),
          ),
        );

        await tester.pump();

        expect(find.text('Failed to load viewer window'), findsOneWidget);
        expect(find.text('viewer failed'), findsOneWidget);
        expect(find.text('Retry'), findsOneWidget);
      },
    );
  });
}

Tag _tag({required int id, required String label}) {
  return Tag(
    id: id,
    preferredLabel: label,
    slug: label,
    reviewState: 'confirmed',
    trustScore: 1,
    usageCount: 1,
    createdAt: DateTime.parse('2026-04-05T00:00:00Z'),
  );
}

ViewerWindowContext _context({
  required int selectedIndex,
  required int selectedImageId,
}) {
  return ViewerWindowContext.gallery(
    selectedIndex: selectedIndex,
    selectedImageId: selectedImageId,
    snapshot: const ViewerWindowGallerySnapshot(
      sortBy: 'created_at',
      sortDir: 'desc',
      tagIds: [],
      hasTags: null,
    ),
  );
}

class _RecordingTagService extends TagService {
  _RecordingTagService(this.loads, this.imageTagsById);

  final List<int> loads;
  final Map<int, Map<String, List<Tag>>> imageTagsById;

  @override
  Future<Map<String, List<Tag>>> getImageTags(int imageId) async {
    loads.add(imageId);
    return imageTagsById[imageId] ??
        {'confirmed': const [], 'pending': const [], 'rejected': const []};
  }
}

class _FakeViewerApiService extends ApiService {
  _FakeViewerApiService(this.results, {this.error})
    : _steps = results.map(_ViewerApiStep.result).toList(growable: false);

  _FakeViewerApiService.sequence(this.results)
    : error = null,
      _steps = results.map(_ViewerApiStep.result).toList(growable: false);

  _FakeViewerApiService.steps(List<_ViewerApiStep> steps)
    : results = const [],
      error = null,
      _steps = steps;

  final List<ViewerWindowResult> results;
  final ViewerWindowApiException? error;
  final List<_ViewerApiStep> _steps;
  final List<ViewerWindowRequest> requests = [];

  @override
  Future<ViewerWindowResult> fetchViewerWindow(
    ViewerWindowRequest request,
  ) async {
    requests.add(request);
    if (error != null) {
      throw error!;
    }
    final step = _steps[requests.length - 1];
    if (step.error != null) {
      throw step.error!;
    }
    return step.result!;
  }
}

class _ViewerApiStep {
  const _ViewerApiStep._({this.result, this.error});

  const _ViewerApiStep.result(ViewerWindowResult result)
    : this._(result: result);

  const _ViewerApiStep.error(ViewerWindowApiException error)
    : this._(error: error);

  final ViewerWindowResult? result;
  final ViewerWindowApiException? error;
}

ImageModel _image({required int id, required String filename}) {
  return ImageModel(
    id: id,
    path: '/some/path/$filename',
    filename: filename,
    sourceRoot: '/some',
    fileSize: id * 1024,
    width: 800,
    height: 600,
    format: 'jpeg',
    phash: id,
    thumbnailSmallUrl: null,
    thumbnailLargeUrl: null,
    createdAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
  );
}
