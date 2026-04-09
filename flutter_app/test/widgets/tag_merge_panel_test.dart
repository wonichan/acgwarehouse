import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/tag_management/tag_merge_panel.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _MergePanelTagProvider extends TagProvider {
  _MergePanelTagProvider({http.Client? client})
    : super(TagService(client: client));

  static final List<TagGovernanceRow> _allRows = [
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
  ];

  @override
  List<TagGovernanceRow> get governanceRows => _allRows;

  @override
  bool get isRunningGovernanceAction => false;

  @override
  String? get governanceError => null;

  @override
  Future<void> loadGovernanceTags({String? search}) async {}

  @override
  Future<void> deleteTag(int tagId) async {}

  int mergeSelectionIntoCalls = 0;
  int? lastMergeTargetTagId;

  @override
  Future<TagGovernanceBatchResult> mergeSelectionInto(int targetTagId) async {
    mergeSelectionIntoCalls++;
    lastMergeTargetTagId = targetTagId;
    return const TagGovernanceBatchResult(
      deletedTagIds: <int>[],
      failures: <TagGovernanceFailure>[],
    );
  }
}

void main() {
  testWidgets(
    'merge panel shows source context and searchable target choices',
    (tester) async {
      final mockClient = MockClient((request) async {
        return http.Response('{"tags":[]}', 200);
      });
      final tagProvider = _MergePanelTagProvider(client: mockClient);

      await tester.pumpWidget(
        MultiProvider(
          providers: [
            ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
          ],
          child: fluent.FluentApp(
            home: TagMergePanel(
              sourceRow: _MergePanelTagProvider._allRows.first,
              allRows: _MergePanelTagProvider._allRows,
              onConfirm: (_) async {},
              onCancel: () {},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Source tag context shown
      expect(find.textContaining('long_hair'), findsWidgets);

      // Target choices rendered (excludes source tagId=10)
      expect(find.text('blue_eyes'), findsOneWidget);
      expect(find.text('school_uniform'), findsOneWidget);

      // Search box for filtering targets
      expect(find.byType(fluent.TextBox), findsOneWidget);

      // Confirm button initially disabled (no target selected)
      expect(find.text('确认合并'), findsOneWidget);
    },
  );

  testWidgets('confirm merge calls onConfirm with targetTagId', (tester) async {
    final mockClient = MockClient((request) async {
      return http.Response('{"tags":[]}', 200);
    });
    final tagProvider = _MergePanelTagProvider(client: mockClient);
    int? confirmedTargetId;

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          ChangeNotifierProvider<TagProvider>.value(value: tagProvider),
        ],
        child: fluent.FluentApp(
          home: TagMergePanel(
            sourceRow: _MergePanelTagProvider._allRows.first,
            allRows: _MergePanelTagProvider._allRows,
            onConfirm: (targetTagId) async {
              confirmedTargetId = targetTagId;
            },
            onCancel: () {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Tap a target to select it
    await tester.tap(find.text('school_uniform'));
    await tester.pumpAndSettle();

    // Confirm button should be enabled now
    await tester.tap(find.text('确认合并'));
    await tester.pumpAndSettle();

    // Verify the merge was confirmed with the correct target
    expect(confirmedTargetId, 30);
  });
}
