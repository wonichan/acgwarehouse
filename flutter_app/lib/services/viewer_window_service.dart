import 'dart:convert';
import 'dart:ui';

import 'package:desktop_multi_window/desktop_multi_window.dart';

import '../models/viewer_session.dart';
import '../utils/window_manager.dart';

abstract class ViewerWindowAdapter {
  Future<void> launch(ViewerWindowLaunchRequest request);

  Future<void> restoreWindowState(String windowId);

  Future<void> saveWindowState(String windowId);
}

class ViewerWindowLaunchPayload {
  final String logicalWindowId;
  final String title;
  final ViewerSession session;

  const ViewerWindowLaunchPayload({
    required this.logicalWindowId,
    required this.title,
    required this.session,
  });

  factory ViewerWindowLaunchPayload.fromJson(Map<String, dynamic> json) {
    return ViewerWindowLaunchPayload(
      logicalWindowId: json['logical_window_id'] as String? ?? '',
      title: json['title'] as String? ?? '',
      session: ViewerSession.fromJson(json['session'] as Map<String, dynamic>),
    );
  }

  static ViewerWindowLaunchPayload? fromEncodedJson(String raw) {
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, dynamic>) {
      return null;
    }
    if (decoded['kind'] != 'viewer-window') {
      return null;
    }
    return ViewerWindowLaunchPayload.fromJson(decoded);
  }

  Map<String, dynamic> toJson() {
    return {
      'kind': 'viewer-window',
      'logical_window_id': logicalWindowId,
      'title': title,
      'session': session.toJson(),
    };
  }
}

class ViewerWindowLaunchRequest {
  final String windowId;
  final String title;
  final ViewerSession session;
  final AppWindowPolicy policy;

  const ViewerWindowLaunchRequest({
    required this.windowId,
    required this.title,
    required this.session,
    required this.policy,
  });

  Map<String, dynamic> get arguments {
    return ViewerWindowLaunchPayload(
      logicalWindowId: windowId,
      title: title,
      session: session,
    ).toJson();
  }
}

class ViewerWindowBootstrapData {
  final int windowId;
  final ViewerSession session;
  final AppWindowPolicy policy;

  const ViewerWindowBootstrapData({
    required this.windowId,
    required this.session,
    required this.policy,
  });

  static ViewerWindowBootstrapData? fromCommandLine(List<String> arguments) {
    if (arguments.length < 3 || arguments.first != 'multi_window') {
      return null;
    }

    final windowId = int.tryParse(arguments[1]);
    if (windowId == null) {
      return null;
    }

    final payload = ViewerWindowLaunchPayload.fromEncodedJson(arguments[2]);
    if (payload == null) {
      return null;
    }

    final session = payload.session;
    final title = payload.title.isEmpty
        ? buildViewerWindowTitle(session.selectedItem.filename)
        : payload.title;

    return ViewerWindowBootstrapData(
      windowId: windowId,
      session: session,
      policy: viewerWindowOptions(title),
    );
  }
}

class DesktopMultiWindowViewerWindowAdapter implements ViewerWindowAdapter {
  @override
  Future<void> launch(ViewerWindowLaunchRequest request) async {
    final controller = await DesktopMultiWindow.createWindow(
      jsonEncode(request.arguments),
    );

    final frame = Rect.fromLTWH(
      0,
      0,
      request.policy.size.width,
      request.policy.size.height,
    );
    await controller.setFrame(frame);
    if (request.policy.center) {
      await controller.center();
    }
    await controller.setTitle(request.title);
    await controller.show();
  }

  @override
  Future<void> restoreWindowState(String windowId) async {}

  @override
  Future<void> saveWindowState(String windowId) async {}
}

class ViewerWindowService {
  final ViewerWindowAdapter adapter;
  int _windowCounter = 0;

  ViewerWindowService({required this.adapter});

  static String buildWindowTitle(String filename) {
    return buildViewerWindowTitle(filename);
  }

  Future<void> openSession(ViewerSession session) async {
    _windowCounter += 1;
    final title = buildWindowTitle(session.selectedItem.filename);

    await adapter.launch(
      ViewerWindowLaunchRequest(
        windowId: 'viewer-window-$_windowCounter',
        title: title,
        session: session,
        policy: viewerWindowOptions(title),
      ),
    );
  }
}
