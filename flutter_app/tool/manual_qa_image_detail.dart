import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/providers/config_provider.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/screens/image_detail_screen.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:gallery/widgets/image_metadata_panel.dart';
import 'package:gallery/widgets/image_metadata_pane_theme.dart';
import 'package:http/http.dart' as http;
import 'package:provider/provider.dart';

void main() {
  runApp(const ManualQaApp());
}

class ManualQaApp extends StatelessWidget {
  const ManualQaApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      home: DefaultTabController(
        length: 2,
        child: Scaffold(
          appBar: AppBar(
            title: const Text('Manual QA'),
            bottom: const TabBar(
              tabs: [
                Tab(text: 'Detail Screen'),
                Tab(text: 'Metadata Panel'),
              ],
            ),
          ),
          body: const TabBarView(
            children: [_DetailScreenQaPage(), _MetadataPanelQaPage()],
          ),
        ),
      ),
    );
  }
}

class _DetailScreenQaPage extends StatelessWidget {
  const _DetailScreenQaPage();

  @override
  Widget build(BuildContext context) {
    final image = ImageModel(
      id: 1,
      path: r'E:\picture\very\long\path\to\demo\image_for_manual_qa.jpg',
      filename: 'image_for_manual_qa.jpg',
      sourceRoot: r'E:\picture',
      fileSize: 1024 * 1024 * 64,
      width: 1920,
      height: 1080,
      format: 'jpeg',
      phash: 123,
      thumbnailSmallUrl: '',
      thumbnailLargeUrl: '',
      createdAt: DateTime.utc(2024, 1, 1),
      updatedAt: DateTime.utc(2024, 1, 1),
    );

    return ChangeNotifierProvider(
      create: (_) => ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
      child: ImageDetailScreen(
        image: image,
        tagService: TagService(
          baseUrl: 'http://localhost:8080',
          client: _FakeTagClient(),
        ),
      ),
    );
  }
}

class _MetadataPanelQaPage extends StatelessWidget {
  const _MetadataPanelQaPage();

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider<TagProvider>(
      create: (_) {
        final provider = TagProvider(
          TagService(
            baseUrl: 'http://localhost:8080',
            client: _FakeTagClient(),
          ),
        );
        unawaited(provider.loadImageTags(1));
        return provider;
      },
      child: Builder(
        builder: (context) {
          final theme = ImageMetadataPaneTheme.of(context);
          return Container(
            color: theme.panelSurface,
            padding: const EdgeInsets.all(24),
            child: Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: 360,
                child: ImageMetadataPanel(
                  imageId: 1,
                  metadataSection: Container(
                    margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
                    decoration: theme.sectionDecoration,
                    child: const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('QA metadata stub'),
                    ),
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

class _FakeTagClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final path = request.url.path;
    if (path.endsWith('/api/v1/ai-tags/default-prompt')) {
      return _jsonResponse('{"default_prompt":"default prompt"}');
    }
    if (path.endsWith('/api/v1/images/1/tags')) {
      return _jsonResponse(
        '{"confirmed":[{"id":2,"preferred_label":"confirmed-tag","review_state":"confirmed"}],"pending":[{"id":1,"preferred_label":"pending-tag","review_state":"pending"}],"rejected":[{"id":3,"preferred_label":"rejected-tag","review_state":"rejected"}]}',
      );
    }
    if (path.endsWith('/api/v1/images/1/ai-tags')) {
      return _jsonResponse('{"status":"queued","job_id":77}', statusCode: 202);
    }
    if (path.endsWith('/api/v1/images/1/ai-tags/status')) {
      return _jsonResponse('{"job_id":77,"status":"running"}');
    }
    return _jsonResponse('{}');
  }

  http.StreamedResponse _jsonResponse(String body, {int statusCode = 200}) {
    final bytes = utf8.encode(body);
    return http.StreamedResponse(
      Stream<List<int>>.value(bytes),
      statusCode,
      headers: {'content-type': 'application/json'},
    );
  }
}
