import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/theme_provider.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<ThemeProvider>(
      builder: (context, themeProvider, _) {
        return Scaffold(
          appBar: AppBar(title: const Text('设置')),
          body: ListView(
            children: [
              const ListTile(
                title: Text('外观', style: TextStyle(fontWeight: FontWeight.bold)),
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
