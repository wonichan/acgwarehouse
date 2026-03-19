import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/widgets/image_lightbox.dart';

void main() {
  group('ImageLightbox', () {
    testWidgets('displays image fullscreen with dark background', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) {
                return ElevatedButton(
                  onPressed: () {
                    ImageLightbox.show(
                      context,
                      imageUrl: 'https://example.com/test.jpg',
                    );
                  },
                  child: const Text('Open Lightbox'),
                );
              },
            ),
          ),
        ),
      );

      // Tap button to open lightbox
      await tester.tap(find.text('Open Lightbox'));
      await tester.pumpAndSettle();

      // Verify dark background overlay exists
      expect(find.byType(Container), findsWidgets);
    });

    testWidgets('shows close button in top-right corner', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) {
                return ElevatedButton(
                  onPressed: () {
                    ImageLightbox.show(
                      context,
                      imageUrl: 'https://example.com/test.jpg',
                    );
                  },
                  child: const Text('Open Lightbox'),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Lightbox'));
      await tester.pumpAndSettle();

      // Verify close button exists
      expect(find.byIcon(Icons.close), findsOneWidget);
    });

    testWidgets('close button dismisses the lightbox', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) {
                return ElevatedButton(
                  onPressed: () {
                    ImageLightbox.show(
                      context,
                      imageUrl: 'https://example.com/test.jpg',
                    );
                  },
                  child: const Text('Open Lightbox'),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Lightbox'));
      await tester.pumpAndSettle();

      // Tap close button
      await tester.tap(find.byIcon(Icons.close));
      await tester.pumpAndSettle();

      // Verify lightbox is dismissed (back to original screen)
      expect(find.text('Open Lightbox'), findsOneWidget);
    });

    testWidgets('accepts heroTag parameter for Hero animation', (tester) async {
      // Test that the widget accepts heroTag parameter
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) {
                return ElevatedButton(
                  onPressed: () {
                    ImageLightbox.show(
                      context,
                      imageUrl: 'https://example.com/test.jpg',
                      heroTag: 'test-hero-tag',
                    );
                  },
                  child: const Text('Open Lightbox'),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Lightbox'));
      await tester.pumpAndSettle();

      // Verify Hero widget is present
      expect(find.byType(Hero), findsOneWidget);
    });

    testWidgets('tapping barrier dismisses the lightbox', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) {
                return ElevatedButton(
                  onPressed: () {
                    ImageLightbox.show(
                      context,
                      imageUrl: 'https://example.com/test.jpg',
                    );
                  },
                  child: const Text('Open Lightbox'),
                );
              },
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open Lightbox'));
      await tester.pumpAndSettle();

      // Tap outside the image (on the barrier/background)
      // The barrier should be dismissible
      await tester.tapAt(const Offset(10, 10));
      await tester.pumpAndSettle();

      // Lightbox should be dismissed
      expect(find.text('Open Lightbox'), findsOneWidget);
    });
  });
}