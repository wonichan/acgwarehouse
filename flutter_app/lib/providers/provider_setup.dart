import 'package:provider/provider.dart';
import '../config/api_config.dart';
import '../models/log_models.dart';
import '../providers/config_provider.dart';
import '../providers/image_move_provider.dart';
import '../providers/image_provider.dart';
import '../providers/log_viewer_provider.dart';
import '../providers/monitoring_provider.dart';
import '../providers/navigation_provider.dart';
import '../providers/search_provider.dart';
import '../providers/selection_provider.dart';
import '../providers/tag_provider.dart';
import '../providers/theme_provider.dart';
import '../services/api_service.dart';
import '../services/log_stream_service.dart';
import '../services/monitoring_service.dart';
import '../services/search_service.dart';
import '../services/tag_service.dart';

/// Creates all application providers for the MultiProvider tree.
///
/// [manifestBaseUrl] and [manifestAdminAuth] provide baseline configuration
/// discovered from the runtime manifest file (packaged desktop only).
List<dynamic> createAppProviders({
  String? manifestBaseUrl,
  String? manifestThumbnailBaseUrl,
  String? manifestAdminAuth,
}) {
  return [
    // ConfigProvider must be first - all services depend on it for baseURL
    ChangeNotifierProvider(
      create: (_) => ConfigProvider(
        initialBaseUrl: manifestBaseUrl,
        initialThumbnailBaseUrl: manifestThumbnailBaseUrl,
        initialAdminAuth: manifestAdminAuth,
      ),
    ),
    ProxyProvider<ConfigProvider, ApiService>(
      update: (_, config, previous) {
        if (previous != null && previous.baseUrl == config.baseUrl) {
          return previous;
        }
        final service = ApiService(baseUrl: config.baseUrl);
        previous?.dispose();
        return service;
      },
      dispose: (_, service) => service.dispose(),
    ),
    ProxyProvider<ConfigProvider, TagService>(
      update: (_, config, previous) {
        if (previous != null && previous.baseUrl == config.baseUrl) {
          return previous;
        }
        final service = TagService(baseUrl: config.baseUrl);
        previous?.dispose();
        return service;
      },
      dispose: (_, service) => service.dispose(),
    ),
    ProxyProvider<ConfigProvider, SearchService>(
      update: (_, config, previous) {
        if (previous != null && previous.baseUrl == config.baseUrl) {
          return previous;
        }
        final service = SearchService(baseUrl: config.baseUrl);
        previous?.dispose();
        return service;
      },
      dispose: (_, service) => service.dispose(),
    ),
    ProxyProvider<ConfigProvider, MonitoringService>(
      update: (_, config, previous) {
        if (previous != null &&
            previous.baseUrl == config.baseUrl &&
            previous.basicAuthHeader == config.adminBasicAuthHeader) {
          return previous;
        }
        final service = MonitoringService(
          baseUrl: config.baseUrl,
          basicAuthHeader: config.adminBasicAuthHeader,
        );
        previous?.dispose();
        return service;
      },
      dispose: (_, service) => service.dispose(),
    ),
    ChangeNotifierProxyProvider<ApiService, ImageListProvider>(
      create: (context) =>
          ImageListProvider(context.read<ApiService>())..loadImages(),
      update: (_, service, provider) {
        if (provider == null) {
          return ImageListProvider(service)..loadImages();
        }
        provider.updateApiService(service);
        return provider;
      },
    ),
    ChangeNotifierProxyProvider<TagService, TagProvider>(
      create: (context) => TagProvider(context.read<TagService>()),
      update: (_, service, provider) {
        if (provider == null) {
          return TagProvider(service);
        }
        provider.updateTagService(service);
        return provider;
      },
    ),
    ChangeNotifierProvider(create: (_) => ImageMoveProvider()),
    ChangeNotifierProxyProvider<SearchService, SearchProvider>(
      create: (context) =>
          SearchProvider(service: context.read<SearchService>()),
      update: (_, service, provider) {
        if (provider == null) {
          return SearchProvider(service: service);
        }
        provider.updateService(service);
        return provider;
      },
    ),
    ChangeNotifierProvider(
      create: (context) => MonitoringProvider(
        service: context.read<MonitoringService>(),
        wsUriFactory: () => Uri.parse(
          ApiConfig.monitoringWs(context.read<ConfigProvider>().baseUrl),
        ),
      ),
    ),
    ChangeNotifierProvider(create: (_) => SelectionProvider()),
    Provider(
      create: (context) => LogStreamService(
        baseUrl: context.read<ConfigProvider>().baseUrl,
        basicAuthHeader: context.read<ConfigProvider>().adminBasicAuthHeader,
      ),
    ),
    ChangeNotifierProvider(
      create: (context) => LogViewerProvider(
        service: context.read<LogStreamService>(),
        wsUriFactory: ({required LogSource source, int tail = 200}) =>
            Uri.parse(
              ApiConfig.logStreamWs(
                context.read<ConfigProvider>().baseUrl,
                source: source.name,
                tail: tail,
              ),
            ),
      ),
    ),
    ChangeNotifierProvider(create: (_) => NavigationProvider()),
    ChangeNotifierProvider(create: (_) => ThemeProvider()),
  ];
}
