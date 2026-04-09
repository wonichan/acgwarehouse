import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import '../providers/theme_provider.dart';
import '../providers/config_provider.dart';

/// Fluent 风格设置页面
/// Phase 10 完整实现主题切换等配置
class FluentSettingsPage extends StatelessWidget {
  const FluentSettingsPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer2<ThemeProvider, ConfigProvider>(
      builder: (context, themeProvider, configProvider, _) {
        return ScaffoldPage(
          header: const PageHeader(title: Text('设置')),
          content: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Backend Configuration Section
                Text(
                  '后端配置',
                  style: FluentTheme.of(context).typography.subtitle,
                ),
                const SizedBox(height: 12),
                _BackendUrlCard(configProvider: configProvider),
                const SizedBox(height: 24),

                // Appearance Section
                Text('外观', style: FluentTheme.of(context).typography.subtitle),
                const SizedBox(height: 12),
                _buildThemeTile(
                  context,
                  themeProvider,
                  ThemeMode.system,
                  '跟随系统',
                  FluentIcons.bulleted_list,
                ),
                _buildThemeTile(
                  context,
                  themeProvider,
                  ThemeMode.light,
                  '浅色',
                  FluentIcons.sunny,
                ),
                _buildThemeTile(
                  context,
                  themeProvider,
                  ThemeMode.dark,
                  '深色',
                  FluentIcons.clear_night,
                ),
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

/// Backend URL configuration card with editing capability
class _BackendUrlCard extends StatefulWidget {
  final ConfigProvider configProvider;

  const _BackendUrlCard({required this.configProvider});

  @override
  State<_BackendUrlCard> createState() => _BackendUrlCardState();
}

class _BackendUrlCardState extends State<_BackendUrlCard> {
  late TextEditingController _controller;
  bool _isEditing = false;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: widget.configProvider.baseUrl);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _startEditing() {
    setState(() {
      _isEditing = true;
      _controller.text = widget.configProvider.baseUrl;
    });
  }

  void _cancelEditing() {
    setState(() {
      _isEditing = false;
      _controller.text = widget.configProvider.baseUrl;
    });
  }

  void _saveUrl() {
    final newUrl = _controller.text.trim();
    if (newUrl.isNotEmpty) {
      widget.configProvider.setBaseUrl(newUrl);

      setState(() {
        _isEditing = false;
      });

      displayInfoBar(
        context,
        builder: (_, close) {
          return InfoBar(
            title: const Text('后端地址已更新'),
            content: Text('新地址: $newUrl'),
            severity: InfoBarSeverity.success,
            onClose: close,
          );
        },
      );
    }
  }

  void _resetToDefault() {
    widget.configProvider.resetToDefault();
    _controller.text = widget.configProvider.baseUrl;
    setState(() {
      _isEditing = false;
    });

    displayInfoBar(
      context,
      builder: (_, close) {
        return InfoBar(
          title: const Text('已恢复默认'),
          content: Text('后端地址: ${widget.configProvider.baseUrl}'),
          severity: InfoBarSeverity.info,
          onClose: close,
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_isEditing) {
      return Card(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('后端地址', style: TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            TextBox(
              controller: _controller,
              placeholder: 'http://localhost:8080',
              prefix: const Icon(FluentIcons.server),
            ),
            const SizedBox(height: 8),
            const Text(
              '输入后端 API 地址（不含 /api/v1）',
              style: TextStyle(fontSize: 12, color: Colors.grey),
            ),
            const SizedBox(height: 12),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                HyperlinkButton(
                  onPressed: _cancelEditing,
                  child: const Text('取消'),
                ),
                const SizedBox(width: 8),
                FilledButton(onPressed: _saveUrl, child: const Text('保存')),
              ],
            ),
          ],
        ),
      );
    }

    return Card(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          const Icon(FluentIcons.server, size: 20),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  '后端地址',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
                Text(widget.configProvider.baseUrl),
              ],
            ),
          ),
          if (!widget.configProvider.isDefault)
            IconButton(
              icon: const Icon(FluentIcons.refresh),
              onPressed: _resetToDefault,
            ),
          IconButton(
            icon: const Icon(FluentIcons.edit),
            onPressed: _startEditing,
          ),
        ],
      ),
    );
  }
}
