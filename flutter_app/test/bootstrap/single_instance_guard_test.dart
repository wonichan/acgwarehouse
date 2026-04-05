import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/bootstrap/single_instance_guard.dart';

void main() {
  test('SingleInstanceGuard rejects second acquisition on same port', () async {
    final probe = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = probe.port;
    await probe.close();

    final first = await SingleInstanceGuard.tryAcquire(port: port);
    final second = await SingleInstanceGuard.tryAcquire(port: port);

    expect(first, isNotNull);
    expect(second, isNull);

    await first?.release();
  });
}
