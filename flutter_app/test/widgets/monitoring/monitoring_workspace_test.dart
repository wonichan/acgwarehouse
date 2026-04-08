import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/monitoring_models.dart';
import 'package:gallery/providers/monitoring_provider.dart';
import 'package:gallery/services/monitoring_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

import 'package:gallery/widgets/monitoring/batch_list_section.dart';
import 'package:gallery/widgets/monitoring/monitoring_workspace.dart';

class _WorkspaceMonitoringProvider extends MonitoringProvider {
  _WorkspaceMonitoringProvider({
    required List<BatchRow> batches,
    List<TaskDetail> tasks = const [],
    bool wsConnected = true,
    bool serviceUnavailable = false,
    int? selectedBatchId,
  }) : _batches = batches,
       _tasks = tasks,
       _wsConnected = wsConnected,
       _serviceUnavailable = serviceUnavailable,
       _selectedBatchId = selectedBatchId,
       super(
         service: MonitoringService(
           client: MockClient((_) async => http.Response('{}', 200)),
         ),
         wsUriFactory: () =>
             Uri.parse('ws://localhost:8080/admin/api/monitoring/ws'),
       );

  final List<BatchRow> _batches;
  final List<TaskDetail> _tasks;
  bool _wsConnected;
  bool _serviceUnavailable;
  int? _selectedBatchId;

  int connectCalls = 0;
  int disconnectCalls = 0;
  int retryLoadCalls = 0;
  final List<int?> selectBatchCalls = [];

  @override
  List<BatchRow> get batches => _batches;

  @override
  List<TaskDetail> get tasks => _tasks;

  @override
  bool get wsConnected => _wsConnected;

  @override
  bool get serviceUnavailable => _serviceUnavailable;

  @override
  int? get selectedBatchId => _selectedBatchId;

  @override
  Future<void> connect() async {
    connectCalls += 1;
  }

  @override
  Future<void> disconnect() async {
    disconnectCalls += 1;
  }

  @override
  Future<void> retryLoad() async {
    retryLoadCalls += 1;
    _serviceUnavailable = false;
    notifyListeners();
  }

  @override
  Future<void> selectBatch(int? id) async {
    selectBatchCalls.add(id);
    _selectedBatchId = id;
    notifyListeners();
  }

  void setWsConnected(bool value) {
    _wsConnected = value;
    notifyListeners();
  }
}

void main() {
  final List<BatchRow> sampleBatches = [
    BatchRow(
      id: 101,
      sourceType: 'import',
      summaryLabel: 'Batch Alpha',
      status: 'running',
      totalImages: 20,
      newImages: 18,
      createdAt: DateTime(2026, 4, 5, 10, 30),
      finishedAt: null,
      statusCounts: const {'completed': 10, 'running': 5, 'pending': 5},
      taskTypeCounts: const {'tagging': 20},
      failureGroups: const [],
    ),
  ];

  final sampleTasks = [
    const TaskDetail(
      id: 1,
      batchId: 101,
      imageId: 99,
      imageFilename: 'cover.png',
      taskType: 'tagging',
      status: 'failed',
      errorSummary: 'worker unavailable',
    ),
  ];

  Future<void> pumpWorkspace(
    WidgetTester tester,
    _WorkspaceMonitoringProvider provider,
    Widget child,
  ) async {
    await tester.pumpWidget(
      ChangeNotifierProvider<MonitoringProvider>.value(
        value: provider,
        child: fluent.FluentApp(home: child),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('workspace connects on init and disconnects on dispose', (
    tester,
  ) async {
    final provider = _WorkspaceMonitoringProvider(batches: sampleBatches);

    await pumpWorkspace(
      tester,
      provider,
      const OperationsMonitoringWorkspace(),
    );
    expect(provider.connectCalls, 1);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    expect(provider.disconnectCalls, 1);
  });

  testWidgets('workspace shows disconnect banner with reconnect action', (
    tester,
  ) async {
    final provider = _WorkspaceMonitoringProvider(
      batches: sampleBatches,
      wsConnected: false,
    );

    await pumpWorkspace(
      tester,
      provider,
      const OperationsMonitoringWorkspace(),
    );

    expect(find.text('实时连接已断开，数据可能不是最新。'), findsOneWidget);
    expect(find.text('重新连接'), findsOneWidget);

    await tester.tap(find.text('重新连接'));
    await tester.pumpAndSettle();
    expect(provider.connectCalls, 2);
  });

  testWidgets('workspace shows service unavailable retry state', (
    tester,
  ) async {
    final provider = _WorkspaceMonitoringProvider(
      batches: sampleBatches,
      serviceUnavailable: true,
    );

    await pumpWorkspace(
      tester,
      provider,
      const OperationsMonitoringWorkspace(),
    );

    expect(find.text('服务暂时不可用。检查后端连接后重试。'), findsOneWidget);
    expect(find.text('重试'), findsOneWidget);

    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();
    expect(provider.retryLoadCalls, 1);
  });

  testWidgets('batch list renders status, progress, and timestamps', (
    tester,
  ) async {
    final provider = _WorkspaceMonitoringProvider(
      batches: sampleBatches,
      tasks: sampleTasks,
      selectedBatchId: 101,
    );

    await pumpWorkspace(tester, provider, const BatchListSection());

    expect(find.text('批次任务监控'), findsOneWidget);
    expect(find.text('进行中'), findsOneWidget);
    expect(find.text('50%'), findsOneWidget);
    expect(find.text('2026-04-05 10:30'), findsOneWidget);
    expect(find.text('cover.png'), findsOneWidget);
  });

  testWidgets('batch row tap delegates selection to provider', (tester) async {
    final provider = _WorkspaceMonitoringProvider(batches: sampleBatches);

    await pumpWorkspace(tester, provider, const BatchListSection());

    await tester.tap(find.text('Batch Alpha'));
    await tester.pumpAndSettle();

    expect(provider.selectBatchCalls, [101]);
  });
}
