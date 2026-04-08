import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/duplicate_provider.dart';
import 'package:gallery/services/duplicate_service.dart';
import 'package:mocktail/mocktail.dart';

class MockDuplicateService extends Mock implements DuplicateService {}

void main() {
  group('DuplicateProvider', () {
    late MockDuplicateService service;
    late DuplicateProvider provider;

    setUp(() {
      service = MockDuplicateService();
      provider = DuplicateProvider(service: service);
    });

    test(
      'detectDuplicates exposes task progress and terminal completion',
      () async {
        when(
          () => service.detectDuplicates(threshold: any(named: 'threshold')),
        ).thenAnswer(
          (_) async => const DetectionResult(
            taskId: 'dup-1',
            status: 'queued',
            progress: 0,
            processed: 0,
            total: 10,
            message: 'queued',
          ),
        );

        when(() => service.streamDuplicateTaskEvents('dup-1')).thenAnswer(
          (_) => Stream<DuplicateTaskEvent>.fromIterable([
            const DuplicateTaskEvent(
              event: 'progress',
              payload: DuplicateTaskStatus(
                taskId: 'dup-1',
                status: 'hashing',
                progress: 50,
                processed: 5,
                total: 10,
                message: 'hashing',
                error: null,
                groupsFound: 0,
              ),
            ),
            const DuplicateTaskEvent(
              event: 'terminal',
              payload: DuplicateTaskStatus(
                taskId: 'dup-1',
                status: 'completed',
                progress: 100,
                processed: 10,
                total: 10,
                message: 'completed',
                error: null,
                groupsFound: 2,
              ),
            ),
          ]),
        );

        when(() => service.getDuplicateTaskStatus('dup-1')).thenAnswer(
          (_) async => const DuplicateTaskStatus(
            taskId: 'dup-1',
            status: 'completed',
            progress: 100,
            processed: 10,
            total: 10,
            message: 'completed',
            error: null,
            groupsFound: 2,
          ),
        );

        when(
          () => service.getDuplicateGroups(
            limit: any(named: 'limit'),
            offset: any(named: 'offset'),
          ),
        ).thenAnswer(
          (_) async =>
              const DuplicateListResponse(groups: [], total: 0, hasMore: false),
        );

        await provider.detectDuplicates(threshold: 10);
        await Future<void>.delayed(const Duration(milliseconds: 10));

        expect(provider.taskId, 'dup-1');
        expect(provider.taskStatus, 'completed');
        expect(provider.taskProgress, 100);
        expect(provider.taskProcessed, 10);
        expect(provider.taskTotal, 10);
        expect(provider.isDetecting, isFalse);
      },
    );
  });
}
