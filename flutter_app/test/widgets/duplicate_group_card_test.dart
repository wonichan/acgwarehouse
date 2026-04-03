import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:gallery/widgets/duplicate_group_card.dart';

void main() {
  group('DuplicateGroupCard', () {
    testWidgets(
      'renders real thumbnail when duplicate relation has thumbnail URL',
      (tester) async {
        final group = DuplicateGroup(
          id: 1,
          recommendedImageId: 11,
          similarityThreshold: 10,
          createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
          relations: [
            DuplicateRelation(
              imageId: 11,
              isRecommended: true,
              pHashDistance: 0,
              image: ImageModel(
                id: 11,
                path: '/images/a.jpg',
                filename: 'a.jpg',
                sourceRoot: '/images',
                fileSize: 2048,
                width: 800,
                height: 600,
                format: 'jpg',
                phash: 123,
                thumbnailSmallUrl: 'https://example.com/thumb.jpg',
                thumbnailLargeUrl: 'https://example.com/large.jpg',
                createdAt: DateTime.parse('2024-01-01T00:00:00Z'),
                updatedAt: DateTime.parse('2024-01-01T00:00:00Z'),
              ),
            ),
          ],
        );

        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(body: DuplicateGroupCard(group: group)),
          ),
        );

        expect(find.byType(Image), findsOneWidget);
        expect(find.byIcon(Icons.image), findsNothing);
      },
    );
  });
}
