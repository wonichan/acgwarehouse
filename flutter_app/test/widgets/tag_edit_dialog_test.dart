import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/tag_management/tag_edit_dialog.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _TagEditProvider extends TagProvider {
  _TagEditProvider({http.Client? client})
    : super(TagService(baseUrl: 'http://localhost:8080', client: client));

  String? createdLevel;
  int? createdParentId;
  String? createdLabel;
  int? changedTagId;
  String? changedLevel;
  int? changedParentId;
  int? reparentedTagId;
  int? reparentedParentId;

  @override
  Future<List<Tag>> getParentCandidates(String level) async {
    return [
      Tag(
        id: 99,
        preferredLabel: 'characters',
        slug: 'characters',
        primaryCategory: 'meta',
        reviewState: 'confirmed',
        trustScore: 1,
        usageCount: 10,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'root',
      ),
      Tag(
        id: 100,
        preferredLabel: 'appearance',
        slug: 'appearance',
        primaryCategory: 'meta',
        reviewState: 'confirmed',
        trustScore: 1,
        usageCount: 10,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'parent',
      ),
    ].where((tag) {
      if (level == 'parent') return tag.level == 'root';
      if (level == 'child') return tag.level == 'parent';
      return false;
    }).toList();
  }

  @override
  Future<void> createTag({
    required String preferredLabel,
    String? primaryCategory,
    String? level,
    int? parentId,
  }) async {
    createdLabel = preferredLabel;
    createdLevel = level;
    createdParentId = parentId;
  }

  @override
  Future<void> changeTagLevel(int tagId, String level, {int? parentId}) async {
    changedTagId = tagId;
    changedLevel = level;
    changedParentId = parentId;
  }

  @override
  Future<void> reparentTag(int tagId, int? parentId) async {
    reparentedTagId = tagId;
    reparentedParentId = parentId;
  }
}

void main() {
  fluent.Widget buildDialog(TagProvider provider, fluent.Widget child) {
    return fluent.FluentApp(
      home: ChangeNotifierProvider<TagProvider>.value(
        value: provider,
        child: fluent.NavigationView(content: child),
      ),
    );
  }

  testWidgets('create mode supports parent level with parent selection', (
    tester,
  ) async {
    final provider = _TagEditProvider(
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    await tester.pumpWidget(buildDialog(provider, const TagEditDialog()));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(fluent.TextBox).first, 'heroine');

    await tester.tap(find.byType(fluent.ComboBox<String>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Parent (父级)').last);
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ComboBox<int?>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('characters').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('保存'));
    await tester.pumpAndSettle();

    expect(provider.createdLabel, 'heroine');
    expect(provider.createdLevel, 'parent');
    expect(provider.createdParentId, 99);
  });

  testWidgets('edit mode changing level uses changeTagLevel', (tester) async {
    final provider = _TagEditProvider(
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    const row = TagGovernanceRow(
      tagId: 10,
      preferredLabel: 'long_hair',
      primaryCategory: 'appearance',
      aliases: <String>[],
      usageCount: 1,
      pendingCount: 0,
      confirmedCount: 1,
      rejectedCount: 0,
      aiCount: 0,
      manualCount: 1,
      affectedImageCount: 1,
      canDelete: false,
      level: 'root',
    );

    await tester.pumpWidget(
      buildDialog(provider, const TagEditDialog(row: row)),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ComboBox<String>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Parent (父级)').last);
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ComboBox<int?>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('characters').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('保存'));
    await tester.pumpAndSettle();

    expect(provider.changedTagId, 10);
    expect(provider.changedLevel, 'parent');
    expect(provider.changedParentId, 99);
  });

  testWidgets('edit mode same level with different parent uses reparentTag', (
    tester,
  ) async {
    final provider = _TagEditProvider(
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    const row = TagGovernanceRow(
      tagId: 11,
      preferredLabel: 'hairclip',
      primaryCategory: 'appearance',
      aliases: <String>[],
      usageCount: 1,
      pendingCount: 0,
      confirmedCount: 1,
      rejectedCount: 0,
      aiCount: 0,
      manualCount: 1,
      affectedImageCount: 1,
      canDelete: false,
      level: 'child',
      parentId: 100,
    );

    await tester.pumpWidget(
      buildDialog(provider, const TagEditDialog(row: row)),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(fluent.ComboBox<int?>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('无父标签').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('保存'));
    await tester.pumpAndSettle();

    expect(provider.reparentedTagId, 11);
    expect(provider.reparentedParentId, isNull);
  });
}
