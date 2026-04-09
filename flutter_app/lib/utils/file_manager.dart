import 'dart:io';

import 'package:flutter/foundation.dart';

/// Opens the parent folder of [filePath] in the system file manager.
/// On Windows, uses `/select` to highlight the file in Explorer.
/// On macOS, uses `-R` to reveal in Finder.
/// On Linux, uses `xdg-open` on the directory.
Future<void> openContainingFolder(String filePath) async {
  if (filePath.isEmpty) return;

  try {
    if (Platform.isWindows) {
      await Process.run('explorer', ['/select,', filePath]);
    } else if (Platform.isMacOS) {
      await Process.run('open', ['-R', filePath]);
    } else if (Platform.isLinux) {
      final dir = filePath.substring(
        0,
        filePath.lastIndexOf(Platform.pathSeparator),
      );
      await Process.run('xdg-open', [dir]);
    }
  } catch (e) {
    debugPrint('Failed to open containing folder: $e');
  }
}
