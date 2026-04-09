import 'package:provider/provider.dart';
import '../config/api_config.dart';
import '../models/log_models.dart';
import '../providers/config_provider.dart';
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
  String? manifestAdminAuth,
}) {
  return [
    // ConfigProvider must be first - all services depend on it for baseURL
    ChangeNotifierProvider(
      create: (_) => ConfigProvider(
        initialBaseUrl: manifestBaseUrl,
        initialAdminAuth: manifestAdminAuth,
      ),
    ),
    Provider(
      create: (context) =>
          ApiService(baseUrl: context.read<ConfigProvider>().baseUrl),
    ),
    Provider(
      create: (context) =>
          TagService(baseUrl: context.read<ConfigProvider>().baseUrl),
    ),
    Provider(
      create: (context) =>
          SearchService(baseUrl: context.read<ConfigProvider>().baseUrl),
    ),
    Provider(
      create: (context) => MonitoringService(
        baseUrl: context.read<ConfigProvider>().baseUrl,
        basicAuthHeader: context.read<ConfigProvider>().adminBasicAuthHeader,
      ),
    ),
    ChangeNotifierProvider(
      create: (context) =>
          ImageListProvider(context.read<ApiService>())..loadImages(),
    ),
    ChangeNotifierProvider(
      create: (context) => TagProvider(context.read<TagService>()),
    ),
    ChangeNotifierProvider(
      create: (context) =>
          SearchProvider(service: context.read<SearchService>()),
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
