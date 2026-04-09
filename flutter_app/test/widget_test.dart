import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:gallery/main.dart';

void main() {
  testWidgets('App smoke test - renders child override', (
    WidgetTester tester,
  ) async {
    // Build our app and trigger a frame.
    await tester.pumpWidget(
      const MyApp(childOverride: Text('Hello ACGWarehouse')),
    );

    // Verify that the child override is rendered
    expect(find.text('Hello ACGWarehouse'), findsOneWidget);
  });
}
