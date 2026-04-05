import 'package:web_socket_channel/web_socket_channel.dart';

import 'monitoring_channel_factory_stub.dart'
    if (dart.library.io) 'monitoring_channel_factory_io.dart'
    as monitoring_channel_factory;

WebSocketChannel createMonitoringChannel(
  Uri uri, {
  Map<String, dynamic>? headers,
}) {
  return monitoring_channel_factory.createMonitoringChannel(
    uri,
    headers: headers,
  );
}
