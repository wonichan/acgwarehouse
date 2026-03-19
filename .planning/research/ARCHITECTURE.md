# Architecture Research

**Domain:** Dual-framework Flutter app (Windows Fluent UI + Android Material 3)
**Researched:** 2026-03-20
**Confidence:** HIGH
**Context:** v2.0 UI/UX重构与多端适配 — Adding Windows desktop and Android mobile UI to existing Flutter Web app

## Summary

This research focuses specifically on **dual-framework Flutter architecture** for integrating Windows Fluent UI (`fluent_ui`) alongside existing Material 3 Android/Web UI. The key architectural decision is maintaining a **shared business logic layer** while having **platform-specific presentation layers**.

**Core Principle:** UI frameworks change, business logic stays the same. Providers, services, and models remain unchanged. Only the presentation layer bifurcates into platform-specific implementations.

---

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              PRESENTATION LAYER                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────┐  ┌───────────────────────┐  ┌─────────────────┐  │
│  │    Windows UI Shell   │  │    Android UI Shell   │  │    Web UI Shell │  │
│  │    (fluent_ui)        │  │    (Material 3)       │  │    (Material 3) │  │
│  │                       │  │                       │  │                 │  │
│  │  - NavigationView     │  │  - NavigationBar      │  │  - Same as      │  │
│  │  - NavigationPane     │  │  - NavigationRail     │  │    Android      │  │
│  │  - ScaffoldPage       │  │  - Scaffold           │  │                 │  │
│  │  - FluentThemeData    │  │  - ThemeData          │  │                 │  │
│  └───────────┬───────────┘  └───────────┬───────────┘  └────────┬────────┘  │
│              │                          │                       │           │
│              └──────────────────────────┴───────────────────────┘           │
│                                         │                                   │
├─────────────────────────────────────────┴───────────────────────────────────┤
│                           PLATFORM ABSTRACTION LAYER                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Adaptive Navigation Shell                       │    │
│  │  - Platform detection (defaultTargetPlatform)                       │    │
│  │  - Responsive layout (LayoutBuilder)                                │    │
│  │  - Screen size breakpoints (600px threshold)                        │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                         │                                   │
├─────────────────────────────────────────┴───────────────────────────────────┤
│                           SHARED BUSINESS LOGIC                              │
│                           (EXISTING - UNCHANGED)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │    Providers     │  │    Services      │  │     Models       │          │
│  │                  │  │                  │  │                  │          │
│  │  - ImageListProv │  │  - ApiService    │  │  - ImageModel    │          │
│  │  - TagProvider   │  │  - TagService    │  │  - Tag           │          │
│  │  - SearchProvider│  │  - SearchService │  │  - Collection    │          │
│  │  - etc.          │  │  - etc.          │  │  - etc.          │          │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘          │
│                                         │                                   │
├─────────────────────────────────────────┴───────────────────────────────────┤
│                              DATA LAYER (UNCHANGED)                          │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Go Backend REST API (port 8080)                 │    │
│  │  - /api/images    - /api/tags    - /api/search    - /api/collections│    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `lib/ui/windows/` | Windows Fluent UI screens and widgets | fluent_ui NavigationView, ScaffoldPage |
| `lib/ui/android/` | Android Material 3 screens and widgets | Material Scaffold, NavigationBar |
| `lib/ui/web/` | Web Material 3 screens (shared with Android) | Material Scaffold, responsive layout |
| `lib/ui/shared/` | Platform-agnostic widgets (images, cards) | Custom widgets using theme abstraction |
| `lib/core/` | Navigation, theme, platform detection | Adaptive shell, theme provider |
| `lib/providers/` | State management (EXISTING - unchanged) | Provider ChangeNotifier |
| `lib/services/` | API and business logic (EXISTING - unchanged) | HTTP services |
| `lib/models/` | Data models (EXISTING - unchanged) | Immutable data classes |

---

## Recommended Project Structure

### New Directory Structure

```
flutter_app/lib/
├── main.dart                    # Entry point with platform-aware app shell
├── app.dart                     # AdaptiveApp widget (MaterialApp/FluentApp)
│
├── core/                        # NEW: Cross-platform infrastructure
│   ├── navigation/              # Navigation shell and routing
│   │   ├── adaptive_shell.dart  # Platform-aware navigation shell
│   │   ├── router.dart          # Optional: go_router configuration
│   │   └── routes.dart          # Route definitions
│   ├── theme/                   # Theme management
│   │   ├── app_theme.dart       # Shared theme constants (purple accent)
│   │   ├── fluent_theme.dart    # Windows Fluent theme config
│   │   └── material_theme.dart  # Material 3 theme config
│   └── platform/                # Platform utilities
│       └── platform_info.dart   # Platform detection helpers
│
├── ui/                          # NEW: Platform-specific UI layers
│   ├── windows/                 # Windows Fluent UI
│   │   ├── screens/             # Windows-specific screens
│   │   │   ├── gallery_screen.dart
│   │   │   ├── search_screen.dart
│   │   │   ├── duplicate_screen.dart
│   │   │   └── settings_screen.dart
│   │   ├── widgets/             # Windows-specific widgets
│   │   │   └── windows_image_grid.dart
│   │   └── shell.dart           # NavigationView shell
│   │
│   ├── android/                 # Android Material 3 UI
│   │   ├── screens/             # Android-specific screens
│   │   │   ├── gallery_screen.dart
│   │   │   ├── search_screen.dart
│   │   │   └── duplicate_screen.dart
│   │   ├── widgets/             # Android-specific widgets
│   │   └── shell.dart           # Scaffold with NavigationBar/Rail
│   │
│   ├── web/                     # Web UI (can extend android/)
│   │   └── shell.dart           # Web-specific shell
│   │
│   └── shared/                  # Shared UI components
│       ├── widgets/             # Platform-agnostic widgets
│       │   ├── image_card.dart  # Works with both themes
│       │   ├── tag_chip.dart    # Themed via abstraction
│       │   └── loading_indicator.dart
│       └── screens/             # Complex shared screens (detail views)
│           └── image_detail_screen.dart
│
├── providers/                   # EXISTING - State management (unchanged)
│   ├── image_provider.dart
│   ├── tag_provider.dart
│   ├── search_provider.dart
│   ├── selection_provider.dart
│   ├── collection_provider.dart
│   └── duplicate_provider.dart
│
├── services/                    # EXISTING - Business logic (unchanged)
│   ├── api_service.dart
│   ├── tag_service.dart
│   ├── search_service.dart
│   ├── batch_service.dart
│   ├── collection_service.dart
│   └── duplicate_service.dart
│
├── models/                      # EXISTING - Data models (unchanged)
│   ├── image.dart
│   ├── tag.dart
│   ├── tag_alias.dart
│   └── collection.dart
│
├── widgets/                     # EXISTING - Can migrate to ui/shared/ over time
│   ├── image_grid.dart
│   ├── image_masonry.dart
│   ├── tag_filter_drawer.dart
│   └── ...
│
├── screens/                     # EXISTING - To be replaced by ui/android/ screens
│   └── ...
│
└── config/                      # EXISTING - Configuration (unchanged)
    └── api_config.dart
```

### Structure Rationale

| Directory | Rationale |
|-----------|-----------|
| `ui/windows/` | Isolated fluent_ui code prevents Material imports, avoiding widget naming conflicts |
| `ui/android/` | Material 3 specific implementation for mobile; responsive patterns for tablets |
| `ui/shared/` | Reusable components that work with both themes via abstraction layer |
| `core/` | Infrastructure that both UI layers depend on; navigation, themes, platform detection |
| `providers/` at root | Kept unchanged to minimize migration effort; works identically with both UI frameworks |
| `services/` at root | Stateless HTTP clients; no UI framework dependency |

---

## Architectural Patterns

### Pattern 1: Platform-Aware App Shell

**What:** Single entry point that detects platform and renders appropriate UI framework  
**When to use:** When supporting multiple UI frameworks in one codebase  
**Trade-offs:** Adds complexity at entry point, but isolates platform differences cleanly

```dart
// lib/app.dart
import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'dart:io' show Platform;
import 'ui/windows/shell.dart' as windows;
import 'ui/android/shell.dart' as android;

class AdaptiveApp extends StatelessWidget {
  const AdaptiveApp({super.key});

  @override
  Widget build(BuildContext context) {
    // Platform detection
    if (kIsWeb) {
      return const MaterialAppShell(); // Web uses Material
    }
    
    if (Platform.isWindows) {
      return const FluentAppShell(); // Windows uses Fluent
    }
    
    // Android/iOS use Material
    return const MaterialAppShell();
  }
}

// Windows shell
class FluentAppShell extends StatelessWidget {
  const FluentAppShell({super.key});
  
  @override
  Widget build(BuildContext context) {
    return FluentApp(
      title: 'ACGWarehouse',
      theme: FluentThemeData(
        accentColor: Colors.purple, // Anime-style purple
        brightness: Brightness.light,
      ),
      darkTheme: FluentThemeData(
        accentColor: Colors.purple,
        brightness: Brightness.dark,
      ),
      themeMode: ThemeMode.system,
      home: const windows.WindowsNavigationShell(),
    );
  }
}

// Android/Mobile shell
class MaterialAppShell extends StatelessWidget {
  const MaterialAppShell({super.key});
  
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ACGWarehouse',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.purple, // Match Windows accent
          brightness: Brightness.light,
        ),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.purple,
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
      ),
      themeMode: ThemeMode.system,
      home: const android.AndroidNavigationShell(),
    );
  }
}
```

### Pattern 2: Adaptive Navigation (Material)

**What:** Navigation component that switches between NavigationRail and NavigationBar based on screen width  
**When to use:** Responsive layouts that adapt to phone vs tablet/desktop  
**Trade-offs:** Two navigation implementations, but optimal UX per form factor

```dart
// lib/ui/android/shell.dart
import 'package:flutter/material.dart';

class AndroidNavigationShell extends StatefulWidget {
  const AndroidNavigationShell({super.key});

  @override
  State<AndroidNavigationShell> createState() => _AndroidNavigationShellState();
}

class _AndroidNavigationShellState extends State<AndroidNavigationShell> {
  int _selectedIndex = 0;

  // Platform-specific screen imports
  static final List<Widget> _screens = [
    const GalleryScreen(),      // from ui/android/screens/
    const SearchScreen(),
    const DuplicateScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final isLargeScreen = constraints.maxWidth >= 600;

        if (isLargeScreen) {
          // Tablet/Large screen: NavigationRail on left
          return Scaffold(
            body: Row(
              children: [
                NavigationRail(
                  selectedIndex: _selectedIndex,
                  onDestinationSelected: (index) {
                    setState(() => _selectedIndex = index);
                  },
                  labelType: NavigationRailLabelType.all,
                  destinations: const [
                    NavigationRailDestination(
                      icon: Icon(Icons.photo_library_outlined),
                      selectedIcon: Icon(Icons.photo_library),
                      label: Text('图库'),
                    ),
                    NavigationRailDestination(
                      icon: Icon(Icons.search_outlined),
                      selectedIcon: Icon(Icons.search),
                      label: Text('搜索'),
                    ),
                    NavigationRailDestination(
                      icon: Icon(Icons.content_copy_outlined),
                      selectedIcon: Icon(Icons.content_copy),
                      label: Text('重复检测'),
                    ),
                  ],
                ),
                const VerticalDivider(thickness: 1, width: 1),
                Expanded(child: _screens[_selectedIndex]),
              ],
            ),
          );
        }

        // Phone: NavigationBar at bottom
        return Scaffold(
          body: _screens[_selectedIndex],
          bottomNavigationBar: NavigationBar(
            selectedIndex: _selectedIndex,
            onDestinationSelected: (index) {
              setState(() => _selectedIndex = index);
            },
            destinations: const [
              NavigationDestination(
                icon: Icon(Icons.photo_library_outlined),
                selectedIcon: Icon(Icons.photo_library),
                label: '图库',
              ),
              NavigationDestination(
                icon: Icon(Icons.search_outlined),
                selectedIcon: Icon(Icons.search),
                label: '搜索',
              ),
              NavigationDestination(
                icon: Icon(Icons.content_copy_outlined),
                selectedIcon: Icon(Icons.content_copy),
                label: '重复检测',
              ),
            ],
          ),
        );
      },
    );
  }
}
```

### Pattern 3: Windows Navigation (Fluent UI)

**What:** NavigationView with adaptive pane display mode  
**When to use:** Windows desktop applications  
**Trade-offs:** Windows-specific, but native feel with auto-adapting layout

```dart
// lib/ui/windows/shell.dart
import 'package:fluent_ui/fluent_ui.dart';

class WindowsNavigationShell extends StatefulWidget {
  const WindowsNavigationShell({super.key});

  @override
  State<WindowsNavigationShell> createState() => _WindowsNavigationShellState();
}

class _WindowsNavigationShellState extends State<WindowsNavigationShell> {
  int _selectedIndex = 0;

  @override
  Widget build(BuildContext context) {
    return NavigationView(
      appBar: NavigationAppBar(
        title: const Text('ACGWarehouse'),
        actions: Row(
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            IconButton(
              icon: const Icon(FluentIcons.settings),
              onPressed: () {
                setState(() => _selectedIndex = 3); // Settings
              },
            ),
          ],
        ),
      ),
      pane: NavigationPane(
        selected: _selectedIndex,
        onChanged: (index) => setState(() => _selectedIndex = index),
        displayMode: PaneDisplayMode.auto, // Auto-adapts: open/compact/minimal
        items: [
          PaneItem(
            icon: const Icon(FluentIcons.photo2),
            title: const Text('图库'),
            body: const GalleryScreen(),
          ),
          PaneItem(
            icon: const Icon(FluentIcons.search),
            title: const Text('搜索'),
            body: const SearchScreen(),
          ),
          PaneItem(
            icon: const Icon(FluentIcons.copy),
            title: const Text('重复检测'),
            body: const DuplicateScreen(),
          ),
        ],
        footerItems: [
          PaneItem(
            icon: const Icon(FluentIcons.settings),
            title: const Text('设置'),
            body: const SettingsScreen(),
          ),
        ],
      ),
    );
  }
}
```

### Pattern 4: Shared Provider Access

**What:** Providers remain at root level, accessed identically from any UI layer  
**When to use:** State management should be framework-agnostic  
**Trade-offs:** None — Provider works with any Flutter widgets

```dart
// main.dart - Provider setup remains UNCHANGED
void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider(create: (_) => ApiService()),
        Provider(create: (_) => TagService()),
        ChangeNotifierProvider(create: (context) => ImageListProvider(context.read<ApiService>())),
        ChangeNotifierProvider(create: (context) => TagProvider(context.read<TagService>())),
        ChangeNotifierProvider(create: (context) => SearchProvider(service: context.read<SearchService>())),
        ChangeNotifierProvider(create: (context) => DuplicateProvider(service: context.read<DuplicateService>())),
        ChangeNotifierProvider(create: (context) => CollectionProvider(service: context.read<CollectionService>())),
        ChangeNotifierProvider(create: (_) => SelectionProvider()),
      ],
      child: const AdaptiveApp(), // NEW: Platform-aware shell
    );
  }
}

// Usage is IDENTICAL in Windows and Android screens
class GalleryScreen extends StatelessWidget {
  const GalleryScreen({super.key});
  
  @override
  Widget build(BuildContext context) {
    // Same Consumer pattern works with both fluent_ui and Material widgets
    return Consumer<ImageListProvider>(
      builder: (context, provider, child) {
        if (provider.isLoading && provider.images.isEmpty) {
          // Windows: ProgressRing, Android: CircularProgressIndicator
          return const Center(child: CircularProgressIndicator());
        }
        
        // Build grid using shared data
        return GridView.builder(
          itemCount: provider.images.length,
          itemBuilder: (context, index) {
            final image = provider.images[index];
            return ImageCard(image: image); // Shared widget
          },
        );
      },
    );
  }
}
```

### Pattern 5: Theme Abstraction for Shared Widgets

**What:** Shared widgets detect current theme and adapt styling  
**When to use:** For reusable components like cards, chips, lists  
**Trade-offs:** Limited to properties both frameworks support

```dart
// lib/ui/shared/widgets/image_card.dart
import 'package:flutter/material.dart' as material;
import 'package:fluent_ui/fluent_ui.dart' as fluent;
import '../../models/image.dart';

class ImageCard extends StatelessWidget {
  final ImageModel image;
  final VoidCallback? onTap;
  
  const ImageCard({super.key, required this.image, this.onTap});
  
  @override
  Widget build(BuildContext context) {
    // Detect current UI framework
    if (fluent.FluentTheme.maybeOf(context) != null) {
      return _buildFluentCard(context);
    }
    return _buildMaterialCard(context);
  }
  
  Widget _buildMaterialCard(BuildContext context) {
    return material.Card(
      clipBehavior: material.Clip.antiAlias,
      child: material.InkWell(
        onTap: onTap,
        child: material.Column(
          crossAxisAlignment: material.CrossAxisAlignment.stretch,
          children: [
            material.Expanded(
              child: material.CachedNetworkImage(
                imageUrl: image.thumbnailPath,
                fit: material.BoxFit.cover,
              ),
            ),
            material.Padding(
              padding: const material.EdgeInsets.all(8.0),
              child: material.Text(
                image.filename,
                maxLines: 1,
                overflow: material.TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ),
    );
  }
  
  Widget _buildFluentCard(BuildContext context) {
    return fluent.Card(
      padding: fluent.EdgeInsets.zero,
      child: fluent.Button(
        style: fluent.ButtonState.all(fluent.ButtonStyle(
          padding: fluent.ButtonState.all(fluent.EdgeInsets.zero),
        )),
        onPressed: onTap,
        child: fluent.Column(
          crossAxisAlignment: fluent.CrossAxisAlignment.stretch,
          children: [
            fluent.Expanded(
              child: fluent.CachedNetworkImage(
                imageUrl: image.thumbnailPath,
                fit: fluent.BoxFit.cover,
              ),
            ),
            fluent.Padding(
              padding: const fluent.EdgeInsets.all(8.0),
              child: fluent.Text(
                image.filename,
                maxLines: 1,
                overflow: fluent.TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
```

---

## Data Flow

### State Management Flow (Unchanged)

```
[MultiProvider (root)]
    ↓ (providers are global, injected in main.dart)
┌───────────────────┬───────────────────┬───────────────────┐
│  Windows Shell    │  Android Shell    │  Web Shell        │
│  (FluentApp)      │  (MaterialApp)    │  (MaterialApp)    │
└─────────┬─────────┴─────────┬─────────┴─────────┬─────────┘
          │                   │                   │
          └───────────────────┴───────────────────┘
                              │
                    Consumer<T>/Provider.of<T>()
                              │
                    [Screen Widget reads/writes state]
                              │
                    [notifyListeners() → UI Rebuild]
```

### Navigation Flow

```
[Shell maintains selectedIndex]
    ↓
[Switches body widget based on index]
    ↓
[Screen Widget builds UI]
    ↓
[Consumer reads from Provider]
    ↓
[User interaction → Provider method call]
    ↓
[notifyListeners() → UI update]
```

---

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Current (single user) | No changes needed; current Provider setup is optimal |
| Large image collections | Already handled with pagination; add virtualization if needed |
| Future multi-user | Add auth layer; user-scoped providers |

### Performance Priorities

1. **First bottleneck:** Image grid with 10k+ images — Already handled with pagination and lazy loading
2. **Second bottleneck:** Memory usage — Use `cached_network_image` (already installed), add `AutomaticKeepAliveClientMixin`

---

## Anti-Patterns

### Anti-Pattern 1: Mixing UI Frameworks in Same Widget

**What people do:** Import both `material.dart` and `fluent_ui.dart` in same file  
**Why it's wrong:** Widget naming conflicts (Card, Button, Icon), theme confusion, larger bundle  
**Do this instead:** Create separate screen files per platform; share only business logic

```dart
// ❌ BAD
import 'package:flutter/material.dart';
import 'package:fluent_ui/fluent_ui.dart';

class MyScreen extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Card(child: Text('Which Card?')); // Conflict!
  }
}

// ✅ GOOD - Separate files per platform
// lib/ui/windows/gallery_screen.dart
import 'package:fluent_ui/fluent_ui.dart';
class WindowsGalleryScreen extends StatelessWidget { ... }

// lib/ui/android/gallery_screen.dart
import 'package:flutter/material.dart';
class AndroidGalleryScreen extends StatelessWidget { ... }
```

### Anti-Pattern 2: Platform Detection in Every Widget

**What people do:** Check `Platform.isWindows` in every build method  
**Why it's wrong:** Scattered logic, harder to maintain, test difficulty  
**Do this instead:** Detect once at app root (AdaptiveApp), route to appropriate shell

### Anti-Pattern 3: Duplicating Business Logic

**What people do:** Copy provider logic between Windows and Android screens  
**Why it's wrong:** Maintenance nightmare, bugs multiply  
**Do this instead:** Keep providers/services at root level; screens only read/write via providers

### Anti-Pattern 4: Using `dart:io` Platform in Web Build

**What people do:** Use `Platform.isWindows` without guarding for web  
**Why it's wrong:** Web builds fail because `dart:io` doesn't exist in browser  
**Do this instead:** Use `kIsWeb` check first, or use `defaultTargetPlatform` from `flutter/foundation.dart`

```dart
// ✅ Correct platform detection
import 'package:flutter/foundation.dart' show kIsWeb, defaultTargetPlatform;

String getPlatformName() {
  if (kIsWeb) return 'web';
  return defaultTargetPlatform.name; // 'android', 'windows', etc.
}
```

---

## Integration Points

### External Services (Unchanged)

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Go Backend (port 8080) | HTTP REST API | Existing ApiService works unchanged |
| AI APIs (Qwen/Doubao) | Via Go backend | No direct Flutter integration needed |

### Internal Boundaries (Unchanged)

| Boundary | Communication | Notes |
|----------|---------------|-------|
| UI Layer ↔ Providers | Provider.of<T>(), Consumer<T> | Same pattern for all UI frameworks |
| Providers ↔ Services | Direct method calls | Services are stateless, inject into providers |
| Services ↔ Backend | HTTP (ApiService) | Existing implementation unchanged |

---

## New Components Required

| Component | Location | Purpose |
|-----------|----------|---------|
| `AdaptiveApp` | `lib/app.dart` | Platform detection and app shell routing |
| `FluentAppShell` | `lib/app.dart` | Windows FluentApp wrapper with theme |
| `MaterialAppShell` | `lib/app.dart` | Android/Web MaterialApp wrapper with theme |
| `WindowsNavigationShell` | `lib/ui/windows/shell.dart` | Fluent NavigationView container |
| `AndroidNavigationShell` | `lib/ui/android/shell.dart` | Material Scaffold with adaptive NavigationBar/Rail |
| `WindowsGalleryScreen` | `lib/ui/windows/screens/` | Fluent-styled gallery |
| `AndroidGalleryScreen` | `lib/ui/android/screens/` | Material-styled gallery |
| `AppTheme` | `lib/core/theme/` | Shared color palette (purple accent) |

## Modified Components

| Component | Change Required |
|-----------|-----------------|
| `main.dart` | Replace `MaterialApp` with `AdaptiveApp` |
| `pubspec.yaml` | Add `fluent_ui: ^4.9.0` dependency |
| Existing screens | Can remain as reference; new screens in `ui/` directories |

---

## Build Configuration

### pubspec.yaml Additions

```yaml
dependencies:
  fluent_ui: ^4.9.0       # Windows Fluent Design
  system_theme: ^3.1.0    # Optional: System accent color on Windows
  window_manager: ^0.4.3  # Optional: Window control on desktop
```

### Platform Build Commands

```bash
# Windows desktop
flutter build windows --release

# Android
flutter build apk --release
flutter build appbundle --release

# Web (existing)
flutter build web --release

# Development
flutter run -d windows    # Windows debug
flutter run -d chrome     # Web debug
flutter run -d <device>   # Android debug
```

---

## Sources

- **Fluent UI Documentation** — https://context7.com/bdlukaa/fluent_ui (HIGH confidence)
- **Flutter Adaptive Layout Guide** — https://docs.flutter.dev/ui/adaptive-responsive/large-screens (HIGH confidence)
- **Flutter Platform Adaptations** — https://docs.flutter.dev/platform-integration/platform-adaptations (HIGH confidence)
- **go_router ShellRoute** — https://pub.dev/documentation/go_router/latest/topics/Configuration-topic (HIGH confidence)
- **Provider Documentation** — https://pub.dev/packages/provider (HIGH confidence)
- **GitHub: Immich Adaptive Navigation** — https://github.com/immich-app/immich (MEDIUM confidence - real-world reference)

---

## Quality Gate Checklist

- [x] Integration points identified (Providers, Services, Backend API)
- [x] New vs modified components explicitly listed
- [x] Build order considers dependencies (Providers/Services first, UI last)
- [x] Platform-specific code isolated in separate directories
- [x] Shared business logic remains unchanged
- [x] Theme consistency across platforms (purple accent)

---

*Architecture research for: ACGWarehouse v2.0 Dual-Platform UI*  
*Researched: 2026-03-20*