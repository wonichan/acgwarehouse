import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/bootstrap/runtime_manifest_loader.dart';
import 'package:gallery/bootstrap/runtime_manifest_loader_io.dart';
import 'package:gallery/config/api_config.dart';

void main() {
  group('RuntimeManifestLoader', () {
    setUp(() {
      ApiConfig.resetToDefault();
      runtimeManifestEnvironmentProvider = () => Platform.environment;
      runtimeManifestExecutablePathProvider = () => Platform.resolvedExecutable;
    });

    test('applies go.base_url from valid manifest before API usage', () async {
      final loader = RuntimeManifestLoader(
        readText: (_) async =>
            '{"version":1,"generated_at":"2026-04-04T10:00:00Z","go":{"base_url":"http://127.0.0.1:51423","ready":true,"admin_basic_auth":"Basic ZGVtbzpkZW1v"}}',
      );

      final result = await loader.load(
        isDevelopmentMode: false,
        isDesktopTarget: true,
      );

      expect(result.source, RuntimeManifestSource.manifest);
      expect(result.appliedBaseUrl, 'http://127.0.0.1:51423');
      expect(ApiConfig.hostUrl, 'http://127.0.0.1:51423');
      expect(ApiConfig.adminBasicAuthHeader, 'Basic ZGVtbzpkZW1v');
    });

    test(
      'uses localhost fallback only in development when manifest missing',
      () async {
        final loader = RuntimeManifestLoader(readText: (_) async => null);

        final result = await loader.load(
          isDevelopmentMode: true,
          isDesktopTarget: true,
        );

        expect(result.source, RuntimeManifestSource.devFallback);
        expect(result.appliedBaseUrl, ApiConfig.developmentFallbackHostUrl);
        expect(ApiConfig.hostUrl, ApiConfig.developmentFallbackHostUrl);
        expect(ApiConfig.adminBasicAuthHeader, isNull);
      },
    );

    test(
      'does not fallback in non-development when manifest missing',
      () async {
        final loader = RuntimeManifestLoader(readText: (_) async => null);

        final result = await loader.load(
          isDevelopmentMode: false,
          isDesktopTarget: true,
        );

        expect(result.source, RuntimeManifestSource.none);
        expect(result.appliedBaseUrl, isNull);
        expect(ApiConfig.hostUrl, ApiConfig.developmentFallbackHostUrl);
      },
    );

    test('does not ignore manifest in development mode', () async {
      final loader = RuntimeManifestLoader(
        readText: (_) async =>
            '{"version":1,"generated_at":"2026-04-04T10:00:00Z","go":{"base_url":"http://127.0.0.1:60001","ready":true}}',
      );

      final result = await loader.load(
        isDevelopmentMode: true,
        isDesktopTarget: true,
      );

      expect(result.source, RuntimeManifestSource.manifest);
      expect(result.appliedBaseUrl, 'http://127.0.0.1:60001');
      expect(ApiConfig.hostUrl, 'http://127.0.0.1:60001');
      expect(ApiConfig.hostUrl, isNot(ApiConfig.developmentFallbackHostUrl));
    });

    test('resolveRuntimeManifestPath prefers ACG_RUNTIME_MANIFEST_PATH', () {
      runtimeManifestEnvironmentProvider = () => <String, String>{
        'ACG_RUNTIME_MANIFEST_PATH': r'C:\bundle\runtime\runtime-manifest.json',
      };

      expect(
        resolveRuntimeManifestPath(),
        r'C:\bundle\runtime\runtime-manifest.json',
      );
    });

    test(
      'resolveRuntimeManifestPath falls back to bundle-local packaged path before temp fallback',
      () async {
        final tempDir = await Directory.systemTemp.createTemp(
          'manifest-loader-test',
        );
        addTearDown(() async {
          if (await tempDir.exists()) {
            await tempDir.delete(recursive: true);
          }
        });
        final bundleDir = Directory(
          '${tempDir.path}${Platform.pathSeparator}bundle',
        );
        await Directory(
          '${bundleDir.path}${Platform.pathSeparator}runtime${Platform.pathSeparator}bin',
        ).create(recursive: true);
        await File(
          '${bundleDir.path}${Platform.pathSeparator}runtime${Platform.pathSeparator}bin${Platform.pathSeparator}acgwarehouse-server.exe',
        ).writeAsString('go');

        runtimeManifestEnvironmentProvider = () => const <String, String>{};
        runtimeManifestExecutablePathProvider = () =>
            '${bundleDir.path}${Platform.pathSeparator}ACGWarehouse.exe';

        expect(
          resolveRuntimeManifestPath(),
          '${bundleDir.path}${Platform.pathSeparator}runtime${Platform.pathSeparator}runtime-manifest.json',
        );
      },
    );
  });
}
