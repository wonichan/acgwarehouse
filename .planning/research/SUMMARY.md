# Project Research Summary

**Project:** ACGWarehouse v2.0 UI/UX 重构与多端适配
**Domain:** Anime Image Gallery — Multi-platform Flutter UI (Windows Desktop + Android Mobile)
**Researched:** 2026-03-20
**Confidence:** HIGH

## Executive Summary

ACGWarehouse is an anime image gallery management system being extended from Flutter Web to support Windows desktop (Fluent Design) and Android mobile (Material 3) platforms. The v2.0 milestone focuses on dual-platform UI/UX refactoring while preserving the existing Go backend and Provider-based state management.

The recommended approach uses a **platform-aware app shell** pattern: a single entry point detects the platform and routes to either `FluentApp` (Windows) or `MaterialApp` (Android/Web). Business logic (Providers, Services, Models) remains shared and unchanged, while presentation layers are platform-specific. This architecture ensures native feel on each platform without duplicating business logic.

Key risks include dual MaterialApp/FluentApp nesting errors, theme type confusion between frameworks, and state loss during navigation switches. These are mitigated by: (1) conditional root widget selection at app entry, (2) unified theme interface with platform-specific implementations, and (3) Provider-managed navigation state above the widget tree. Development should proceed Windows-first due to higher complexity (new fluent_ui integration), then Android.

## Key Findings

### Recommended Stack

**Existing Backend (unchanged):**
- Go 1.24.x + Gin — HTTP server, excellent concurrency
- govips/libvips — Image processing, 5x faster than ImageMagick
- SQLite (ncruces/go-sqlite3) or PostgreSQL (pgx) — Dual database support

**Frontend Core:**
- Flutter 3.27.x+ + Dart 3.6.x+ — Cross-platform UI framework
- Provider ^6.1.5 — State management (existing, works with both UI frameworks)
- waterfall_flow — Pinterest-style gallery grid

**v2.0 New Dependencies:**
- **fluent_ui ^4.15.0** — Windows Fluent Design UI (NavigationView, NavigationPane, ScaffoldPage)
- **system_theme ^3.2.0** — Windows system accent color for native Fluent look
- **window_manager ^0.5.1** — Windows desktop window controls (size, position, title bar)

**NOT Required:**
- responsive_builder — Use native `LayoutBuilder` with 600px/900px breakpoints
- flutter_platform_widgets — Use manual `Platform.isWindows` detection
- Riverpod/BLoC — Keep existing Provider setup

### Expected Features

**Must have (MVP v2.0):**
- **NavigationView (Windows)** — Core Windows navigation with `PaneDisplayMode.auto`
- **NavigationBar (Android)** — Material 3 bottom navigation for phones
- **NavigationRail (Android tablets)** — Side navigation for large screens
- **Responsive Layout** — `LayoutBuilder` with 600px breakpoint
- **Anime Theming** — Soft pink-purple color scheme (`Color(0xFFED79B5)` seed)
- **Light/Dark Theme** — System-aware theme switching

**Should have (v2.x):**
- **Keyboard Shortcuts** — Desktop efficiency (Ctrl+N, Delete, Arrow keys)
- **CommandBar Actions** — Windows toolbar for page actions
- **Hover Effects** — Desktop mouse interaction refinement
- **Extended NavigationRail** — Labels visible on tablets in landscape

**Defer (v3+):**
- Custom touch gestures
- Animated screen transitions
- Arrow key gallery navigation

### Architecture Approach

**Pattern: Platform-Aware App Shell**

Single entry point (`AdaptiveApp`) detects platform and renders the appropriate UI framework shell. Business logic layer (Providers, Services, Models) remains unchanged and platform-agnostic. Presentation layer bifurcates into `ui/windows/` (fluent_ui) and `ui/android/` (Material 3).

**Major components:**
1. **AdaptiveApp** (`lib/app.dart`) — Platform detection, routes to FluentApp/MaterialApp
2. **WindowsNavigationShell** (`lib/ui/windows/shell.dart`) — NavigationView with PaneDisplayMode.auto
3. **AndroidNavigationShell** (`lib/ui/android/shell.dart`) — Scaffold with NavigationBar/Rail
4. **Core Theme** (`lib/core/theme/`) — Unified theme constants (purple accent)
5. **Shared Widgets** (`lib/ui/shared/`) — Platform-agnostic cards, chips, indicators

**Key principle:** UI frameworks change, business logic stays the same.

### Critical Pitfalls

1. **Dual MaterialApp/FluentApp Nesting** — Never nest these wrappers. Use conditional return at root: `if (Platform.isWindows) return FluentApp(...) else return MaterialApp(...)`

2. **Theme Data Type Confusion** — `Theme.of(context)` returns Material theme, `FluentTheme.of(context)` returns Fluent theme. Create unified theme interface with platform-specific implementations.

3. **Platform.isX in Widget Build** — `Platform.isWindows` crashes on web. Use `defaultTargetPlatform` from `flutter/foundation.dart` or wrap in `kIsWeb` guard.

4. **State Loss During Navigation Switch** — NavigationRail↔NavigationBar switch rebuilds widget tree. Keep navigation state in Provider above navigation widgets, use `PageStorage` for scroll position.

5. **Mismatched Navigation Patterns** — NavigationBar belongs on phones (<600px), NavigationRail on tablets/desktop. Windows uses `NavigationView` with `PaneDisplayMode.auto` for automatic adaptation.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Architecture Foundation
**Rationale:** Must establish platform detection and app shell before any UI work. Prevents dual-app nesting pitfall.
**Delivers:** AdaptiveApp root widget, platform detection, FluentApp/MaterialApp shells, Provider integration verified
**Addresses:** Core infrastructure from ARCHITECTURE.md
**Avoids:** Pitfall #11 (Dual MaterialApp/FluentApp), Pitfall #15 (Platform.isX in build)

### Phase 2: Windows Fluent UI Shell
**Rationale:** Windows has higher complexity (new fluent_ui package). Implement first while Android uses existing Material patterns.
**Delivers:** NavigationView shell, gallery/search/settings screens with Fluent widgets, Windows window controls
**Uses:** fluent_ui, window_manager, system_theme
**Implements:** WindowsNavigationShell, Windows-specific screens
**Avoids:** Pitfall #14 (Theme type confusion) by isolating Fluent theme

### Phase 3: Android Material 3 Adaptive Layout
**Rationale:** Build on existing Material knowledge. Implement responsive navigation (Bar/Rail switch).
**Delivers:** Adaptive NavigationBar/Rail, responsive grid, Material 3 theme with anime colors
**Uses:** LayoutBuilder, NavigationBar, NavigationRail, ColorScheme.fromSeed()
**Implements:** AndroidNavigationShell, responsive breakpoint logic
**Avoids:** Pitfall #12 (Hardcoded breakpoints), Pitfall #13 (Mismatched navigation), Pitfall #16 (State loss)

### Phase 4: Unified Theme & Polish
**Rationale:** With both platforms working, unify theme interface and add finishing touches.
**Delivers:** Shared theme constants, keyboard shortcuts (Windows), hover effects, extended NavigationRail, dark mode verification
**Addresses:** Differentiators from FEATURES.md (Anime theming, keyboard shortcuts)
**Avoids:** Pitfall #14 (Theme type confusion) — unified interface

### Phase Ordering Rationale

- **Windows before Android:** fluent_ui is new dependency with learning curve; Android reuses existing Material knowledge
- **Architecture first:** Platform detection and app shell are prerequisites for all UI work
- **Shared business logic:** Providers, Services, Models remain unchanged throughout — no migration risk
- **Responsive last:** Both platforms need base navigation working before responsive enhancements

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2:** fluent_ui specific patterns (ContentDialog, InfoBar, CommandBar) — Context7 available but may need code examples
- **Phase 4:** Keyboard shortcuts implementation (Shortcuts + Actions widgets) — well-documented but complex focus management

Phases with standard patterns (skip research-phase):
- **Phase 1:** Platform detection and conditional rendering — established Flutter pattern
- **Phase 3:** Material 3 NavigationBar/Rail — well-documented, official Flutter guides

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All packages verified on pub.dev, Context7 documentation available, existing project already uses Provider |
| Features | HIGH | Flutter official docs for adaptive layouts, Material 3 migration complete, fluent_ui 3.1k+ likes |
| Architecture | HIGH | Platform-aware shell pattern documented in Flutter docs, Provider proven in production |
| Pitfalls | HIGH | All pitfalls sourced from Context7, Flutter docs, and real GitHub code patterns |

**Overall confidence:** HIGH

### Gaps to Address

- **fluent_ui version compatibility:** Research verified ^4.15.0 works with Flutter 3.27+. Verify during `flutter pub get`.
- **Windows window_manager setup:** Requires native Windows configuration — follow package README during Phase 2.
- **Web compatibility:** Existing Web build must continue working. Test `kIsWeb` guards thoroughly.

## Sources

### Primary (HIGH confidence)
- Context7: `/bdlukaa/fluent_ui` — NavigationView, NavigationPane, FluentThemeData, ContentDialog
- Context7: `/websites/flutter_dev` — LayoutBuilder, Platform detection, NavigationRail, Actions & Shortcuts
- pub.dev: fluent_ui ^4.15.0, system_theme ^3.2.0, window_manager ^0.5.1 — Version verification
- Flutter Official: Adaptive-responsive layouts, Material 3 migration guides

### Secondary (MEDIUM confidence)
- GitHub: bdlukaa/fluent_ui examples — NavigationView patterns, PaneDisplayMode.auto usage
- GitHub: immich-app/immich — Real-world adaptive navigation, ColorScheme.fromSeed() usage
- Material Design: Responsive layout guidelines, NavigationRail specifications

### Tertiary (Context from existing codebase)
- Existing Provider setup — Confirmed works with both MaterialApp and FluentApp
- Existing Go backend — Unchanged, REST API continues serving all platforms

---
*Research completed: 2026-03-20*
*Ready for roadmap: yes*