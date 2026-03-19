import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, TargetPlatform;

/// 平台自适应应用入口
/// Windows 使用 Fluent UI，其他平台使用 Material 3
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
    // 平台检测：Windows 使用 Fluent UI
    final isWindows = defaultTargetPlatform == TargetPlatform.windows;

    if (isWindows) {
      return fluentAppBuilder();
    } else {
      return materialAppBuilder();
    }
  }
}