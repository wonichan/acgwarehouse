import 'dart:io';

Map<String, String> Function() runtimeManifestEnvironmentProvider = () =>
    Platform.environment;
String Function() runtimeManifestExecutablePathProvider = () =>
    Platform.resolvedExecutable;

Future<String?> readManifestText(String path) async {
  final file = File(path);
  if (!await file.exists()) {
    return null;
  }
  return file.readAsString();
}

String resolveRuntimeManifestPath() {
  final configured =
      runtimeManifestEnvironmentProvider()['ACG_RUNTIME_MANIFEST_PATH'];
  if (configured != null && configured.trim().isNotEmpty) {
    return configured.trim();
  }

  final executablePath = runtimeManifestExecutablePathProvider().trim();
  if (executablePath.isNotEmpty) {
    final bundleDir = File(executablePath).parent.path;
    final packagedServer = File(
      '$bundleDir${Platform.pathSeparator}runtime${Platform.pathSeparator}bin${Platform.pathSeparator}acgwarehouse-server.exe',
    );
    if (packagedServer.existsSync()) {
      return '$bundleDir${Platform.pathSeparator}runtime${Platform.pathSeparator}runtime-manifest.json';
    }
  }

  final separator = Platform.pathSeparator;
  return '${Directory.systemTemp.path}${separator}acgwarehouse${separator}runtime-manifest.json';
}
