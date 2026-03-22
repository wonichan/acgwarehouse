import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/theme_provider.dart';

/// Fluent 风格设置页面
/// Phase 10 完整实现主题切换等配置
class FluentSettingsPage extends StatelessWidget {
  const FluentSettingsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<ThemeProvider>(
      builder: (context, themeProvider, _) {
        return ScaffoldPage(
          header: const PageHeader(
            title: Text('设置'),
          ),
          content: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('外观', style: FluentTheme.of(context).typography.subtitle),
                const SizedBox(height: 12),
                _buildThemeTile(context, themeProvider, ThemeMode.system, '跟随系统', FluentIcons.bulleted_list),
                _buildThemeTile(context, themeProvider, ThemeMode.light, '浅色', FluentIcons.sunny),
                _buildThemeTile(context, themeProvider, ThemeMode.dark, '深色', FluentIcons.clear_night),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildThemeTile(
    BuildContext context,
    ThemeProvider provider,
    ThemeMode mode,
    String label,
    IconData icon,
  ) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: ListTile.selectable(
        selected: provider.themeMode == mode,
        leading: Icon(icon),
        title: Text(label),
        onPressed: () => provider.setThemeMode(mode),
      ),
    );
  }
}
