import '../models/viewer_session.dart';
import '../utils/window_manager.dart';

abstract class ViewerWindowAdapter {
  Future<void> launch(ViewerWindowLaunchRequest request);

  Future<void> restoreWindowState(String windowId);

  Future<void> saveWindowState(String windowId);
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
