import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/config/api_config.dart';

void main() {
  const testBase = 'http://localhost:8080';

  group('ApiConfig endpoint builders', () {
    test('baseUrlOf adds /api/v1 suffix', () {
      expect(ApiConfig.baseUrlOf(testBase), 'http://localhost:8080/api/v1');
    });

    test('images returns full image list URL', () {
      expect(ApiConfig.images(testBase), 'http://localhost:8080/api/v1/images');
    });

    test('imageDetail injects ID', () {
      expect(
        ApiConfig.imageDetail(testBase, 42),
        'http://localhost:8080/api/v1/images/42',
      );
    });

    test('tags returns full tags URL', () {
      expect(ApiConfig.tags(testBase), 'http://localhost:8080/api/v1/tags');
    });

    test('imageTags combines image ID', () {
      expect(
        ApiConfig.imageTags(testBase, 7),
        'http://localhost:8080/api/v1/images/7/tags',
      );
    });

    test('monitoringWs converts http to ws', () {
      expect(
        ApiConfig.monitoringWs(testBase),
        'ws://localhost:8080/admin/api/monitoring/ws',
      );
    });

    test('logStreamWs includes source and tail', () {
      expect(
        ApiConfig.logStreamWs(testBase, source: 'go', tail: 100),
        'ws://localhost:8080/admin/api/logs/ws?source=go&tail=100',
      );
    });

    test('admin endpoints use hostUrl without /api/v1', () {
      expect(
        ApiConfig.adminOverview(testBase),
        'http://localhost:8080/admin/api/task-platform/overview',
      );
      expect(
        ApiConfig.adminBatches(testBase),
        'http://localhost:8080/admin/api/task-batches',
      );
    });

    test('adminTasks adds batchId query parameter', () {
      expect(
        ApiConfig.adminTasks(testBase, batchId: 42),
        'http://localhost:8080/admin/api/tasks?batch_id=42',
      );
      expect(
        ApiConfig.adminTasks(testBase),
        'http://localhost:8080/admin/api/tasks',
      );
    });

    test('developmentFallbackHostUrl is the expected default', () {
      expect(ApiConfig.developmentFallbackHostUrl, 'http://localhost:8080');
    });

    test('resolveThumbnailUrl builds absolute URL from relative path', () {
      expect(
        ApiConfig.resolveThumbnailUrl(
          'acg/thumbnails/20260419/example-large.jpg',
          thumbnailBaseUrl: 'http://118.25.139.30:19003',
        ),
        'http://118.25.139.30:19003/acg/thumbnails/20260419/example-large.jpg',
      );
    });

    test('resolveThumbnailUrl keeps existing absolute URL', () {
      expect(
        ApiConfig.resolveThumbnailUrl(
          'https://cdn.example.com/acg/thumbnails/example-large.jpg',
          thumbnailBaseUrl: 'http://118.25.139.30:19003',
        ),
        'https://cdn.example.com/acg/thumbnails/example-large.jpg',
      );
    });
  });
}
