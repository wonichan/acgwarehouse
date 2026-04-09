import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/tag_management/tag_bulk_action_bar.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _BulkBarTagProvider extends TagProvider {
  _BulkBarTagProvider({http.Client? client})
    : _selectedIds = {10, 30},
      super(TagService(baseUrl: 'http://localhost:8080', client: client));

  final Set<int> _selectedIds;

  @override
  List<TagGovernanceRow> get governanceRows => [
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
  ];

  @override
  Set<int> get selectedGovernanceIds => _selectedIds;

  @override
  bool get isRunningGovernanceAction => false;

  @override
  String? get governanceError => null;

  @override
  Future<void> loadGovernanceTags({String? search}) async {}

  int cleanupCalls = 0;
  int? mergeIntoTargetId;

  @override
  Future<TagGovernanceBatchResult> cleanupSelectedUnusedTags() async {
    cleanupCalls++;
    return const TagGovernanceBatchResult(
      deletedTagIds: <int>[],
      failures: <TagGovernanceFailure>[],
    );
  }

  @override
  Future<TagGovernanceBatchResult> mergeSelectionInto(int targetTagId) async {
    mergeIntoTargetId = targetTagId;
    return const TagGovernanceBatchResult(
      deletedTagIds: <int>[],
      failures: <TagGovernanceFailure>[],
    );
  }

  @override
  void clearGovernanceSelection() {
    _selectedIds.clear();
    notifyListeners();
  }
}

void main() {
  testWidgets(
    'bulk action bar shows selected count and cleanup/merge controls',
    (tester) async {
      final mockClient = MockClient((request) async {
        return http.Response('{"tags":[]}', 200);
      });
      final tagProvider = _BulkBarTagProvider(client: mockClient);

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
          ],
          child: fluent.FluentApp(
            home: TagBulkActionBar(
              onCleanup: () async {},
              onMergeInto: (_) async {},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Shows selected count
      expect(find.textContaining('2 已选中'), findsOneWidget);

      // Shows cleanup and merge controls
      expect(find.text('清理已选中'), findsOneWidget);
      expect(find.text('合并到...'), findsOneWidget);
      expect(find.text('清除选择'), findsOneWidget);
    },
  );

  testWidgets('cleanup calls onCleanup callback', (tester) async {
    final mockClient = MockClient((request) async {
      return http.Response('{"tags":[]}', 200);
    });
    final tagProvider = _BulkBarTagProvider(client: mockClient);
    int cleanupCalls = 0;

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
        ],
        child: fluent.FluentApp(
          home: TagBulkActionBar(
            onCleanup: () async {
              cleanupCalls++;
            },
            onMergeInto: (_) async {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('清理已选中'));
    await tester.pumpAndSettle();

    expect(cleanupCalls, 1);
  });
}
