import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/providers/image_provider.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/api_service.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/tag_management/tag_management_workspace.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

/// Subclass that records setTagFilter calls without making real HTTP requests.
class _TrackingImageListProvider extends ImageListProvider {
  _TrackingImageListProvider()
    : super(ApiService(baseUrl: 'http://localhost:8080'));

  int setTagFilterCalls = 0;
  List<int> lastTagFilter = const [];

  @override
  Future<void> setTagFilter(List<int> tagIds) async {
    setTagFilterCalls++;
    lastTagFilter = List<int>.from(tagIds);
  }
}

/// Subclass that provides governance rows and tracks provider calls.
class _WorkspaceTagProvider extends TagProvider {
  _WorkspaceTagProvider({http.Client? client})
    : _rows = [
        const TagGovernanceRow(
          tagId: 10,
          preferredLabel: 'long_hair',
          primaryCategory: 'appearance',
          aliases: ['longhair'],
          usageCount: 42,
          pendingCount: 5,
          confirmedCount: 35,
          rejectedCount: 2,
          aiCount: 30,
          manualCount: 12,
          affectedImageCount: 42,
          canDelete: false,
          level: 'root',
        ),
        const TagGovernanceRow(
          tagId: 20,
          preferredLabel: 'blue_eyes',
          primaryCategory: 'appearance',
          aliases: <String>[],
          usageCount: 0,
          pendingCount: 0,
          confirmedCount: 0,
          rejectedCount: 0,
          aiCount: 0,
          manualCount: 0,
          affectedImageCount: 0,
          canDelete: true,
          level: 'parent',
          parentId: 10,
        ),
        const TagGovernanceRow(
          tagId: 30,
          preferredLabel: 'school_uniform',
          primaryCategory: 'clothing',
          aliases: ['seifuku'],
          usageCount: 15,
          pendingCount: 2,
          confirmedCount: 13,
          rejectedCount: 0,
          aiCount: 10,
          manualCount: 5,
          affectedImageCount: 15,
          canDelete: false,
          level: 'child',
          parentId: 20,
        ),
      ],
      super(TagService(baseUrl: 'http://localhost:8080', client: client));

  final List<TagGovernanceRow> _rows;
  TagDeletePreview? _preview;

  @override
  List<TagGovernanceRow> get governanceRows => _rows;

  @override
  Map<String, dynamic>? get tagTree => {
    'tree': [
      {
        'tag_id': 10,
        'preferred_label': 'long_hair',
        'level': 'root',
        'children': [
          {
            'tag_id': 20,
            'preferred_label': 'blue_eyes',
            'level': 'parent',
            'children': [
              {
                'tag_id': 30,
                'preferred_label': 'school_uniform',
                'level': 'child',
                'children': [],
              },
            ],
          },
        ],
      },
      {
        'tag_id': 40,
        'preferred_label': 'landscape',
        'level': 'root',
        'tree_usage_count': 5,
        'children': [],
      },
    ],
  };

  @override
  bool get isRunningGovernanceAction => false;

  @override
  String? get governanceError => null;

  @override
  TagDeletePreview? get deletePreview => _preview;

  @override
  Future<void> loadGovernanceTags({String? search}) async {}

  @override
  Future<void> loadDeletePreview(int tagId) async {
    final row = _rows.firstWhere((item) => item.tagId == tagId);
    _preview = TagDeletePreview(
      tagId: row.tagId,
      preferredLabel: row.preferredLabel,
      affectedImageCount: row.affectedImageCount,
      canDelete: row.canDelete,
      blockingReason: row.canDelete ? null : 'merge_or_reclassify_required',
    );
  }

  @override
  Future<void> deleteTag(int tagId) async {}
}

class _SearchTreeWorkspaceTagProvider extends TagProvider {
  _SearchTreeWorkspaceTagProvider({http.Client? client})
    : super(TagService(baseUrl: 'http://localhost:8080', client: client));

  final List<TagGovernanceRow> _rows = const [
    TagGovernanceRow(
      tagId: 10,
      preferredLabel: 'long_hair',
      primaryCategory: 'appearance',
      aliases: ['hair'],
      usageCount: 42,
      pendingCount: 4,
      confirmedCount: 38,
      rejectedCount: 0,
      aiCount: 25,
      manualCount: 17,
      affectedImageCount: 42,
      canDelete: false,
      level: 'root',
      directUsageCount: 12,
      treeUsageCount: 42,
      directAiCount: 8,
      treeAiCount: 25,
      directManualCount: 4,
      treeManualCount: 17,
    ),
    TagGovernanceRow(
      tagId: 20,
      preferredLabel: 'blue_eyes',
      primaryCategory: 'appearance',
      aliases: ['eyes'],
      usageCount: 20,
      pendingCount: 2,
      confirmedCount: 18,
      rejectedCount: 0,
      aiCount: 10,
      manualCount: 10,
      affectedImageCount: 20,
      canDelete: false,
      level: 'parent',
      parentId: 10,
      directUsageCount: 8,
      treeUsageCount: 20,
      directAiCount: 3,
      treeAiCount: 10,
      directManualCount: 5,
      treeManualCount: 10,
    ),
    TagGovernanceRow(
      tagId: 30,
      preferredLabel: 'school_uniform',
      primaryCategory: 'clothing',
      aliases: ['seifuku'],
      usageCount: 15,
      pendingCount: 2,
      confirmedCount: 13,
      rejectedCount: 0,
      aiCount: 10,
      manualCount: 5,
      affectedImageCount: 15,
      canDelete: false,
      level: 'child',
      parentId: 20,
      directUsageCount: 15,
      treeUsageCount: 15,
      directAiCount: 10,
      treeAiCount: 10,
      directManualCount: 5,
      treeManualCount: 5,
    ),
  ];

  @override
  List<TagGovernanceRow> get governanceRows => _rows;

  @override
  Map<String, dynamic>? get tagTree => {
    'tree': [
      {
        'tag_id': 10,
        'preferred_label': 'long_hair',
        'level': 'root',
        'tree_usage_count': 42,
        'children': [
          {
            'tag_id': 20,
            'preferred_label': 'blue_eyes',
            'level': 'parent',
            'tree_usage_count': 20,
            'children': [
              {
                'tag_id': 30,
                'preferred_label': 'school_uniform',
                'level': 'child',
                'tree_usage_count': 15,
                'children': [],
              },
            ],
          },
        ],
      },
    ],
  };

  @override
  bool get isRunningGovernanceAction => false;

  @override
  String? get governanceError => null;

  @override
  Future<void> loadGovernanceTags({String? search}) async {
    notifyListeners();
  }
}

void main() {
  testWidgets(
    'workspace shows summary stats, search box, governance list with row actions',
    (tester) async {
      final mockClient = MockClient((request) async {
        return http.Response('{"tags":[]}', 200);
      });

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>(
              create: (_) => _WorkspaceTagProvider(client: mockClient),
            ),
            ChangeNotifierProvider<NavigationProvider>(
              create: (_) => NavigationProvider(),
            ),
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => _TrackingImageListProvider(),
            ),
          ],
          child: const fluent.FluentApp(home: TagManagementWorkspace()),
        ),
      );
      await tester.pumpAndSettle();

      // Summary stats visible
      expect(find.text('总计使用量'), findsWidgets);
      expect(find.text('AI 生成'), findsWidgets);
      expect(find.text('手动'), findsWidgets);

      // Search box
      expect(find.byType(fluent.TextBox), findsOneWidget);

      // Row actions for each governance row
      expect(find.text('编辑'), findsWidgets);
      expect(find.text('合并'), findsWidgets);
      expect(find.text('删除'), findsWidgets);
      expect(find.text('查看受影响图片'), findsWidgets);

      // Governance rows rendered
      expect(find.text('long_hair'), findsOneWidget);
      expect(find.text('blue_eyes'), findsOneWidget);
      expect(find.text('school_uniform'), findsOneWidget);
    },
  );

  testWidgets('workspace renders governance rows from hierarchy tree data', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      return http.Response('{"tags":[]}', 200);
    });

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<TagProvider>(
            create: (_) => _WorkspaceTagProvider(client: mockClient),
          ),
          ChangeNotifierProvider<NavigationProvider>(
            create: (_) => NavigationProvider(),
          ),
          ChangeNotifierProvider<ImageListProvider>(
            create: (_) => _TrackingImageListProvider(),
          ),
        ],
        child: const fluent.FluentApp(home: TagManagementWorkspace()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('root'), findsWidgets);
    expect(find.text('parent'), findsWidgets);
    expect(find.text('child'), findsWidgets);
    expect(find.text('long_hair'), findsOneWidget);
    expect(find.text('blue_eyes'), findsOneWidget);
    expect(find.text('school_uniform'), findsOneWidget);
    expect(find.text('landscape'), findsNothing);
  });

  testWidgets(
    'workspace search keeps matched descendant ancestor path visible',
    (tester) async {
      final mockClient = MockClient(
        (request) async => http.Response('{"tags":[]}', 200),
      );

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>(
              create: (_) =>
                  _SearchTreeWorkspaceTagProvider(client: mockClient),
            ),
            ChangeNotifierProvider<NavigationProvider>(
              create: (_) => NavigationProvider(),
            ),
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => _TrackingImageListProvider(),
            ),
          ],
          child: const fluent.FluentApp(home: TagManagementWorkspace()),
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(fluent.TextBox), 'school');
      await tester.pumpAndSettle();

      expect(find.text('long_hair'), findsOneWidget);
      expect(find.text('blue_eyes'), findsOneWidget);
      expect(find.text('school_uniform'), findsOneWidget);
    },
  );

  testWidgets(
    'delete confirmation shows affected image count and blocks used tags',
    (tester) async {
      final mockClient = MockClient((request) async {
        return http.Response('{"tags":[]}', 200);
      });
      final tagProvider = _WorkspaceTagProvider(client: mockClient);

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
            ChangeNotifierProvider<NavigationProvider>(
              create: (_) => NavigationProvider(),
            ),
            ChangeNotifierProvider<ImageListProvider>(
              create: (_) => _TrackingImageListProvider(),
            ),
          ],
          child: const fluent.FluentApp(home: TagManagementWorkspace()),
        ),
      );
      await tester.pumpAndSettle();

      // Tap Delete on the first row (long_hair, canDelete=false, 42 affected images)
      final deleteButtons = find.text('删除');
      expect(deleteButtons, findsWidgets);

      await tester.tap(deleteButtons.first);
      await tester.pumpAndSettle();

      // Confirmation dialog should show the exact affected image count
      expect(find.textContaining('42'), findsWidgets);
      // And mention affected image context
      expect(find.textContaining('受影响的图片'), findsWidgets);
    },
  );

  testWidgets(
    'View affected images applies tag filter and switches to gallery',
    (tester) async {
      final mockClient = MockClient((request) async {
        return http.Response('{"tags":[]}', 200);
      });
      final tagProvider = _WorkspaceTagProvider(client: mockClient);
      final imageProvider = _TrackingImageListProvider();
      final navProvider = NavigationProvider();
      navProvider.setSelectedIndex(NavigationProvider.tagManagementIndex);

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
            ChangeNotifierProvider<NavigationProvider>.value(
              value: navProvider,
            ),
            ChangeNotifierProvider<ImageListProvider>.value(
              value: imageProvider,
            ),
          ],
          child: const fluent.FluentApp(home: TagManagementWorkspace()),
        ),
      );
      await tester.pumpAndSettle();

      // Tap "View affected images" on the first row
      final viewButtons = find.text('查看受影响图片');
      expect(viewButtons, findsWidgets);

      await tester.tap(viewButtons.first);
      await tester.pumpAndSettle();

      // Verify image provider received the tag filter
      expect(imageProvider.setTagFilterCalls, 1);
      expect(imageProvider.lastTagFilter, [10]);

      // Verify tag provider visible selection state stays in sync
      expect(tagProvider.selectedTagIds, {10});

      // Verify navigation switched to gallery
      expect(navProvider.selectedIndex, NavigationProvider.galleryIndex);
    },
  );
}
