import 'dart:convert';

import '../config/api_config.dart';
import 'runtime_manifest_loader_stub.dart'
    if (dart.library.io) 'runtime_manifest_loader_io.dart'
    as runtime_manifest_io;

typedef RuntimeManifestTextReader = Future<String?> Function(String path);

enum RuntimeManifestSource { manifest, devFallback, none }

class RuntimeManifestLoadResult {
  final RuntimeManifestSource source;
  final String? appliedBaseUrl;
  final String? appliedAdminBasicAuth;

  const RuntimeManifestLoadResult({
    required this.source,
    this.appliedBaseUrl,
    this.appliedAdminBasicAuth,
  });
}

class RuntimeManifestLoader {
  RuntimeManifestLoader({
    RuntimeManifestTextReader? readText,
    String? manifestPath,
  }) : _readText = readText ?? runtime_manifest_io.readManifestText,
       _manifestPath =
           manifestPath ?? runtime_manifest_io.resolveRuntimeManifestPath();

  final RuntimeManifestTextReader _readText;
  final String _manifestPath;

  Future<RuntimeManifestLoadResult> load({
    required bool isDevelopmentMode,
    required bool isDesktopTarget,
  }) async {
    if (!isDesktopTarget || _manifestPath.isEmpty) {
      return const RuntimeManifestLoadResult(
        source: RuntimeManifestSource.none,
      );
    }

    final text = await _readText(_manifestPath);
    final discoveredBaseUrl = _extractGoBaseUrl(text);
    final discoveredAdminBasicAuth = _extractAdminBasicAuth(text);
    if (discoveredBaseUrl != null) {
      return RuntimeManifestLoadResult(
        source: RuntimeManifestSource.manifest,
        appliedBaseUrl: discoveredBaseUrl,
        appliedAdminBasicAuth: discoveredAdminBasicAuth,
      );
    }

    if (isDevelopmentMode) {
      return RuntimeManifestLoadResult(
        source: RuntimeManifestSource.devFallback,
        appliedBaseUrl: ApiConfig.developmentFallbackHostUrl,
      );
    }

    return const RuntimeManifestLoadResult(source: RuntimeManifestSource.none);
  }

  String? _extractGoBaseUrl(String? raw) {
    if (raw == null || raw.trim().isEmpty) {
      return null;
    }

    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        return null;
      }

      final go = decoded['go'];
      if (go is! Map<String, dynamic>) {
        return null;
      }

      final baseUrl = go['base_url'];
      if (baseUrl is! String || baseUrl.trim().isEmpty) {
        return null;
      }

      final normalized = baseUrl.trim().endsWith('/')
          ? baseUrl.trim().substring(0, baseUrl.trim().length - 1)
          : baseUrl.trim();
      final parsed = Uri.tryParse(normalized);
      if (parsed == null || !parsed.hasScheme || parsed.host.isEmpty) {
        return null;
      }

      return normalized;
    } catch (_) {
      return null;
    }
  }

  String? _extractAdminBasicAuth(String? raw) {
    if (raw == null || raw.trim().isEmpty) {
      return null;
    }

    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        return null;
      }

      final go = decoded['go'];
      if (go is! Map<String, dynamic>) {
        return null;
      }

      final adminBasicAuth = go['admin_basic_auth'];
      if (adminBasicAuth is! String || adminBasicAuth.trim().isEmpty) {
        return null;
      }

      return adminBasicAuth.trim();
    } catch (_) {
      return null;
    }
  }
}
