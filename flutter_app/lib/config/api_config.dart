class ApiConfig {
  static const String baseUrl = 'http://localhost:8080/api/v1';

  static String get images => '$baseUrl/images';
  static String imageDetail(int id) => '$baseUrl/images/$id';
  static String get importStatus => '$baseUrl/images/import-status';

  // Tag endpoints
  static String get tags => '$baseUrl/tags';
  static String tagDetail(int id) => '$baseUrl/tags/$id';
  static String tagAliases(int id) => '$baseUrl/tags/$id/aliases';
  static String tagAlias(int tagId, int aliasId) => '$baseUrl/tags/$tagId/aliases/$aliasId';

  // Image tag endpoints
  static String imageTags(int imageId) => '$baseUrl/images/$imageId/tags';
  static String imageTag(int imageId, int tagId) => '$baseUrl/images/$imageId/tags/$tagId';
  static String tagReview(int imageId, int tagId) => '$baseUrl/images/$imageId/tags/$tagId/review';
  static String batchTagReview(int imageId) => '$baseUrl/images/$imageId/tags/batch-review';

  // AI tag endpoints
  static String triggerAITags(int imageId) => '$baseUrl/images/$imageId/ai-tags';
  static String aiTagStatus(int imageId) => '$baseUrl/images/$imageId/ai-tags/status';
  static String get batchAITags => '$baseUrl/images/batch-ai-tags';

  // Duplicate detection endpoints
  static String get duplicates => '$baseUrl/duplicates';
  static String duplicateDetail(int id) => '$baseUrl/duplicates/$id';
  static String get detectDuplicates => '$baseUrl/duplicates/detect';

  // Search endpoints
  static String get search => '$baseUrl/search';
  static String get searchByFilename => '$baseUrl/search/filename';
}
