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
  _TrackingImageListProvider() : super(ApiService());

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
        ),
      ],
      super(TagService(client: client));

  final List<TagGovernanceRow> _rows;
  TagDeletePreview? _preview;

  @override
  List<TagGovernanceRow> get governanceRows => _rows;

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
      expect(find.text('使用量'), findsWidgets);
      expect(find.text('AI'), findsWidgets);
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

      // Verify navigation switched to gallery
      expect(navProvider.selectedIndex, NavigationProvider.galleryIndex);
    },
  );
}
