import 'dart:io';

const int defaultSingleInstancePort = 38473;

class SingleInstanceGuard {
  SingleInstanceGuard._(this._socket);

  final ServerSocket _socket;
  bool _released = false;

  static Future<SingleInstanceGuard?> tryAcquire({
    int port = defaultSingleInstancePort,
    InternetAddress? address,
  }) async {
    try {
      final socket = await ServerSocket.bind(
        address ?? InternetAddress.loopbackIPv4,
        port,
        shared: false,
      );
      return SingleInstanceGuard._(socket);
    } on SocketException {
      return null;
    }
  }

  Future<void> release() async {
    if (_released) {
      return;
    }
    _released = true;
    await _socket.close();
  }
}
