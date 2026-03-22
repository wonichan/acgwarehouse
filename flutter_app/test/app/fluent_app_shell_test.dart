import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:gallery/app/fluent_app_shell.dart';
import 'package:gallery/providers/navigation_provider.dart';
import 'package:gallery/screens/gallery_screen.dart';
import 'package:gallery/screens/search_screen.dart';
import 'package:gallery/screens/duplicate_screen.dart';
import 'package:gallery/screens/tag_management_screen.dart';
import 'package:gallery/widgets/fluent_settings_page.dart';

void main() {
  testWidgets('FluentAppShell exposes five navigation items and matching pages', (tester) async {
    final navProvider = NavigationProvider();

    await tester.pumpWidget(
      ChangeNotifierProvider<NavigationProvider>.value(
        value: navProvider,
        child: const fluent.FluentApp(
          home: FluentAppShell(),
        ),
      ),
    );

    expect(find.text('图库'), findsWidgets);
    expect(find.text('重复检测'), findsWidgets);
    expect(find.text('搜索'), findsWidgets);
    expect(find.text('标签管理'), findsWidgets);
    expect(find.text('设置'), findsWidgets);
    expect(find.byType(GalleryScreen), findsOneWidget);

    navProvider.setSelectedIndex(1);
    await tester.pumpAndSettle();
    expect(find.byType(DuplicateScreen), findsOneWidget);

    navProvider.setSelectedIndex(2);
    await tester.pumpAndSettle();
    expect(find.byType(SearchScreen), findsOneWidget);

    navProvider.setSelectedIndex(3);
    await tester.pumpAndSettle();
    expect(find.byType(TagManagementScreen), findsOneWidget);

    navProvider.setSelectedIndex(4);
    await tester.pumpAndSettle();
    expect(find.byType(FluentSettingsPage), findsOneWidget);
  });
}
