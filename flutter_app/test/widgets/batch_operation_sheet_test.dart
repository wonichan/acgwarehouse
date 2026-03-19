import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/providers/selection_provider.dart';
import 'package:gallery/widgets/batch_operation_sheet.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('shows AI generate tags button and triggers callback', (tester) async {
    var tapped = false;

    await tester.pumpWidget(
      ChangeNotifierProvider(
        create: (_) => SelectionProvider(),
        child: MaterialApp(
          home: Scaffold(
            body: BatchOperationSheet(
              onGenerateAITags: () {
                tapped = true;
              },
            ),
          ),
        ),
      ),
    );

    expect(find.text('AI生成标签'), findsOneWidget);
    expect(find.byIcon(Icons.auto_awesome), findsOneWidget);

    await tester.tap(find.text('AI生成标签'));
    await tester.pumpAndSettle();

    expect(tapped, isTrue);
  });
}
