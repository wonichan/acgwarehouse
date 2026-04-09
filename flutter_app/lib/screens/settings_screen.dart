import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/theme_provider.dart';
import '../providers/config_provider.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer2<ThemeProvider, ConfigProvider>(
      builder: (context, themeProvider, configProvider, _) {
        return Scaffold(
          appBar: AppBar(title: const Text('设置')),
          body: ListView(
            children: [
              // Backend Configuration Section
              const ListTile(
                title: Text(
                  '后端配置',
                  style: TextStyle(fontWeight: FontWeight.bold),
                ),
              ),
              _BackendUrlTile(configProvider: configProvider),
              const Divider(),

              // Appearance Section
              const ListTile(
                title: Text(
                  '外观',
                  style: TextStyle(fontWeight: FontWeight.bold),
                ),
              ),
              RadioListTile<ThemeMode>(
                title: const Text('跟随系统'),
                secondary: const Icon(Icons.brightness_auto),
                value: ThemeMode.system,
                groupValue: themeProvider.themeMode,
                onChanged: (value) => themeProvider.setThemeMode(value!),
              ),
              RadioListTile<ThemeMode>(
                title: const Text('浅色'),
                secondary: const Icon(Icons.light_mode),
                value: ThemeMode.light,
                groupValue: themeProvider.themeMode,
                onChanged: (value) => themeProvider.setThemeMode(value!),
              ),
              RadioListTile<ThemeMode>(
                title: const Text('深色'),
                secondary: const Icon(Icons.dark_mode),
                value: ThemeMode.dark,
                groupValue: themeProvider.themeMode,
                onChanged: (value) => themeProvider.setThemeMode(value!),
              ),
              const Divider(),
            ],
          ),
        );
      },
    );
  }
}

/// Backend URL configuration tile with editing capability
class _BackendUrlTile extends StatefulWidget {
  final ConfigProvider configProvider;

  const _BackendUrlTile({required this.configProvider});

  @override
  State<_BackendUrlTile> createState() => _BackendUrlTileState();
}

class _BackendUrlTileState extends State<_BackendUrlTile> {
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

  void _saveUrl() async {
    final newUrl = _controller.text.trim();
    if (newUrl.isNotEmpty) {
      await widget.configProvider.setBaseUrl(newUrl);

      setState(() {
        _isEditing = false;
      });

      // Show confirmation
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('后端地址已更新为: $newUrl'),
          duration: const Duration(seconds: 2),
        ),
      );

      // Note: Services created before this change will still use old URL
      // User needs to refresh the app or we could rebuild providers
      showDialog(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('提示'),
          content: const Text('后端地址已更新。部分页面可能需要刷新才能生效。'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('知道了'),
            ),
          ],
        ),
      );
    }
  }

  void _resetToDefault() {
    widget.configProvider.resetToDefault();
    _controller.text = widget.configProvider.baseUrl;
    setState(() {
      _isEditing = false;
    });

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('已恢复默认后端地址'),
        duration: Duration(seconds: 2),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_isEditing) {
      return ListTile(
        title: const Text('后端地址'),
        subtitle: TextField(
          controller: _controller,
          decoration: InputDecoration(
            hintText: 'http://localhost:8080',
            helperText: '输入后端 API 地址（不含 /api/v1）',
            suffixIcon: IconButton(
              icon: const Icon(Icons.clear),
              onPressed: _cancelEditing,
            ),
          ),
          autofocus: true,
          onSubmitted: (_) => _saveUrl(),
        ),
        trailing: IconButton(
          icon: const Icon(Icons.check),
          onPressed: _saveUrl,
        ),
      );
    }

    return ListTile(
      title: const Text('后端地址'),
      subtitle: Text(widget.configProvider.baseUrl),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (!widget.configProvider.isDefault)
            IconButton(
              icon: const Icon(Icons.restore),
              tooltip: '恢复默认',
              onPressed: _resetToDefault,
            ),
          IconButton(
            icon: const Icon(Icons.edit),
            tooltip: '编辑',
            onPressed: _startEditing,
          ),
        ],
      ),
    );
  }
}
