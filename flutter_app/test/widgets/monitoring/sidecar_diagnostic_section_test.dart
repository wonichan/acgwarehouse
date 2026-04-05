import 'dart:async';

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/monitoring_models.dart';
import 'package:gallery/providers/monitoring_provider.dart';
import 'package:gallery/services/monitoring_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

import 'package:gallery/widgets/monitoring/sidecar_diagnostic_section.dart';

class _SidecarMonitoringProvider extends MonitoringProvider {
  _SidecarMonitoringProvider({
    required MonitoringOverview overview,
    RestartImpact? restartImpact,
  }) : _overview = overview,
       _restartImpact = restartImpact,
       super(
         service: MonitoringService(
           client: MockClient((_) async => http.Response('{}', 200)),
         ),
         wsUriFactory: () =>
             Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
       );

  MonitoringOverview _overview;
  RestartImpact? _restartImpact;
  bool _isRestarting = false;
  int restartCalls = 0;
  Completer<void>? restartCompleter;

  @override
  MonitoringOverview get overview => _overview;

  @override
  RestartImpact? get restartImpact => _restartImpact;

  @override
  bool get isRestarting => _isRestarting;

  @override
  Future<void> restartSidecar() async {
    restartCalls += 1;
    _isRestarting = true;
    notifyListeners();
    if (restartCompleter != null) {
      await restartCompleter!.future;
    }
    _restartImpact ??= const RestartImpact(interruptedTaskCount: 4);
    _isRestarting = false;
    notifyListeners();
  }
}

MonitoringOverview _overviewForState({
  required String state,
  String? lastErrorSummary,
  int pendingTasks = 6,
}) {
  return MonitoringOverview.fromJson({
    'health': {'status': 'ok', 'message': 'healthy'},
    'queue': {
      'is_running': true,
      'is_paused': false,
      'queue_size': 7,
      'worker_count': 3,
    },
    'sidecar': {
      'state': state,
      'last_probe_at': '2026-04-05T10:45:00Z',
      'last_probe_result': state == 'ready' ? 'ok' : 'warn',
      'last_error_summary': lastErrorSummary ?? '',
    },
    'batches': {'running': 1},
    'tasks': {'pending': pendingTasks, 'running': 2},
  });
}

void main() {
  Future<void> pumpSection(
    WidgetTester tester,
    _SidecarMonitoringProvider provider,
  ) async {
    await tester.pumpWidget(
      ChangeNotifierProvider<MonitoringProvider>.value(
        value: provider,
        child: const fluent.FluentApp(
          home: fluent.ScaffoldPage(content: SidecarDiagnosticSection()),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('renders ready, degraded, and stopped status states', (
    tester,
  ) async {
    final readyProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'ready'),
    );
    await pumpSection(tester, readyProvider);
    expect(find.text('就绪'), findsWidgets);
    expect(find.byKey(const Key('sidecar-status-band-ready')), findsOneWidget);

    final degradedProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(
        state: 'degraded',
        lastErrorSummary: 'Probe slow',
      ),
    );
    await pumpSection(tester, degradedProvider);
    expect(find.text('降级'), findsWidgets);
    expect(
      find.byKey(const Key('sidecar-status-band-degraded')),
      findsOneWidget,
    );

    final stoppedProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(
        state: 'stopped',
        lastErrorSummary: 'Sidecar unavailable',
      ),
    );
    await pumpSection(tester, stoppedProvider);
    expect(find.text('已停止'), findsWidgets);
    expect(
      find.byKey(const Key('sidecar-status-band-stopped')),
      findsOneWidget,
    );
  });

  testWidgets('restart button only enables for degraded or stopped states', (
    tester,
  ) async {
    final readyProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'ready'),
    );
    await pumpSection(tester, readyProvider);
    var button = tester.widget<fluent.Button>(
      find
          .ancestor(
            of: find.text('重启 Sidecar'),
            matching: find.byType(fluent.Button),
          )
          .first,
    );
    expect(button.onPressed, isNull);

    final degradedProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'degraded'),
    );
    await pumpSection(tester, degradedProvider);
    button = tester.widget<fluent.Button>(
      find
          .ancestor(
            of: find.text('重启 Sidecar'),
            matching: find.byType(fluent.Button),
          )
          .first,
    );
    expect(button.onPressed, isNotNull);
  });

  testWidgets('restart opens confirmation dialog with impact count', (
    tester,
  ) async {
    final provider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'degraded'),
    );

    await pumpSection(tester, provider);
    await tester.tap(find.text('重启 Sidecar'));
    await tester.pumpAndSettle();

    expect(find.text('确认重启 Sidecar'), findsOneWidget);
    expect(find.text('重启将中断正在进行的 2 个计算任务。确定要继续吗？'), findsOneWidget);
  });

  testWidgets('confirm restart calls provider and shows loading state', (
    tester,
  ) async {
    final provider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'stopped'),
    );
    provider.restartCompleter = Completer<void>();

    await pumpSection(tester, provider);
    await tester.tap(find.text('重启 Sidecar'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('确认重启'));
    await tester.pump();

    expect(provider.restartCalls, 1);
    expect(find.byType(fluent.ProgressRing), findsOneWidget);

    provider.restartCompleter!.complete();
    await tester.pumpAndSettle();
  });

  testWidgets('shows queue depth, active workers, and pending tasks metrics', (
    tester,
  ) async {
    final provider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'ready', pendingTasks: 9),
    );

    await pumpSection(tester, provider);

    expect(find.text('队列深度'), findsOneWidget);
    expect(find.text('7'), findsOneWidget);
    expect(find.text('活跃 Worker'), findsOneWidget);
    expect(find.text('3'), findsOneWidget);
    expect(find.text('待处理任务'), findsOneWidget);
    expect(find.text('9'), findsOneWidget);
  });

  testWidgets('shows last error summary or empty state', (tester) async {
    final erroredProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(
        state: 'degraded',
        lastErrorSummary: 'network timeout at 10:45',
      ),
    );
    await pumpSection(tester, erroredProvider);
    expect(find.textContaining('network timeout'), findsOneWidget);

    final emptyProvider = _SidecarMonitoringProvider(
      overview: _overviewForState(state: 'ready', lastErrorSummary: ''),
    );
    await pumpSection(tester, emptyProvider);
    expect(find.text('近期无错误记录'), findsOneWidget);
  });
}
