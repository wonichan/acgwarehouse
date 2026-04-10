import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/services/batch_service.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  group('BatchService', () {
    test(
      'batchDeleteImages posts selected image ids to /api/v1/batch/images/delete',
      () async {
        final mockClient = MockClient((request) async {
          if (request.url.path == '/api/v1/batch/images/delete') {
            final body = jsonDecode(request.body);
            expect(body['image_ids'], equals([1, 2, 3]));
            return http.Response(jsonEncode({'images_deleted': 3}), 200);
          }
          return http.Response('Not Found', 404);
        });

        final service = BatchService(
          client: mockClient,
          baseUrl: 'http://test.com',
        );
        final deletedCount = await service.batchDeleteImages([1, 2, 3]);

        expect(deletedCount, 3);
      },
    );
  });
}
