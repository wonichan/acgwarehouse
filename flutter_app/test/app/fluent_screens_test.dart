import 'dart:convert';
import 'dart:ui' show Size;

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart' show Autocomplete, MaterialApp, Widget;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/gallery_filter_state.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/batch_service.dart';
import 'package:gallery/services/search_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/screens/gallery_screen.dart';
import 'package:gallery/widgets/fluent_image_card.dart';
import 'package:gallery/widgets/tag_management/tag_management_workspace.dart';

void main() {
  Widget buildFluentGalleryTestApp({
    required http.Client client,
    ImageListProvider? imageProvider,
    TagProvider? tagProvider,
    SelectionProvider? selectionProvider,
    BatchService? batchService,
  }) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ImageListProvider>(
          create: (_) =>
              imageProvider ??
              ImageListProvider(
                ApiService(baseUrl: 'http://localhost:8080', client: client),
              ),
        ),
        ChangeNotifierProvider<TagProvider>(
          create: (_) =>
              tagProvider ??
              TagProvider(
                TagService(baseUrl: 'http://localhost:8080', client: client),
              ),
        ),
        ChangeNotifierProvider<NavigationProvider>(
          create: (_) => NavigationProvider(),
        ),
        ChangeNotifierProvider<ConfigProvider>(
          create: (_) =>
              ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
        ),
        ChangeNotifierProvider<SelectionProvider>(
          create: (_) => selectionProvider ?? SelectionProvider(),
        ),
      ],
      child: fluent.FluentApp(
        home: FluentGalleryPage(batchService: batchService),
      ),
    );
  }

  testWidgets(
    'FluentTagManagementPage hosts TagManagementWorkspace in ScaffoldPage',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>(
              create: (_) => TagProvider(
                TagService(
                  baseUrl: 'http://localhost:8080',
                  client: mockClient,
                ),
              ),
            ),
          ],
          child: const fluent.FluentApp(home: FluentTagManagementPage()),
        ),
      );

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.byType(fluent.PageHeader), findsOneWidget);
      expect(find.text('标签治理'), findsOneWidget);
      expect(find.byType(TagManagementWorkspace), findsOneWidget);
    },
  );

  testWidgets(
    'FluentGalleryPage keeps page content and exposes top filter action',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/images')) {
          return http.Response('{"images":[],"total":0,"has_more":false}', 200);
        }
        if (request.url.path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      final imageProvider = ImageListProvider(
        ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
      );

      await tester.pumpWidget(
        buildFluentGalleryTestApp(
          client: mockClient,
          imageProvider: imageProvider,
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.text('图库'), findsWidgets);
      expect(find.text('标签筛选'), findsOneWidget);
      expect(find.text('未打标签'), findsOneWidget);
      expect(find.text('选择'), findsOneWidget);

      await tester.tap(find.text('排序').first);
      await tester.pumpAndSettle();

      expect(find.text('源文件创建时间（新→旧）'), findsOneWidget);
      expect(find.text('源文件创建时间（旧→新）'), findsOneWidget);
      expect(find.text('源文件大小（大→小）'), findsOneWidget);
      expect(find.text('源文件大小（小→大）'), findsOneWidget);
      expect(find.text('源文件文件名（A-Z）'), findsOneWidget);
      expect(find.text('源文件文件名（Z-A）'), findsOneWidget);

      await tester.tap(find.text('源文件大小（小→大）'));
      await tester.pumpAndSettle();

      expect(imageProvider.sortField, SortField.fileSize);
      expect(imageProvider.sortAsc, isTrue);
    },
  );

  testWidgets('FluentGalleryPage shows browse-mode toolbar actions', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":1,"preferred_label":"test","slug":"test","review_state":"confirmed","trust_score":0.8,"usage_count":1,"created_at":"2024-01-01T00:00:00Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    await tester.pumpWidget(buildFluentGalleryTestApp(client: mockClient));
    await tester.pumpAndSettle();

    expect(find.text('排序'), findsOneWidget);
    expect(find.text('刷新'), findsOneWidget);
    expect(find.text('批量AI标签'), findsOneWidget);
    expect(find.text('选择'), findsOneWidget);
    expect(find.text('全选'), findsNothing);
  });

  testWidgets(
    'FluentGalleryPage has sort and batch AI tag actions in command bar',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/images')) {
          return http.Response('{"images":[],"total":0,"has_more":false}', 200);
        }
        if (request.url.path.endsWith('/api/v1/tags')) {
          return http.Response('{"tags":[]}', 200);
        }
        return http.Response('{}', 200);
      });

      await tester.pumpWidget(buildFluentGalleryTestApp(client: mockClient));
      await tester.pumpAndSettle();

      // CommandBar should have Sort button
      expect(find.text('排序'), findsOneWidget);
      expect(find.byIcon(fluent.FluentIcons.sort), findsOneWidget);

      // CommandBar should have Refresh button
      expect(find.text('刷新'), findsOneWidget);
      expect(find.byIcon(fluent.FluentIcons.refresh), findsOneWidget);

      // CommandBar should have 批量AI标签 button
      expect(find.text('批量AI标签'), findsOneWidget);
      expect(find.byIcon(fluent.FluentIcons.auto_enhance_on), findsOneWidget);
    },
  );

  testWidgets(
    'FluentSearchPage still renders search body inside ScaffoldPage',
    (tester) async {
      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<SearchProvider>(
              create: (_) => SearchProvider(
                service: SearchService(
                  baseUrl: 'http://localhost:8080',
                  client: MockClient((_) async => http.Response('{}', 200)),
                ),
              ),
            ),
          ],
          child: const fluent.FluentApp(home: FluentSearchPage()),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.byType(fluent.TextBox), findsOneWidget);
    },
  );

  testWidgets('FluentGalleryPage exposes batch AI trigger action', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    Map<String, dynamic>? batchRequestBody;
    String? batchRequestPath;
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith(
        '/api/v1/images/batch-ai-tags/regenerate',
      )) {
        batchRequestPath = request.url.path;
        batchRequestBody = jsonDecode(request.body) as Map<String, dynamic>?;
        return http.Response('{"job_ids":[101],"status":"queued"}', 202);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => imageProvider,
          ),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(
              TagService(baseUrl: 'http://localhost:8080', client: mockClient),
            ),
          ),
          ChangeNotifierProvider<NavigationProvider>(
            create: (_) => NavigationProvider(),
          ),
          ChangeNotifierProvider<ConfigProvider>(
            create: (_) =>
                ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
          ),
          ChangeNotifierProvider<SelectionProvider>(
            create: (_) => SelectionProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: FluentGalleryPage()),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('批量AI标签'), findsOneWidget);

    await tester.tap(find.text('批量AI标签'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('批量触发 AI 标签'), findsOneWidget);
    expect(find.textContaining('1 张图片'), findsOneWidget);

    await tester.tap(find.text('确认').first);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(batchRequestBody, isNotNull);
    expect(batchRequestPath, equals('/api/v1/images/batch-ai-tags/regenerate'));
    expect(batchRequestBody!.containsKey('image_ids'), isFalse);
    expect(batchRequestBody!['sort_by'], equals('created_at'));
    expect(batchRequestBody!['sort_dir'], equals('desc'));
  });

  testWidgets('FluentGalleryPage opens in-window detail on image double tap', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      if (request.url.path.endsWith('/api/v1/images/1/tags')) {
        return http.Response(
          '{"confirmed":[],"pending":[],"rejected":[]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.byType(FluentImageCard), findsOneWidget);

    await tester.tap(find.byType(FluentImageCard), warnIfMissed: false);
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(find.byType(FluentImageCard), warnIfMissed: false);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('图片详情'), findsOneWidget);
  });

  testWidgets('FluentGalleryPage enters selection mode and switches toolbar', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('选择'), findsOneWidget);
    expect(find.text('批量AI标签'), findsOneWidget);

    await tester.tap(find.text('选择'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('批量AI标签'), findsNothing);
    expect(find.text('全不选'), findsOneWidget);
    expect(find.text('全选'), findsOneWidget);
    expect(find.text('批量添加标签 (0)'), findsOneWidget);
    expect(find.text('批量删除 (0)'), findsOneWidget);
    expect(find.text('退出选择模式'), findsOneWidget);
  });

  testWidgets('全不选 keeps selection mode active and disables batch actions', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);
    final selectionProvider = SelectionProvider()
      ..enterSelectionMode()
      ..toggleSelection(1);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
        selectionProvider: selectionProvider,
        batchService: BatchService(
          client: mockClient,
          baseUrl: 'http://localhost:8080',
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    await tester.tap(find.text('全不选'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(selectionProvider.isSelectionMode, isTrue);
    expect(selectionProvider.selectedCount, 0);
    expect(find.text('全选'), findsOneWidget);
    expect(find.text('退出选择模式'), findsOneWidget);
    expect(find.text('批量添加标签 (0)'), findsOneWidget);
    expect(find.text('批量删除 (0)'), findsOneWidget);
  });

  testWidgets('退出选择模式 clears selection and restores browse toolbar', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);
    final selectionProvider = SelectionProvider()
      ..enterSelectionMode()
      ..toggleSelection(1);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
        selectionProvider: selectionProvider,
        batchService: BatchService(
          client: mockClient,
          baseUrl: 'http://localhost:8080',
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    await tester.tap(find.text('退出选择模式'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(selectionProvider.isSelectionMode, isFalse);
    expect(selectionProvider.selectedCount, 0);
    expect(find.text('选择'), findsOneWidget);
    expect(find.text('批量AI标签'), findsOneWidget);
    expect(find.text('全选'), findsNothing);
  });

  testWidgets('batch delete confirms, refreshes, and exits selection mode', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    Map<String, dynamic>? deleteBody;
    var imageRequestCount = 0;
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        imageRequestCount += 1;
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      if (request.url.path.endsWith('/api/v1/batch/images/delete')) {
        deleteBody = jsonDecode(request.body) as Map<String, dynamic>;
        return http.Response('{"images_deleted":1}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);
    final selectionProvider = SelectionProvider()
      ..enterSelectionMode()
      ..toggleSelection(1);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
        selectionProvider: selectionProvider,
        batchService: BatchService(
          client: mockClient,
          baseUrl: 'http://localhost:8080',
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    await tester.tap(find.text('批量删除 (1)'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('确认批量删除'), findsOneWidget);
    expect(find.textContaining('1 张图片'), findsOneWidget);
    expect(find.text('取消'), findsOneWidget);
    expect(find.text('确认删除'), findsOneWidget);

    await tester.tap(find.text('确认删除'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(deleteBody, isNotNull);
    expect(deleteBody!['image_ids'], equals([1]));
    expect(imageRequestCount, greaterThanOrEqualTo(2));
    expect(selectionProvider.isSelectionMode, isFalse);
    expect(find.text('删除成功'), findsOneWidget);

    await tester.tap(find.text('知道了'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('选择'), findsOneWidget);
    expect(find.text('批量AI标签'), findsOneWidget);
  });

  testWidgets('batch add success refreshes and exits selection mode', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    Map<String, dynamic>? addBody;
    var imageRequestCount = 0;
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        imageRequestCount += 1;
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":101,"preferred_label":"tag1","slug":"tag1","review_state":"confirmed","trust_score":0.9,"usage_count":10,"created_at":"2024-01-01T00:00:00Z"}]}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/batch/tags/add')) {
        addBody = jsonDecode(request.body) as Map<String, dynamic>;
        return http.Response('{"images_updated":1}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);
    final selectionProvider = SelectionProvider()
      ..enterSelectionMode()
      ..toggleSelection(1);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
        selectionProvider: selectionProvider,
        batchService: BatchService(
          client: mockClient,
          baseUrl: 'http://localhost:8080',
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    await tester.tap(find.text('批量添加标签 (1)'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('批量添加标签'), findsOneWidget);
    await tester.tap(find.text('tag1'));
    await tester.pump();
    await tester.tap(find.text('确认添加'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(addBody, isNotNull);
    expect(addBody!['image_ids'], equals([1]));
    expect(addBody!['tag_ids'], equals([101]));
    expect(imageRequestCount, greaterThanOrEqualTo(2));
    expect(selectionProvider.isSelectionMode, isFalse);
    expect(find.text('批量添加成功'), findsOneWidget);

    await tester.tap(find.text('知道了'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    expect(find.text('选择'), findsOneWidget);
  });

  testWidgets('batch add failure preserves selection mode', (tester) async {
    await tester.binding.setSurfaceSize(const Size(2200, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":101,"preferred_label":"tag1","slug":"tag1","review_state":"confirmed","trust_score":0.9,"usage_count":10,"created_at":"2024-01-01T00:00:00Z"}]}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/batch/tags/add')) {
        return http.Response('oops', 500);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );
    await imageProvider.loadImages(refresh: true);
    final selectionProvider = SelectionProvider()
      ..enterSelectionMode()
      ..toggleSelection(1);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(
        client: mockClient,
        imageProvider: imageProvider,
        selectionProvider: selectionProvider,
        batchService: BatchService(
          client: mockClient,
          baseUrl: 'http://localhost:8080',
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));

    await tester.tap(find.text('批量添加标签 (1)'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    await tester.tap(find.text('tag1'));
    await tester.pump();
    await tester.tap(find.text('确认添加'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('批量添加失败'), findsOneWidget);
    expect(selectionProvider.isSelectionMode, isTrue);
    expect(selectionProvider.selectedCount, 1);
  });

  testWidgets('GalleryScreen renders inline search bar and untagged toggle', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":1,"preferred_label":"tag1","slug":"tag1","review_state":"confirmed","trust_score":0.9,"usage_count":10,"created_at":"2024-01-01T00:00:00Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>.value(value: imageProvider),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(
              TagService(baseUrl: 'http://localhost:8080', client: mockClient),
            ),
          ),
          ChangeNotifierProvider<SelectionProvider>(
            create: (_) => SelectionProvider(),
          ),
        ],
        child: const MaterialApp(home: GalleryScreen()),
      ),
    );
    await tester.pumpAndSettle();

    // The Material GalleryScreen should have search functionality
    // Find the Autocomplete for tag search
    expect(find.byType(Autocomplete<Tag>), findsOneWidget);

    // Should have the "未打标签" toggle
    expect(find.text('未打标签'), findsOneWidget);

    // Verify the toggle is present and tappable
    final untaggedToggle = find.text('未打标签');
    expect(untaggedToggle, findsOneWidget);

    // Tap the "未打标签" toggle and verify it changes the filter
    await tester.tap(untaggedToggle);
    await tester.pumpAndSettle();

    // After tapping, the filter should be applied (hasTagsFilter should be false)
    expect(imageProvider.hasTagsFilter, equals(false));

    // Tap again to toggle off
    await tester.tap(untaggedToggle);
    await tester.pumpAndSettle();

    // Filter should be cleared (null means no filter)
    expect(imageProvider.hasTagsFilter, isNull);
  });

  testWidgets('FluentGalleryPage top untagged action keeps selected tags', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response('{"images":[],"total":0,"has_more":false}', 200);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response(
          '{"tags":[{"id":1,"preferred_label":"tag1","slug":"tag1","review_state":"confirmed","trust_score":0.9,"usage_count":10,"created_at":"2024-01-01T00:00:00Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(
      ApiService(baseUrl: 'http://localhost:8080', client: mockClient),
    );

    await imageProvider.applyFilter(GalleryFilterState(exactTagIds: {1}));
    expect(imageProvider.selectedTagIds, [1]);

    await tester.pumpWidget(
      buildFluentGalleryTestApp(client: mockClient, imageProvider: imageProvider),
    );
    await tester.pumpAndSettle();

    final untaggedToggle = find.text('未打标签');
    expect(untaggedToggle, findsOneWidget);

    await tester.tap(untaggedToggle);
    await tester.pumpAndSettle();

    expect(imageProvider.hasTagsFilter, isFalse);
    expect(imageProvider.selectedTagIds, [1]);

    await tester.tap(untaggedToggle);
    await tester.pumpAndSettle();

    expect(imageProvider.hasTagsFilter, isNull);
    expect(imageProvider.selectedTagIds, [1]);
  });
}
