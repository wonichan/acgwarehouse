import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

/// Window manager utility for Windows desktop
/// Handles window initialization, sizing, and title management
class AppWindowManager {
  static const double defaultWidth = 1280.0;
  static const double defaultHeight = 720.0;
  static const double minWidth = 800.0;
  static const double minHeight = 600.0;

  /// Initialize window manager with default settings
  static Future<void> ensureInitialized() async {
    // Must call WidgetsFlutterBinding.ensureInitialized() before
    await windowManager.ensureInitialized();

    const windowOptions = WindowOptions(
      size: Size(defaultWidth, defaultHeight),
      minimumSize: Size(minWidth, minHeight),
      center: true,
      backgroundColor: Colors.transparent,
      skipTaskbar: false,
      titleBarStyle: TitleBarStyle.normal,
      title: 'ACGWarehouse',
    );

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