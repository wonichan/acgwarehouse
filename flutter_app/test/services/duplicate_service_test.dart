import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:http/http.dart' as http;
import 'package:mocktail/mocktail.dart';

class MockHttpClient extends Mock implements http.Client {}

class FakeUri extends Fake implements Uri {}

void main() {
  late DuplicateService duplicateService;
  late MockHttpClient mockClient;

  setUpAll(() {
    registerFallbackValue(FakeUri());
  });

  setUp(() {
    mockClient = MockHttpClient();
    duplicateService = DuplicateService(client: mockClient);
  });

  group('DuplicateService', () {
    test('parses detect response as async task payload', () async {
      const responseBody = '''
      {
        "task_id": "dup-123",
        "status": "queued",
        "progress": 0,
        "processed": 0,
        "total": 0,
        "message": "queued"
      }
      ''';

      when(
        () => mockClient.post(
          any(),
          headers: any(named: 'headers'),
          body: any(named: 'body'),
        ),
      ).thenAnswer((_) async => http.Response(responseBody, 200));

      final result = await duplicateService.detectDuplicates(threshold: 10);
      expect(result.taskId, 'dup-123');
      expect(result.status, 'queued');
      expect(result.progress, 0);
      expect(result.processed, 0);
      expect(result.total, 0);
    });

    test('parses duplicate task status response', () async {
      const responseBody = '''
      {
        "task_id": "dup-123",
        "status": "hashing",
        "progress": 42.5,
        "processed": 85,
        "total": 200,
        "message": "hashing",
        "groups_found": 3
      }
      ''';

      when(
        () => mockClient.get(any(), headers: any(named: 'headers')),
      ).thenAnswer((_) async => http.Response(responseBody, 200));

      final status = await duplicateService.getDuplicateTaskStatus('dup-123');
      expect(status.taskId, 'dup-123');
      expect(status.status, 'hashing');
      expect(status.progress, 42.5);
      expect(status.processed, 85);
      expect(status.total, 200);
      expect(status.groupsFound, 3);
    });

    test(
      'parses duplicate group list using backend group/images shape',
      () async {
        const responseBody = '''
      {
        "groups": [
          {
            "group": {
              "id": 7,
              "recommended_image_id": 101,
              "similarity_threshold": 10,
              "created_at": "2024-01-01T00:00:00Z"
            },
            "images": [
              {
                "id": 101,
                "path": "/images/a.jpg",
                "filename": "a.jpg",
                "source_root": "/images",
                "width": 1000,
                "height": 800,
                "file_size": 2048,
                "format": "jpg",
                "phash": 123,
                "thumbnail_small_url": "https://example.com/thumb-a.jpg",
                "thumbnail_large_url": "https://example.com/large-a.jpg",
                "created_at": "2024-01-01T00:00:00Z",
                "updated_at": "2024-01-01T00:00:00Z",
                "is_recommended": true,
                "file_hash": "hash-a",
                "phash_distance": 0
              },
              {
                "id": 102,
                "path": "/images/b.jpg",
                "filename": "b.jpg",
                "source_root": "/images",
                "width": 900,
                "height": 700,
                "file_size": 1024,
                "format": "jpg",
                "phash": 456,
                "thumbnail_small_url": "https://example.com/thumb-b.jpg",
                "thumbnail_large_url": "https://example.com/large-b.jpg",
                "created_at": "2024-01-01T00:00:00Z",
                "updated_at": "2024-01-01T00:00:00Z",
                "is_recommended": false,
                "file_hash": "hash-b",
                "phash_distance": 3
              }
            ]
          }
        ],
        "total": 1,
        "has_more": false
      }
      ''';

        when(
          () => mockClient.get(any(), headers: any(named: 'headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final response = await duplicateService.getDuplicateGroups();
        final groups = response.groups;

        expect(groups, hasLength(1));
        expect(groups.first.id, 7);
        expect(groups.first.recommendedImageId, 101);
        expect(groups.first.relations, hasLength(2));
        expect(groups.first.relations.first.imageId, 101);
        expect(groups.first.relations.first.isRecommended, isTrue);
        expect(
          groups.first.relations.first.image?.thumbnailSmallUrl,
          'https://example.com/thumb-a.jpg',
        );
        expect(groups.first.relations.last.pHashDistance, 3);
      },
    );

    test(
      'parses duplicate group detail using backend group/images shape',
      () async {
        const responseBody = '''
      {
        "group": {
          "id": 8,
          "recommended_image_id": 201,
          "similarity_threshold": 5,
          "created_at": "2024-01-02T00:00:00Z"
        },
        "images": [
          {
            "id": 201,
            "path": "/images/c.jpg",
            "filename": "c.jpg",
            "source_root": "/images",
            "width": 1200,
            "height": 900,
            "file_size": 4096,
            "format": "jpg",
            "phash": 789,
            "thumbnail_small_url": "https://example.com/thumb-c.jpg",
            "thumbnail_large_url": "https://example.com/large-c.jpg",
            "created_at": "2024-01-02T00:00:00Z",
            "updated_at": "2024-01-02T00:00:00Z",
            "is_recommended": true,
            "file_hash": "hash-c",
            "phash_distance": 0
          },
          {
            "id": 202,
            "path": "/images/d.jpg",
            "filename": "d.jpg",
            "source_root": "/images",
            "width": 1180,
            "height": 880,
            "file_size": 4000,
            "format": "jpg",
            "phash": 790,
            "thumbnail_small_url": "https://example.com/thumb-d.jpg",
            "thumbnail_large_url": "https://example.com/large-d.jpg",
            "created_at": "2024-01-02T00:00:00Z",
            "updated_at": "2024-01-02T00:00:00Z",
            "is_recommended": false,
            "file_hash": "hash-d",
            "phash_distance": 2
          }
        ]
      }
      ''';

        when(
          () => mockClient.get(any(), headers: any(named: 'headers')),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final group = await duplicateService.getDuplicateGroup(8);

        expect(group.id, 8);
        expect(group.recommendedImageId, 201);
        expect(group.relations, hasLength(2));
        expect(group.relations.last.imageId, 202);
        expect(
          group.relations.last.image?.thumbnailSmallUrl,
          'https://example.com/thumb-d.jpg',
        );
        expect(group.relations.last.pHashDistance, 2);
      },
    );
  });
}
