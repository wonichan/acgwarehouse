import 'package:flutter/foundation.dart';

/// Configuration provider for runtime-configurable settings
///
/// Supports modifying backend API URL at runtime, useful for:
/// - Development: switching between local/remote servers
/// - Production: allowing users to configure custom backend
class ConfigProvider extends ChangeNotifier {
  // Default values
  static const String _defaultBaseUrl = 'http://localhost:8080';

  String _baseUrl = _defaultBaseUrl;

  /// Current backend API base URL (without /api/v1 suffix)
  String get baseUrl => _baseUrl;

  /// Full API base URL with /api/v1 suffix
  String get apiBaseUrl => '$baseUrl/api/v1';

  /// Sets the backend base URL
  ///
  /// [url] should be the base URL without the /api/v1 path suffix
  /// Example: 'http://localhost:8080' or 'https://api.example.com'
  void setBaseUrl(String url) {
    // Normalize URL - remove trailing slash
    final normalized = url.endsWith('/')
        ? url.substring(0, url.length - 1)
        : url;

    if (_baseUrl != normalized) {
      _baseUrl = normalized;
      notifyListeners();
      debugPrint('ConfigProvider: baseUrl changed to $normalized');
    }
  }

  /// Resets to default configuration
  void resetToDefault() {
    if (_baseUrl != _defaultBaseUrl) {
      _baseUrl = _defaultBaseUrl;
      notifyListeners();
      debugPrint('ConfigProvider: baseUrl reset to default');
    }
  }

  /// Checks if current configuration matches default
  bool get isDefault => _baseUrl == _defaultBaseUrl;
}
