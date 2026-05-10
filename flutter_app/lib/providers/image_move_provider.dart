import 'package:flutter/foundation.dart';

import '../models/image_move.dart';
import '../models/tag.dart';

class ImageMoveProvider extends ChangeNotifier {
  final List<String> _sourceDirs = <String>[];

  Tag? _selectedTag;
  String? _targetDir;
  ImageMovePreview? _preview;
  ImageMoveResult? _result;
  List<ImageMoveBatch> _history = const <ImageMoveBatch>[];
  ImageMoveBatch? _activeJob;
  bool _isPreviewLoading = false;
  bool _isExecuting = false;
  bool _isHistoryLoading = false;
  bool _allowTargetInsideSource = false;
  String? _errorMessage;
  String? _successMessage;
  String _tagQuery = '';
  String _conflict = 'skip';
  int _selectedTabIndex = 0;

  List<String> get sourceDirs => List.unmodifiable(_sourceDirs);
  Tag? get selectedTag => _selectedTag;
  String? get targetDir => _targetDir;
  ImageMovePreview? get preview => _preview;
  ImageMoveResult? get result => _result;
  List<ImageMoveBatch> get history => _history;
  ImageMoveBatch? get activeJob => _activeJob;
  bool get isPreviewLoading => _isPreviewLoading;
  bool get isExecuting => _isExecuting;
  bool get isHistoryLoading => _isHistoryLoading;
  bool get allowTargetInsideSource => _allowTargetInsideSource;
  String? get errorMessage => _errorMessage;
  String? get successMessage => _successMessage;
  String get tagQuery => _tagQuery;
  String get conflict => _conflict;
  int get selectedTabIndex => _selectedTabIndex;

  bool get isBusy => _isPreviewLoading || _isExecuting;

  bool get canPreview =>
      !isBusy &&
      _sourceDirs.isNotEmpty &&
      _selectedTag != null &&
      (_targetDir?.trim().isNotEmpty ?? false);

  bool get canExecute => !isBusy && _preview != null && _preview!.movable > 0;

  void addSourceDir(String dir) {
    final normalized = dir.trim();
    if (normalized.isEmpty || _sourceDirs.contains(normalized)) return;
    _sourceDirs.add(normalized);
    _clearResponses();
    notifyListeners();
  }

  void removeSourceDir(String dir) {
    if (!_sourceDirs.remove(dir)) return;
    _clearResponses();
    notifyListeners();
  }

  void clearSourceDirs() {
    if (_sourceDirs.isEmpty) return;
    _sourceDirs.clear();
    _clearResponses();
    notifyListeners();
  }

  void setSelectedTag(Tag tag) {
    _selectedTag = tag;
    _clearResponses();
    notifyListeners();
  }

  void setTargetDir(String dir) {
    final normalized = dir.trim();
    if (normalized.isEmpty) return;
    _targetDir = normalized;
    _clearResponses();
    notifyListeners();
  }

  void setConflict(String conflict) {
    if (_conflict == conflict) return;
    _conflict = conflict;
    _clearResponses();
    notifyListeners();
  }

  void setAllowTargetInsideSource(bool value) {
    if (_allowTargetInsideSource == value) return;
    _allowTargetInsideSource = value;
    _clearResponses();
    notifyListeners();
  }

  void setTagQuery(String value) {
    final normalized = value.trim();
    if (_tagQuery == normalized) return;
    _tagQuery = normalized;
    notifyListeners();
  }

  void setSelectedTabIndex(int value) {
    if (_selectedTabIndex == value) return;
    _selectedTabIndex = value;
    notifyListeners();
  }

  void startPreview() {
    _isPreviewLoading = true;
    _errorMessage = null;
    _successMessage = null;
    _result = null;
    notifyListeners();
  }

  void finishPreview(ImageMovePreview preview) {
    _preview = preview;
    _isPreviewLoading = false;
    if (preview.totalMatched > 1000) {
      _successMessage = '命中数量较大，可创建后台任务执行';
    }
    notifyListeners();
  }

  void failPreview(String message) {
    _isPreviewLoading = false;
    _errorMessage = message;
    notifyListeners();
  }

  void startExecute() {
    _isExecuting = true;
    _errorMessage = null;
    _successMessage = null;
    notifyListeners();
  }

  void finishExecute(ImageMoveResult result) {
    _result = result;
    _isExecuting = false;
    _successMessage = '移动完成';
    notifyListeners();
  }

  void failExecute(String message) {
    _isExecuting = false;
    _errorMessage = message;
    notifyListeners();
  }

  void startHistoryLoad() {
    _isHistoryLoading = true;
    notifyListeners();
  }

  void finishHistoryLoad(List<ImageMoveBatch> history) {
    _history = history;
    _isHistoryLoading = false;
    notifyListeners();
  }

  void failHistoryLoad(String message) {
    _isHistoryLoading = false;
    _errorMessage = message;
    notifyListeners();
  }

  void setActiveJob(ImageMoveBatch? job) {
    _activeJob = job;
    notifyListeners();
  }

  void setJobCreated(ImageMoveBatch job) {
    _activeJob = job;
    _selectedTabIndex = 1;
    _isExecuting = false;
    _successMessage = '后台任务已创建';
    notifyListeners();
  }

  ImageMoveRequest buildRequest() {
    final tag = _selectedTag;
    final target = _targetDir;
    if (tag == null || target == null) {
      throw StateError('image move request is incomplete');
    }
    return ImageMoveRequest(
      sourceDirs: sourceDirs,
      tagId: tag.id,
      targetDir: target,
      conflict: _conflict,
      allowTargetInsideSource: _allowTargetInsideSource,
    );
  }

  void _clearResponses() {
    _preview = null;
    _result = null;
    _errorMessage = null;
    _successMessage = null;
  }
}
