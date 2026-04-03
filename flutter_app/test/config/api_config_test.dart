import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/config/api_config.dart';

void main() {
  group('ApiConfig runtime override', () {
    setUp(() {
      ApiConfig.resetToDefault();
    });

    test('normalizes trailing slash when updating host URL', () {
      ApiConfig.updateBaseUrl('http://127.0.0.1:54321/');

      expect(ApiConfig.hostUrl, 'http://127.0.0.1:54321');
      expect(ApiConfig.baseUrl, 'http://127.0.0.1:54321/api/v1');
    });

    test('applies dev fallback explicitly', () {
      ApiConfig.updateBaseUrl('http://127.0.0.1:54321');
      ApiConfig.applyDevelopmentFallback();

      expect(ApiConfig.hostUrl, ApiConfig.developmentFallbackHostUrl);
      expect(ApiConfig.isDefault, isTrue);
    });
  });
}
