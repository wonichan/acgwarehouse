import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:http/http.dart' as http;
import 'package:gallery/services/tag_service.dart';

import 'tag_service_test.mocks.dart';

@GenerateMocks([http.Client])
void main() {
  late TagService tagService;
  late MockClient mockClient;

  setUp(() {
    mockClient = MockClient();
    tagService = TagService(client: mockClient);
  });

  group('TagService', () {
    group('getTagStatistics', () {
      test('parses wrapped {"stats": [...]} response from backend', () async {
        // Arrange - backend returns wrapped response
        final responseBody = '''
        {
          "stats": [
            {
              "tag_id": 1,
              "label": "anime",
              "usage_count": 100,
              "pending_count": 5,
              "confirmed_count": 95,
              "ai_count": 80,
              "manual_count": 20
            }
          ]
        }
        ''';

        when(mockClient.get(any))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await tagService.getTagStatistics();

        // Assert
        expect(result.length, 1);
        expect(result[0].tagId, 1);
        expect(result[0].label, 'anime');
        expect(result[0].usageCount, 100);
        expect(result[0].pendingCount, 5);
        expect(result[0].confirmedCount, 95);
        expect(result[0].aiCount, 80);
        expect(result[0].manualCount, 20);

        verify(mockClient.get(any)).called(1);
      });

      test('returns empty list when stats array is empty', () async {
        // Arrange
        final responseBody = '''
        {
          "stats": []
        }
        ''';

        when(mockClient.get(any))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await tagService.getTagStatistics();

        // Assert
        expect(result, isEmpty);
      });

      test('handles multiple statistics entries', () async {
        // Arrange
        final responseBody = '''
        {
          "stats": [
            {
              "tag_id": 1,
              "label": "anime",
              "usage_count": 100
            },
            {
              "tag_id": 2,
              "label": "manga",
              "usage_count": 50
            }
          ]
        }
        ''';

        when(mockClient.get(any))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await tagService.getTagStatistics();

        // Assert
        expect(result.length, 2);
        expect(result[0].tagId, 1);
        expect(result[1].tagId, 2);
      });

      test('throws exception on non-200 status code', () async {
        // Arrange
        when(mockClient.get(any))
            .thenAnswer((_) async => http.Response('Error', 500));

        // Act & Assert
        expect(
          () => tagService.getTagStatistics(),
          throwsA(isA<Exception>()),
        );
      });
    });
  });
}