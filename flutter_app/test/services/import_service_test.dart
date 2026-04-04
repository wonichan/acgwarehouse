import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/config/api_config.dart';
import 'package:gallery/services/import_service.dart';
import 'package:http/http.dart' as http;
import 'package:mocktail/mocktail.dart';

class MockHttpClient extends Mock implements http.Client {}

class FakeUri extends Fake implements Uri {}

void main() {
  late ImportService importService;
  late MockHttpClient mockClient;

  setUpAll(() {
    registerFallbackValue(FakeUri());
  });

  setUp(() {
    mockClient = MockHttpClient();
    importService = ImportService(client: mockClient);
  });

  test('posts to product-facing image scan endpoint', () async {
    when(
      () => mockClient.post(any(), headers: any(named: 'headers')),
    ).thenAnswer(
      (_) async =>
          http.Response(jsonEncode({'status': 'queued', 'job_id': 12}), 202),
    );

    await importService.triggerImport();

    final captured =
        verify(
              () =>
                  mockClient.post(captureAny(), headers: any(named: 'headers')),
            ).captured.single
            as Uri;
    expect(captured.toString(), ApiConfig.imageScan);
  });

  test('parses queued response with job id', () async {
    when(
      () => mockClient.post(any(), headers: any(named: 'headers')),
    ).thenAnswer(
      (_) async =>
          http.Response(jsonEncode({'status': 'queued', 'job_id': 88}), 202),
    );

    final result = await importService.triggerImport();

    expect(result.status, 'queued');
    expect(result.jobId, 88);
  });

  test('throws ImportTriggerException on failure response', () async {
    when(
      () => mockClient.post(any(), headers: any(named: 'headers')),
    ).thenAnswer(
      (_) async => http.Response(
        jsonEncode({'status': 'failed', 'error': 'queue unavailable'}),
        500,
      ),
    );

    await expectLater(
      importService.triggerImport(),
      throwsA(isA<ImportTriggerException>()),
    );
  });
}
