import 'dart:io';

Future<String?> readManifestText(String path) async {
  final file = File(path);
  if (!await file.exists()) {
    return null;
  }
  return file.readAsString();
}

String resolveRuntimeManifestPath() {
  final configured = Platform.environment['ACG_RUNTIME_MANIFEST_PATH'];
  if (configured != null && configured.trim().isNotEmpty) {
    return configured.trim();
  }

  final separator = Platform.pathSeparator;
  return '${Directory.systemTemp.path}${separator}acgwarehouse${separator}runtime-manifest.json';
}
