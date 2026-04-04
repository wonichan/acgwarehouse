import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

const double defaultWidth = 1280.0;
const double defaultHeight = 720.0;
const double minWidth = 800.0;
const double minHeight = 600.0;
const double viewerDefaultWidth = 1440.0;
const double viewerDefaultHeight = 900.0;
const double viewerMinWidth = 960.0;
const double viewerMinHeight = 640.0;

@immutable
class AppWindowPolicy {
  final Size size;
  final Size minimumSize;
  final bool center;
  final Color backgroundColor;
  final bool skipTaskbar;
  final TitleBarStyle titleBarStyle;
  final String title;
  final bool rememberWindowState;

  const AppWindowPolicy({
    required this.size,
    required this.minimumSize,
    required this.center,
    required this.backgroundColor,
    required this.skipTaskbar,
    required this.titleBarStyle,
    required this.title,
    required this.rememberWindowState,
  });

  WindowOptions toWindowOptions() {
    return WindowOptions(
      size: size,
      minimumSize: minimumSize,
      center: center,
      backgroundColor: backgroundColor,
      skipTaskbar: skipTaskbar,
      titleBarStyle: titleBarStyle,
      title: title,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) {
      return true;
    }
    return other is AppWindowPolicy &&
        other.size == size &&
        other.minimumSize == minimumSize &&
        other.center == center &&
        other.backgroundColor == backgroundColor &&
        other.skipTaskbar == skipTaskbar &&
        other.titleBarStyle == titleBarStyle &&
        other.title == title &&
        other.rememberWindowState == rememberWindowState;
  }

  @override
  int get hashCode => Object.hash(
    size,
    minimumSize,
    center,
    backgroundColor,
    skipTaskbar,
    titleBarStyle,
    title,
    rememberWindowState,
  );
}

AppWindowPolicy mainWindowOptions([String title = 'ACGWarehouse']) {
  return AppWindowPolicy(
    size: const Size(defaultWidth, defaultHeight),
    minimumSize: const Size(minWidth, minHeight),
    center: true,
    backgroundColor: Colors.transparent,
    skipTaskbar: false,
    titleBarStyle: TitleBarStyle.normal,
    title: title,
    rememberWindowState: true,
  );
}

AppWindowPolicy viewerWindowOptions(String title) {
  return AppWindowPolicy(
    size: const Size(viewerDefaultWidth, viewerDefaultHeight),
    minimumSize: const Size(viewerMinWidth, viewerMinHeight),
    center: true,
    backgroundColor: Colors.transparent,
    skipTaskbar: false,
    titleBarStyle: TitleBarStyle.normal,
    title: title,
    rememberWindowState: false,
  );
}

String buildViewerWindowTitle(String filename) {
  return 'ACGWarehouse Viewer — $filename';
}

/// Window manager utility for Windows desktop
/// Handles window initialization, sizing, and title management
class AppWindowManager {
  /// Initialize window manager with default settings
  static Future<void> ensureInitialized({AppWindowPolicy? policy}) async {
    // Must call WidgetsFlutterBinding.ensureInitialized() before
    await windowManager.ensureInitialized();

    final resolvedPolicy = policy ?? mainWindowOptions();
    final windowOptions = resolvedPolicy.toWindowOptions();

    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
    });
  }

  /// Set window title
  static Future<void> setTitle(String title) async {
    await windowManager.setTitle(title);
  }

  /// Get current window size
  static Future<Size> getSize() async {
    return await windowManager.getSize();
  }

  /// Check if window is maximized
  static Future<bool> isMaximized() async {
    return await windowManager.isMaximized();
  }

  /// Maximize or restore window
  static Future<void> toggleMaximize() async {
    if (await isMaximized()) {
      await windowManager.unmaximize();
    } else {
      await windowManager.maximize();
    }
  }

  /// Minimize window
  static Future<void> minimize() async {
    await windowManager.minimize();
  }

  /// Close window
  static Future<void> close() async {
    await windowManager.close();
  }
}
