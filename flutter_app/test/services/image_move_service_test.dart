import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image_move.dart';
import 'package:gallery/services/image_move_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  group('ImageMoveService', () {
    test('preview posts move request and parses preview response', () async {
      final client = MockClient((request) async {
        expect(request.method, 'POST');
        expect(request.url.path, '/api/v1/image-moves/preview');

        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['source_dirs'], ['E:/picture/output']);
        expect(body['tag_id'], 7);
        expect(body['target_dir'], 'E:/picture/archive');
        expect(body['conflict'], 'skip');
        expect(body['limit'], 1000);

        return http.Response(
          jsonEncode({
            'total_matched': 2,
            'movable': 1,
            'skipped': 1,
            'items': [
              {
                'image_id': 10,
                'filename': 'alpha.png',
                'source_path': 'E:/picture/output/alpha.png',
                'target_path': 'E:/picture/archive/alpha.png',
                'status': 'movable',
              },
              {
                'image_id': 11,
                'filename': 'beta.png',
                'source_path': 'E:/picture/output/beta.png',
                'target_path': 'E:/picture/archive/beta.png',
                'status': 'skipped',
                'reason': 'target_exists',
              },
            ],
          }),
          200,
        );
      });

      final service = ImageMoveService(
        baseUrl: 'http://localhost:8080',
        client: client,
      );

      final preview = await service.preview(
        const ImageMoveRequest(
          sourceDirs: ['E:/picture/output'],
          tagId: 7,
          targetDir: 'E:/picture/archive',
        ),
      );

      expect(preview.totalMatched, 2);
      expect(preview.movable, 1);
      expect(preview.skipped, 1);
      expect(preview.items.last.reason, 'target_exists');
    });

    test('execute posts move request and parses result response', () async {
      final client = MockClient((request) async {
        expect(request.url.path, '/api/v1/image-moves/execute');
        return http.Response(
          jsonEncode({
            'total_matched': 2,
            'moved': 1,
            'skipped': 0,
            'failed': 1,
            'items': [
              {
                'image_id': 10,
                'filename': 'alpha.png',
                'source_path': 'E:/picture/output/alpha.png',
                'target_path': 'E:/picture/archive/alpha.png',
                'status': 'moved',
              },
              {
                'image_id': 12,
                'filename': 'gamma.png',
                'source_path': 'E:/picture/output/gamma.png',
                'target_path': 'E:/picture/archive/gamma.png',
                'status': 'failed',
                'reason': 'move_failed',
              },
            ],
          }),
          200,
        );
      });

      final service = ImageMoveService(
        baseUrl: 'http://localhost:8080',
        client: client,
      );

      final result = await service.execute(
        const ImageMoveRequest(
          sourceDirs: ['E:/picture/output'],
          tagId: 7,
          targetDir: 'E:/picture/archive',
        ),
      );

      expect(result.moved, 1);
      expect(result.failed, 1);
      expect(result.items.last.status, 'failed');
      expect(result.items.last.reason, 'move_failed');
    });

    test('createJob and history use phase 3 endpoints', () async {
      final service = ImageMoveService(
        baseUrl: 'http://localhost:8080',
        client: MockClient((request) async {
          if (request.url.path == '/api/v1/image-moves/jobs') {
            final body = jsonDecode(request.body) as Map<String, dynamic>;
            expect(body['conflict'], 'rename');
            return http.Response(
              jsonEncode({
                'id': 12,
                'tag_id': 7,
                'source_dirs': ['E:/src'],
                'target_dir': 'E:/dst',
                'conflict_strategy': 'rename',
                'total_matched': 2,
                'moved': 0,
                'skipped': 0,
                'failed': 0,
                'status': 'queued',
                'created_at': '2026-05-10 00:00:00',
                'progress': {
                  'total': 2,
                  'processed': 0,
                  'moved': 0,
                  'skipped': 0,
                  'failed': 0,
                },
              }),
              202,
            );
          }

          expect(request.url.path, '/api/v1/image-moves/history');
          return http.Response(
            jsonEncode({
              'items': [
                {
                  'id': 12,
                  'tag_id': 7,
                  'source_dirs': ['E:/src'],
                  'target_dir': 'E:/dst',
                  'conflict_strategy': 'rename',
                  'total_matched': 2,
                  'moved': 2,
                  'skipped': 0,
                  'failed': 0,
                  'status': 'completed',
                  'created_at': '2026-05-10 00:00:00',
                  'progress': {
                    'total': 2,
                    'processed': 2,
                    'moved': 2,
                    'skipped': 0,
                    'failed': 0,
                  },
                },
              ],
            }),
            200,
          );
        }),
      );

      final job = await service.createJob(
        const ImageMoveRequest(
          sourceDirs: ['E:/src'],
          tagId: 7,
          targetDir: 'E:/dst',
          conflict: 'rename',
        ),
      );
      expect(job.id, 12);
      expect(job.status, 'queued');

      final history = await service.history();
      expect(history.single.status, 'completed');
    });
  });
}
