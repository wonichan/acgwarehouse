import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:http/http.dart' as http;
import 'package:gallery/services/api_service.dart';

import 'api_service_test.mocks.dart';

@GenerateMocks([http.Client])
void main() {
  late ApiService apiService;
  late MockClient mockClient;

  setUp(() {
    mockClient = MockClient();
    apiService = ApiService(client: mockClient);
  });

  group('ApiService', () {
    group('fetchImages', () {
      test('fetches images without tag filter', () async {
        // Arrange
        final responseBody = '''
        {
          "items": [
            {
              "id": 1,
              "path": "/path/to/image1.jpg",
              "filename": "image1.jpg",
              "source_root": "/path/to",
              "file_size": 102400,
              "width": 800,
              "height": 600,
              "format": "jpg",
              "phash": 12345,
              "thumbnail_small_url": "http://example.com/thumb1.jpg",
              "thumbnail_large_url": "http://example.com/large1.jpg",
              "created_at": "2024-01-01T00:00:00Z",
              "updated_at": "2024-01-01T00:00:00Z"
            }
          ],
          "next_cursor": "cursor123",
          "has_more": true
        }
        ''';
        
        when(mockClient.get(any, headers: anyNamed('headers')))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await apiService.fetchImages();

        // Assert
        expect(result.items.length, 1);
        expect(result.items[0].id, 1);
        expect(result.nextCursor, 'cursor123');
        expect(result.hasMore, true);
        
        final captured = verify(mockClient.get(captureAny, headers: anyNamed('headers'))).captured.single as Uri;
        expect(captured.path, contains('/api/v1/images'));
      });

      test('serializes tagIds into query parameters', () async {
        // Arrange
        final responseBody = '''
        {
          "items": [],
          "next_cursor": null,
          "has_more": false
        }
        ''';
        
        when(mockClient.get(any, headers: anyNamed('headers')))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        await apiService.fetchImages(tagIds: [1, 2, 3]);

        // Assert
        final captured = verify(mockClient.get(captureAny, headers: anyNamed('headers'))).captured.single as Uri;
        expect(captured.queryParameters['tag_ids'], '1,2,3');
      });

      test('combines tagIds with other query parameters', () async {
        // Arrange
        final responseBody = '''
        {
          "items": [],
          "next_cursor": null,
          "has_more": false
        }
        ''';
        
        when(mockClient.get(any, headers: anyNamed('headers')))
            .thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        await apiService.fetchImages(
          tagIds: [5, 10],
          cursor: 'abc123',
          limit: 50,
          sortBy: 'filename',
          sortDir: 'asc',
        );

        // Assert
        final captured = verify(mockClient.get(captureAny, headers: anyNamed('headers'))).captured.single as Uri;
        expect(captured.queryParameters['tag_ids'], '5,10');
        expect(captured.queryParameters['cursor'], 'abc123');
        expect(captured.queryParameters['limit'], '50');
        expect(captured.queryParameters['sort_by'], 'filename');
        expect(captured.queryParameters['sort_dir'], 'asc');
      });
    });
  });
}