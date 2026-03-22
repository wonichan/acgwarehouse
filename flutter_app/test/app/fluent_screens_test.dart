import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';

import 'package:gallery/app/fluent_screens.dart';
import 'package:gallery/screens/tag_management_screen.dart';

void main() {
  testWidgets('FluentTagManagementPage wraps TagManagementScreen in ScaffoldPage', (tester) async {
    await tester.pumpWidget(
      const fluent.FluentApp(
        home: FluentTagManagementPage(),
      ),
    );

    expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
    expect(find.byType(fluent.PageHeader), findsOneWidget);
    expect(find.text('标签管理'), findsWidgets);
    expect(find.byType(TagManagementScreen), findsOneWidget);
  });
}
