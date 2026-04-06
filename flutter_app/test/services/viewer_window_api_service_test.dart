import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/viewer_window_context.dart';
import 'package:gallery/models/viewer_window_result.dart';
import 'package:gallery/services/api_service.dart';
import 'package:http/http.dart' as http;
import 'package:mocktail/mocktail.dart';

void main() {
  late ApiService apiService;
  late MockClient mockClient;

  setUp(() {
    mockClient = MockClient();
    apiService = ApiService(
      client: mockClient,
      baseUrl: 'http://localhost:8080',
    );
    registerFallbackValue(Uri.parse('http://localhost/fallback'));
  });

  group('ApiService.fetchViewerWindow', () {
    final request = ViewerWindowRequest(
      context: ViewerWindowContext.search(
        selectedIndex: 42,
        selectedImageId: 123,
        snapshot: const ViewerWindowSearchSnapshot(
          query: 'touhou',
          tagIds: [7, 8],
          sortBy: 'relevance',
          sortOrder: 'desc',
        ),
      ),
    );

    test(
      'posts typed request and parses typed viewer window response',
      () async {
        when(
          () => mockClient.post(
            any(),
            headers: any(named: 'headers'),
            body: any(named: 'body'),
          ),
        ).thenAnswer(
          (_) async => http.Response('''
              {
                "items": [
                  {
                    "id": 123,
                    "path": "E:/images/example.jpg",
                    "filename": "example.jpg",
                    "source_root": "E:/images",
                    "file_size": 123456,
                    "width": 1280,
                    "height": 720,
                    "format": "jpg",
                    "phash": 999,
                    "thumbnail_small_url": "/thumbs/example-small.jpg",
                    "thumbnail_large_url": "/thumbs/example.jpg",
                    "created_at": "2026-04-01T00:00:00Z",
                    "updated_at": "2026-04-01T00:00:00Z"
                  }
                ],
                "window_start_index": 38,
                "selected_index": 42,
                "selected_index_in_window": 4,
                "total": 615,
                "has_previous": true,
                "has_next": true
              }
              ''', 200),
        );

        final result = await apiService.fetchViewerWindow(request);

        expect(result, isA<ViewerWindowResult>());
        expect(result.items.single.id, 123);
        expect(result.selectedIndexInWindow, 4);
        expect(result.windowStartIndex, 38);
        expect(result.total, 615);
        expect(result.hasPrevious, isTrue);
        expect(result.hasNext, isTrue);

        final captured = verify(
          () => mockClient.post(
            captureAny(),
            headers: any(named: 'headers'),
            body: captureAny(named: 'body'),
          ),
        );
        expect((captured.captured[0] as Uri).path, '/api/v1/viewer/window');
        expect(captured.captured[1], contains('"selected_index":42'));
        expect(captured.captured[1], contains('"selected_image_id":123'));
        expect(captured.captured[1], contains('"q":"touhou"'));
      },
    );

    test('maps 400 responses to invalid viewer request errors', () async {
      when(
        () => mockClient.post(
          any(),
          headers: any(named: 'headers'),
          body: any(named: 'body'),
        ),
      ).thenAnswer(
        (_) async => http.Response(
          '{"error":"invalid_viewer_request","message":"selected_index is out of range"}',
          400,
        ),
      );

      await expectLater(
        () => apiService.fetchViewerWindow(request),
        throwsA(
          isA<ViewerWindowApiException>()
              .having((e) => e.statusCode, 'statusCode', 400)
              .having((e) => e.error, 'error', 'invalid_viewer_request'),
        ),
      );
    });

    test('maps 409 responses to snapshot drift errors', () async {
      when(
        () => mockClient.post(
          any(),
          headers: any(named: 'headers'),
          body: any(named: 'body'),
        ),
      ).thenAnswer(
        (_) async => http.Response(
          '{"error":"viewer_snapshot_drift","message":"selected image no longer matches"}',
          409,
        ),
      );

      await expectLater(
        () => apiService.fetchViewerWindow(request),
        throwsA(
          isA<ViewerWindowApiException>()
              .having((e) => e.statusCode, 'statusCode', 409)
              .having((e) => e.error, 'error', 'viewer_snapshot_drift'),
        ),
      );
    });

    test('maps 500 responses to viewer window failure errors', () async {
      when(
        () => mockClient.post(
          any(),
          headers: any(named: 'headers'),
          body: any(named: 'body'),
        ),
      ).thenAnswer(
        (_) async => http.Response(
          '{"error":"viewer_window_failed","message":"failed to load viewer window"}',
          500,
        ),
      );

      await expectLater(
        () => apiService.fetchViewerWindow(request),
        throwsA(
          isA<ViewerWindowApiException>()
              .having((e) => e.statusCode, 'statusCode', 500)
              .having((e) => e.error, 'error', 'viewer_window_failed'),
        ),
      );
    });
  });
}

class MockClient extends Mock implements http.Client {}
