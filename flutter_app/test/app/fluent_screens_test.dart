import 'dart:convert';
import 'dart:ui' show Size;

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart'
    show Autocomplete, GlobalKey, Material, MaterialApp, TextField;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/search_provider.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
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
              create: (_) => TagProvider(TagService(client: mockClient)),
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

      final imageProvider = ImageListProvider(ApiService(client: mockClient));

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => imageProvider,
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

      expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
      expect(find.text('图库'), findsWidgets);
      // The new design uses a tokenized search bar in the header instead of a filter button
      expect(find.byType(fluent.AutoSuggestBox<Tag>), findsOneWidget);

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

  testWidgets('FluentGalleryPage has inline search bar in header', (
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

    // Should find the inline search bar (AutoSuggestBox for tokenized search)
    expect(find.byType(fluent.AutoSuggestBox<Tag>), findsOneWidget);
    // Should find TextBox within the autocomplete
    expect(find.byType(fluent.TextBox), findsWidgets);
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
    await tester.binding.setSurfaceSize(const Size(1400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    Map<String, dynamic>? batchRequestBody;
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/images')) {
        return http.Response(
          '{"images":[{"id":1,"path":"C:/images/alpha.png","filename":"alpha.png","source_root":"C:/images","file_size":2048,"width":800,"height":600,"format":"png","phash":123,"thumbnail_small_url":"http://example.com/thumb.png","thumbnail_large_url":"http://example.com/large.png","created_at":"2026-04-05T00:00:00.000Z","updated_at":"2026-04-05T00:00:00.000Z"}],"total":1,"has_more":false}',
          200,
        );
      }
      if (request.url.path.endsWith('/api/v1/images/batch-ai-tags')) {
        batchRequestBody = jsonDecode(request.body) as Map<String, dynamic>?;
        return http.Response('{"job_ids":[101],"status":"queued"}', 202);
      }
      if (request.url.path.endsWith('/api/v1/tags')) {
        return http.Response('{"tags":[]}', 200);
      }
      return http.Response('{}', 200);
    });

    final imageProvider = ImageListProvider(ApiService(client: mockClient));
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => imageProvider,
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
    expect(batchRequestBody!.containsKey('image_ids'), isFalse);
    expect(batchRequestBody!['sort_by'], equals('created_at'));
    expect(batchRequestBody!['sort_dir'], equals('desc'));
  });

  testWidgets('FluentGalleryPage opens in-window detail on image double tap', (
    tester,
  ) async {
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

    final imageProvider = ImageListProvider(ApiService(client: mockClient));
    await imageProvider.loadImages(refresh: true);

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => imageProvider,
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
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.byType(FluentImageCard), findsOneWidget);

    await tester.tap(find.byType(FluentImageCard));
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(find.byType(FluentImageCard));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('图片详情'), findsOneWidget);
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

    final imageProvider = ImageListProvider(ApiService(client: mockClient));

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<ImageListProvider>.value(value: imageProvider),
          ChangeNotifierProvider<TagProvider>(
            create: (_) => TagProvider(TagService(client: mockClient)),
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
}
