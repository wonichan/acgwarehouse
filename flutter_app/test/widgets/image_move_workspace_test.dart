import 'dart:convert';

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/image_move_provider.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/image_move_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_move/image_move_workspace.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _WorkspaceTagProvider extends TagProvider {
  _WorkspaceTagProvider() : super(TagService(baseUrl: 'http://localhost:8080'));

  @override
  List<Tag> get allTags => [
    Tag(
      id: 7,
      preferredLabel: '调月莉音',
      slug: 'rio',
      reviewState: 'confirmed',
      trustScore: 1,
      usageCount: 3,
      createdAt: DateTime.parse('2026-05-10T00:00:00.000Z'),
    ),
  ];

  @override
  bool get isLoading => false;

  @override
  String? get error => null;

  @override
  Future<void> loadTags() async {}
}

class _TrackingImageListProvider extends ImageListProvider {
  _TrackingImageListProvider()
    : super(ApiService(baseUrl: 'http://localhost:8080'));

  int refreshCalls = 0;

  @override
  Future<void> loadImages({bool refresh = false}) async {
    if (refresh) {
      refreshCalls++;
    }
  }
}

void main() {
  Widget buildApp({
    required ImageMoveService service,
    required List<String?> pickedDirs,
    _TrackingImageListProvider? imageProvider,
    ImageMoveProvider? moveProvider,
  }) {
    var pickIndex = 0;
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ImageMoveProvider>.value(
          value: moveProvider ?? ImageMoveProvider(),
        ),
        ChangeNotifierProvider<TagProvider>(
          create: (_) => _WorkspaceTagProvider(),
        ),
        ChangeNotifierProvider<ImageListProvider>.value(
          value: imageProvider ?? _TrackingImageListProvider(),
        ),
        ChangeNotifierProvider<ConfigProvider>(
          create: (_) =>
              ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
        ),
      ],
      child: fluent.FluentApp(
        home: ImageMoveWorkspace(
          imageMoveService: service,
          directoryPicker: () async => pickedDirs[pickIndex++],
        ),
      ),
    );
  }

  testWidgets('disables preview until source, tag, and target are selected', (
    tester,
  ) async {
    final moveProvider = ImageMoveProvider();
    final service = ImageMoveService(
      baseUrl: 'http://localhost:8080',
      client: MockClient((_) async => http.Response('{}', 500)),
    );

    await tester.pumpWidget(
      buildApp(
        service: service,
        pickedDirs: ['E:/picture/output', 'E:/picture/archive'],
        moveProvider: moveProvider,
      ),
    );
    await tester.pump();

    expect(find.text('预览'), findsOneWidget);
    expect(moveProvider.canPreview, isFalse);

    await tester.tap(find.text('添加来源目录'));
    await tester.pumpAndSettle();
    expect(moveProvider.canPreview, isFalse);

    await tester.tap(find.text('调月莉音 (3)'));
    await tester.pumpAndSettle();
    expect(moveProvider.canPreview, isFalse);

    await tester.tap(find.text('选择目标目录'));
    await tester.pumpAndSettle();
    expect(moveProvider.canPreview, isTrue);
  });

  testWidgets('shows preview conflicts and execution failures', (tester) async {
    final client = MockClient((request) async {
      if (request.url.path.endsWith('/preview')) {
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['source_dirs'], ['E:/picture/output']);
        expect(body['tag_id'], 7);
        expect(body['target_dir'], 'E:/picture/archive');
        expect(body['conflict'], 'skip');
        return http.Response(
          jsonEncode({
            'total_matched': 2,
            'movable': 1,
            'skipped': 1,
            'items': [
              {
                'image_id': 1,
                'filename': 'alpha.png',
                'source_path': 'E:/picture/output/alpha.png',
                'target_path': 'E:/picture/archive/alpha.png',
                'status': 'movable',
              },
              {
                'image_id': 2,
                'filename': 'beta.png',
                'source_path': 'E:/picture/output/beta.png',
                'target_path': 'E:/picture/archive/beta.png',
                'status': 'skipped',
                'reason': 'target_exists',
              },
            ],
          }),
          200,
        );
      }

      return http.Response(
        jsonEncode({
          'total_matched': 2,
          'moved': 1,
          'skipped': 0,
          'failed': 1,
          'items': [
            {
              'image_id': 1,
              'filename': 'alpha.png',
              'source_path': 'E:/picture/output/alpha.png',
              'target_path': 'E:/picture/archive/alpha.png',
              'status': 'moved',
            },
            {
              'image_id': 2,
              'filename': 'beta.png',
              'source_path': 'E:/picture/output/beta.png',
              'target_path': 'E:/picture/archive/beta.png',
              'status': 'failed',
              'reason': 'move_failed',
            },
          ],
        }),
        200,
      );
    });
    final service = ImageMoveService(
      baseUrl: 'http://localhost:8080',
      client: client,
    );

    await tester.pumpWidget(
      buildApp(
        service: service,
        pickedDirs: ['E:/picture/output', 'E:/picture/archive'],
      ),
    );

    await tester.tap(find.text('添加来源目录'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('调月莉音 (3)'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('选择目标目录'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('预览'));
    await tester.pumpAndSettle();

    expect(find.text('命中 2'), findsOneWidget);
    expect(find.text('可移动 1'), findsOneWidget);
    expect(find.text('原因：目标已存在'), findsOneWidget);

    await tester.tap(find.text('开始移动'));
    await tester.pumpAndSettle();

    expect(find.text('移动完成'), findsOneWidget);
    expect(find.text('已移动 1'), findsOneWidget);
    expect(find.text('失败 1'), findsOneWidget);
    expect(find.text('原因：移动失败'), findsOneWidget);
  });

  testWidgets('keeps move selections after workspace is rebuilt', (
    tester,
  ) async {
    final moveProvider = ImageMoveProvider();
    final service = ImageMoveService(
      baseUrl: 'http://localhost:8080',
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    await tester.pumpWidget(
      buildApp(
        service: service,
        pickedDirs: ['E:/picture/output', 'E:/picture/archive'],
        moveProvider: moveProvider,
      ),
    );

    await tester.tap(find.text('添加来源目录'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('调月莉音 (3)'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('选择目标目录'));
    await tester.pumpAndSettle();

    expect(find.text('E:/picture/output'), findsOneWidget);
    expect(find.text('已选择：调月莉音（3）'), findsOneWidget);
    expect(find.text('E:/picture/archive'), findsOneWidget);

    await tester.pumpWidget(
      buildApp(
        service: service,
        pickedDirs: ['unused'],
        moveProvider: moveProvider,
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('E:/picture/output'), findsOneWidget);
    expect(find.text('已选择：调月莉音（3）'), findsOneWidget);
    expect(find.text('E:/picture/archive'), findsOneWidget);
  });

  testWidgets('refresh gallery button delegates to image provider', (
    tester,
  ) async {
    final imageProvider = _TrackingImageListProvider();
    final service = ImageMoveService(
      baseUrl: 'http://localhost:8080',
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    await tester.pumpWidget(
      buildApp(
        service: service,
        pickedDirs: const [],
        imageProvider: imageProvider,
      ),
    );
    await tester.tap(find.text('刷新图库'));
    await tester.pumpAndSettle();

    expect(imageProvider.refreshCalls, 1);
  });
}
