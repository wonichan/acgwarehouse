import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/screens/gallery_screen.dart';

void main() {
  group('GalleryScreen', () {
    testWidgets('builds without error', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert - widget should build without throwing
      expect(find.byType(GalleryScreen), findsOneWidget);
    });

    testWidgets('has app bar with title', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert
      expect(find.text('ACGWarehouse'), findsOneWidget);
    });

    testWidgets('shows action buttons in app bar', (tester) async {
      // Act
      await tester.pumpWidget(const MaterialApp(
        home: GalleryScreen(),
      ));
      
      // Assert - has filter, sort, and manage tags buttons
      expect(find.byIcon(Icons.filter_list), findsOneWidget);
      expect(find.byIcon(Icons.sort), findsOneWidget);
      expect(find.byIcon(Icons.label_outline), findsOneWidget);
    });
  });
}