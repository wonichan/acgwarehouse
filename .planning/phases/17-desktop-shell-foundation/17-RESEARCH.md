# Phase 17: Desktop Shell Foundation - Research

**Researched:** 2026-04-04
**Domain:** Flutter desktop shell composition, Fluent UI shell restructuring, gallery grid behavior, desktop filter panel accessibility, import entry integration
**Confidence:** HIGH

## Summary

Phase 17 should be implemented as a targeted Flutter desktop shell refactor, not a broad frontend rewrite. The current codebase already has the core pieces needed for the phase: `NavigationView` shelling, a dedicated search page, provider-backed tag filtering, a stable image loading pipeline, and square-grid rendering. The work is to reorganize those pieces into a Windows Photos-inspired desktop shell with a custom top toolbar, stable square gallery tiles, and a persistent right-side filter panel.

**Primary recommendation:** keep `fluent_ui` + `provider` as the phase stack, retain `NavigationPane` for view switching, move shell-level actions into a custom `TitleBar` toolbar, keep search results on the existing dedicated search page, make grid mode the default/stable primary mode, replace the gallery tag dialog with an always-available right-side filter panel, and add a thin product-facing import action that delegates to the existing Go manual-scan backend path instead of inventing a new import subsystem.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use a **custom top bar** as the desktop shell core instead of continuing with only loose page-level `PageHeader + CommandBar` composition.
- **D-02:** Use **top-bar-led shell + retained left navigation**. The top bar owns search, import, and settings; the left pane remains for view switching only.
- **D-03:** The top bar must carry both window-shell semantics and product action semantics, inspired by Windows Photos without pixel-perfect cloning.
- **D-04:** Search entry must be a **persistent top-bar search box**, not just a button.
- **D-05:** Search results stay in the **existing dedicated search view**; Phase 17 does not merge gallery and search data models.
- **D-06:** The **square grid** is the primary browsing mode for `DSK-02`.
- **D-07:** Existing `masonry` capability may remain, but it is **not the Phase 17 center of gravity**.
- **D-08:** The square grid should prioritize a **stable tile feel** with uniform sizes and only limited responsive adjustment.
- **D-09:** Tag filtering must use a **right-side collapsible panel**, not a dialog popup.
- **D-10:** Filter interaction is **apply-immediately** — tag and untagged changes refresh gallery without an Apply button.
- **D-11:** Existing selected-tag chips in the gallery title area may remain, but only as **status feedback / quick-clear**, not as the main filter UI.
- **D-12:** The top bar must expose an **import entry** alongside search and settings.
- **D-13:** Phase 17 import scope is **desktop-shell access to existing import capability**, not a new import center or operations module.
- **D-14:** Prefer reusing the existing backend manual-scan path and known import status contract hints. If the desktop frontend lacks a direct entry, add the UI/access layer without changing the Go/Python responsibility boundary.

### Agent's Discretion
- Exact split of drag region, title text, and action area in the custom top bar
- Overflow behavior for narrow window widths
- Exact grid breakpoint thresholds and spacing values
- Filter panel animation and default width
- Import button form (simple button vs split button vs status button)

### Deferred Ideas (OUT OF SCOPE)
- Independent viewer window / filmstrip / viewer interactions (Phase 18)
- Desktop tag governance flows (Phase 19)
- Import monitoring / sidecar diagnostics entry (Phase 20)
- Large-gallery performance program and sidecar recovery UX (Phase 22)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DSK-01 | User can access search, import, and settings from the desktop top toolbar | Custom shell toolbar, search handoff, settings routing, thin import action |
| DSK-02 | User can browse gallery in square grid layout | Stable square tile defaults, grid-first shell behavior, limited responsive columns |
| DSK-03 | User can filter gallery content by tags in an accessible filter panel | Persistent right-side panel, provider-backed immediate filter updates, keyboard/focus-aware layout |
</phase_requirements>

## Standard Stack

### Core UI Stack
| Library | Current Status | Use in Phase 17 | Why |
|---------|----------------|-----------------|-----|
| `fluent_ui` | already in `pubspec.yaml` | Shell, title bar, navigation pane, buttons, text boxes, panels | Existing Windows desktop UI system; avoids re-platforming |
| `provider` | already in `pubspec.yaml` | Shell state composition with `NavigationProvider`, `ImageListProvider`, `SearchProvider`, `TagProvider` | Existing state management pattern throughout app |
| `window_manager` | already in `pubspec.yaml` | Preserve drag/move semantics inside custom title bar | Existing desktop shell dependency |
| `flutter_staggered_grid_view` | already in `pubspec.yaml` | Preserve masonry implementation as non-primary mode | Existing gallery behavior, no new dependency |

### Backend Surface
| Component | Current Status | Use in Phase 17 | Why |
|-----------|----------------|-----------------|-----|
| `AdminService.TriggerScan()` | implemented | Reuse as backend scan trigger | Existing manual import path already queues `manual_scan` |
| `POST /admin/api/actions/scan` | implemented | Source behavior to delegate from product-facing endpoint | Keeps Phase 17 import work thin |
| `POST /api/v1/images/scan` | placeholder | Recommended thin product-facing endpoint | Matches existing route placeholder and desktop product flow |
| `GET /api/v1/images/import-status` | hinted in Flutter config | Recommended lightweight status endpoint | Lets toolbar show basic action/result feedback without Phase 20 scope creep |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing `provider` | Riverpod / Bloc | Unnecessary architectural churn for a shell-only phase |
| Retaining `NavigationPane` | Full custom sidebar | More code, more regressions, less leverage from current tests |
| Right-side panel in page body | Overlay dialog / flyout | Violates D-09 and weakens persistent accessibility |
| Thin `/api/v1/images/scan` wrapper | Direct admin API call from Flutter | Couples desktop product flow to admin auth surface |
| Grid-first stabilization | New virtualization package | Premature for Phase 17; performance work belongs to Phase 22 |

## Architecture Patterns

### Pattern 1: Shell-Orchestrated Toolbar
**What:** Move cross-view actions into the app shell title bar and keep view-specific content inside page bodies.
**Use here:** Search box, import button, settings button, and gallery filter toggle all belong to the shell layer.

```dart
class FluentAppShell extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Consumer<NavigationProvider>(
      builder: (context, nav, _) {
        return NavigationView(
          titleBar: TitleBar(
            title: DesktopShellTopBar(
              selectedIndex: nav.selectedIndex,
              onNavigate: nav.setSelectedIndex,
            ),
          ),
          pane: NavigationPane(...),
        );
      },
    );
  }
}
```

**Why this fits:** It implements D-01/D-02/D-03 cleanly, reduces duplicated command bars, and keeps shell semantics in one place.

### Pattern 2: Search Handoff, Not Search Merge
**What:** The top-bar search box should submit into `SearchProvider`, then navigate to the existing search page.
**Use here:** Preserve the current search page while upgrading entry placement.

```dart
onSubmitted: (query) async {
  if (query.trim().isEmpty) return;
  await context.read<SearchProvider>().search(query: query.trim());
  context.read<NavigationProvider>().setSelectedIndex(NavigationProvider.searchIndex);
}
```

**Why this fits:** It directly implements D-04/D-05 and avoids reworking gallery/search contracts.

### Pattern 3: Gallery Workspace Layout
**What:** Compose the gallery page as a horizontal workspace: `gallery content + right filter panel`.
**Use here:** Replace the current filter dialog with an in-layout panel that can collapse.

```dart
Row(
  children: [
    Expanded(child: FluentGalleryContent(...)),
    if (shell.showFilterPanel)
      SizedBox(
        width: 320,
        child: GalleryFilterPanel(...),
      ),
  ],
)
```

**Why this fits:** It satisfies D-09/D-10, keeps filters visible, and is easy to test with widget tests.

### Pattern 4: Provider-Driven Immediate Filters
**What:** Continue using `TagProvider` selection state and `ImageListProvider` fetch methods, but wire them from panel interactions rather than a modal dialog.
**Use here:** Toggling a checkbox updates tags and reloads images immediately.

```dart
onChanged: (selected) async {
  tagProvider.toggleTag(tag.id);
  await imageProvider.setTagFilter(tagProvider.selectedTagIds.toList());
}
```

**Why this fits:** The current providers already implement the immediate-reload behavior needed for D-10.

### Pattern 5: Thin Import Endpoint
**What:** Expose a small product-facing image scan endpoint in Go that delegates to the existing manual scan job path.
**Use here:** The toolbar import button gets a clean desktop-safe endpoint without pulling Phase 20 concerns into Phase 17.

```go
func (h *ImageHandler) TriggerImport(c *gin.Context) {
    jobID, err := h.adminService.TriggerScan(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "status": "queued"})
}
```

**Why this fits:** It honors D-12/D-13/D-14 and leverages the existing `manual_scan` backend.

## Recommended File Shape

| File | Recommendation |
|------|----------------|
| `flutter_app/lib/app/fluent_app_shell.dart` | Keep as shell root but extract top-bar and shell state wiring helpers out of the giant widget tree |
| `flutter_app/lib/app/fluent_screens.dart` | Split gallery-specific shell concerns from search screen concerns; remove page-level command bar responsibilities that migrate upward |
| `flutter_app/lib/widgets/fluent_gallery_content.dart` | Keep grid/masonry rendering here; make square grid the stable default and leave masonry secondary |
| `flutter_app/lib/providers/navigation_provider.dart` | Extend with shell actions / filter panel visibility / search entry helpers only if they are true shell concerns |
| `flutter_app/lib/services/...` | Add a small import/scan service rather than burying HTTP calls in widgets |
| `internal/handler/routes.go` + image handler files | Implement the thin product-facing import entry using existing scan backend logic |

## TDD Planning Guidance

Phase 17 is a good fit for **widget-first and provider-first TDD**, not visual-only planning. The highest-value red-green loops are:

1. **Shell contract tests first**
   - Top bar renders persistent search input, import button, settings button
   - Submitting search navigates to search page
   - Settings action switches to settings page

2. **Gallery workspace tests first**
   - Gallery renders grid-first defaults
   - Filter panel visibility toggles from shell/gallery action
   - Tag toggles call provider methods and update chips / filter state

3. **Backend import endpoint tests first**
   - `POST /api/v1/images/scan` returns `202` and queued job id
   - Failure path returns structured error

4. **Service/client tests first**
   - Flutter import action service calls the product endpoint
   - UI feedback state maps queued/error responses correctly

## Atomic Commit Strategy

Use small commits aligned to passing tests:

1. `test(17-01): add failing desktop shell toolbar coverage`
2. `feat(17-01): wire custom top bar search settings and shell actions`
3. `test(17-02): add failing gallery workspace and filter panel tests`
4. `feat(17-02): ship grid-first gallery workspace and persistent filter panel`
5. `test(17-03): add failing import endpoint and client coverage`
6. `feat(17-03): expose desktop import action and toolbar integration`

If a refactor is needed after green, use a separate commit such as:
- `refactor(17-02): extract reusable shell workspace widgets`

## Common Pitfalls

### Pitfall 1: Hiding shell behavior inside page command bars
If search/import/settings stay page-local, the shell never becomes consistent and `DSK-01` is only partially met.

**Avoid by:** moving those actions into the shell title bar and leaving only page-specific controls in pages.

### Pitfall 2: Fake persistent filter panel implemented as a dialog or flyout
This breaks D-09 even if the visuals look similar.

**Avoid by:** making the filter panel part of the gallery page layout tree.

### Pitfall 3: Treating masonry and grid as equal Phase 17 priorities
That spreads effort away from the locked square-grid requirement.

**Avoid by:** making grid the default and validating square-tile behavior first.

### Pitfall 4: Shipping an import button with no real backend contract
That creates a dead shell affordance and fails D-12/D-13.

**Avoid by:** either delegating through `/api/v1/images/scan` or explicitly wiring the product path to the existing scan backend before the toolbar UI lands.

### Pitfall 5: Pulling Phase 20 monitoring into Phase 17
Import badges, live task dashboards, and sidecar diagnostics expand scope.

**Avoid by:** limiting Phase 17 import feedback to action-level queued/success/error state only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Desktop navigation framework | Full custom window shell | Existing `NavigationView` + `NavigationPane` | Already integrated and tested |
| Search result rendering | New merged gallery/search page | Existing `FluentSearchPage` + `SearchProvider` | Matches D-05 and reduces risk |
| Filter state model | New duplicated filter store | Existing `TagProvider` + `ImageListProvider` | Current code already supports immediate filter reload |
| Import orchestration | New import pipeline | Existing `AdminService.TriggerScan()` manual scan job path | Reuses proven backend behavior |

## Validation Architecture

### Fast Feedback Loop
- **Shell widgets:** `flutter test test/app/fluent_app_shell_test.dart`
- **Gallery workspace:** `flutter test test/widgets/fluent_gallery_content_test.dart test/app/fluent_screens_test.dart`
- **Providers:** `flutter test test/providers/image_provider_has_tags_test.dart test/providers/navigation_provider_test.dart test/providers/tag_provider_test.dart`
- **Go import route:** `go test ./internal/handler ./internal/service -run 'Test(AdminHandler_TriggerScan|AdminService_TriggerScan)' -count=1`

### Phase-Specific Additions Recommended
- Add a dedicated shell toolbar widget test file for search/import/settings behavior
- Add a gallery filter panel widget test file for right-side panel presence and immediate apply behavior
- Add a Flutter service test for import action client
- Add Go handler tests for `/api/v1/images/scan` response shape

### Manual Verification That Still Matters
- Narrow-width desktop window behavior for top-bar overflow
- Keyboard focus order across search box, import button, settings button, and filter controls
- Visual confirmation that grid tiles feel stable and square under window resize

## Final Recommendation

Split Phase 17 into **three execution plans**:

1. **Shell contract plan** — custom top bar, search handoff, settings access, shell state/tests
2. **Gallery workspace plan** — grid-first gallery page, persistent right filter panel, immediate filter behavior/tests
3. **Import access plan** — thin Go import endpoint, Flutter import client/button wiring, minimal action feedback/tests

This split maps cleanly to the three phase requirements, supports TDD-oriented execution, and keeps import integration from blocking shell and gallery progress unnecessarily.
