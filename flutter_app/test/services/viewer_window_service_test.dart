import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_session.dart';
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
        final session = ViewerSession(
          source: ViewerSessionSource.gallery,
          items: const [
            ViewerSessionItem(
              imageId: 1,
              path: 'C:/images/alpha.png',
              filename: 'alpha.png',
              sourceRoot: 'C:/images',
              fileSize: 2048,
              width: 800,
              height: 600,
              format: 'png',
              thumbnailSmallUrl: '/thumbs/alpha-small.jpg',
              thumbnailLargeUrl: '/thumbs/alpha.jpg',
              createdAtIso8601: '2026-04-05T00:00:00.000Z',
              updatedAtIso8601: '2026-04-05T00:00:00.000Z',
            ),
          ],
          initialSelectedIndex: 0,
        );

        await service.openSession(session);

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

        await service.openSession(
          ViewerSession(
            source: ViewerSessionSource.gallery,
            items: const [
              ViewerSessionItem(
                imageId: 1,
                path: 'C:/images/alpha.png',
                filename: 'alpha.png',
                sourceRoot: 'C:/images',
                fileSize: 2048,
                width: 800,
                height: 600,
                format: 'png',
                thumbnailSmallUrl: '/thumbs/alpha-small.jpg',
                thumbnailLargeUrl: '/thumbs/alpha.jpg',
                createdAtIso8601: '2026-04-05T00:00:00.000Z',
                updatedAtIso8601: '2026-04-05T00:00:00.000Z',
              ),
            ],
            initialSelectedIndex: 0,
          ),
        );
        await service.openSession(
          ViewerSession(
            source: ViewerSessionSource.search,
            items: const [
              ViewerSessionItem(
                imageId: 2,
                path: 'C:/images/beta.png',
                filename: 'beta.png',
                sourceRoot: 'C:/images',
                fileSize: 2048,
                width: 800,
                height: 600,
                format: 'png',
                thumbnailSmallUrl: '/thumbs/beta-small.jpg',
                thumbnailLargeUrl: '/thumbs/beta.jpg',
                createdAtIso8601: '2026-04-05T00:00:00.000Z',
                updatedAtIso8601: '2026-04-05T00:00:00.000Z',
              ),
            ],
            initialSelectedIndex: 0,
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
        final session = ViewerSession(
          source: ViewerSessionSource.search,
          items: const [
            ViewerSessionItem(
              imageId: 9,
              path: 'C:/images/launch-target.png',
              filename: 'launch-target.png',
              sourceRoot: 'C:/images',
              fileSize: 2048,
              width: 800,
              height: 600,
              format: 'png',
              thumbnailSmallUrl: '/thumbs/launch-target-small.jpg',
              thumbnailLargeUrl: '/thumbs/launch-target.jpg',
              createdAtIso8601: '2026-04-05T00:00:00.000Z',
              updatedAtIso8601: '2026-04-05T00:00:00.000Z',
            ),
          ],
          initialSelectedIndex: 0,
        );

        await service.openSession(session);

        final payload = adapter.launches.single.arguments;
        expect(payload['kind'], 'viewer-window');
        expect(payload['session']['source'], 'search');
        expect(payload['session']['items'][0]['filename'], 'launch-target.png');
      },
    );

    test('parses viewer-window bootstrap arguments for secondary startup', () {
      final data = ViewerWindowBootstrapData.fromCommandLine([
        'multi_window',
        '7',
        '{"kind":"viewer-window","session":{"source":"gallery","items":[{"image_id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","thumbnail_small_url":"/thumbs/alpha-small.jpg","thumbnail_large_url":"/thumbs/alpha.jpg","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"initial_selected_index":0}}',
      ]);

      expect(data, isNotNull);
      expect(data!.windowId, 7);
      expect(data.session.selectedItem.filename, 'alpha.png');
      expect(data.policy.title, 'ACGWarehouse Viewer — alpha.png');
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
