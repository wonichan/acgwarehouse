import 'dart:async';
import 'dart:convert';
import 'dart:io';

typedef PackagedProcessStarter =
    Future<Process> Function({
      required String executable,
      required List<String> arguments,
      required Map<String, String> environment,
      required String workingDirectory,
    });
typedef PackagedPortAllocator = Future<int> Function();
typedef PackagedDelay = Future<void> Function(Duration duration);
typedef PackagedShutdownRequest = Future<void> Function(Uri shutdownUri);
typedef PackagedProcessTerminator = Future<void> Function(Process process);

const _packagedServerHost = '127.0.0.1';

enum StartupFailureType { go, python, startupChain }

class StartupFailure {
  const StartupFailure({
    required this.type,
    required this.title,
    required this.message,
    required this.logPaths,
  });

  final StartupFailureType type;
  final String title;
  final String message;
  final List<String> logPaths;
}

class PackagedDesktopBootstrapLayout {
  const PackagedDesktopBootstrapLayout({
    required this.rootDir,
    required this.runtimeDir,
    required this.logsDir,
    required this.diagnosticsDir,
    required this.manifestPath,
    required this.startupDiagnosticPath,
    required this.serverExecutablePath,
    required this.sidecarExecutablePath,
  });

  final String rootDir;
  final String runtimeDir;
  final String logsDir;
  final String diagnosticsDir;
  final String manifestPath;
  final String startupDiagnosticPath;
  final String serverExecutablePath;
  final String sidecarExecutablePath;
}

class PackagedDesktopBootstrapResult {
  const PackagedDesktopBootstrapResult._({
    required this.isPackagedLaunch,
    required this.isSuccess,
    this.failure,
  });

  const PackagedDesktopBootstrapResult.skipped()
    : this._(isPackagedLaunch: false, isSuccess: true);

  const PackagedDesktopBootstrapResult.success({required bool isPackagedLaunch})
    : this._(isPackagedLaunch: isPackagedLaunch, isSuccess: true);

  const PackagedDesktopBootstrapResult.failure({
    required bool isPackagedLaunch,
    required StartupFailure failure,
  }) : this._(
         isPackagedLaunch: isPackagedLaunch,
         isSuccess: false,
         failure: failure,
       );

  final bool isPackagedLaunch;
  final bool isSuccess;
  final StartupFailure? failure;
}

class PackagedDesktopBootstrap {
  PackagedDesktopBootstrap({
    String? executablePath,
    bool? isPackagedWindowsDesktop,
    Duration? startupTimeout,
    Duration? shutdownTimeout,
    Duration? pollInterval,
    PackagedProcessStarter? processStarter,
    PackagedPortAllocator? portAllocator,
    PackagedDelay? delay,
    PackagedShutdownRequest? shutdownRequest,
    PackagedProcessTerminator? processTerminator,
  }) : executablePath = executablePath ?? Platform.resolvedExecutable,
       isPackagedWindowsDesktop =
           isPackagedWindowsDesktop ??
           (Platform.isWindows && bool.fromEnvironment('dart.vm.product')),
       startupTimeout = startupTimeout ?? const Duration(seconds: 20),
       shutdownTimeout = shutdownTimeout ?? const Duration(seconds: 5),
       pollInterval = pollInterval ?? const Duration(milliseconds: 150),
       _processStarter = processStarter ?? _defaultProcessStarter,
       _portAllocator = portAllocator ?? _allocateLoopbackPort,
       _delay = delay ?? Future<void>.delayed,
       _shutdownRequest = shutdownRequest ?? _defaultShutdownRequest,
       _processTerminator = processTerminator ?? _defaultProcessTerminator;

  final String executablePath;
  final bool isPackagedWindowsDesktop;
  final Duration startupTimeout;
  final Duration shutdownTimeout;
  final Duration pollInterval;
  final PackagedProcessStarter _processStarter;
  final PackagedPortAllocator _portAllocator;
  final PackagedDelay _delay;
  final PackagedShutdownRequest _shutdownRequest;
  final PackagedProcessTerminator _processTerminator;

  Process? _childProcess;
  Uri? _baseUri;

  PackagedDesktopBootstrapLayout resolveLayout() {
    final rootDir = File(executablePath).parent.path;
    final runtimeDir = _join(rootDir, 'runtime');
    final diagnosticsDir = _join(runtimeDir, 'diagnostics');
    return PackagedDesktopBootstrapLayout(
      rootDir: rootDir,
      runtimeDir: runtimeDir,
      logsDir: _join(runtimeDir, 'logs'),
      diagnosticsDir: diagnosticsDir,
      manifestPath: _join(runtimeDir, 'runtime-manifest.json'),
      startupDiagnosticPath: _join(diagnosticsDir, 'startup-error.json'),
      serverExecutablePath: _join(runtimeDir, 'bin', 'acgwarehouse-server.exe'),
      sidecarExecutablePath: _join(
        runtimeDir,
        'python-sidecar',
        'acgwarehouse-sidecar.exe',
      ),
    );
  }

  Future<PackagedDesktopBootstrapResult> startIfNeeded() async {
    if (!isPackagedWindowsDesktop) {
      return const PackagedDesktopBootstrapResult.skipped();
    }

    final layout = resolveLayout();
    final logPaths = _defaultLogPaths(layout);

    if (!await File(layout.serverExecutablePath).exists()) {
      return PackagedDesktopBootstrapResult.failure(
        isPackagedLaunch: true,
        failure: StartupFailure(
          type: StartupFailureType.go,
          title: 'Go runtime failed to start',
          message:
              'Missing packaged Go executable at ${layout.serverExecutablePath}. Check bundle integrity and logs: ${logPaths.join(', ')}',
          logPaths: logPaths,
        ),
      );
    }

    if (!await File(layout.sidecarExecutablePath).exists()) {
      return PackagedDesktopBootstrapResult.failure(
        isPackagedLaunch: true,
        failure: StartupFailure(
          type: StartupFailureType.python,
          title: 'Python sidecar failed to start',
          message:
              'Missing packaged Python sidecar at ${layout.sidecarExecutablePath}. Check bundle integrity and logs: ${logPaths.join(', ')}',
          logPaths: logPaths,
        ),
      );
    }

    await Directory(layout.logsDir).create(recursive: true);
    await Directory(layout.diagnosticsDir).create(recursive: true);

    final serverPort = await _portAllocator();
    final sidecarPort = await _portAllocator();
    _childProcess = await _processStarter(
      executable: layout.serverExecutablePath,
      arguments: const <String>[],
      environment: <String, String>{
        'SERVER_HOST': _packagedServerHost,
        'SERVER_PORT': '$serverPort',
        'ACG_RUNTIME_ROOT': layout.runtimeDir,
        'ACG_RUNTIME_MANIFEST_PATH': layout.manifestPath,
        'ACG_DIAGNOSTICS_DIR': layout.diagnosticsDir,
        'ACG_LOGS_DIR': layout.logsDir,
        'ACG_SIDECAR_EXECUTABLE': layout.sidecarExecutablePath,
        'ACG_SIDECAR_PORT': '$sidecarPort',
      },
      workingDirectory: layout.rootDir,
    );

    final outcome = await _waitForStartup(layout, logPaths);
    if (outcome.isSuccess) {
      _baseUri = await _readBaseUri(layout.manifestPath);
    }
    return outcome;
  }

  Future<void> shutdown() async {
    final process = _childProcess;
    _childProcess = null;
    final baseUri = _baseUri;
    _baseUri = null;
    if (process == null) {
      return;
    }

    if (baseUri != null) {
      try {
        await _shutdownRequest(baseUri.resolve('/shutdown'));
      } catch (_) {}
    }

    try {
      await process.exitCode.timeout(shutdownTimeout);
      return;
    } on TimeoutException {
      await _processTerminator(process);
      await process.exitCode.catchError((_) => 1);
    }
  }

  Future<PackagedDesktopBootstrapResult> _waitForStartup(
    PackagedDesktopBootstrapLayout layout,
    List<String> logPaths,
  ) async {
    final manifestFile = File(layout.manifestPath);
    final diagnosticFile = File(layout.startupDiagnosticPath);
    final deadline = DateTime.now().add(startupTimeout);

    while (DateTime.now().isBefore(deadline)) {
      if (await diagnosticFile.exists()) {
        return PackagedDesktopBootstrapResult.failure(
          isPackagedLaunch: true,
          failure: await _readFailure(diagnosticFile, logPaths),
        );
      }
      if (await manifestFile.exists()) {
        return const PackagedDesktopBootstrapResult.success(
          isPackagedLaunch: true,
        );
      }
      await _delay(pollInterval);
    }

    return PackagedDesktopBootstrapResult.failure(
      isPackagedLaunch: true,
      failure: StartupFailure(
        type: StartupFailureType.startupChain,
        title: 'Application startup did not complete',
        message:
            'Timed out waiting for runtime-manifest.json. Check the packaged startup logs: ${logPaths.join(', ')}',
        logPaths: logPaths,
      ),
    );
  }

  Future<StartupFailure> _readFailure(
    File diagnosticFile,
    List<String> fallbackLogPaths,
  ) async {
    try {
      final decoded = jsonDecode(await diagnosticFile.readAsString());
      if (decoded is! Map<String, dynamic>) {
        throw const FormatException(
          'Startup diagnostic must be a JSON object.',
        );
      }

      final component =
          (decoded['component'] as String?)?.trim() ?? 'startup_chain';
      final message = (decoded['message'] as String?)?.trim();
      final rawLogPaths = decoded['log_paths'];
      final logPaths = rawLogPaths is List
          ? rawLogPaths
                .whereType<String>()
                .where((path) => path.trim().isNotEmpty)
                .toList(growable: false)
          : fallbackLogPaths;

      final type = switch (component) {
        'go' => StartupFailureType.go,
        'python' => StartupFailureType.python,
        _ => StartupFailureType.startupChain,
      };
      final title = switch (type) {
        StartupFailureType.go => 'Go runtime failed to start',
        StartupFailureType.python => 'Python sidecar failed to start',
        StartupFailureType.startupChain =>
          'Application startup did not complete',
      };

      final details = message == null || message.isEmpty
          ? 'Check the packaged logs: ${logPaths.join(', ')}'
          : '$message. Check the packaged logs: ${logPaths.join(', ')}';

      return StartupFailure(
        type: type,
        title: title,
        message: details,
        logPaths: logPaths,
      );
    } catch (_) {
      return StartupFailure(
        type: StartupFailureType.startupChain,
        title: 'Application startup did not complete',
        message:
            'Startup diagnostics were unreadable. Check the packaged logs: ${fallbackLogPaths.join(', ')}',
        logPaths: fallbackLogPaths,
      );
    }
  }

  Future<Uri?> _readBaseUri(String manifestPath) async {
    final manifestFile = File(manifestPath);
    if (!await manifestFile.exists()) {
      return null;
    }

    try {
      final decoded = jsonDecode(await manifestFile.readAsString());
      if (decoded is! Map<String, dynamic>) {
        return null;
      }
      final go = decoded['go'];
      if (go is! Map<String, dynamic>) {
        return null;
      }
      final baseUrl = go['base_url'];
      if (baseUrl is! String || baseUrl.trim().isEmpty) {
        return null;
      }
      return Uri.tryParse(baseUrl.trim());
    } catch (_) {
      return null;
    }
  }

  static Future<Process> _defaultProcessStarter({
    required String executable,
    required List<String> arguments,
    required Map<String, String> environment,
    required String workingDirectory,
  }) async {
    final process = await Process.start(
      executable,
      arguments,
      environment: environment,
      workingDirectory: workingDirectory,
      mode: ProcessStartMode.normal,
    );

    final logsDir = environment['ACG_LOGS_DIR'];
    if (logsDir != null && logsDir.trim().isNotEmpty) {
      await Directory(logsDir).create(recursive: true);
      final sink = File(
        _join(logsDir, 'go.log'),
      ).openWrite(mode: FileMode.writeOnlyAppend);
      var openStreams = 2;

      void closeSinkIfDone() {
        openStreams -= 1;
        if (openStreams == 0) {
          unawaited(sink.flush().whenComplete(sink.close));
        }
      }

      process.stdout.listen(sink.add, onDone: closeSinkIfDone);
      process.stderr.listen(sink.add, onDone: closeSinkIfDone);
    }

    return process;
  }

  static Future<int> _allocateLoopbackPort() async {
    final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = socket.port;
    await socket.close();
    return port;
  }

  static Future<void> _defaultShutdownRequest(Uri shutdownUri) async {
    final client = HttpClient();
    try {
      final request = await client.postUrl(shutdownUri);
      final response = await request.close();
      await response.drain<void>();
    } finally {
      client.close(force: true);
    }
  }

  static Future<void> _defaultProcessTerminator(Process process) async {
    if (Platform.isWindows) {
      try {
        await Process.run('taskkill', <String>[
          '/PID',
          '${process.pid}',
          '/T',
          '/F',
        ]);
        return;
      } catch (_) {}
    }

    process.kill();
  }

  static List<String> _defaultLogPaths(PackagedDesktopBootstrapLayout layout) {
    return <String>[
      _join(layout.logsDir, 'go.log'),
      _join(layout.logsDir, 'python-sidecar.log'),
      _join(layout.logsDir, 'flutter-bootstrap.log'),
    ];
  }
}

String _join(String first, [String? second, String? third, String? fourth]) {
  final segments = <String>[first];
  for (final segment in <String?>[second, third, fourth]) {
    if (segment != null && segment.isNotEmpty) {
      segments.add(segment);
    }
  }
  return segments.join(Platform.pathSeparator);
}
