# Feature Research: Windows Fluent UI + Android Material 3

**Domain:** Multi-platform Flutter UI (Windows Desktop + Android Mobile)
**Milestone:** v2.0 UI/UX 重构与多端适配
**Researched:** 2026-03-20
**Confidence:** HIGH

> **Note:** This research focuses on NEW features for the v2.0 UI milestone. Core product features (gallery, tagging, search, etc.) are already implemented in v1.0. See original FEATURES.md for core product feature landscape.

---

## Executive Summary

This research covers the feature landscape for adding Windows Fluent UI and Android Material 3 interfaces to an existing Flutter Web image gallery application. The focus is on platform-specific UI components, responsive layouts, adaptive navigation, and input handling patterns. Windows users expect Fluent Design patterns (NavigationView, CommandBar, keyboard shortcuts) while Android users expect Material 3 patterns (NavigationBar, touch interactions, ripples). The key challenge is creating a unified codebase that feels native on both platforms while maintaining the anime-style aesthetic.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Platform | Complexity | Notes |
|---------|--------------|----------|------------|-------|
| **NavigationView** | Standard Windows navigation pattern | Windows | MEDIUM | Uses `fluent_ui` package with `PaneDisplayMode.auto` for responsive adaptation |
| **NavigationBar** | Standard Android bottom navigation | Android | LOW | Material 3 `NavigationBar` with `NavigationDestination` items |
| **NavigationRail** | Expected for tablets/large screens | Both | MEDIUM | Switches from NavigationBar at 600px+ width threshold |
| **Keyboard Shortcuts** | Desktop users expect accelerator keys | Windows | MEDIUM | `Shortcuts` + `Actions` widgets with `LogicalKeySet` |
| **Hover Effects** | Mouse interaction expected on desktop | Windows | LOW | `MouseRegion` for custom hover states |
| **Touch Ripples** | Material touch feedback expected | Android | LOW | `InkWell` provides built-in ripples |
| **Light/Dark Theme** | Users expect theme switching | Both | LOW | `FluentThemeData` / `ColorScheme.fromSeed()` |
| **Responsive Grid** | Gallery adapts to window size | Both | MEDIUM | `LayoutBuilder` + `SliverGridDelegateWithMaxCrossAxisExtent` |
| **Back Button** | Android hardware/gesture back | Android | LOW | `WillPopScope` / `PopScope` for handling |
| **ContentDialog** | Standard Windows confirmation dialogs | Windows | LOW | `ContentDialog` with actions |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Platform | Complexity | Notes |
|---------|-------------------|----------|------------|-------|
| **Anime-style Theming** | Soft pink-purple color scheme appeals to target audience | Both | LOW | `ColorScheme.fromSeed(seedColor: Color(0xFFED79B5))` for pink theme |
| **Dual-Platform Native Feel** | Windows feels like Windows, Android feels like Android | Both | HIGH | Platform-adaptive widgets, not lowest-common-denominator |
| **Adaptive Navigation** | Automatically switches NavigationRail/NavigationBar | Both | MEDIUM | 600px breakpoint, smooth transition |
| **CommandBar Actions** | Windows File Explorer-style toolbar | Windows | MEDIUM | Primary and secondary items with overflow |
| **Keyboard Gallery Navigation** | Arrow keys for browsing, space for select | Windows | MEDIUM | Focus management + Shortcuts widget |
| **Extended NavigationRail** | Shows labels on tablets in landscape | Android | LOW | `extended: true` based on width |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Custom Touch Gestures** | Want unique interactions | Conflicts with platform conventions, confusing users | Use standard `GestureDetector` patterns, Material `InkWell` |
| **Platform-Identical UI** | "Same experience everywhere" | Ignores platform conventions, feels foreign on both | Platform-adaptive widgets that feel native on each |
| **Over-Animated Transitions** | Want "smooth" experience | Performance issues, motion sickness, slow interactions | Subtle animations, respect system animation settings |
| **Custom Navigation Patterns** | Want unique navigation | Users can't transfer muscle memory, increases learning curve | Follow Fluent/Material navigation patterns |
| **Hamburger Menu Everywhere** | Want simple navigation | Hides options, requires extra tap, poor on large screens | Use NavigationRail on large screens, NavigationBar on mobile |


---

## Feature Dependencies

```
Responsive Layout System
    └──requires──> LayoutBuilder/MediaQuery measurement
    └──enables──> Adaptive Navigation
                       └──requires──> NavigationRail (large) + NavigationBar (small)

Keyboard Shortcuts
    └──requires──> Focus Management (Focus widget)
    └──requires──> Shortcuts widget + Actions widget
    └──requires──> LogicalKeySet for key combinations

Anime Theming
    └──requires──> FluentThemeData.accentColor (Windows)
    └──requires──> ColorScheme.fromSeed(seedColor) (Android)
    └──requires──> Light/dark mode support

Hover Effects
    └──requires──> MouseRegion widget
    └──requires──> State management for hover state
    └──conflicts──> Touch-only interactions (need separate paths)

Touch Interactions
    └──requires──> InkWell for ripples (Android)
    └──requires──> GestureDetector for gestures
    └──conflicts──> Mouse-only interactions (need fallback)

CommandBar
    └──requires──> ScaffoldPage.header with PageHeader
    └──enables──> Toolbar actions in page header
    └──enables──> Overflow menu for secondary actions
```

### Dependency Notes

- **Responsive Layout enables Adaptive Navigation:** Cannot switch between NavigationRail and NavigationBar without first measuring screen width
- **Keyboard Shortcuts require Focus:** Shortcuts only fire when the widget tree has focus; wrap in `Focus(autofocus: true)`
- **Hover Effects conflict with Touch:** `MouseRegion.onEnter/onExit` don't fire on touch devices; need separate interaction handlers
- **Anime Theming affects both platforms:** Must define consistent accent color across FluentThemeData and ColorScheme

---

## Platform-Specific Components

### Windows Fluent UI Components

| Component | Purpose | When to Use |
|-----------|---------|-------------|
| `NavigationView` | Main app navigation structure | Always for main navigation |
| `NavigationPane` | Side navigation panel | With NavigationView |
| `PaneItem` | Navigation item with icon + title | NavigationPane.items |
| `CommandBar` | Toolbar with primary/secondary actions | Page headers |
| `CommandBarButton` | Action button in CommandBar | CommandBar.primaryItems |
| `ScaffoldPage` | Page layout with header | All content pages |
| `PageHeader` | Page title + CommandBar | ScaffoldPage.header |
| `ContentDialog` | Modal dialog | Confirmations, forms |
| `InfoBar` | Toast notifications | Success/error feedback |
| `Expander` | Collapsible sections | Settings panels |
| `FluentThemeData` | Theme configuration | FluentApp.theme |

**Display Modes (PaneDisplayMode):**
- `auto` - Automatically switches based on width (recommended)
- `expanded` - Full pane with icons + labels
- `compact` - Collapsed, icons only
- `minimal` - Hidden, hamburger menu
- `top` - Horizontal navigation bar

### Android Material 3 Components

| Component | Purpose | When to Use |
|-----------|---------|-------------|
| `NavigationBar` | Bottom navigation bar | Main navigation (width < 600px) |
| `NavigationDestination` | Navigation item | NavigationBar.destinations |
| `NavigationRail` | Side navigation rail | Large screens (width >= 600px) |
| `NavigationRailDestination` | Rail navigation item | NavigationRail.destinations |
| `InkWell` | Touch ripple container | All tappable items |
| `ColorScheme.fromSeed()` | Generate theme from color | Theme configuration |
| `ThemeData(useMaterial3: true)` | Material 3 theme | MaterialApp.theme |

**NavigationRail Types:**
- `labelType: NavigationRailLabelType.all` - Always show labels
- `labelType: NavigationRailLabelType.selected` - Show selected label only
- `extended: true` - Full width with labels always visible

---

## Responsive Design Patterns

### Breakpoint Strategy

| Width Range | Navigation | Grid Columns | Notes |
|-------------|------------|--------------|-------|
| 0-599px | NavigationBar | 2-3 columns | Phone portrait |
| 600-839px | NavigationRail (compact) | 4-5 columns | Phone landscape, small tablet |
| 840+px | NavigationRail (extended) | 6+ columns | Tablet, desktop |

### Implementation Pattern

```dart
// Responsive navigation switch
LayoutBuilder(
  builder: (context, constraints) {
    final isLargeScreen = constraints.maxWidth >= 600;
    
    if (isLargeScreen) {
      // Windows: NavigationView with PaneDisplayMode.auto
      // Android: NavigationRail (extended if >= 840px)
      return NavigationRail(...);
    } else {
      // Android: NavigationBar
      return NavigationBar(...);
    }
  },
)

// Responsive grid
SliverGridDelegateWithMaxCrossAxisExtent(
  maxCrossAxisExtent: 200, // Item max width
  mainAxisSpacing: 8,
  crossAxisSpacing: 8,
)
```

---

## Keyboard Shortcuts Reference

### Common Desktop Shortcuts

| Shortcut | Action | Implementation |
|----------|--------|----------------|
| `Ctrl+N` | New item | `SingleActivator(LogicalKeyboardKey.keyN, control: true)` |
| `Delete` | Delete selected | `SingleActivator(LogicalKeyboardKey.delete)` |
| `Ctrl+A` | Select all | `SingleActivator(LogicalKeyboardKey.keyA, control: true)` |
| `Escape` | Cancel/close | `SingleActivator(LogicalKeyboardKey.escape)` |
| `Arrow Keys` | Navigate grid | Focus traversal |
| `Space` | Select/toggle | `SingleActivator(LogicalKeyboardKey.space)` |

### Implementation Pattern

```dart
Shortcuts(
  shortcuts: const <ShortcutActivator, Intent>{
    SingleActivator(LogicalKeyboardKey.keyN, control: true): CreateNewItemIntent(),
    SingleActivator(LogicalKeyboardKey.delete): DeleteSelectedIntent(),
  },
  child: Actions(
    actions: <Type, Action<Intent>>{
      CreateNewItemIntent: CallbackAction<CreateNewItemIntent>(
        onInvoke: (intent) => _createNewItem(),
      ),
      DeleteSelectedIntent: CallbackAction<DeleteSelectedIntent>(
        onInvoke: (intent) => _deleteSelected(),
      ),
    },
    child: Focus(autofocus: true, child: content),
  ),
)
```

---

## Theming: Anime-Style Soft Pink-Purple

### Color Scheme Definition

```dart
// Material 3 (Android)
final lightTheme = ThemeData(
  useMaterial3: true,
  colorScheme: ColorScheme.fromSeed(
    seedColor: const Color(0xFFED79B5), // Soft pink
    brightness: Brightness.light,
  ),
);

final darkTheme = ThemeData(
  useMaterial3: true,
  colorScheme: ColorScheme.fromSeed(
    seedColor: const Color(0xFFD3BBFF), // Soft purple for dark
    brightness: Brightness.dark,
  ),
);

// Fluent UI (Windows)
final fluentLightTheme = FluentThemeData(
  brightness: Brightness.light,
  accentColor: Colors.pink.toAccentColor(),
);

final fluentDarkTheme = FluentThemeData(
  brightness: Brightness.dark,
  accentColor: Colors.purple.toAccentColor(),
);
```

### Accent Color Options

| Theme | Light Seed | Dark Seed | Notes |
|-------|-----------|-----------|-------|
| Pink | `Color(0xFFED79B5)` | `Color(0xFFED79B5)` | Primary recommendation |
| Purple | `Color(0xFF6F43C0)` | `Color(0xFFD3BBFF)` | Alternative |
| Lavender | `Color(0xFFB39DDB)` | `Color(0xFF9575CD)` | Subtle option |

---

## MVP Definition

### Launch With (v2.0)

Minimum viable product — what's needed to validate the dual-platform UI.

- [x] **NavigationView (Windows)** — Core Windows navigation pattern
- [x] **NavigationBar (Android)** — Core Android navigation pattern
- [x] **Responsive Layout** — Adapts to screen size
- [x] **Adaptive Navigation** — Rail/Bar switch at breakpoint
- [x] **Anime Theming** — Pink-purple color scheme
- [x] **Light/Dark Theme** — Theme switching support

### Add After Validation (v2.x)

Features to add once core is working.

- [ ] **Keyboard Shortcuts** — Desktop efficiency (trigger: positive Windows feedback)
- [ ] **CommandBar Actions** — Toolbar for page actions (trigger: need for bulk operations)
- [ ] **Extended NavigationRail** — Labels on tablets (trigger: tablet user feedback)
- [ ] **Hover Effects** — Desktop interaction refinement (trigger: desktop usability testing)

### Future Consideration (v3+)

Features to defer until product-market fit is established.

- [ ] **Custom Touch Gestures** — Advanced mobile interactions (defer: standard patterns sufficient)
- [ ] **Animated Transitions** — Screen transition animations (defer: performance priority)
- [ ] **Keyboard Gallery Navigation** — Arrow key browsing (defer: complex focus management)


---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| NavigationView (Windows) | HIGH | MEDIUM | P1 |
| NavigationBar (Android) | HIGH | LOW | P1 |
| Responsive Layout | HIGH | MEDIUM | P1 |
| Adaptive Navigation | HIGH | MEDIUM | P1 |
| Anime Theming | MEDIUM | LOW | P1 |
| Light/Dark Theme | HIGH | LOW | P1 |
| Keyboard Shortcuts | MEDIUM | MEDIUM | P2 |
| CommandBar | MEDIUM | MEDIUM | P2 |
| Hover Effects | LOW | LOW | P2 |
| Extended NavigationRail | LOW | LOW | P3 |

**Priority key:**
- P1: Must have for launch — core platform UI patterns
- P2: Should have, add when possible — desktop refinements
- P3: Nice to have, future consideration — polish features

---

## Integration with Existing Features

### Dependencies on Existing v1.0 Features

| New Feature | Depends On Existing | Integration Notes |
|-------------|---------------------|-------------------|
| NavigationView | Page routing, state management | Shared Provider/BLoC state |
| NavigationBar | Page routing, state management | Same navigation logic as Web |
| Responsive Grid | Image gallery, pagination | Same data fetching, different layout |
| Theming | Theme preferences | Extend existing theme system |
| Keyboard Shortcuts | Selection, batch operations | Connect to existing action handlers |

### Code Reuse Strategy

```
Shared Layer:
├── State Management (Provider/BLoC)
├── API Services
├── Data Models
├── Business Logic
└── Theme Configuration (colors, constants)

Platform-Specific Layer:
├── Windows: fluent_ui widgets
├── Android: Material 3 widgets
└── Web: Existing implementation
```

---

## Sources

### High Confidence (Context7 + Official Docs)

- Flutter Official Documentation: https://docs.flutter.dev/ui/adaptive-responsive/general
- Flutter Material 3 Migration: https://docs.flutter.dev/release/breaking-changes/material-3-migration
- Flutter Actions & Shortcuts: https://docs.flutter.dev/ui/interactivity/actions-and-shortcuts
- fluent_ui Package: https://context7.com/bdlukaa/fluent_ui
- Microsoft Fluent UI: https://developer.microsoft.com/en-us/fluentui

### Medium Confidence (GitHub Examples)

- Immich App Theme: https://github.com/immich-app/immich (ColorScheme.fromSeed usage)
- PixEz Flutter: https://github.com/Notsfsssf/pixez-flutter (NavigationView with PaneDisplayMode.auto)
- fluent_ui Examples: https://github.com/bdlukaa/fluent_ui/tree/master/example

### Real-World Patterns Verified

- `ColorScheme.fromSeed()` for Material 3 theming — widely adopted
- `PaneDisplayMode.auto` for responsive NavigationView — recommended pattern
- 600px breakpoint for NavigationRail/NavigationBar — Material guideline
- `LayoutBuilder` for responsive layouts — Flutter standard approach

---

*Feature research for: Windows Fluent UI + Android Material 3 dual-platform Flutter app*
*Researched: 2026-03-20*
