/// API configuration for backend endpoints
///
/// baseUrl can be updated at runtime via [updateBaseUrl].
/// Default: 'http://localhost:8080/api/v1'
class ApiConfig {
  // Default base URL without /api/v1 suffix
  static const String developmentFallbackHostUrl = 'http://localhost:8080';
  static const String _defaultHostUrl = developmentFallbackHostUrl;
  static String _hostUrl = _defaultHostUrl;

  /// Current host URL (without /api/v1 suffix)
  /// Example: 'http://localhost:8080'
  static String get hostUrl => _hostUrl;

  /// Current base URL (with /api/v1 suffix)
  /// Example: 'http://localhost:8080/api/v1'
  static String get baseUrl => '$hostUrl/api/v1';

  /// Updates the host URL
  ///
  /// [url] should be the base URL without /api/v1 suffix.
  /// Example: 'http://localhost:8080'
  static void updateBaseUrl(String url) {
    // Normalize URL - remove trailing slash
    final normalized = url.endsWith('/')
        ? url.substring(0, url.length - 1)
        : url;
    _hostUrl = normalized;
  }

  /// Resets to default configuration
  static void resetToDefault() {
    _hostUrl = _defaultHostUrl;
  }

  /// Applies the explicit development fallback host.
  static void applyDevelopmentFallback() {
    _hostUrl = developmentFallbackHostUrl;
  }

  /// Checks if current configuration matches default
  static bool get isDefault => _hostUrl == _defaultHostUrl;

  // Image endpoints
  static String get images => '$baseUrl/images';
  static String imageDetail(int id) => '$baseUrl/images/$id';
  static String get importStatus => '$baseUrl/images/import-status';

  // Tag endpoints
  static String get tags => '$baseUrl/tags';
  static String tagDetail(int id) => '$baseUrl/tags/$id';
  static String tagAliases(int id) => '$baseUrl/tags/$id/aliases';
  static String tagAlias(int tagId, int aliasId) =>
      '$baseUrl/tags/$tagId/aliases/$aliasId';

  // Image tag endpoints
  static String imageTags(int imageId) => '$baseUrl/images/$imageId/tags';
  static String imageTag(int imageId, int tagId) =>
      '$baseUrl/images/$imageId/tags/$tagId';
  static String tagReview(int imageId, int tagId) =>
      '$baseUrl/images/$imageId/tags/$tagId/review';
  static String batchTagReview(int imageId) =>
      '$baseUrl/images/$imageId/tags/batch-review';

  // AI tag endpoints
  static String triggerAITags(int imageId) =>
      '$baseUrl/images/$imageId/ai-tags';
  static String aiTagStatus(int imageId) =>
      '$baseUrl/images/$imageId/ai-tags/status';
  static String get batchAITags => '$baseUrl/images/batch-ai-tags';
  static String get defaultAIPrompt => '$baseUrl/ai-tags/default-prompt';

  // Duplicate detection endpoints
  static String get duplicates => '$baseUrl/duplicates';
  static String duplicateDetail(int id) => '$baseUrl/duplicates/$id';
  static String get detectDuplicates => '$baseUrl/duplicates/detect';

  // Search endpoints
  static String get search => '$baseUrl/search';
  static String get searchByFilename => '$baseUrl/search/filename';
}
