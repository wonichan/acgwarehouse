import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:http/http.dart' as http;
import 'package:gallery/config/api_config.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/services/tag_service.dart';

import 'tag_service_test.mocks.dart';

@GenerateMocks([http.Client])
void main() {
  late TagService tagService;
  late MockClient mockClient;

  setUp(() {
    mockClient = MockClient();
    tagService = TagService(
      baseUrl: 'http://localhost:8080',
      client: mockClient,
    );
  });

  group('TagService', () {
    group('fetchTags', () {
      test('sends limit and offset for default tag paging', () async {
        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response('{"tags":[],"total":0}', 200));

        await tagService.fetchTags(limit: 30, offset: 60);

        final captured =
            verify(mockClient.get(captureAny)).captured.single as Uri;
        expect(captured.path, '/api/v1/tags');
        expect(captured.queryParameters['limit'], '30');
        expect(captured.queryParameters['offset'], '60');
        expect(captured.queryParameters.containsKey('search'), isFalse);
      });

      test('includes search query when provided', () async {
        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response('{"tags":[],"total":0}', 200));

        await tagService.fetchTags(search: 'hair', limit: 20, offset: 0);

        final captured =
            verify(mockClient.get(captureAny)).captured.single as Uri;
        expect(captured.path, '/api/v1/tags');
        expect(captured.queryParameters['search'], 'hair');
        expect(captured.queryParameters['limit'], '20');
        expect(captured.queryParameters['offset'], '0');
      });
    });

    group('triggerAITags', () {
      test('parses accepted batch response with job_ids list', () async {
        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer(
          (_) async => http.Response(
            '{"batch_id":1,"status":"queued","job_ids":[321]}',
            202,
          ),
        );

        final jobId = await tagService.triggerAITags(123);

        expect(jobId, 321);
      });

      test('keeps compatibility with legacy job_id response', () async {
        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer(
          (_) async => http.Response('{"status":"queued","job_id":888}', 202),
        );

        final jobId = await tagService.triggerAITags(123);

        expect(jobId, 888);
      });

      test('reuses existing job when backend skips repeated trigger', () async {
        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer(
          (_) async => http.Response(
            '{"status":"skipped","created_tasks":0,"skipped_tasks":1}',
            202,
          ),
        );
        when(mockClient.get(any)).thenAnswer(
          (_) async => http.Response('{"job_id":456,"status":"running"}', 200),
        );

        final jobId = await tagService.triggerAITags(123);

        expect(jobId, 456);
        final captured =
            verify(mockClient.get(captureAny)).captured.single as Uri;
        expect(
          captured.toString(),
          ApiConfig.aiTagStatus('http://localhost:8080', 123),
        );
      });
    });

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

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

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

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

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

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        // Act
        final result = await tagService.getTagStatistics();

        // Assert
        expect(result.length, 2);
        expect(result[0].tagId, 1);
        expect(result[1].tagId, 2);
      });

      test('throws exception on non-200 status code', () async {
        // Arrange
        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response('Error', 500));

        // Act & Assert
        expect(() => tagService.getTagStatistics(), throwsA(isA<Exception>()));
      });
    });

    group('governance contracts', () {
      test('createTag parses wrapped reused response contract', () async {
        final responseBody = '''
        {
          "id": 101,
          "reused": true,
          "tag": {
            "id": 101,
            "preferred_label": "anime-girl",
            "slug": "anime-girl",
            "review_state": "confirmed",
            "trust_score": 1.0,
            "usage_count": 42,
            "created_at": "2024-01-15T10:30:00Z",
            "level": "child",
            "parent_id": 10
          }
        }
        ''';

        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final tag = await tagService.createTag(
          preferredLabel: 'anime-girl',
          level: 'child',
          parentId: 10,
        );

        expect(tag.id, 101);
        expect(tag.preferredLabel, 'anime-girl');
        expect(tag.level, 'child');
        expect(tag.parentId, 10);
      });

      test('fetchGovernanceTags parses typed governance rows', () async {
        final responseBody = '''
        {
          "rows": [
            {
              "tag_id": 101,
              "preferred_label": "anime-girl",
              "primary_category": "character",
              "aliases": ["animegirl", "2d-girl"],
              "usage_count": 42,
              "pending_count": 3,
              "confirmed_count": 37,
              "rejected_count": 2,
              "ai_count": 28,
              "manual_count": 14,
              "direct_ai_count": 12,
              "tree_ai_count": 28,
              "direct_manual_count": 6,
              "tree_manual_count": 14,
              "affected_image_count": 42,
              "can_delete": false
            }
          ],
          "total": 1
        }
        ''';

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final rows = await tagService.fetchGovernanceTags(search: 'anime');

        expect(rows, hasLength(1));
        expect(rows.first, isA<TagGovernanceRow>());
        expect(rows.first.tagId, 101);
        expect(rows.first.primaryCategory, 'character');
        expect(rows.first.aliases, ['animegirl', '2d-girl']);
        expect(rows.first.affectedImageCount, 42);
        expect(rows.first.canDelete, false);
        expect(rows.first.directAiCount, 12);
        expect(rows.first.treeAiCount, 28);
        expect(rows.first.directManualCount, 6);
        expect(rows.first.treeManualCount, 14);

        final captured =
            verify(mockClient.get(captureAny)).captured.single as Uri;
        expect(captured.path, '/api/v1/tags/governance');
        expect(captured.queryParameters['search'], 'anime');
        expect(captured.queryParameters['limit'], '500');
      });

      test('fetchDeletePreview parses delete preview contract', () async {
        final responseBody = '''
        {
          "tag_id": 101,
          "preferred_label": "anime-girl",
          "affected_image_count": 42,
          "can_delete": false,
          "blocking_reason": "merge_or_reclassify_required"
        }
        ''';

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final preview = await tagService.fetchDeletePreview(101);

        expect(preview, isA<TagDeletePreview>());
        expect(preview.tagId, 101);
        expect(preview.preferredLabel, 'anime-girl');
        expect(preview.affectedImageCount, 42);
        expect(preview.canDelete, false);
        expect(preview.blockingReason, 'merge_or_reclassify_required');

        final captured =
            verify(mockClient.get(captureAny)).captured.single as Uri;
        expect(captured.path, '/api/v1/tags/101/delete-preview');
      });

      test('mergeTagInto sends exact target_tag_id body', () async {
        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer((_) async => http.Response('{}', 200));

        await tagService.mergeTagInto(101, 102);

        final captured = verify(
          mockClient.post(
            captureAny,
            headers: captureAnyNamed('headers'),
            body: captureAnyNamed('body'),
          ),
        );

        final uri = captured.captured[0] as Uri;
        final headers = captured.captured[1] as Map<String, String>;
        final body = captured.captured[2] as String;

        expect(uri.path, '/api/v1/tags/101/merge');
        expect(headers['Content-Type'], 'application/json');
        expect(body, '{"target_tag_id":102}');
      });

      test('alias CRUD wrappers hit alias endpoints', () async {
        final aliasesResponse = '''
        {
          "aliases": [
            {"id": 1, "label": "waifu", "alias_type": "synonym"}
          ]
        }
        ''';

        when(
          mockClient.get(any),
        ).thenAnswer((_) async => http.Response(aliasesResponse, 200));
        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer(
          (_) async => http.Response(
            '{"id":2,"label":"best-girl","alias_type":"synonym"}',
            201,
          ),
        );
        when(
          mockClient.delete(any),
        ).thenAnswer((_) async => http.Response('{}', 200));

        final aliases = await tagService.getTagAliases(101);
        await tagService.addTagAlias(101, 'best-girl', 'synonym');
        await tagService.deleteTagAlias(101, 2);

        expect(aliases, ['waifu']);
        verify(
          mockClient.get(
            Uri.parse(ApiConfig.tagAliases('http://localhost:8080', 101)),
          ),
        ).called(1);
        verify(
          mockClient.post(
            Uri.parse(ApiConfig.tagAliases('http://localhost:8080', 101)),
            headers: {'Content-Type': 'application/json'},
            body: '{"label":"best-girl","alias_type":"synonym"}',
          ),
        ).called(1);
        verify(
          mockClient.delete(
            Uri.parse(ApiConfig.tagAlias('http://localhost:8080', 101, 2)),
          ),
        ).called(1);
      });

      test('batchCleanupTags posts selected ids and parses result', () async {
        final responseBody = '''
        {
          "deleted": [{"tag_id": 1, "preferred_label": "unused-a"}],
          "blocked": [{"tag_id": 2, "preferred_label": "used-b", "message": "tag is still in use"}],
          "failed": [{"tag_id": 3, "preferred_label": "broken-c", "message": "db timeout"}]
        }
        ''';

        when(
          mockClient.post(
            any,
            headers: anyNamed('headers'),
            body: anyNamed('body'),
          ),
        ).thenAnswer((_) async => http.Response(responseBody, 200));

        final result = await tagService.batchCleanupTags([1, 2, 3]);

        expect(result, isA<TagGovernanceBatchResult>());
        expect(result.deletedTagIds, [1]);
        expect(result.failures, hasLength(2));
        expect(result.failures.first.tagId, 2);
        expect(result.failures.first.preferredLabel, 'used-b');

        final captured = verify(
          mockClient.post(
            captureAny,
            headers: captureAnyNamed('headers'),
            body: captureAnyNamed('body'),
          ),
        );

        final uri = captured.captured[0] as Uri;
        final body = captured.captured[2] as String;
        expect(uri.path, '/api/v1/tags/batch/cleanup');
        expect(body, '{"tag_ids":[1,2,3]}');
      });
    });
  });
}
