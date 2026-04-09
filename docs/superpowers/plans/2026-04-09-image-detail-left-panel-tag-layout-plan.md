# Image Detail Left Panel and Tag Card Layout Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the image detail metadata pane so the basic info card, AI card, and tag card follow the approved layered-sidebar design while preserving existing tag and AI behaviors.

**Architecture:** Keep the current `ImageMetadataPanel` composition and `image_detail_screen.dart` split layout, then implement the redesign as focused UI refactors inside the existing metadata section builders. Use widget tests first to lock in the new hierarchy and preserve existing AI/tag interactions, then update shared theme tokens and chip styling only where needed for the new emphasis model.

**Tech Stack:** Flutter 3.x, Material 3, Provider, flutter_test

---

## Working Directory

Unless a step explicitly says otherwise, run Flutter commands from:

`flutter_app/`

## Spec Reference

- Design spec: `docs/superpowers/specs/2026-04-09-image-detail-left-panel-tag-layout-design.md`

## File Map

### Modify
- `flutter_app/lib/screens/image_detail_screen.dart`
  - Rework `_buildMetadataSection` and `_buildMetadataRow` to support the new overview-card rhythm, reordered field presentation, and path-specific rendering.
- `flutter_app/lib/widgets/image_metadata_panel.dart`
  - Rework `_buildAITagSection` and `_buildTagsSection` to match the new header/body hierarchy and state-group ordering.
- `flutter_app/lib/widgets/image_metadata_pane_theme.dart`
  - Add any missing pane-level spacing/color/border tokens needed by the redesigned sections.
- `flutter_app/lib/widgets/tag_chip.dart`
  - Adjust chip emphasis only if the redesigned tag group hierarchy cannot be achieved with the existing styles.
- `flutter_app/test/screens/image_detail_screen_test.dart`
  - Add/adjust widget assertions for metadata-pane structure, desktop layout, and path presentation behavior.
- `flutter_app/test/widgets/image_metadata_panel_test.dart`
  - Add/adjust tests for AI card hierarchy, custom-prompt visibility, and grouped tag order.
- `flutter_app/test/widgets/tag_chip_test.dart`
  - Add/adjust tests only if chip typography/color/deemphasis behavior changes.

### Keep unchanged
- `TagProvider` loading and mutation behavior
- `ImageMetadataPanel` polling / retry behavior (`_startPolling`, `_loadImageTagsWithRetry`)
- AI trigger API behavior
- Tag confirm / reject / merge / edit / remove callbacks
- Image preview area and filmstrip behavior

## Chunk 1: Lock the target behavior with tests

### Task 1: Extend metadata-pane and AI/tag widget tests

**Files:**
- Modify: `flutter_app/test/screens/image_detail_screen_test.dart`
- Modify: `flutter_app/test/widgets/image_metadata_panel_test.dart`
- Possibly modify: `flutter_app/test/widgets/tag_chip_test.dart`

- [ ] **Step 1: Add a failing desktop metadata-pane test for the new overview structure**

Add assertions to `flutter_app/test/screens/image_detail_screen_test.dart` that verify:

- the pane still renders in desktop mode
- the metadata section still exists
- the basic info area exposes the expected labels (`文件名`, `尺寸`, `格式`, `大小`, `导入时间`, `路径`)
- the path row can be targeted independently from the compact metadata rows
- the path value `Text` uses `maxLines: 1` and `TextOverflow.ellipsis`
- the full path remains accessible via a `Tooltip`
- the copy affordance exists and is targetable via tooltip text

Suggested test shape:

```dart
Widget buildDetailScreenHarness() {
  return ChangeNotifierProvider<ConfigProvider>(
    create: (_) => ConfigProvider(initialBaseUrl: 'http://localhost:8080'),
    child: MaterialApp(
      theme: ThemeData.dark(),
      home: ImageDetailScreen(image: image),
    ),
  );
}

testWidgets('renders overview metadata rows with a dedicated path row', (
  WidgetTester tester,
) async {
  await tester.binding.setSurfaceSize(const Size(1400, 900));
  addTearDown(() => tester.binding.setSurfaceSize(null));

  await tester.pumpWidget(buildDetailScreenHarness());
  await tester.pump();

  expect(find.text('文件名'), findsOneWidget);
  expect(find.text('路径'), findsOneWidget);
  expect(find.byKey(const Key('metadata-path-row')), findsOneWidget);

  final pathText = tester.widget<Text>(
    find.byKey(const Key('metadata-path-value')),
  );
  expect(pathText.maxLines, 1);
  expect(pathText.overflow, TextOverflow.ellipsis);
  expect(find.byTooltip('复制路径'), findsOneWidget);
});
```

- [ ] **Step 2: Run the metadata-pane test and confirm it fails for the new path-row key/structure**

Run from `flutter_app/`:

Run:

```bash
flutter test test/screens/image_detail_screen_test.dart
```

Expected: FAIL because the new path-row targeting structure, truncated path text, tooltip access, and copy affordance do not exist yet.

- [ ] **Step 3: Add failing AI/tag panel tests for the approved grouping and card hierarchy**

Extend `flutter_app/test/widgets/image_metadata_panel_test.dart` with cases that verify:

- the custom prompt toggle row remains visible before prompt expansion
- the prompt editor appears only after enabling custom prompt
- while AI work is active, the primary action remains visible in a disabled/loading state and the status pill stays secondary beside it
- the tag section renders groups in the order `待确认 -> 已确认 -> 已拒绝`
- each rendered group shows its own count or equivalent inline total

Create a local test harness in that file using `MockClient` responses rather than a fictional helper. Seed `confirmed`, `pending`, and `rejected` through the `/api/v1/images/:id/tags` response body so the widget renders deterministic groups.

Suggested test shape:

```dart
Widget buildMetadataPanelHarness(http.Client client) {
  return MaterialApp(
    home: Scaffold(
      body: ChangeNotifierProvider<TagProvider>(
        create: (_) => TagProvider(
          TagService(baseUrl: 'http://localhost:8080', client: client),
        ),
        child: const ImageMetadataPanel(
          imageId: 1,
          metadataSection: SizedBox.shrink(),
        ),
      ),
    ),
  );
}

testWidgets('shows prompt editor only after enabling custom prompt', (
  WidgetTester tester,
) async {
  final mockClient = MockClient((request) async {
    if (request.url.path.endsWith('/api/v1/ai-tags/default-prompt')) {
      return http.Response('{"default_prompt":"default prompt"}', 200);
    }
    if (request.url.path.endsWith('/api/v1/images/1/tags')) {
      return http.Response(
        '{"confirmed":[],"pending":[{"id":1,"preferred_label":"待确认标签","slug":"pending-1","review_state":"pending","trust_score":0.8,"usage_count":1,"created_at":"2024-01-15T10:30:00Z"}],"rejected":[]}',
        200,
      );
    }
    return http.Response('{}', 200);
  });

  await tester.pumpWidget(buildMetadataPanelHarness(mockClient));
  await tester.pumpAndSettle();

  expect(find.text('自定义提示词'), findsOneWidget);
  expect(find.byType(TextField), findsNothing);

  await tester.tap(find.byType(Switch));
  await tester.pump();

  expect(find.byType(TextField), findsOneWidget);
});
```

- [ ] **Step 4: Run the panel tests and confirm they fail on the old layout/order**

Run from `flutter_app/`:

Run:

```bash
flutter test test/widgets/image_metadata_panel_test.dart
```

Expected: FAIL because the current layout keeps the old grouping order and does not yet expose the new hierarchy/count treatment. The prompt-toggle visibility assertion may already pass; only keep failure expectations for genuinely new behavior.

- [ ] **Step 5: If chip styling changes are needed, add a failing tag-chip test first**

Only if `tag_chip.dart` must change, add a test that verifies the rejected style remains visually deemphasized and pending stays stronger.

Example:

```dart
testWidgets('keeps rejected chips visually deemphasized', (tester) async {
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: TagChip(tag: sampleTag, style: TagChipStyle.rejected),
      ),
    ),
  );

  final textWidget = tester.widget<Text>(find.text('测试标签'));
  expect(textWidget.style?.decoration, TextDecoration.lineThrough);
});
```

- [ ] **Step 6: Run the optional chip test file if touched**

Run:

```bash
flutter test test/widgets/tag_chip_test.dart
```

Expected: FAIL only if the new assertions target behavior that is not implemented yet.

## Chunk 2: Rebuild the basic information card and shared pane tokens

### Task 2: Refactor the overview metadata section for compact scanning

**Files:**
- Modify: `flutter_app/lib/screens/image_detail_screen.dart:523-624`
- Modify if needed: `flutter_app/lib/widgets/image_metadata_pane_theme.dart:11-51`
- Verify with: `flutter_app/test/screens/image_detail_screen_test.dart`

- [ ] **Step 1: Introduce dedicated rendering helpers for compact rows and the path row**

Refactor `_buildMetadataSection` so it no longer renders all fields through one uniform `_buildMetadataRow` path.

Target structure:

```dart
Widget _buildMetadataSection(...) {
  return Container(
    margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
    decoration: paneTheme.sectionDecoration,
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _buildMetadataHeader(...),
          _buildCompactMetadataRow(...文件名...),
          _buildCompactMetadataGrid(...尺寸..., ...格式...),
          _buildCompactMetadataGrid(...大小..., ...导入时间...),
          _buildPathMetadataRow(...),
        ],
      ),
    ),
  );
}
```

- [ ] **Step 2: Implement the dedicated path row behavior**

The path row should:

- render on its own line
- default to single-line ellipsis
- expose a stable test key such as `Key('metadata-path-row')`
- expose a stable value key such as `Key('metadata-path-value')`
- include a lightweight copy affordance or button without turning the card into a toolbar
- preserve access to the full value through tooltip/title/copy behavior

Minimal implementation sketch:

```dart
Widget _buildPathMetadataRow(...) {
  return Row(
    key: const Key('metadata-path-row'),
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      _buildMetadataLabel('路径'),
      Expanded(
        child: Tooltip(
          message: value,
          child: Text(
            key: const Key('metadata-path-value'),
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ),
      IconButton(
        tooltip: '复制路径',
        onPressed: () => Clipboard.setData(ClipboardData(text: value)),
        icon: const Icon(Icons.copy_outlined, size: 16),
      ),
    ],
  );
}
```

- [ ] **Step 3: Add deterministic test coverage for tooltip and copy behavior**

If clipboard copy is implemented, add a widget test that taps the copy affordance and verifies the clipboard contents through Flutter test clipboard APIs or a mocked platform channel.

At minimum, the screen test must verify:

- `find.byTooltip('复制路径')`
- the path text widget uses `maxLines: 1`
- the path text widget uses `TextOverflow.ellipsis`
- a `Tooltip` with the full path is present

- [ ] **Step 4: Add only the pane-theme tokens needed by the new metadata rhythm**

If current tokens are insufficient, add narrowly-scoped helpers in `image_metadata_pane_theme.dart`, for example:

```dart
double get compactRowGap => 10;
double get sectionHeaderGap => 12;
Color get subtleIconColor => colorScheme.onSurfaceVariant;
```

Do not introduce a new theme system.

- [ ] **Step 5: Run the screen test and make it pass**

Run:

```bash
flutter test test/screens/image_detail_screen_test.dart
```

Expected: PASS.

- [ ] **Step 6: Run analyzer diagnostics on the touched files**

Run:

```bash
flutter analyze lib/screens/image_detail_screen.dart lib/widgets/image_metadata_pane_theme.dart
```

Expected: No errors.

## Chunk 3: Rebuild the AI card and grouped tag card

### Task 3: Refactor the AI card hierarchy without changing AI trigger/polling semantics

**Files:**
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart:314-462`
- Verify with: `flutter_app/test/widgets/image_metadata_panel_test.dart`

- [ ] **Step 1: Rework the AI card header/body layout around one primary action**

Preserve the existing trigger/polling logic, but change the presentation so it matches the approved spec:

- header = icon + title + primary button + status
- body = visible custom-prompt toggle row + short helper text
- prompt editor appears only when `_useCustomPrompt` is true
- while AI is idle, the primary button remains the strongest control in the card
- while AI is queued/running (`_isAITriggered == true`), keep the primary action visible but disabled (or visibly loading) and render the status pill beside it as secondary information

Target shape:

```dart
Column(
  crossAxisAlignment: CrossAxisAlignment.start,
  children: [
    _buildAIHeader(...),
    const SizedBox(height: 12),
    _buildPromptToggleRow(...),
    if (_useCustomPrompt) ...[
      const SizedBox(height: 12),
      _buildPromptEditor(...),
    ] else
      _buildAIHelperText(...),
  ],
)
```

- [ ] **Step 2: Preserve polling and trigger semantics exactly as they are**

Do not change:

- `_triggerAITags`
- `_startPolling`
- `_loadImageTagsWithRetry`
- the request/response contract with `TagProvider`

Only move presentation code and conditional rendering. The approved presentation rule is: status must not replace the primary action visually.

- [ ] **Step 3: Run the AI/prompt widget tests and make them pass**

Run:

```bash
flutter test test/widgets/image_metadata_panel_test.dart
```

Expected: PASS for prompt-visibility and AI status behavior.

### Task 4: Refactor the tag card into ordered grouped subsections

**Files:**
- Modify: `flutter_app/lib/widgets/image_metadata_panel.dart:465-591`
- Modify if needed: `flutter_app/lib/widgets/tag_chip.dart:42-167`
- Verify with: `flutter_app/test/widgets/image_metadata_panel_test.dart`
- Verify if touched: `flutter_app/test/widgets/tag_chip_test.dart`

- [ ] **Step 1: Reorder the groups to pending → confirmed → rejected**

Update the section render order while preserving each group’s existing actions.

Implementation direction:

```dart
final groups = [
  _TagGroupConfig(label: '待确认', tags: pending, style: TagChipStyle.pending),
  _TagGroupConfig(label: '已确认', tags: confirmed, style: TagChipStyle.confirmed),
  _TagGroupConfig(label: '已拒绝', tags: rejected, style: TagChipStyle.rejected),
];
```

- [ ] **Step 2: Give each subsection explicit label/count structure**

Render each non-empty group with:

- subsection title
- count badge or inline count text
- chip wrap area

Suggested rendering helper:

```dart
Widget _buildTagGroup({
  required String label,
  required int count,
  required List<Tag> tags,
  required List<Widget> chips,
}) {
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Row(
        children: [Text(label), const SizedBox(width: 6), Text('($count)')],
      ),
      const SizedBox(height: 8),
      Wrap(spacing: 10, runSpacing: 8, children: chips),
    ],
  );
}
```

- [ ] **Step 3: Keep rejected chips accessible but visually quieter**

Prefer solving this in the group styling first. Touch `tag_chip.dart` only if the existing rejected style is insufficient.

If `tag_chip.dart` must change, keep the delta minimal, for example:

```dart
case TagChipStyle.rejected:
  return Colors.grey.shade500;
```

and avoid changing action availability.

- [ ] **Step 4: Preserve the existing empty state**

When all three groups are empty, keep the `暂无标签` fallback.

- [ ] **Step 5: Run the metadata-panel tests and make them pass**

Run:

```bash
flutter test test/widgets/image_metadata_panel_test.dart
```

Expected: PASS.

- [ ] **Step 6: Run the chip test file if touched**

Run:

```bash
flutter test test/widgets/tag_chip_test.dart
```

Expected: PASS.

## Chunk 4: Final verification and cleanup

### Task 5: Verify the whole UI change without widening scope

**Files:**
- Verify: `flutter_app/lib/screens/image_detail_screen.dart`
- Verify: `flutter_app/lib/widgets/image_metadata_panel.dart`
- Verify if touched: `flutter_app/lib/widgets/image_metadata_pane_theme.dart`
- Verify if touched: `flutter_app/lib/widgets/tag_chip.dart`

- [ ] **Step 1: Run the focused widget test set together**

Run from `flutter_app/`:

Run:

```bash
flutter test test/screens/image_detail_screen_test.dart test/widgets/image_metadata_panel_test.dart test/widgets/tag_chip_test.dart
```

Expected: PASS, or note that `tag_chip_test.dart` may be omitted if untouched.

- [ ] **Step 2: Run analyzer on all touched UI files**

Run from `flutter_app/`:

Run:

```bash
flutter analyze lib/screens/image_detail_screen.dart lib/widgets/image_metadata_panel.dart lib/widgets/image_metadata_pane_theme.dart lib/widgets/tag_chip.dart
```

Expected: No errors.

- [ ] **Step 3: Do a manual smoke check in Flutter desktop/web dev mode**

Run from `flutter_app/`:

```bash
flutter run -d chrome
```

Manual verification checklist:

- desktop width still shows the two-column details layout
- compact width still reuses the same metadata pane without overflow
- metadata card scans in the new order
- path truncates cleanly and copy works
- custom prompt toggle remains visible, prompt editor appears only when enabled
- tag groups render as pending → confirmed → rejected
- pending actions, confirmed edit/merge, and rejected delete still work

- [ ] **Step 4: Commit the UI refactor**

Run from `flutter_app/`:

```bash
git add lib/screens/image_detail_screen.dart lib/widgets/image_metadata_panel.dart lib/widgets/image_metadata_pane_theme.dart lib/widgets/tag_chip.dart test/screens/image_detail_screen_test.dart test/widgets/image_metadata_panel_test.dart test/widgets/tag_chip_test.dart
git commit -m "feat: refine image detail metadata pane hierarchy"
```

If `tag_chip.dart` or `tag_chip_test.dart` were untouched, remove them from the commit command.

## Notes for the implementing agent

- Prefer minimal structural helpers over broad widget extraction unless repetition becomes painful.
- Keep the implementation theme-aware; do not reintroduce hardcoded light surfaces.
- Do not alter API contracts, provider interfaces, or tag action semantics.
- If a test requires a new stable key or tooltip, add the narrowest possible hook for that test.

Plan complete and saved to `docs/superpowers/plans/2026-04-09-image-detail-left-panel-tag-layout-plan.md`. Ready to execute?
