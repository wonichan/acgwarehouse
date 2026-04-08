import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/duplicate_provider.dart';
import 'package:gallery/screens/duplicate_screen.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:provider/provider.dart';

class _NoopDuplicateService extends DuplicateService {}

class _FakeDetectingProvider extends DuplicateProvider {
  _FakeDetectingProvider() : super(service: _NoopDuplicateService());

  @override
  bool get isDetecting => true;

  @override
  String? get taskId => 'dup-1';

  @override
  String get taskStatus => 'hashing';

  @override
  double get taskProgress => 50.0;

  @override
  int get taskProcessed => 5;

  @override
  int get taskTotal => 10;

  @override
  bool get isLoading => false;

  @override
  List<DuplicateGroup> get groups => const [];
}

void main() {
  testWidgets('renders live progress details while detecting', (
    WidgetTester tester,
  ) async {
    final provider = _FakeDetectingProvider();

    await tester.pumpWidget(
      ChangeNotifierProvider<DuplicateProvider>.value(
        value: provider,
        child: const MaterialApp(home: DuplicateScreen()),
      ),
    );

    expect(find.textContaining('任务: dup-1'), findsOneWidget);
    expect(find.textContaining('阶段: hashing'), findsOneWidget);
    expect(find.textContaining('进度: 50.0% (5/10)'), findsOneWidget);
    expect(find.byType(LinearProgressIndicator), findsOneWidget);
  });
}
