import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../config/api_config.dart';

/// Configuration provider - Single Source of Truth for all runtime configuration.
///
/// Manages:
/// - Backend base URL
/// - Admin Basic Auth header
///
/// Persists settings via SharedPreferences so they survive app restarts.
class ConfigProvider extends ChangeNotifier {
  static const String _baseUrlKey = 'config_baseUrl';
  static const String _adminAuthKey = 'config_adminAuth';
  static const String _thumbnailBaseUrlKey = 'config_thumbnailBaseUrl';
  static const String _defaultBaseUrl = ApiConfig.developmentFallbackHostUrl;

  ConfigProvider({
    String? initialBaseUrl,
    String? initialAdminAuth,
    String? initialThumbnailBaseUrl,
  })
    : _baseUrl = initialBaseUrl ?? _defaultBaseUrl,
      _adminBasicAuthHeader = initialAdminAuth?.trim().isEmpty == true
          ? null
          : initialAdminAuth,
      _thumbnailBaseUrl = initialThumbnailBaseUrl?.trim().isEmpty == true
          ? null
          : initialThumbnailBaseUrl;

  String _baseUrl;
  String? _adminBasicAuthHeader;
  String? _thumbnailBaseUrl;

  /// Current backend base URL (without /api/v1 suffix).
  /// Example: 'http://localhost:8080'
  String get baseUrl => _baseUrl;

  /// Full API base URL with /api/v1 suffix.
  /// Example: 'http://localhost:8080/api/v1'
  String get apiBaseUrl => '$baseUrl/api/v1';

  /// Admin Basic Auth header for monitoring endpoints.
  String? get adminBasicAuthHeader => _adminBasicAuthHeader;

  /// Frontend-accessible thumbnail base URL used for resolving relative paths.
  String? get thumbnailBaseUrl => _thumbnailBaseUrl;

  String? resolveThumbnailUrl(String? rawUrl) => ApiConfig.resolveThumbnailUrl(
    rawUrl,
    thumbnailBaseUrl: _thumbnailBaseUrl,
  );

  /// Sets the backend base URL.
  ///
  /// [url] should be the base URL without the /api/v1 path suffix.
  /// Example: 'http://localhost:8080' or 'https://api.example.com'
  Future<void> setBaseUrl(String url) async {
    final normalized = url.endsWith('/')
        ? url.substring(0, url.length - 1)
        : url;

    if (_baseUrl != normalized) {
      _baseUrl = normalized;
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_baseUrlKey, normalized);
      notifyListeners();
      debugPrint('ConfigProvider: baseUrl changed to $normalized');
    }
  }

  /// Sets the admin Basic Auth header.
  Future<void> setAdminBasicAuthHeader(String? value) async {
    final normalized = value?.trim();
    final newAuth = (normalized == null || normalized.isEmpty)
        ? null
        : normalized;

    if (_adminBasicAuthHeader != newAuth) {
      _adminBasicAuthHeader = newAuth;
      final prefs = await SharedPreferences.getInstance();
      if (newAuth == null) {
        await prefs.remove(_adminAuthKey);
      } else {
        await prefs.setString(_adminAuthKey, newAuth);
      }
      notifyListeners();
      debugPrint('ConfigProvider: adminBasicAuthHeader updated');
    }
  }

  Future<void> setThumbnailBaseUrl(String? value) async {
    final normalized = value?.trim();
    final newValue = (normalized == null || normalized.isEmpty)
        ? null
        : (normalized.endsWith('/')
              ? normalized.substring(0, normalized.length - 1)
              : normalized);

    if (_thumbnailBaseUrl != newValue) {
      _thumbnailBaseUrl = newValue;
      final prefs = await SharedPreferences.getInstance();
      if (newValue == null) {
        await prefs.remove(_thumbnailBaseUrlKey);
      } else {
        await prefs.setString(_thumbnailBaseUrlKey, newValue);
      }
      notifyListeners();
      debugPrint('ConfigProvider: thumbnailBaseUrl updated');
    }
  }

  /// Loads persisted config from SharedPreferences.
  static Future<ConfigProvider> loadPersisted() async {
    final prefs = await SharedPreferences.getInstance();
    return ConfigProvider(
      initialBaseUrl: prefs.getString(_baseUrlKey),
      initialAdminAuth: prefs.getString(_adminAuthKey),
      initialThumbnailBaseUrl: prefs.getString(_thumbnailBaseUrlKey),
    );
  }

  /// Resets all configuration to defaults.
  Future<void> resetToDefault() async {
    if (_baseUrl != _defaultBaseUrl ||
        _adminBasicAuthHeader != null ||
        _thumbnailBaseUrl != null) {
      _baseUrl = _defaultBaseUrl;
      _adminBasicAuthHeader = null;
      _thumbnailBaseUrl = null;
      final prefs = await SharedPreferences.getInstance();
      await prefs.remove(_baseUrlKey);
      await prefs.remove(_adminAuthKey);
      await prefs.remove(_thumbnailBaseUrlKey);
      notifyListeners();
      debugPrint('ConfigProvider: reset to default');
    }
  }

  /// Checks if current configuration matches default.
  bool get isDefault =>
      _baseUrl == _defaultBaseUrl &&
      _adminBasicAuthHeader == null &&
      _thumbnailBaseUrl == null;
}
