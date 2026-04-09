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
    apiService = ApiService(
      baseUrl: 'http://localhost:8080',
      client: mockClient,
    );
  });

  group('ApiService', () {
    group('fetchImages', () {
      test('parses backend response with images array', () async {
        // Arrange - backend returns 'images' array, not 'items'
        final responseBody = '''
        {
          "images": [
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
              "created_at": "2024-01-01T00:00:00Z",
              "updated_at": "2024-01-01T00:00:00Z"
            }
          ],
          "next_cursor": "20",
          "has_more": true,
          "total": 100
        }
        ''';

        when(
          mockClient.get(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await apiService.fetchImages();

        // Assert
        expect(result.items.length, 1);
        expect(result.items[0].id, 1);
        expect(result.nextCursor, '20');
        expect(result.hasMore, true);
        expect(result.total, 100);

        final captured =
            verify(
                  mockClient.get(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.path, contains('/api/v1/images'));
      });

      test('parses empty next_cursor when has_more is false', () async {
        // Arrange - last page has empty next_cursor
        final responseBody = '''
        {
          "images": [
            {
              "id": 3,
              "path": "/path/to/image3.jpg",
              "filename": "image3.jpg",
              "source_root": "/path/to",
              "file_size": 51200,
              "width": 400,
              "height": 300,
              "format": "jpg",
              "phash": 0,
              "created_at": "2024-01-01T00:00:00Z",
              "updated_at": "2024-01-01T00:00:00Z"
            }
          ],
          "next_cursor": "",
          "has_more": false,
          "total": 3
        }
        ''';

        when(
          mockClient.get(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await apiService.fetchImages();

        // Assert
        expect(result.nextCursor, isEmpty);
        expect(result.hasMore, false);
      });

      test('sends offset parameter instead of cursor for pagination', () async {
        // Arrange
        final responseBody = '''
        {
          "images": [],
          "next_cursor": "",
          "has_more": false,
          "total": 0
        }
        ''';

        when(
          mockClient.get(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act - use offset for next page
        await apiService.fetchImages(offset: 20);

        // Assert
        final captured =
            verify(
                  mockClient.get(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.queryParameters['offset'], '20');
      });

      test('serializes tagIds into query parameters', () async {
        // Arrange
        final responseBody = '''
        {
          "images": [],
          "next_cursor": "",
          "has_more": false,
          "total": 0
        }
        ''';

        when(
          mockClient.get(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        await apiService.fetchImages(tagIds: [1, 2, 3]);

        // Assert
        final captured =
            verify(
                  mockClient.get(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.queryParameters['tag_ids'], '1,2,3');
      });

      test('combines all query parameters correctly', () async {
        // Arrange
        final responseBody = '''
        {
          "images": [],
          "next_cursor": "",
          "has_more": false,
          "total": 0
        }
        ''';

        when(
          mockClient.get(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        await apiService.fetchImages(
          tagIds: [5, 10],
          offset: 40,
          limit: 50,
          sortBy: 'filename',
          sortDir: 'asc',
        );

        // Assert
        final captured =
            verify(
                  mockClient.get(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.queryParameters['tag_ids'], '5,10');
        expect(captured.queryParameters['offset'], '40');
        expect(captured.queryParameters['limit'], '50');
        expect(captured.queryParameters['sort_by'], 'filename');
        expect(captured.queryParameters['sort_dir'], 'asc');
      });
    });

    group('image actions', () {
      test('openImageSourceFile sends POST to action endpoint', () async {
        when(
          mockClient.post(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response('{"status":"ok"}', 200));

        await apiService.openImageSourceFile(42);

        final captured =
            verify(
                  mockClient.post(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.path, contains('/api/v1/images/42/open-source'));
      });

      test('permanentDeleteImage sends DELETE to action endpoint', () async {
        when(
          mockClient.delete(any, headers: anyNamed('headers')),
        ).thenAnswer((_) async => http.Response('{"status":"deleted"}', 200));

        await apiService.permanentDeleteImage(7);

        final captured =
            verify(
                  mockClient.delete(captureAny, headers: anyNamed('headers')),
                ).captured.single
                as Uri;
        expect(captured.path, contains('/api/v1/images/7/permanent'));
      });
    });
  });
}
