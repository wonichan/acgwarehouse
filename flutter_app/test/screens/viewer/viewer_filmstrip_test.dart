import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/image.dart';
import 'package:gallery/screens/viewer/viewer_filmstrip.dart';

void main() {
  testWidgets('renders only current local window and shows global count', (
    tester,
  ) async {
    var tappedIndex = -1;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ViewerFilmstrip(
            items: [
              _image(id: 10, filename: 'ten.jpg'),
              _image(id: 11, filename: 'eleven.jpg'),
            ],
            selectedIndexInWindow: 1,
            selectedIndex: 10,
            total: 25,
            onIndexChanged: (index) {
              tappedIndex = index;
            },
          ),
        ),
      ),
    );

    expect(find.text('11 of 25'), findsOneWidget);
    expect(find.byType(GestureDetector), findsNWidgets(2));

    await tester.tap(find.byType(GestureDetector).first);
    expect(tappedIndex, 0);
  });
}

ImageModel _image({required int id, required String filename}) {
  return ImageModel(
    id: id,
    path: 'C:/images/$filename',
    filename: filename,
    sourceRoot: 'C:/images',
    fileSize: 2048,
    width: 800,
    height: 600,
    format: 'png',
    phash: id,
    thumbnailSmallUrl: null,
    thumbnailLargeUrl: null,
    createdAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
    updatedAt: DateTime.parse('2026-04-05T00:00:00.000Z'),
  );
}
