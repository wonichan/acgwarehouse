import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;

import 'providers/image_provider.dart';
import 'providers/tag_provider.dart';
import 'providers/duplicate_provider.dart';
import 'providers/search_provider.dart';
import 'providers/navigation_provider.dart';
import 'services/api_service.dart';
import 'services/tag_service.dart';
import 'services/duplicate_service.dart';
import 'services/search_service.dart';
import 'app/adaptive_app.dart';
import 'app/fluent_app_shell.dart';
import 'app/material_app_shell.dart';
import 'utils/window_manager.dart';

void main() async {
  // Ensure Flutter binding is initialized
  WidgetsFlutterBinding.ensureInitialized();

  // Initialize window manager for Windows desktop
  if (defaultTargetPlatform == TargetPlatform.windows) {
    await AppWindowManager.ensureInitialized();
  }

  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider(create: (_) => ApiService()),
        Provider(create: (_) => TagService()),
        Provider(create: (_) => DuplicateService()),
        Provider(create: (_) => SearchService()),
        ChangeNotifierProvider(
            create: (context) =>
                ImageListProvider(context.read<ApiService>())..loadImages()),
        ChangeNotifierProvider(
            create: (context) => TagProvider(context.read<TagService>())),
        ChangeNotifierProvider(
            create: (context) =>
                DuplicateProvider(service: context.read<DuplicateService>())),
        ChangeNotifierProvider(
            create: (context) =>
                SearchProvider(service: context.read<SearchService>())),
        ChangeNotifierProvider(create: (_) => NavigationProvider()),
      ],
      child: const AdaptiveApp(
        fluentAppBuilder: _buildFluentApp,
        materialAppBuilder: _buildMaterialApp,
      ),
    );
  }
}

/// FluentApp - Windows 桌面端
Widget _buildFluentApp() {
  return fluent.FluentApp(
    title: 'ACGWarehouse',
    theme: fluent.FluentThemeData(
      accentColor: fluent.Colors.blue,
    ),
    home: const FluentAppShell(),
    // ScaffoldMessenger is needed for dialogs to show SnackBar feedback
    builder: (context, child) {
      return ScaffoldMessenger(
        child: child ?? const SizedBox.shrink(),
      );
    },
  );
}

/// MaterialApp - Android/Web 平台
Widget _buildMaterialApp() {
  return MaterialApp(
    title: 'ACGWarehouse',
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
      useMaterial3: true,
    ),
    home: const MaterialAppShell(),
  );
}