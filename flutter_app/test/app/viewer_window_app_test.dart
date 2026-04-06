import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/viewer_window_app.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/viewer_window_context.dart';
import 'package:gallery/models/viewer_window_result.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/services/viewer_window_service.dart';
import 'package:gallery/utils/window_manager.dart';

void main() {
  testWidgets('ViewerWindowApp renders viewer workspace content', (
    tester,
  ) async {
    final apiService = _FakeApiService(
      result: ViewerWindowResult(
        items: [_image(id: 1, filename: 'alpha.png')],
        windowStartIndex: 0,
        selectedIndex: 0,
        selectedIndexInWindow: 0,
        total: 25,
        hasPrevious: false,
        hasNext: true,
      ),
    );
    final recordedTitles = <String>[];
    final tagProvider = TagProvider(_FakeTagService());
    const context = ViewerWindowContext.gallery(
      selectedIndex: 0,
      selectedImageId: 1,
      snapshot: ViewerWindowGallerySnapshot(
        sortBy: 'created_at',
        sortDir: 'desc',
        tagIds: [],
        hasTags: null,
      ),
    );

    await tester.pumpWidget(
      ViewerWindowApp(
        bootstrapData: ViewerWindowBootstrapData(
          windowId: 1,
          context: context,
          session: buildLegacyViewerSession(
            context: context,
            title: ViewerWindowService.buildWindowTitle('placeholder.png'),
          ),
          policy: viewerWindowOptions(
            ViewerWindowService.buildWindowTitle('alpha.png'),
          ),
        ),
        apiService: apiService,
        tagProvider: tagProvider,
        titleSetter: (title) async {
          recordedTitles.add(title);
        },
      ),
    );
    await tester.pump();
    await tester.pump();

    expect(find.byType(fluent.FluentApp), findsOneWidget);
    expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
    expect(find.text('Viewer - alpha.png'), findsOneWidget);
    expect(find.text('Image Details'), findsOneWidget);
    expect(
      recordedTitles.last,
      ViewerWindowService.buildWindowTitle('alpha.png'),
    );
    expect(apiService.requests.single.context.selectedImageId, 1);

    tagProvider.dispose();
  });
}

class _FakeApiService extends ApiService {
  _FakeApiService({required this.result});

  final ViewerWindowResult result;
  final List<ViewerWindowRequest> requests = [];

  @override
  Future<ViewerWindowResult> fetchViewerWindow(
    ViewerWindowRequest request,
  ) async {
    requests.add(request);
    return result;
  }
}

class _FakeTagService extends TagService {
  @override
  Future<Map<String, List<Tag>>> getImageTags(int imageId) async {
    return {'confirmed': const [], 'pending': const [], 'rejected': const []};
  }
}

ImageModel _image({required int id, required String filename}) {
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
    thumbnailSmallUrl: null,
    thumbnailLargeUrl: null,
    createdAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
  );
}
