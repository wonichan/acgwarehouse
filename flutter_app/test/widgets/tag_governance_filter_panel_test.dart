import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag_governance_filter.dart';
import 'package:gallery/widgets/tag_management/tag_governance_filter_panel.dart';

void main() {
  testWidgets('summary chips use explicit readable text color', (tester) async {
    await tester.pumpWidget(
      FluentApp(
        home: ScaffoldPage(
          content: TagGovernanceFilterPanel(
            draftFilter: const TagGovernanceFilterState(),
            appliedFilter: const TagGovernanceFilterState(
              levels: {'root'},
              minUsageCount: 5,
            ),
            onDraftChanged: (_) {},
            onApply: () {},
            onReset: () {},
          ),
        ),
      ),
    );

    final levelText = tester.widget<Text>(find.text('层级: root'));
    final usageText = tester.widget<Text>(find.text('使用量: 5+'));

    expect(levelText.style?.color, isNotNull);
    expect(usageText.style?.color, isNotNull);
  });
}
