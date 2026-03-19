import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, TargetPlatform, kIsWeb;

/// 平台自适应应用入口
/// Windows 桌面使用 Fluent UI，Web/Android/iOS 使用 Material 3
class AdaptiveApp extends StatelessWidget {
  final Widget Function() fluentAppBuilder;
  final Widget Function() materialAppBuilder;

  const AdaptiveApp({
    super.key,
    required this.fluentAppBuilder,
    required this.materialAppBuilder,
  });

  @override
  Widget build(BuildContext context) {
    // Web 平台始终使用 Material UI
    // Windows 桌面使用 Fluent UI
    // 其他平台使用 Material UI
    final bool useFluent = !kIsWeb &&
        defaultTargetPlatform == TargetPlatform.windows;

    if (useFluent) {
      return fluentAppBuilder();
    } else {
      return materialAppBuilder();
    }
  }
}