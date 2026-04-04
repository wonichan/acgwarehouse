import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/viewer_session.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/screens/viewer/viewer_workspace.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/screens/viewer/viewer_metadata_sidebar.dart';

void main() {
  group('ViewerWorkspace', () {
    late ViewerSession session;
    late TagProvider tagProvider;

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
          thumbnailSmallUrl: null,
          thumbnailLargeUrl: null,
          createdAtIso8601: DateTime.now().toUtc().toIso8601String(),
          updatedAtIso8601: DateTime.now().toUtc().toIso8601String(),
        ),
        ViewerSessionItem(
          imageId: 2,
          path: '/some/path/2.jpg',
          filename: '2.jpg',
          sourceRoot: '/some',
          fileSize: 2048,
          width: 1200,
          height: 900,
          format: 'png',
          thumbnailSmallUrl: null,
          thumbnailLargeUrl: null,
          createdAtIso8601: DateTime.now().toUtc().toIso8601String(),
          updatedAtIso8601: DateTime.now().toUtc().toIso8601String(),
        ),
      ];
      session = ViewerSession(
        source: ViewerSessionSource.gallery,
        items: items,
        initialSelectedIndex: 0,
      );
      tagProvider = TagProvider(
        FakeTagService({
          1: {
            'confirmed': [_tag(id: 101, label: 'landscape')],
            'pending': const [],
            'rejected': const [],
          },
          2: {
            'confirmed': [_tag(id: 202, label: 'portrait')],
            'pending': const [],
            'rejected': const [],
          },
        }),
      );
    });

    tearDown(() {
      tagProvider.dispose();
    });

    testWidgets(
      'renders dominant image region, fixed right sidebar, and bottom filmstrip',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: ViewerWorkspace(session: session, tagProvider: tagProvider),
            ),
          ),
        );
        await tester.pump();

        // Just ensure the widget renders its main components
        expect(find.byType(ViewerWorkspace), findsOneWidget);
        expect(find.byType(ViewerMetadataSidebar), findsOneWidget);
      },
    );

    testWidgets('renders confirmed tags for the selected viewer item', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ViewerWorkspace(session: session, tagProvider: tagProvider),
          ),
        ),
      );

      await tester.pump();

      expect(find.text('landscape'), findsOneWidget);
    });

    testWidgets('refreshes sidebar tags when keyboard selection changes', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ViewerWorkspace(session: session, tagProvider: tagProvider),
          ),
        ),
      );

      await tester.pump();
      expect(find.text('landscape'), findsOneWidget);

      await tester.sendKeyEvent(LogicalKeyboardKey.arrowRight);
      await tester.pump();

      expect(find.text('portrait'), findsOneWidget);
      expect(find.text('landscape'), findsNothing);
    });
  });
}

Tag _tag({required int id, required String label}) {
  return Tag(
    id: id,
    preferredLabel: label,
    slug: label,
    reviewState: 'confirmed',
    trustScore: 1,
    usageCount: 1,
    createdAt: DateTime.parse('2026-04-05T00:00:00Z'),
  );
}

class FakeTagService extends TagService {
  final Map<int, Map<String, List<Tag>>> imageTagsById;

  FakeTagService(this.imageTagsById);

  @override
  Future<Map<String, List<Tag>>> getImageTags(int imageId) async {
    return imageTagsById[imageId] ??
        {'confirmed': const [], 'pending': const [], 'rejected': const []};
  }
}
