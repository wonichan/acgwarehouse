import 'dart:async';
import 'dart:io' as io;
import 'dart:ui' as ui show AppExitResponse;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:window_manager/window_manager.dart'
    show windowManager, WindowListener, WindowOptions, TitleBarStyle;

import 'providers/provider_setup.dart';
import 'providers/theme_provider.dart';
import 'bootstrap/packaged_desktop_bootstrap.dart';
import 'bootstrap/single_instance_guard.dart';
import 'bootstrap/runtime_manifest_loader.dart';
import 'app/adaptive_app.dart';
import 'app/fluent_app_shell.dart';
import 'app/material_app_shell.dart';
import 'theme/app_theme.dart';
import 'theme/app_theme.dart';
import 'widgets/desktop_material_theme_bridge.dart';
import 'widgets/startup/startup_failure_screen.dart';
import 'widgets/startup/startup_progress_screen.dart';

void main(List<String> args) async {
  // Ensure Flutter binding is initialized
  WidgetsFlutterBinding.ensureInitialized();

  SingleInstanceGuard? singleInstanceGuard;

  if (!kIsWeb && defaultTargetPlatform == TargetPlatform.windows) {
    singleInstanceGuard = await SingleInstanceGuard.tryAcquire();
    if (singleInstanceGuard == null) {
      return;
    }
  }

  final packagedBootstrap = PackagedDesktopBootstrap();

  // Initialize window manager only for main window.
  if (defaultTargetPlatform == TargetPlatform.windows) {
    await windowManager.ensureInitialized();
    await windowManager.waitUntilReadyToShow(
      const WindowOptions(
        size: Size(1280, 720),
        minimumSize: Size(800, 600),
        center: true,
        backgroundColor: Colors.transparent,
        titleBarStyle: TitleBarStyle.normal,
        title: 'ACGWarehouse',
      ),
      () async {
        await windowManager.show();
        await windowManager.focus();
      },
    );
  }

  runApp(
    PackagedDesktopLaunchApp(
      packagedBootstrap: packagedBootstrap,
      singleInstanceGuard: singleInstanceGuard,
    ),
  );
}

class PackagedDesktopLaunchApp extends StatefulWidget {
  const PackagedDesktopLaunchApp({
    super.key,
    required this.packagedBootstrap,
    this.singleInstanceGuard,
    this.runtimeManifestLoader,
    this.isDevelopmentMode = !kReleaseMode,
    this.isDesktopTarget = !kIsWeb,
    this.childOverride,
  });

  final PackagedDesktopBootstrap packagedBootstrap;
  final SingleInstanceGuard? singleInstanceGuard;
  final RuntimeManifestLoader? runtimeManifestLoader;
  final bool isDevelopmentMode;
  final bool isDesktopTarget;
  final Widget? childOverride;

  @override
  State<PackagedDesktopLaunchApp> createState() =>
      _PackagedDesktopLaunchAppState();
}

class _PackagedDesktopLaunchAppState extends State<PackagedDesktopLaunchApp> {
  bool _isStarting = true;
  StartupFailure? _startupFailure;
  RuntimeManifestLoadResult? _manifestResult;

  @override
  void initState() {
    super.initState();
    unawaited(_start());
  }

  Future<void> _start() async {
    final result = await widget.packagedBootstrap.startIfNeeded();

    if (result.isSuccess) {
      _manifestResult =
          await (widget.runtimeManifestLoader ?? RuntimeManifestLoader()).load(
            isDevelopmentMode: widget.isDevelopmentMode,
            isDesktopTarget: widget.isDesktopTarget,
          );
    }

    if (!mounted) {
      return;
    }

    setState(() {
      _isStarting = false;
      _startupFailure = result.failure;
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_isStarting) {
      return const MaterialApp(
        home: StartupProgressScreen(
          title: '正在启动 ACGWarehouse',
          message: '正在启动后端服务，请稍候…',
        ),
      );
    }

    return MyApp(
      startupFailure: _startupFailure,
      packagedBootstrap: widget.packagedBootstrap,
      singleInstanceGuard: widget.singleInstanceGuard,
      manifestResult: _manifestResult,
      childOverride: widget.childOverride,
    );
  }
}

class MyApp extends StatelessWidget {
  const MyApp({
    super.key,
    this.startupFailure,
    this.packagedBootstrap,
    this.singleInstanceGuard,
    this.manifestResult,
    this.childOverride,
  });

  final StartupFailure? startupFailure;
  final PackagedDesktopBootstrap? packagedBootstrap;
  final SingleInstanceGuard? singleInstanceGuard;
  final Widget? childOverride;
  final RuntimeManifestLoadResult? manifestResult;

  @override
  Widget build(BuildContext context) {
    if (startupFailure != null) {
      return MaterialApp(home: StartupFailureScreen(failure: startupFailure!));
    }

    return MultiProvider(
      // ignore: invalid_use_of_visible_for_testing_member
      providers: [
        ...createAppProviders(
          manifestBaseUrl: manifestResult?.appliedBaseUrl,
          manifestAdminAuth: manifestResult?.appliedAdminBasicAuth,
        ),
      ],
      child: _ThemeBootstrapper(
        packagedBootstrap: packagedBootstrap,
        singleInstanceGuard: singleInstanceGuard,
        child:
            childOverride ??
            AdaptiveApp(
              fluentAppBuilder: _buildFluentApp,
              materialAppBuilder: _buildMaterialApp,
            ),
      ),
    );
  }
}

class _ThemeBootstrapper extends StatefulWidget {
  final Widget child;
  final PackagedDesktopBootstrap? packagedBootstrap;
  final SingleInstanceGuard? singleInstanceGuard;

  const _ThemeBootstrapper({
    required this.child,
    this.packagedBootstrap,
    this.singleInstanceGuard,
  });

  @override
  State<_ThemeBootstrapper> createState() => _ThemeBootstrapperState();
}

class _ThemeBootstrapperState extends State<_ThemeBootstrapper>
    with WindowListener, WidgetsBindingObserver {
  bool _scheduled = false;
  bool _closeHookAttached = false;
  bool _isClosing = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    unawaited(_syncWindowCloseHook());
  }

  @override
  void didUpdateWidget(covariant _ThemeBootstrapper oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.packagedBootstrap != widget.packagedBootstrap) {
      unawaited(_syncWindowCloseHook());
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    if (_closeHookAttached) {
      windowManager.removeListener(this);
      _closeHookAttached = false;
      unawaited(windowManager.setPreventClose(false));
    }
    if (!_isClosing) {
      unawaited(widget.packagedBootstrap?.shutdown() ?? Future<void>.value());
      unawaited(widget.singleInstanceGuard?.release() ?? Future<void>.value());
    }
    super.dispose();
  }

  @override
  Future<void> onWindowClose() async {
    if (!_closeHookAttached) {
      return;
    }

    await _handleCloseRequest();
  }

  @override
  Future<ui.AppExitResponse> didRequestAppExit() async {
    await _handleCloseRequest();
    return ui.AppExitResponse.cancel;
  }

  Future<void> _handleCloseRequest() async {
    if (_isClosing) {
      return;
    }

    setState(() {
      _isClosing = true;
    });

    try {
      // Run shutdown and release in parallel to avoid sequential delays
      final shutdownFuture =
          widget.packagedBootstrap?.shutdown() ?? Future.value();
      final releaseFuture =
          widget.singleInstanceGuard?.release() ?? Future.value();

      // Wait for both with a hard timeout to prevent hanging indefinitely
      // If Go server or other resources fail to shutdown cleanly,
      // we must proceed to force close to avoid orphan processes.
      await Future.wait<void>([shutdownFuture, releaseFuture]).timeout(
        const Duration(seconds: 10),
        onTimeout: () {
          debugPrint('Shutdown timed out. Proceeding with forced exit.');
          return <void>[];
        },
      );
    } catch (e) {
      debugPrint('Shutdown error: $e');
    } finally {
      windowManager.removeListener(this);
      _closeHookAttached = false;
      await windowManager.setPreventClose(false);
      await windowManager.destroy();

      // Force exit if the Dart VM is still hanging after UI destruction.
      // This ensures file handles are released and the process terminates
      // even if the underlying engine is stuck waiting for a resource.
      Future<void>.delayed(const Duration(seconds: 2), () {
        io.exit(0);
      });
    }
  }

  Future<void> _syncWindowCloseHook() async {
    final shouldAttach =
        defaultTargetPlatform == TargetPlatform.windows &&
        widget.packagedBootstrap != null;

    if (shouldAttach && !_closeHookAttached) {
      windowManager.addListener(this);
      _closeHookAttached = true;
      await windowManager.setPreventClose(true);
      return;
    }

    if (!shouldAttach && _closeHookAttached) {
      windowManager.removeListener(this);
      _closeHookAttached = false;
      await windowManager.setPreventClose(false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!_scheduled) {
      _scheduled = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        context.read<ThemeProvider>().initialize();
      });
    }

    return Directionality(
      textDirection: TextDirection.ltr,
      child: Stack(
        children: [
          IgnorePointer(ignoring: _isClosing, child: widget.child),
          if (_isClosing)
            const BlockingProgressOverlay(
              title: '正在关闭 ACGWarehouse',
              message: '正在关闭后端服务，请稍候…',
            ),
        ],
      ),
    );
  }
}

/// FluentApp - Windows 桌面端
Widget _buildFluentApp() {
  return Consumer<ThemeProvider>(
    builder: (context, themeProvider, _) {
      final brightness = switch (themeProvider.themeMode) {
        ThemeMode.dark => Brightness.dark,
        ThemeMode.light => Brightness.light,
        ThemeMode.system => MediaQuery.platformBrightnessOf(context),
      };

      return fluent.FluentApp(
        title: 'ACGWarehouse',
        theme: AppTheme.getFluentTheme(brightness),
        home: const FluentAppShell(),
        builder: (context, child) {
          return DesktopMaterialThemeBridge(
            brightness: brightness,
            child: child ?? const SizedBox.shrink(),
          );
        },
      );
    },
  );
}

/// MaterialApp - Android/Web 平台
Widget _buildMaterialApp() {
  return Consumer<ThemeProvider>(
    builder: (context, themeProvider, _) {
      final brightness = switch (themeProvider.themeMode) {
        ThemeMode.dark => Brightness.dark,
        ThemeMode.light => Brightness.light,
        ThemeMode.system => MediaQuery.platformBrightnessOf(context),
      };

      return MaterialApp(
        title: 'ACGWarehouse',
        theme: AppTheme.getMaterialTheme(brightness),
        themeMode: themeProvider.themeMode,
        home: const MaterialAppShell(),
      );
    },
  );
}
