import 'package:flutter/foundation.dart';

import 'runtime_manifest_loader.dart';

Future<RuntimeManifestLoadResult> configureViewerWindowRuntime({
  RuntimeManifestLoader? loader,
  bool isDevelopmentMode = !kReleaseMode,
  bool isDesktopTarget = !kIsWeb,
}) {
  return (loader ?? RuntimeManifestLoader()).load(
    isDevelopmentMode: isDevelopmentMode,
    isDesktopTarget: isDesktopTarget,
  );
}
