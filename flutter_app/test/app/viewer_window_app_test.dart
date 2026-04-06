import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/app/viewer_window_app.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/services/viewer_window_service.dart';
import 'package:gallery/utils/window_manager.dart';

void main() {
  testWidgets('ViewerWindowApp renders viewer workspace content', (
    tester,
  ) async {
    final session = ViewerSession(
      source: ViewerSessionSource.gallery,
      items: const [
        ViewerSessionItem(
          imageId: 1,
          path: 'C:/images/alpha.png',
          filename: 'alpha.png',
          sourceRoot: 'C:/images',
          fileSize: 2048,
          width: 800,
          height: 600,
          format: 'png',
          thumbnailSmallUrl: null,
          thumbnailLargeUrl: null,
          createdAtIso8601: '2026-04-05T00:00:00.000Z',
          updatedAtIso8601: '2026-04-05T00:00:00.000Z',
        ),
      ],
      initialSelectedIndex: 0,
    );

    await tester.pumpWidget(
      ViewerWindowApp(
        bootstrapData: ViewerWindowBootstrapData(
          windowId: 1,
          session: session,
          policy: viewerWindowOptions(
            ViewerWindowService.buildWindowTitle('alpha.png'),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(fluent.FluentApp), findsOneWidget);
    expect(find.byType(fluent.ScaffoldPage), findsOneWidget);
    expect(find.text('Viewer - alpha.png'), findsOneWidget);
    expect(find.text('Image Details'), findsOneWidget);
  });
}
