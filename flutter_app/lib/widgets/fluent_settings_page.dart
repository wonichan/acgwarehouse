import 'package:fluent_ui/fluent_ui.dart';

/// Fluent 风格设置页面
/// Phase 10 完整实现主题切换等配置
class FluentSettingsPage extends StatelessWidget {
  const FluentSettingsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return ScaffoldPage(
      header: const PageHeader(
        title: Text('设置'),
      ),
      content: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(
              FluentIcons.settings,
              size: 64,
            ),
            const SizedBox(height: 16),
            Text(
              '设置功能开发中...',
              style: FluentTheme.of(context).typography.subtitle,
            ),
            const SizedBox(height: 8),
            Text(
              '将在后续版本中添加主题切换、API 配置等功能',
              style: FluentTheme.of(context).typography.body,
            ),
          ],
        ),
      ),
    );
  }
}
