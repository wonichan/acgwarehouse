import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/models/viewer_window_context.dart';
import 'package:gallery/services/viewer_window_service.dart';
import 'package:gallery/utils/window_manager.dart';

void main() {
  group('ViewerWindowService', () {
    test('formats viewer window title with exact Phase 18 copy', () {
      expect(
        ViewerWindowService.buildWindowTitle('sample.png'),
        'ACGWarehouse Viewer — sample.png',
      );
    });

    test(
      'maps launch requests to centered viewer defaults without persistence',
      () async {
        final adapter = FakeViewerWindowAdapter();
        final service = ViewerWindowService(adapter: adapter);
        final context = ViewerWindowContext.gallery(
          selectedIndex: 0,
          selectedImageId: 1,
          snapshot: const ViewerWindowGallerySnapshot(
            sortBy: 'created_at',
            sortDir: 'desc',
            tagIds: [5, 8],
            hasTags: null,
          ),
        );

        await service.openWindow(
          selectedFilename: 'alpha.png',
          context: context,
        );

        expect(adapter.launches, hasLength(1));
        expect(adapter.restoreStateCalls, 0);
        expect(adapter.saveStateCalls, 0);

        final launch = adapter.launches.single;
        expect(launch.title, 'ACGWarehouse Viewer — alpha.png');
        expect(launch.windowId, 'viewer-window-1');
        expect(
          launch.policy,
          viewerWindowOptions('ACGWarehouse Viewer — alpha.png'),
        );
        expect(launch.policy.center, isTrue);
        expect(launch.policy.size, const Size(1440, 900));
      },
    );

    test(
      'issues distinct launch requests for multiple viewer sessions',
      () async {
        final adapter = FakeViewerWindowAdapter();
        final service = ViewerWindowService(adapter: adapter);

        await service.openWindow(
          selectedFilename: 'alpha.png',
          context: ViewerWindowContext.gallery(
            selectedIndex: 0,
            selectedImageId: 1,
            snapshot: const ViewerWindowGallerySnapshot(
              sortBy: 'created_at',
              sortDir: 'desc',
              tagIds: [],
              hasTags: null,
            ),
          ),
        );
        await service.openWindow(
          selectedFilename: 'beta.png',
          context: ViewerWindowContext.search(
            selectedIndex: 3,
            selectedImageId: 2,
            snapshot: const ViewerWindowSearchSnapshot(
              query: 'beta',
              tagIds: [9],
              sortBy: 'relevance',
              sortOrder: 'desc',
            ),
          ),
        );

        expect(adapter.launches, hasLength(2));
        expect(adapter.launches.first.windowId, 'viewer-window-1');
        expect(adapter.launches.last.windowId, 'viewer-window-2');
        expect(adapter.launches.first.title, 'ACGWarehouse Viewer — alpha.png');
        expect(adapter.launches.last.title, 'ACGWarehouse Viewer — beta.png');
      },
    );

    test(
      'encodes viewer-window bootstrap payload for spawned windows',
      () async {
        final adapter = FakeViewerWindowAdapter();
        final service = ViewerWindowService(adapter: adapter);
        final context = ViewerWindowContext.search(
          selectedIndex: 4,
          selectedImageId: 9,
          snapshot: const ViewerWindowSearchSnapshot(
            query: 'launch target',
            tagIds: [4, 5],
            sortBy: 'relevance',
            sortOrder: 'desc',
          ),
        );

        await service.openWindow(
          selectedFilename: 'launch-target.png',
          context: context,
        );

        final payload = adapter.launches.single.arguments;
        expect(payload['kind'], 'viewer-window');
        expect(payload['context']['source'], 'search');
        expect(payload['context']['selected_index'], 4);
        expect(payload['context']['selected_image_id'], 9);
        expect(payload['context']['snapshot']['q'], 'launch target');
        expect(payload['context']['snapshot']['tag_ids'], [4, 5]);
      },
    );

    test('parses viewer-window bootstrap arguments for secondary startup', () {
      final data = ViewerWindowBootstrapData.fromCommandLine([
        'multi_window',
        '7',
        '{"kind":"viewer-window","context":{"source":"gallery","selected_index":12,"selected_image_id":1,"snapshot":{"sort_by":"created_at","sort_dir":"desc","tag_ids":[1,2],"has_tags":false}}}',
      ]);

      expect(data, isNotNull);
      expect(data!.windowId, 7);
      expect(data.context.source, ViewerWindowSource.gallery);
      expect(data.context.selectedIndex, 12);
      expect(data.context.selectedImageId, 1);
      expect(
        (data.context.snapshot as ViewerWindowGallerySnapshot).hasTags,
        isFalse,
      );
      expect(data.policy.title, 'ACGWarehouse Viewer');
    });

    test(
      'keeps main-shell startup path when viewer bootstrap args are absent',
      () {
        expect(ViewerWindowBootstrapData.fromCommandLine(const []), isNull);
        expect(
          ViewerWindowBootstrapData.fromCommandLine(const ['main']),
          isNull,
        );
      },
    );

    test(
      'rejects legacy openSession path to avoid losing context state',
      () async {
        final adapter = FakeViewerWindowAdapter();
        final service = ViewerWindowService(adapter: adapter);

        await expectLater(
          () => service.openSession(
            const ViewerSession(
              source: ViewerSessionSource.search,
              items: [
                ViewerSessionItem(
                  imageId: 1,
                  path: 'C:/images/alpha.png',
                  filename: 'alpha.png',
                  sourceRoot: 'C:/images',
                  fileSize: 2048,
                  width: 800,
                  height: 600,
                  format: 'png',
                  thumbnailSmallUrl: null,
                  thumbnailLargeUrl: null,
                  createdAtIso8601: '2026-04-05T00:00:00.000Z',
                  updatedAtIso8601: '2026-04-05T00:00:00.000Z',
                ),
              ],
              initialSelectedIndex: 0,
            ),
          ),
          throwsA(isA<UnsupportedError>()),
        );
        expect(adapter.launches, isEmpty);
      },
    );

    test('rejects malformed bootstrap json instead of throwing', () {
      expect(
        ViewerWindowBootstrapData.fromCommandLine([
          'multi_window',
          '7',
          '{"kind":"viewer-window","context":',
        ]),
        isNull,
      );
    });

    test('rejects bootstrap payloads missing selected index', () {
      expect(
        ViewerWindowBootstrapData.fromCommandLine([
          'multi_window',
          '7',
          '{"kind":"viewer-window","context":{"source":"gallery","selected_image_id":1,"snapshot":{"sort_by":"created_at","sort_dir":"desc","tag_ids":[],"has_tags":null}}}',
        ]),
        isNull,
      );
    });

    test('rejects bootstrap payloads missing selected image id', () {
      expect(
        ViewerWindowBootstrapData.fromCommandLine([
          'multi_window',
          '7',
          '{"kind":"viewer-window","context":{"source":"search","selected_index":5,"snapshot":{"q":"alpha","tag_ids":[],"sort_by":"relevance","sort_order":"desc"}}}',
        ]),
        isNull,
      );
    });
  });
}

class FakeViewerWindowAdapter implements ViewerWindowAdapter {
  final List<ViewerWindowLaunchRequest> launches = [];
  int restoreStateCalls = 0;
  int saveStateCalls = 0;

  @override
  Future<void> launch(ViewerWindowLaunchRequest request) async {
    launches.add(request);
  }

  @override
  Future<void> restoreWindowState(String windowId) async {
    restoreStateCalls += 1;
  }

  @override
  Future<void> saveWindowState(String windowId) async {
    saveStateCalls += 1;
  }
}
