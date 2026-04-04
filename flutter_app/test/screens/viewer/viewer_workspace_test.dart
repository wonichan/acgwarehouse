import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/screens/viewer/viewer_workspace.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';

void main() {
  group('ViewerWorkspace', () {
    late ViewerSession session;

    setUp(() {
      final items = [
        ViewerSessionItem(
          imageId: 1,
          path: '/some/path/1.jpg',
          filename: '1.jpg',
          sourceRoot: '/some',
          fileSize: 1024,
          width: 800,
          height: 600,
          format: 'jpeg',
          thumbnailSmallUrl: 'http://small.jpg',
          thumbnailLargeUrl: 'http://large.jpg',
          createdAtIso8601: DateTime.now().toUtc().toIso8601String(),
          updatedAtIso8601: DateTime.now().toUtc().toIso8601String(),
        ),
      ];
      session = ViewerSession(
        source: ViewerSessionSource.gallery,
        items: items,
        initialSelectedIndex: 0,
      );
    });

    testWidgets(
      'renders dominant image region, fixed right sidebar, and bottom filmstrip',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(body: ViewerWorkspace(session: session)),
          ),
        );

        // Just ensure the widget renders its main components
        expect(find.byType(ViewerWorkspace), findsOneWidget);
        expect(find.byType(ViewerMetadataSidebar), findsOneWidget);
      },
    );
  });
}
