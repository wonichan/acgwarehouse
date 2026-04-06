import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/bootstrap/runtime_manifest_loader.dart';
import 'package:gallery/bootstrap/viewer_window_runtime_bootstrap.dart';
import 'package:gallery/config/api_config.dart';

void main() {
  group('configureViewerWindowRuntime', () {
    setUp(ApiConfig.resetToDefault);

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
        expect(ApiConfig.hostUrl, 'http://127.0.0.1:51423');
      },
    );
  });
}
