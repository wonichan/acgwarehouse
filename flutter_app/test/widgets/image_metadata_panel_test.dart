import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

class _MetadataTagProvider extends TagProvider {
  _MetadataTagProvider({http.Client? client})
    : super(TagService(baseUrl: 'http://localhost:8080', client: client));

  List<Tag> _filtered = const [];

  @override
  Map<String, List<Tag>> get imageTags => {
    'confirmed': const [],
    'pending': [
      Tag(
        id: 1,
        preferredLabel: 'heroine',
        slug: 'heroine',
        reviewState: 'pending',
        trustScore: 1,
        usageCount: 3,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'child',
        parentId: 20,
      ),
    ],
    'rejected': const [],
  };

  @override
  List<Tag> get filteredTags => _filtered;

  @override
  Future<String> getDefaultAIPrompt() async => 'default';

  @override
  Future<void> loadImageTags(int imageId) async {}

  @override
  Future<void> searchTags(String query) async {
    _filtered = [
      Tag(
        id: 2,
        preferredLabel: 'supporting-cast',
        slug: 'supporting-cast',
        reviewState: 'confirmed',
        trustScore: 1,
        usageCount: 8,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'child',
        parentId: 20,
      ),
      Tag(
        id: 3,
        preferredLabel: 'characters-root',
        slug: 'characters-root',
        reviewState: 'confirmed',
        trustScore: 1,
        usageCount: 12,
        createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
        level: 'root',
      ),
    ];
  }
}

void main() {
  testWidgets('image-detail merge dialog hides cross-level targets', (
    tester,
  ) async {
    final provider = _MetadataTagProvider(
      client: MockClient((_) async => http.Response('{}', 200)),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: ChangeNotifierProvider<TagProvider>.value(
          value: provider,
          child: const Scaffold(
            body: ImageMetadataPanel(
              imageId: 1,
              metadataSection: SizedBox.shrink(),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final gesture = await tester.createGesture(kind: PointerDeviceKind.mouse);
    addTearDown(gesture.removePointer);
    await gesture.addPointer(location: tester.getCenter(find.text('heroine')));
    await gesture.moveTo(tester.getCenter(find.text('heroine')));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.merge_type).first);
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField).last, 'cast');
    await tester.pumpAndSettle();

    expect(find.text('supporting-cast'), findsOneWidget);
    expect(find.text('characters-root'), findsNothing);
  });
}
