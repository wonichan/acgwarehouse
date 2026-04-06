import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/bootstrap/packaged_desktop_bootstrap.dart';
import 'package:gallery/main.dart';
import 'package:gallery/widgets/startup/startup_failure_screen.dart';
import 'package:gallery/widgets/startup/startup_progress_screen.dart';

void main() {
  group('PackagedDesktopBootstrap', () {
    late Directory tempDir;

    setUp(() async {
      tempDir = await Directory.systemTemp.createTemp(
        'packaged-bootstrap-test',
      );
    });

    tearDown(() async {
      if (await tempDir.exists()) {
        await tempDir.delete(recursive: true);
      }
    });

    test('resolves bundle-local runtime layout from executable path', () async {
      final harness = await _BootstrapHarness.create(tempDir);
      final bootstrap = harness.createBootstrap();

      final layout = bootstrap.resolveLayout();

      expect(layout.rootDir, harness.bundleDir.path);
      expect(layout.runtimeDir, _join(harness.bundleDir.path, 'runtime'));
      expect(
        layout.serverExecutablePath,
        _join(
          harness.bundleDir.path,
          'runtime',
          'bin',
          'acgwarehouse-server.exe',
        ),
      );
      expect(
        layout.sidecarExecutablePath,
        _join(
          harness.bundleDir.path,
          'runtime',
          'python-sidecar',
          'acgwarehouse-sidecar.exe',
        ),
      );
      expect(
        layout.manifestPath,
        _join(harness.bundleDir.path, 'runtime', 'runtime-manifest.json'),
      );
      expect(layout.logsDir, _join(harness.bundleDir.path, 'runtime', 'logs'));
      expect(
        layout.startupDiagnosticPath,
        _join(
          harness.bundleDir.path,
          'runtime',
          'diagnostics',
          'startup-error.json',
        ),
      );
    });

    test(
      'starts packaged go runtime with required environment variables',
      () async {
        final harness = await _BootstrapHarness.create(tempDir);
        final manifestFile = File(
          _join(harness.bundleDir.path, 'runtime', 'runtime-manifest.json'),
        );
        final bootstrap = harness.createBootstrap(
          onStart: () async {
            await manifestFile.writeAsString(
              '{"version":1,"generated_at":"2026-04-05T12:00:00Z","go":{"base_url":"http://127.0.0.1:19090","ready":true}}',
            );
          },
        );

        final result = await bootstrap.startIfNeeded();

        expect(result.isSuccess, isTrue);
        expect(harness.processStarts, hasLength(1));
        final started = harness.processStarts.single;
        expect(started.executable, endsWith('acgwarehouse-server.exe'));
        expect(
          started.environment['ACG_RUNTIME_ROOT'],
          harness.runtimeDir.path,
        );
        expect(started.environment['SERVER_HOST'], '127.0.0.1');
        expect(started.environment['SERVER_PORT'], isNotEmpty);
        expect(int.parse(started.environment['SERVER_PORT']!), greaterThan(0));
        expect(
          started.environment['ACG_RUNTIME_MANIFEST_PATH'],
          manifestFile.path,
        );
        expect(
          started.environment['ACG_DIAGNOSTICS_DIR'],
          _join(harness.runtimeDir.path, 'diagnostics'),
        );
        expect(
          started.environment['ACG_LOGS_DIR'],
          _join(harness.runtimeDir.path, 'logs'),
        );
        expect(
          started.environment['ACG_SIDECAR_EXECUTABLE'],
          _join(
            harness.runtimeDir.path,
            'python-sidecar',
            'acgwarehouse-sidecar.exe',
          ),
        );
        expect(started.environment['ACG_SIDECAR_PORT'], isNotEmpty);
        expect(
          int.parse(started.environment['ACG_SIDECAR_PORT']!),
          greaterThan(0),
        );
      },
    );

    test(
      'maps startup diagnostic component values to distinct failures',
      () async {
        final cases = <String, ({String title, StartupFailureType type})>{
          'go': (
            title: 'Go runtime failed to start',
            type: StartupFailureType.go,
          ),
          'python': (
            title: 'Python sidecar failed to start',
            type: StartupFailureType.python,
          ),
          'startup_chain': (
            title: 'Application startup did not complete',
            type: StartupFailureType.startupChain,
          ),
        };

        for (final entry in cases.entries) {
          final harness = await _BootstrapHarness.create(tempDir);
          final diagnosticFile = File(
            _join(
              harness.bundleDir.path,
              'runtime',
              'diagnostics',
              'startup-error.json',
            ),
          );
          final bootstrap = harness.createBootstrap(
            onStart: () async {
              await diagnosticFile.writeAsString(
                jsonEncode({
                  'component': entry.key,
                  'message': '${entry.key} exploded',
                  'log_paths': [
                    _join(harness.bundleDir.path, 'runtime', 'logs', 'go.log'),
                    _join(
                      harness.bundleDir.path,
                      'runtime',
                      'logs',
                      'python-sidecar.log',
                    ),
                  ],
                  'timestamp': '2026-04-05T12:00:00Z',
                }),
              );
            },
          );

          final result = await bootstrap.startIfNeeded();

          expect(result.isSuccess, isFalse);
          expect(result.failure, isNotNull);
          expect(result.failure!.type, entry.value.type);
          expect(result.failure!.title, entry.value.title);
          expect(result.failure!.message, contains('${entry.key} exploded'));
          expect(result.failure!.logPaths, isNotEmpty);
        }
      },
    );

    test(
      'returns startup-chain failure when manifest and diagnostic never appear',
      () async {
        final harness = await _BootstrapHarness.create(tempDir);
        final bootstrap = harness.createBootstrap();

        final result = await bootstrap.startIfNeeded();

        expect(result.isSuccess, isFalse);
        expect(result.failure, isNotNull);
        expect(result.failure!.type, StartupFailureType.startupChain);
        expect(result.failure!.message, contains('runtime-manifest.json'));
        expect(result.failure!.message, contains('go.log'));
        expect(result.failure!.message, contains('python-sidecar.log'));
      },
    );

    testWidgets('MyApp renders StartupFailureScreen when bootstrap fails', (
      tester,
    ) async {
      const failure = StartupFailure(
        type: StartupFailureType.python,
        title: 'Python sidecar failed to start',
        message: 'python exploded',
        logPaths: <String>['C:/bundle/runtime/logs/python-sidecar.log'],
      );

      await tester.pumpWidget(const MyApp(startupFailure: failure));

      expect(find.byType(StartupFailureScreen), findsOneWidget);
      expect(find.text('Python sidecar failed to start'), findsOneWidget);
      expect(find.text('python exploded'), findsOneWidget);
      expect(
        find.text('C:/bundle/runtime/logs/python-sidecar.log'),
        findsOneWidget,
      );
    });

    test('shutdown requests /shutdown then force-kills after timeout', () async {
      final harness = await _BootstrapHarness.create(tempDir);
      final manifestFile = File(
        _join(harness.bundleDir.path, 'runtime', 'runtime-manifest.json'),
      );
      final process = _BlockingProcess();
      Uri? shutdownUri;
      Process? terminatedProcess;
      final bootstrap = harness.createBootstrap(
        processFactory: () => process,
        shutdownTimeout: const Duration(milliseconds: 1),
        shutdownRequest: (uri) async {
          shutdownUri = uri;
        },
        processTerminator: (target) async {
          terminatedProcess = target;
          target.kill();
        },
        onStart: () async {
          await manifestFile.writeAsString(
            '{"version":1,"generated_at":"2026-04-05T12:00:00Z","go":{"base_url":"http://127.0.0.1:19090","ready":true}}',
          );
        },
      );

      final startResult = await bootstrap.startIfNeeded();
      expect(startResult.isSuccess, isTrue);

      await bootstrap.shutdown();

      expect(shutdownUri, isNotNull);
      expect(shutdownUri.toString(), 'http://127.0.0.1:19090/shutdown');
      expect(terminatedProcess, same(process));
    });

    test('shutdown ignores shutdown request timeout and still exits', () async {
      final harness = await _BootstrapHarness.create(tempDir);
      final manifestFile = File(
        _join(harness.bundleDir.path, 'runtime', 'runtime-manifest.json'),
      );
      final process = _BlockingProcess();
      Process? terminatedProcess;
      final bootstrap = harness.createBootstrap(
        processFactory: () => process,
        shutdownTimeout: const Duration(milliseconds: 1),
        shutdownRequest: (_) async {
          throw TimeoutException('shutdown request timeout');
        },
        processTerminator: (target) async {
          terminatedProcess = target;
          target.kill();
        },
        onStart: () async {
          await manifestFile.writeAsString(
            '{"version":1,"generated_at":"2026-04-05T12:00:00Z","go":{"base_url":"http://127.0.0.1:19090","ready":true}}',
          );
        },
      );

      final startResult = await bootstrap.startIfNeeded();
      expect(startResult.isSuccess, isTrue);

      await bootstrap.shutdown();

      expect(terminatedProcess, same(process));
    });

    testWidgets('MyApp requests packaged bootstrap shutdown on dispose', (
      tester,
    ) async {
      final bootstrap = _ObservedBootstrap();

      await tester.pumpWidget(
        MyApp(
          packagedBootstrap: bootstrap,
          childOverride: const SizedBox.shrink(),
        ),
      );
      await tester.pumpWidget(const SizedBox.shrink());

      expect(bootstrap.shutdownCalls, 1);
    });

    testWidgets(
      'PackagedDesktopLaunchApp shows startup progress while booting',
      (tester) async {
        final bootstrap = _PendingStartupBootstrap();

        await tester.pumpWidget(
          PackagedDesktopLaunchApp(
            packagedBootstrap: bootstrap,
            isDesktopTarget: true,
            isDevelopmentMode: false,
          ),
        );

        expect(find.byType(StartupProgressScreen), findsOneWidget);
        expect(find.text('正在启动 ACGWarehouse'), findsOneWidget);
      },
    );

    testWidgets(
      'PackagedDesktopLaunchApp renders app content after startup succeeds',
      (tester) async {
        final bootstrap = _SuccessfulStartupBootstrap();

        await tester.pumpWidget(
          PackagedDesktopLaunchApp(
            packagedBootstrap: bootstrap,
            isDesktopTarget: false,
            isDevelopmentMode: false,
            childOverride: const SizedBox(key: Key('ready-child')),
          ),
        );
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 50));

        expect(find.byKey(const Key('ready-child')), findsOneWidget);
      },
    );
  });
}

class _BootstrapHarness {
  _BootstrapHarness({
    required this.bundleDir,
    required this.runtimeDir,
    required this.processStarts,
  });

  final Directory bundleDir;
  final Directory runtimeDir;
  final List<_RecordedProcessStart> processStarts;

  static Future<_BootstrapHarness> create(Directory tempDir) async {
    final bundleDir = Directory(_join(tempDir.path, 'bundle'));
    final runtimeDir = Directory(_join(bundleDir.path, 'runtime'));
    await Directory(_join(runtimeDir.path, 'bin')).create(recursive: true);
    await Directory(
      _join(runtimeDir.path, 'python-sidecar'),
    ).create(recursive: true);
    await Directory(_join(runtimeDir.path, 'logs')).create(recursive: true);
    await Directory(
      _join(runtimeDir.path, 'diagnostics'),
    ).create(recursive: true);
    await File(
      _join(bundleDir.path, 'ACGWarehouse.exe'),
    ).writeAsString('launcher');
    await File(
      _join(runtimeDir.path, 'bin', 'acgwarehouse-server.exe'),
    ).writeAsString('go');
    await File(
      _join(runtimeDir.path, 'python-sidecar', 'acgwarehouse-sidecar.exe'),
    ).writeAsString('python');
    return _BootstrapHarness(
      bundleDir: bundleDir,
      runtimeDir: runtimeDir,
      processStarts: <_RecordedProcessStart>[],
    );
  }

  PackagedDesktopBootstrap createBootstrap({
    Future<void> Function()? onStart,
    Process Function()? processFactory,
    Duration? shutdownTimeout,
    PackagedShutdownRequest? shutdownRequest,
    PackagedProcessTerminator? processTerminator,
  }) {
    return PackagedDesktopBootstrap(
      executablePath: _join(bundleDir.path, 'ACGWarehouse.exe'),
      isPackagedWindowsDesktop: true,
      startupTimeout: const Duration(milliseconds: 25),
      shutdownTimeout: shutdownTimeout,
      pollInterval: const Duration(milliseconds: 1),
      shutdownRequest: shutdownRequest,
      processTerminator: processTerminator,
      processStarter:
          ({
            required executable,
            required arguments,
            required environment,
            required workingDirectory,
          }) async {
            processStarts.add(
              _RecordedProcessStart(
                executable: executable,
                arguments: arguments,
                environment: environment,
                workingDirectory: workingDirectory,
              ),
            );
            if (onStart != null) {
              await onStart();
            }
            return processFactory?.call() ?? _FakeProcess();
          },
      portAllocator: () async => 41001 + processStarts.length,
      delay: (_) async {},
    );
  }
}

class _RecordedProcessStart {
  _RecordedProcessStart({
    required this.executable,
    required this.arguments,
    required this.environment,
    required this.workingDirectory,
  });

  final String executable;
  final List<String> arguments;
  final Map<String, String> environment;
  final String workingDirectory;
}

class _FakeProcess implements Process {
  @override
  bool kill([ProcessSignal signal = ProcessSignal.sigterm]) => true;

  @override
  Future<int> get exitCode async => 0;

  @override
  int get pid => 4242;

  @override
  IOSink get stdin => throw UnimplementedError();

  @override
  Stream<List<int>> get stderr => const Stream.empty();

  @override
  Stream<List<int>> get stdout => const Stream.empty();
}

class _BlockingProcess implements Process {
  final Completer<int> _exitCode = Completer<int>();
  int killCount = 0;

  @override
  bool kill([ProcessSignal signal = ProcessSignal.sigterm]) {
    killCount += 1;
    if (!_exitCode.isCompleted) {
      _exitCode.complete(0);
    }
    return true;
  }

  @override
  Future<int> get exitCode => _exitCode.future;

  @override
  int get pid => 5252;

  @override
  IOSink get stdin => throw UnimplementedError();

  @override
  Stream<List<int>> get stderr => const Stream.empty();

  @override
  Stream<List<int>> get stdout => const Stream.empty();
}

class _ObservedBootstrap extends PackagedDesktopBootstrap {
  _ObservedBootstrap()
    : super(isPackagedWindowsDesktop: false, executablePath: 'ignored.exe');

  int shutdownCalls = 0;

  @override
  Future<void> shutdown() async {
    shutdownCalls += 1;
  }
}

class _PendingStartupBootstrap extends PackagedDesktopBootstrap {
  _PendingStartupBootstrap()
    : _completer = Completer<PackagedDesktopBootstrapResult>(),
      super(isPackagedWindowsDesktop: false, executablePath: 'ignored.exe');

  final Completer<PackagedDesktopBootstrapResult> _completer;

  @override
  Future<PackagedDesktopBootstrapResult> startIfNeeded() => _completer.future;
}

class _SuccessfulStartupBootstrap extends PackagedDesktopBootstrap {
  _SuccessfulStartupBootstrap()
    : super(isPackagedWindowsDesktop: false, executablePath: 'ignored.exe');

  @override
  Future<PackagedDesktopBootstrapResult> startIfNeeded() async {
    return const PackagedDesktopBootstrapResult.success(
      isPackagedLaunch: false,
    );
  }
}

String _join(String first, [String? second, String? third, String? fourth]) {
  final segments = [first, second, third, fourth]
      .whereType<String>()
      .where((segment) => segment.isNotEmpty)
      .toList(growable: false);
  return segments.join(Platform.pathSeparator);
}
