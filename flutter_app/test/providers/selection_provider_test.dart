import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/selection_provider.dart';

void main() {
  group('SelectionProvider', () {
    test('clearSelection keeps selection mode active', () {
      final provider = SelectionProvider();
      provider.enterSelectionMode();
      provider.toggleSelection(1);

      provider.clearSelection();

      expect(provider.isSelectionMode, isTrue);
      expect(provider.selectedImageIds, isEmpty);
    });

    test('exitSelectionMode clears selection and disables mode', () {
      final provider = SelectionProvider();
      provider.enterSelectionMode();
      provider.selectAll([1, 2, 3]);

      provider.exitSelectionMode();

      expect(provider.isSelectionMode, isFalse);
      expect(provider.selectedImageIds, isEmpty);
    });

    test('selectAll selects the loaded image ids only', () {
      final provider = SelectionProvider();
      provider.enterSelectionMode();

      provider.selectAll([10, 11, 12]);

      expect(provider.selectedImageIds, {10, 11, 12});
    });
  });
}
