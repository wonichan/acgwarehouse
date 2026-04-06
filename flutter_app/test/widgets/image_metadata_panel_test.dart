import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

void main() {
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

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ChangeNotifierProvider<TagProvider>(
              create: (_) => TagProvider(TagService(client: mockClient)),
              child: const ImageMetadataPanel(
                imageId: 1,
                metadataSection: SizedBox.shrink(),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('生成'));
      await tester.pump();
      await tester.pump(const Duration(seconds: 2));

      expect(find.text('分析中...'), findsOneWidget);
      expect(find.textContaining('触发 AI 标签失败'), findsNothing);
    },
  );
}
