import 'dart:convert';
import 'dart:ui';

import 'package:desktop_multi_window/desktop_multi_window.dart';

import '../models/viewer_session.dart';
import '../models/viewer_window_context.dart';
import '../utils/window_manager.dart';

abstract class ViewerWindowAdapter {
  Future<void> launch(ViewerWindowLaunchRequest request);

  Future<void> restoreWindowState(String windowId);

  Future<void> saveWindowState(String windowId);
}

class ViewerWindowLaunchPayload {
  final String logicalWindowId;
  final String title;
  final ViewerWindowContext context;

  const ViewerWindowLaunchPayload({
    required this.logicalWindowId,
    required this.title,
    required this.context,
  });

  factory ViewerWindowLaunchPayload.fromJson(Map<String, dynamic> json) {
    final context = ViewerWindowContext.tryFromJson(
      json['context'] as Map<String, dynamic>? ?? const {},
    );
    if (context == null) {
      throw const FormatException('Invalid viewer window context payload');
    }

    return ViewerWindowLaunchPayload(
      logicalWindowId: json['logical_window_id'] as String? ?? '',
      title: json['title'] as String? ?? '',
      context: context,
    );
  }

  static ViewerWindowLaunchPayload? fromEncodedJson(String raw) {
    Object? decoded;
    try {
      decoded = jsonDecode(raw);
    } on FormatException {
      return null;
    }

    if (decoded is! Map<String, dynamic>) {
      return null;
    }
    if (decoded['kind'] != 'viewer-window') {
      return null;
    }
    try {
      return ViewerWindowLaunchPayload.fromJson(decoded);
    } on FormatException {
      return null;
    }
  }

  Map<String, dynamic> toJson() {
    return {
      'kind': 'viewer-window',
      'logical_window_id': logicalWindowId,
      'title': title,
      'context': context.toJson(),
    };
  }
}

class ViewerWindowLaunchRequest {
  final String windowId;
  final String title;
  final ViewerWindowContext context;
  final AppWindowPolicy policy;

  const ViewerWindowLaunchRequest({
    required this.windowId,
    required this.title,
    required this.context,
    required this.policy,
  });

  Map<String, dynamic> get arguments {
    return ViewerWindowLaunchPayload(
      logicalWindowId: windowId,
      title: title,
      context: context,
    ).toJson();
  }
}

class ViewerWindowBootstrapData {
  final int windowId;
  final ViewerWindowContext context;
  final ViewerSession session;
  final AppWindowPolicy policy;

  const ViewerWindowBootstrapData({
    required this.windowId,
    this.context = const ViewerWindowContext.gallery(
      selectedIndex: 0,
      selectedImageId: 0,
      snapshot: ViewerWindowGallerySnapshot(
        sortBy: 'created_at',
        sortDir: 'desc',
        tagIds: [],
        hasTags: null,
      ),
    ),
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

    final title = payload.title.isEmpty ? 'ACGWarehouse Viewer' : payload.title;

    return ViewerWindowBootstrapData(
      windowId: windowId,
      context: payload.context,
      session: buildLegacyViewerSession(context: payload.context, title: title),
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

  Future<void> openWindow({
    required String selectedFilename,
    required ViewerWindowContext context,
  }) async {
    _windowCounter += 1;
    final title = buildWindowTitle(selectedFilename);

    await adapter.launch(
      ViewerWindowLaunchRequest(
        windowId: 'viewer-window-$_windowCounter',
        title: title,
        context: context,
        policy: viewerWindowOptions(title),
      ),
    );
  }

  Future<void> openSession(ViewerSession session) {
    throw UnsupportedError(
      'openSession discards launch-context state; use openWindow instead.',
    );
  }
}
