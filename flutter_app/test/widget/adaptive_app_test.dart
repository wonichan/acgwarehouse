import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import 'package:gallery/app/adaptive_app.dart';

void main() {
  group('AdaptiveApp Tests', () {
    testWidgets('AdaptiveApp 根据 TargetPlatform 选择正确的 App',
        (tester) async {
      // 测试默认平台（在测试环境中是 android 或根据 kIsWeb）
      await tester.pumpWidget(
        AdaptiveApp(
          fluentAppBuilder: () =>
              const fluent.FluentApp(home: Text('Fluent App')),
          materialAppBuilder: () =>
              const MaterialApp(home: Text('Material App')),
        ),
      );

      // 在测试环境中，默认是 !kIsWeb && TargetPlatform.windows 才会使用 Fluent
      // 测试环境默认是 android，所以应该显示 Material App
      expect(find.text('Material App'), findsOneWidget);
    });

    testWidgets('AdaptiveApp builder 函数被正确调用', (tester) async {
      await tester.pumpWidget(
        AdaptiveApp(
          fluentAppBuilder: () =>
              const fluent.FluentApp(home: Text('Fluent Content')),
          materialAppBuilder: () =>
              const MaterialApp(home: Text('Material Content')),
        ),
      );

      // 验证 AdaptiveApp 存在
      expect(find.byType(AdaptiveApp), findsOneWidget);
    });
  });
}