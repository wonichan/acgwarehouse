/// API endpoint builder - no runtime state, pure functions.
///
/// All URL construction derives from [baseUrl] parameter.
/// Configuration state lives exclusively in ConfigProvider.
///
/// Example usage:
/// ```dart
/// const baseUrl = 'http://localhost:8080';
/// final url = ApiConfig.imageDetail(baseUrl, 42);
/// // => 'http://localhost:8080/api/v1/images/42'
/// ```
class ApiConfig {
  ApiConfig._(); // prevent instantiation

  static const String developmentFallbackHostUrl = 'http://localhost:8080';

  // ---- Core helpers ----

  /// Returns base API URL with /api/v1 suffix.
  static String baseUrlOf(String baseUrl) => '$baseUrl/api/v1';

  // ---- Image endpoints ----
  static String images(String baseUrl) => '${baseUrlOf(baseUrl)}/images';
  static String imageDetail(String baseUrl, int id) =>
      '${baseUrlOf(baseUrl)}/images/$id';
  static String imageScan(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/images/scan';
  static String importStatus(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/images/import-status';

  // ---- Tag endpoints ----
  static String tags(String baseUrl) => '${baseUrlOf(baseUrl)}/tags';
  static String tagDetail(String baseUrl, int id) =>
      '${baseUrlOf(baseUrl)}/tags/$id';
  static String tagAliases(String baseUrl, int id) =>
      '${baseUrlOf(baseUrl)}/tags/$id/aliases';
  static String tagAlias(String baseUrl, int tagId, int aliasId) =>
      '${baseUrlOf(baseUrl)}/tags/$tagId/aliases/$aliasId';

  // ---- Image tag endpoints ----
  static String imageTags(String baseUrl, int imageId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/tags';
  static String imageTag(String baseUrl, int imageId, int tagId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/tags/$tagId';
  static String tagReview(String baseUrl, int imageId, int tagId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/tags/$tagId/review';
  static String batchTagReview(String baseUrl, int imageId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/tags/batch-review';

  // ---- AI tag endpoints ----
  static String triggerAITags(String baseUrl, int imageId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/ai-tags';
  static String aiTagStatus(String baseUrl, int imageId) =>
      '${baseUrlOf(baseUrl)}/images/$imageId/ai-tags/status';
  static String batchAITags(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/images/batch-ai-tags';
  static String batchRegenerateAITags(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/images/batch-ai-tags/regenerate';
  static String defaultAIPrompt(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/ai-tags/default-prompt';

  // ---- Search endpoints ----
  static String search(String baseUrl) => '${baseUrlOf(baseUrl)}/search';
  static String searchByFilename(String baseUrl) =>
      '${baseUrlOf(baseUrl)}/search/filename';

  // ---- Admin monitoring endpoints (under /admin/api, not /api/v1) ----
  static String adminOverview(String baseUrl) =>
      '$baseUrl/admin/api/task-platform/overview';
  static String adminBatches(String baseUrl) =>
      '$baseUrl/admin/api/task-batches';
  static String adminTasks(String baseUrl, {int? batchId}) {
    var url = '$baseUrl/admin/api/tasks';
    if (batchId != null) url += '?batch_id=$batchId';
    return url;
  }

  static String retryBatch(String baseUrl, int batchId) =>
      '$baseUrl/admin/api/actions/task-batches/$batchId/retry-failed';
  static String retryTask(String baseUrl, int taskId) =>
      '$baseUrl/admin/api/actions/tasks/$taskId/retry-failed';
  static String monitoringWs(String baseUrl) {
    final wsHost = baseUrl.replaceFirst('http', 'ws');
    return '$wsHost/admin/api/monitoring/ws';
  }

  static String logStreamWs(
    String baseUrl, {
    required String source,
    int tail = 200,
  }) {
    final wsHost = baseUrl.replaceFirst('http', 'ws');
    return '$wsHost/admin/api/logs/ws?source=$source&tail=$tail';
  }
}
