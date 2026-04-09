import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/bootstrap/runtime_manifest_loader.dart';
import 'package:gallery/bootstrap/viewer_window_runtime_bootstrap.dart';
import 'package:gallery/config/api_config.dart';

void main() {
  group('configureViewerWindowRuntime', () {
    test(
      'applies packaged runtime manifest for secondary viewer windows',
      () async {
        final loader = RuntimeManifestLoader(
          readText: (_) async =>
              '{"version":1,"go":{"base_url":"http://127.0.0.1:51423","ready":true}}',
        );

        final result = await configureViewerWindowRuntime(
          loader: loader,
          isDevelopmentMode: false,
          isDesktopTarget: true,
        );

        expect(result.source, RuntimeManifestSource.manifest);
        expect(result.appliedBaseUrl, 'http://127.0.0.1:51423');
      },
    );

    test(
      'falls back to development default when manifest is missing',
      () async {
        final loader = RuntimeManifestLoader(readText: (_) async => null);

        final result = await configureViewerWindowRuntime(
          loader: loader,
          isDevelopmentMode: true,
          isDesktopTarget: true,
        );

        expect(result.source, RuntimeManifestSource.devFallback);
        expect(result.appliedBaseUrl, ApiConfig.developmentFallbackHostUrl);
      },
    );
  });
}
