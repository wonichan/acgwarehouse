import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
  Widget buildHarness(http.Client client, {bool preloadImageTags = false}) {
    return MaterialApp(
      home: Scaffold(
        body: ChangeNotifierProvider<TagProvider>(
          create: (_) {
            final provider = TagProvider(
              TagService(baseUrl: 'http://localhost:8080', client: client),
            );
            if (preloadImageTags) {
              provider.loadImageTags(1);
            }
            return provider;
          },
          child: const ImageMetadataPanel(
            imageId: 1,
            metadataSection: SizedBox.shrink(),
          ),
        ),
      ),
    );
  }

  testWidgets(
    'reuses existing AI task status when trigger response is skipped',
    (tester) async {
      final mockClient = MockClient((request) async {
        if (request.url.path.endsWith('/api/v1/ai-tags/default-prompt')) {
          return http.Response('{"default_prompt":"default prompt"}', 200);
        }
        if (request.url.path.endsWith('/api/v1/images/1/ai-tags')) {
          return http.Response(
            '{"status":"skipped","created_tasks":0,"skipped_tasks":1}',
            202,
          );
        }
        if (request.url.path.endsWith('/api/v1/images/1/ai-tags/status')) {
          return http.Response('{"job_id":77,"status":"running"}', 200);
        }
        if (request.url.path.endsWith('/api/v1/images/1/tags')) {
          return http.Response(
            '{"confirmed":[],"pending":[],"rejected":[]}',
            200,
          );
        }
        return http.Response('{}', 200);
      });

      await tester.pumpWidget(buildHarness(mockClient));
      await tester.pumpAndSettle();

      await tester.tap(find.text('生成'));
      await tester.pump();
      await tester.pump(const Duration(seconds: 2));

      expect(find.text('分析中...'), findsOneWidget);
      expect(find.textContaining('触发 AI 标签失败'), findsNothing);
    },
  );

  testWidgets('keeps the generate action visible while AI work is active', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/ai-tags/default-prompt')) {
        return http.Response('{"default_prompt":"default prompt"}', 200);
      }
      if (request.url.path.endsWith('/api/v1/images/1/ai-tags')) {
        return http.Response('{"status":"queued","job_id":77}', 202);
      }
      if (request.url.path.endsWith('/api/v1/images/1/ai-tags/status')) {
        return http.Response('{"job_id":77,"status":"running"}', 200);
      }
      if (request.url.path.endsWith('/api/v1/images/1/tags')) {
        return http.Response(
          '{"confirmed":[],"pending":[],"rejected":[]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    await tester.pumpWidget(buildHarness(mockClient));
    await tester.pumpAndSettle();

    expect(find.text('自定义提示词'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);

    await tester.tap(find.text('生成'));
    await tester.pump();
    await tester.pump(const Duration(seconds: 2));

    expect(find.text('分析中...'), findsOneWidget);
    expect(find.text('生成'), findsOneWidget);
  });

  testWidgets('renders tag groups as pending then confirmed then rejected', (
    tester,
  ) async {
    final mockClient = MockClient((request) async {
      if (request.url.path.endsWith('/api/v1/ai-tags/default-prompt')) {
        return http.Response('{"default_prompt":"default prompt"}', 200);
      }
      if (request.url.path.endsWith('/api/v1/images/1/tags')) {
        return http.Response(
          '{"confirmed":[{"id":2,"preferred_label":"confirmed-tag","slug":"confirmed-1","review_state":"confirmed","trust_score":0.9,"usage_count":1,"created_at":"2024-01-15T10:30:00Z"}],"pending":[{"id":1,"preferred_label":"pending-tag","slug":"pending-1","review_state":"pending","trust_score":0.8,"usage_count":1,"created_at":"2024-01-15T10:30:00Z"}],"rejected":[{"id":3,"preferred_label":"rejected-tag","slug":"rejected-1","review_state":"rejected","trust_score":0.1,"usage_count":1,"created_at":"2024-01-15T10:30:00Z"}]}',
          200,
        );
      }
      return http.Response('{}', 200);
    });

    await tester.pumpWidget(buildHarness(mockClient, preloadImageTags: true));
    await tester.pumpAndSettle();

    expect(find.text('待确认'), findsOneWidget);
    expect(find.text('已确认'), findsOneWidget);
    expect(find.text('已拒绝'), findsOneWidget);

    final pendingTop = tester.getTopLeft(find.text('待确认')).dy;
    final confirmedTop = tester.getTopLeft(find.text('已确认')).dy;
    final rejectedTop = tester.getTopLeft(find.text('已拒绝')).dy;

    expect(pendingTop, lessThan(confirmedTop));
    expect(confirmedTop, lessThan(rejectedTop));
  });
}
