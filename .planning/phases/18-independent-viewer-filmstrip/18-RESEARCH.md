# Phase 18: Independent Viewer & Filmstrip - Research

**Researched:** 2026-04-05
**Domain:** Flutter Windows desktop multi-window viewer bootstrap, viewer workspace composition, filmstrip navigation, metadata sidebar reuse
**Confidence:** HIGH

## Summary

Phase 18 should be implemented as a **staged desktop viewer expansion**, not as a route-only detail-page enhancement and not as a viewer-stack rewrite. The existing Flutter code already contains the key reusable primitives needed for the phase: `ExtendedImage` gesture behavior, provider-backed gallery/search result state, desktop shell/window initialization, and a reusable metadata/tags sidebar structure. The missing capability is **true independent viewer window hosting** plus a viewer-specific workspace that can render the current image, a result-set-scoped filmstrip, and a fixed right metadata sidebar.

**Primary recommendation:** keep `fluent_ui` + `provider` + `extended_image` as the core UI stack; introduce a dedicated **multi-window host layer** for Windows desktop using a true multi-window plugin (for example `window_manager_plus` or an equivalent package proven to spawn secondary native windows), preserve `window_manager`-style sizing/title conventions for the current window lifecycle, extract a container-agnostic viewer workspace widget tree that can render both in test harnesses and in a real secondary window, and integrate entry points from both gallery and search so double-click opens a non-modal viewer window whose filmstrip follows the originating result set.

This aligns with `VIEW-01` through `VIEW-04`, honors locked decisions `D-01` through `D-09`, preserves Phase 17 shell foundations, and avoids scope creep into tag governance, packaging, operations monitoring, or Phase 22 performance work.

## Locked Decisions to Preserve

The following decisions from `.planning/phases/18-independent-viewer-filmstrip/18-CONTEXT.md` are mandatory and should shape all planning and implementation slices:

- **D-01:** Open a **non-modal independent viewer window** on image double-click; allow multiple viewer windows.
- **D-02:** Do **not** persist viewer window size/position in Phase 18.
- **D-03:** Filmstrip follows the **current result set**, not folder-only scope.
- **D-04:** Filmstrip uses **medium-density thumbnails** with a clearly highlighted current item and visible neighbors.
- **D-05:** Viewer layout is **main image + fixed right metadata sidebar + bottom filmstrip**.
- **D-06:** Sidebar shows the **extended metadata set**: filename, format, resolution, size, path, imported time, tags.
- **D-07:** Double-click zoom semantics are **fit-to-window ↔ 2x**.
- **D-08:** Switching images resets viewer state to **fit-to-window**.
- **D-09:** Keyboard scope is limited to **`←/→` image navigation** and **`Esc` close current viewer**.

Any plan or implementation path that weakens these should be rejected.

## Grounded Findings

### Existing Code Strengths

- `flutter_app/lib/screens/image_gallery_viewer.dart` already proves `ExtendedImage` supports the required desktop viewing baseline: zoom, pan, page-based switching, and double-click zoom toggling.
- `flutter_app/lib/screens/image_detail_screen.dart` already contains reusable metadata and tag presentation sections that map well to the fixed right sidebar required by `D-05` and `D-06`.
- `flutter_app/lib/providers/image_provider.dart` already owns the gallery result set and filtering/sort state, making it the natural data source for a gallery-launched filmstrip session.
- `flutter_app/lib/providers/search_provider.dart` already owns the search result set and pagination/sort state, making it the natural data source for a search-launched filmstrip session.
- `flutter_app/lib/app/fluent_screens.dart`, `flutter_app/lib/widgets/fluent_gallery_content.dart`, and `flutter_app/lib/widgets/fluent_search_content.dart` already concentrate the image-click entry points that should be upgraded from page push behavior to independent viewer launch behavior.
- `flutter_app/lib/utils/window_manager.dart` already standardizes desktop window initialization and title updates for the current process window, so it remains a useful baseline for title/size conventions even though it does **not** create additional native windows.

### Hard Constraints from Research and Product Context

- Existing `window_manager` manages only the current window lifecycle and is **insufficient for true multi-window viewer support**.
- True desktop multi-window support likely needs a **dedicated multi-window plugin** plus Windows runner/bootstrap changes.
- Existing `ExtendedImage` behavior already matches the phase interaction contract closely enough to **prefer reuse over replacement**.
- The app is currently **single-window, route-push based**, so the safest path is a **hybrid staged implementation**: window host/bootstrap first, then viewer workspace, then integration.

## Recommended Architecture

### 1. Split the phase into three architectural layers

1. **Viewer Window Host Layer**
   - Owns secondary-window creation, argument passing, default sizing/title, and close behavior.
   - Bridges platform bootstrap details and app-level viewer session launch requests.
   - Must support multiple concurrent viewer instances to satisfy `D-01`.

2. **Viewer Workspace Layer**
   - Pure Flutter widget composition for the viewer content area.
   - Owns the three-region layout: image stage, fixed right metadata sidebar, bottom filmstrip.
   - Must be container-agnostic so it can render in widget tests and in a real spawned viewer window.

3. **Viewer Session Integration Layer**
   - Converts gallery/search click context into a viewer session payload.
   - Resolves the active image index and the filmstrip result set.
   - Updates viewer title and keyboard navigation behavior.

This separation gives planners a clean sequence: bootstrap the window host without entangling image UI, stabilize the viewer workspace without native window complexity, then wire entry points.

### 2. Use result-set sessions, not folder-scoped sessions

The viewer should launch with a **viewer session model** containing:

- launch source (`gallery` or `search`)
- result items snapshot or resolvable IDs
- initial selected index
- title seed (`filename`)
- optional source query/sort/filter metadata for diagnostics only

The session must be scoped to the originating result set to honor `D-03`, even if the roadmap’s older wording mentions folder-based switching.

### 3. Extract reusable viewer widgets instead of extending detail page directly

Do **not** keep Phase 18 centered on `ImageDetailScreen` or `Navigator.push`. Instead:

- extract a reusable **metadata sidebar widget** from `ImageDetailScreen`
- extract or reimplement a **viewer stage widget** built around `ExtendedImage`
- add a dedicated **filmstrip widget** with explicit selected-item styling
- compose those into a **viewer workspace widget** for desktop use

This preserves reuse while avoiding the anti-pattern of growing a mobile/material detail page into a desktop multi-window viewer host.

## Recommended Stack Decisions

## Core UI Stack

| Library / Surface | Status | Recommendation | Why |
|---|---|---|---|
| `fluent_ui` | already present | Keep for viewer shell chrome, filmstrip/sidebar surfaces, focus semantics | Matches Phase 17 desktop shell baseline and Phase 18 UI spec |
| `provider` | already present | Keep for launch-context access and lightweight viewer session models | Consistent with current app state patterns |
| `extended_image` | already present | Reuse for image rendering, zoom, pan, double-click semantics | Already satisfies `D-07` and most of `VIEW-03` |
| `window_manager` | already present | Keep for current-window lifecycle conventions only | Still useful for sizing/title helpers but not for spawning extra windows |
| true multi-window plugin | not present | Add a dedicated plugin such as `window_manager_plus` or equivalent | Required for `VIEW-01` true non-modal multi-window behavior |

### Stack Recommendations

- **Adopt:** a dedicated multi-window package proven to create secondary Windows windows from Flutter desktop.
- **Retain:** `ExtendedImage` instead of replacing it with a new viewer framework.
- **Retain:** `provider` rather than introducing a new state system for one phase.
- **Avoid:** route-only or dialog-based viewer simulation as the shipped Phase 18 path.
- **Avoid:** mixing Phase 18 with packaging-time shell bootstrap changes beyond what the new window host strictly needs.

## File-Level Anchors

| File | Role in Phase 18 research | Recommendation |
|---|---|---|
| `.planning/phases/18-independent-viewer-filmstrip/18-CONTEXT.md` | Phase boundary and locked decisions | Treat as authoritative source for `D-01` through `D-09` |
| `.planning/phases/18-independent-viewer-filmstrip/18-UI-SPEC.md` | Viewer visual/interaction contract | Treat as authoritative for layout, copy, density, and desktop constraints |
| `.planning/phases/17-desktop-shell-foundation/17-RESEARCH.md` | Prior-phase planning pattern and validation shape | Reuse its staged/TDD-oriented planning style |
| `flutter_app/pubspec.yaml` | Dependency baseline | Add one multi-window dependency only; avoid broader stack churn |
| `flutter_app/lib/main.dart` | App bootstrap | Extend desktop bootstrap to initialize the secondary-window host path |
| `flutter_app/lib/utils/window_manager.dart` | Current window setup helpers | Refactor into shared default window constants/helpers, but do not treat as full multi-window solution |
| `flutter_app/lib/app/fluent_screens.dart` | Current gallery/search detail entry | Replace `Navigator.push` detail launches with viewer-session launch requests |
| `flutter_app/lib/widgets/fluent_gallery_content.dart` | Gallery image click surface | Preserve click/double-click entry semantics and pass current gallery result context |
| `flutter_app/lib/widgets/fluent_search_content.dart` | Search image click surface | Mirror gallery integration for search result contexts |
| `flutter_app/lib/screens/image_gallery_viewer.dart` | Existing gesture/viewer baseline | Mine `ExtendedImage` gesture config and reset semantics; do not keep page-view scaffold as final desktop shell |
| `flutter_app/lib/screens/image_detail_screen.dart` | Existing metadata/tags presentation | Extract reusable metadata/tags sidebar pieces and leave AI/tag-management controls out of Phase 18 scope |
| `flutter_app/lib/providers/image_provider.dart` | Gallery result-set source | Use as gallery-launched filmstrip source |
| `flutter_app/lib/providers/search_provider.dart` | Search result-set source | Use as search-launched filmstrip source |
| `Windows11-Photos-App-ACG-Gallery-Research.md` | Product reference | Borrow independent-window/filmstrip feel, but not editing, slideshow, or cloud features |

## Implementation Shape Recommended for Planning

### Stage A — Window Host / Bootstrap

Deliver true secondary-window capability first.

- Add the chosen multi-window plugin to `flutter_app/pubspec.yaml`.
- Extend `flutter_app/lib/main.dart` so the process can bootstrap either the main shell or a viewer-window entry.
- Refactor `flutter_app/lib/utils/window_manager.dart` into shared default viewer/main-window sizing/title helpers.
- Add a viewer-window launch service or coordinator rather than calling plugin APIs directly from widgets.

**Planning note:** this stage is the architectural prerequisite for `VIEW-01`; do not hide it inside later viewer UI tasks.

### Stage B — Viewer Workspace

Build a testable desktop viewer surface that can render independent of native window creation.

- Create a dedicated viewer workspace under a viewer-focused path such as `flutter_app/lib/screens/viewer/` or `flutter_app/lib/widgets/viewer/`.
- Compose:
  - image stage widget using `ExtendedImage`
  - metadata sidebar widget extracted from `ImageDetailScreen`
  - filmstrip widget bound to a list of `ImageModel`
- Encode `D-07` and `D-08` directly in the viewer stage/session logic.
- Encode copy/title labels from `18-UI-SPEC.md`.

**Planning note:** this stage should be executable under widget tests before native window host integration is fully exercised.

### Stage C — Entry Integration

Wire viewer launches from both gallery and search.

- Replace route-based `_showImageDetail(...)` behavior in `flutter_app/lib/app/fluent_screens.dart` with viewer launch orchestration.
- Ensure gallery launches pull result data from `ImageListProvider.images`.
- Ensure search launches pull result data from `SearchProvider.results`.
- Use double-click as the primary open gesture for Phase 18, preserving non-blocking desktop flow.

**Planning note:** integrate only after the host and workspace layers are stable enough to avoid cross-cutting churn.

## TDD Planning Guidance

Phase 18 is a **mixed TDD phase**: some seams are strong candidates for classic red-green loops, while some desktop-native slices are better validated by harness tests and manual verification.

### TDD-Appropriate Seams

1. **Viewer session model / coordinator**
   - open request builds the correct session payload
   - initial index resolves correctly from clicked image
   - previous/next navigation clamps or disables at boundaries
   - image switch resets zoom state to fit mode (`D-08`)

2. **Viewer workspace widget contracts**
   - renders image stage, right sidebar, and filmstrip together
   - selected thumbnail state is visually distinct by more than color
   - metadata sidebar shows filename, format, resolution, size, path, imported time, tags
   - loading/failure states preserve workspace structure

3. **Metadata/sidebar extraction**
   - extracted sidebar renders the required field groups from `ImageModel`
   - omits out-of-scope editing and AI-tag actions for Phase 18 viewer windows

4. **Entry-point integration services**
   - gallery launch service receives `ImageListProvider.images`
   - search launch service receives `SearchProvider.results`
   - window title formatting follows `ACGWarehouse Viewer — {filename}`

### Non-TDD-Primary UI Slices

The following can still have tests, but should not block the phase on brittle fine-grained TDD loops:

- native Windows runner/bootstrap changes required by the chosen multi-window package
- final drag/focus behavior between viewer window chrome and Flutter content
- desktop keyboard focus behavior across filmstrip/sidebar in real spawned windows
- final visual tuning of filmstrip density and scroll feel

These should be validated by integration smoke checks and manual desktop verification after the core testable seams are green.

## Key Constraints

- **True multi-window is mandatory** for shipped Phase 18 behavior; route push, dialog overlay, or fake detached surfaces are not sufficient for final acceptance.
- **Window memory is explicitly out of scope** in this phase (`D-02`).
- **Result-set scope is authoritative** even if some legacy/product notes mention folder-scoped filmstrips (`D-03`).
- **Viewer state reset on image switch is mandatory** (`D-08`) even if persistent zoom feels tempting.
- **Keyboard scope must stay narrow** to avoid scope creep (`D-09`).
- **Do not widen into tag governance**: the viewer may display tags, but editing/merging/governance belongs to Phase 19.
- **Do not widen into operations diagnostics, packaging, or performance programs**.

## Risks

### Risk 1: Underestimating multi-window bootstrap cost

Secondary-window support may require Windows runner changes, argument dispatch, and plugin-specific lifecycle handling.

**Mitigation:** isolate the window host stage first and prove opening/closing multiple viewer windows before integrating UI entry points.

### Risk 2: Coupling viewer UI directly to provider internals

If the viewer window depends on live provider instances from the main shell, spawned windows may become fragile or impossible to bootstrap cleanly.

**Mitigation:** use a viewer session payload or thin session store boundary rather than direct widget-level provider reach-through.

### Risk 3: Reusing `ImageDetailScreen` too literally

Direct reuse would drag Material-page scaffolding, AI-tag actions, and route-page assumptions into the desktop viewer.

**Mitigation:** extract sidebar widgets and field-formatting helpers, not the whole screen.

### Risk 4: Treating `ExtendedImage` as the problem instead of the window host

A viewer rewrite would add churn without solving the actual blocker.

**Mitigation:** keep the current gesture stack unless a concrete blocker is proven during implementation.

### Risk 5: Letting filmstrip behavior depend on lazy-loading assumptions

Gallery/search providers paginate; a viewer launched from a partial result set may not represent the entire universe.

**Mitigation:** define Phase 18 filmstrip semantics around the **available current result set context at launch time**, not a background attempt to widen scope.

## Anti-Patterns to Avoid

- **Do not** ship Phase 18 as `Navigator.push` to `ImageDetailScreen` with a filmstrip bolted on.
- **Do not** simulate independent windows with dialogs, overlays, or panes inside the main shell.
- **Do not** replace `ExtendedImage` preemptively.
- **Do not** bind the filmstrip to folder membership when the locked decision says result-set scope.
- **Do not** carry over AI tag generation, tag merge/edit flows, slideshow controls, edit/delete/share affordances, or packaging concerns.
- **Do not** add window memory, advanced keyboard shortcuts, or performance infrastructure in this phase.
- **Do not** let native window bootstrap details leak throughout gallery/search widgets; keep launch behavior behind a coordinator/service boundary.

## Testing Strategy

### Automated Tests First

- Widget tests for the viewer workspace layout contract.
- Widget tests for filmstrip selected-state, switch behavior, and neighboring-context visibility rules.
- Widget/provider tests for viewer session navigation and reset-on-switch semantics.
- Service/unit tests for viewer launch coordination and title formatting.

### Desktop Smoke Tests

- Spawn multiple viewer windows from gallery and confirm the main shell remains usable.
- Spawn multiple viewer windows from search and confirm the main shell remains usable.
- Confirm `Esc` closes only the active viewer window.
- Confirm `←/→` changes image within the current viewer session only.

### Manual Visual Verification

- Filmstrip density feels medium rather than cramped.
- Right sidebar remains readable and visually subordinate to the image stage.
- Double-click toggles fit ↔ 2x.
- Switching image resets zoom/pan state.
- Loading/failure states do not collapse the three-region workspace.

## Validation Architecture

The later `VALIDATION.md` for Phase 18 should require evidence for **all** of the following:

### Automated Validation Requirements

- `flutter test` coverage for viewer-session logic, including initial index resolution and previous/next navigation.
- `flutter test` coverage for viewer workspace rendering, including image stage, metadata sidebar, and filmstrip presence in one composed surface.
- `flutter test` coverage for `D-08` reset semantics when switching images.
- `flutter test` coverage for sidebar metadata fields: filename, format, resolution, size, path, imported time, tags.
- `flutter test` or service tests for viewer title formatting and launch request translation from gallery/search contexts.

### Native/Desktop Validation Requirements

- A repeatable desktop smoke command or manual procedure proving that at least **two viewer windows** can be opened concurrently while the main shell remains interactive.
- A repeatable procedure proving `Esc` closes only the current viewer window.
- A repeatable procedure proving `←/→` navigation stays inside the viewer’s own result-set session.
- A repeatable procedure proving each new viewer opens with the default size/position policy and **does not** restore prior window memory.

### Acceptance Evidence Requirements

- Evidence that gallery-launched viewer sessions use gallery result context.
- Evidence that search-launched viewer sessions use search result context.
- Evidence that filmstrip selection is indicated by border/contrast and not color alone.
- Evidence that loading/failure states keep the viewer shell structure intact.
- Evidence that no out-of-scope viewer actions (edit/share/delete/slideshow/tag governance) were introduced.

## Atomic Commit Strategy

Use commit boundaries that mirror the recommended staged architecture:

1. `test(18-01): add failing viewer session and launch coordinator coverage`
2. `feat(18-01): add viewer session model and launch coordinator seam`
3. `test(18-01): extend failing desktop bootstrap coverage for viewer windows`
4. `feat(18-01): add Windows secondary-window bootstrap for viewer host`
5. `refactor(18-01): extract shared desktop window helpers`
6. `test(18-02): add failing viewer workspace and metadata sidebar coverage`
7. `feat(18-02): add reusable viewer workspace shell and metadata sidebar`
8. `test(18-02): add failing viewer stage, filmstrip, and keyboard coverage`
9. `feat(18-02): add viewer stage, filmstrip, and keyboard scope`
10. `test(18-03): add failing gallery and search viewer launch coverage`
11. `feat(18-03): wire independent viewer launches from gallery and search`
12. `test(18-03): extend viewer host coverage for mounted workspace and title updates`
13. `feat(18-03): mount viewer workspace in spawned windows`
14. `docs(18-03): add phase validation procedure for independent viewer windows`

If native bootstrap refactoring becomes necessary after green, keep it isolated:

- `refactor(18-02): extract shared desktop window helpers`

## Final Recommendation

Phase 18 should be planned as **three execution slices**:

1. **Window host/bootstrap slice** — establish true Windows secondary-window support and default viewer window policy.
2. **Viewer workspace slice** — build the reusable desktop viewer surface with `ExtendedImage`, right metadata sidebar, and result-set filmstrip.
3. **Integration slice** — replace gallery/search page-push detail behavior with independent viewer launch orchestration.

This is the lowest-risk path because it tackles the real blocker first (true multi-window hosting), reuses the existing gesture stack instead of rewriting it, preserves Phase 17 shell decisions, and gives downstream planners clear TDD seams plus concrete manual validation targets.
