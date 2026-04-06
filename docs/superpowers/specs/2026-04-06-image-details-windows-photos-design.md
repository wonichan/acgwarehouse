# Image Details Windows Photos Style Design

## Goal

Rework the desktop image details experience so it feels closer to the Windows Photos metadata panel, especially in dark mode, and remove the current white-card-on-dark-background mismatch.

## Problem Summary

The current implementation breaks visual consistency in dark mode:

- `flutter_app/lib/widgets/image_metadata_panel.dart` forces a light panel surface and white cards.
- `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart` hardcodes a light gray sidebar and white metadata card.
- `flutter_app/lib/screens/image_detail_screen.dart` uses rounded panel containers that feel more like generic app cards than a Windows-style information pane.

This creates an unfinished look when the app is in dark mode.

## Design Intent

Target the visual feel of the Windows Photos right-side information panel without doing a pixel-perfect clone.

The approved direction is:

- darker, flatter, more native-feeling information pane
- edge-aligned panel instead of a floating white card
- thin dividers instead of strong card borders and shadows
- restrained hierarchy like a property inspector
- one shared visual language for both the details page and viewer sidebar

## Scope

In scope:

- desktop image details page metadata area
- viewer metadata sidebar
- shared metadata panel styling and layout tokens
- dark mode correctness for metadata surfaces, text, inputs, and status pills

Out of scope:

- changing backend metadata fields
- redesigning the main image viewport interaction model
- changing tag workflows beyond visual treatment
- mobile-first redesign

## UX Principles

1. No white surfaces in dark mode.
2. Metadata should read like an information pane, not a stack of settings cards.
3. Details page and viewer sidebar must feel like the same product surface.
4. Information density should increase slightly, but readability must remain strong.
5. Hierarchy should come from spacing, typography, and subtle surface separation, not heavy decoration.

## Visual Model

### 1. Pane Structure

Use a right-side information pane pattern:

- the metadata area is a dedicated side pane
- the pane is separated from the image region with a thin divider
- the pane itself is mostly flat
- large floating card treatment should be reduced or removed

For the dedicated image details page:

- keep the two-column desktop layout
- left side remains the image viewer
- right side becomes a proper information pane rather than a rounded card block

For the viewer sidebar:

- keep the fixed-width sidebar model
- use the same pane surface, divider, spacing, and typography as the details page

### 2. Surface Hierarchy

Use a three-level dark surface hierarchy:

- **App background**: darkest overall background
- **Info pane background**: slightly elevated from the app background
- **Section surface**: only slightly distinct from the pane, or transparent with dividers

Rules:

- do not use pure white or near-white container backgrounds in dark mode
- do not rely on drop shadows for separation
- prefer low-contrast borders and spacing over prominent cards

### 3. Corners and Borders

Adopt a more native, restrained feel:

- reduce large-radius card feel
- pane outer shape can stay subtly rounded where required by current layout shell
- inner sections should use weak radius or none at all
- borders should be thin, low contrast, and sparse

### 4. Typography Hierarchy

Use a property-panel hierarchy:

- pane title: compact, stable, not oversized
- field label: smaller and lower-contrast
- field value: brighter, more prominent, wraps when needed
- helper text: smallest and muted

This should feel closer to:

- label
- value

than to a dense admin table.

### 5. Grouping Strategy

Use vertical sections with subtle separation.

Recommended order:

1. Basic information
2. Date / import info
3. Size / resolution / format
4. Source / path
5. Tags
6. AI tags

Grouping rules:

- avoid strong boxy section headers
- separate groups primarily with vertical spacing and optional dividers
- avoid many nested cards

### 6. Metadata Row Pattern

Rows should be optimized for scanning:

- label on top or narrow left column depending on width
- value visually emphasized
- long values such as paths can wrap to multiple lines
- row spacing remains compact and consistent

Desktop target:

- compact rows
- clear label/value distinction
- no heavy grid lines

### 7. Interactive Elements Inside the Pane

The pane includes active controls in the AI tag area and tag management area, so dark mode support must cover more than just backgrounds.

Adjust:

- buttons
- status chips
- switches
- text fields
- input borders
- inline helper text

These controls should inherit theme-aware colors and visually belong to the same dark pane.

## Component-Level Design Decisions

### `flutter_app/lib/widgets/image_metadata_panel.dart`

This becomes the main shared presentation container for metadata-related content.

Changes:

- remove hardcoded light panel assumptions
- remove `Colors.white` card surfaces
- stop forcing `Color(0xFFF3F3F3)` panel styling
- introduce theme-derived pane and section colors
- render metadata, tags, and AI tag sections with one shared design language

Design role:

- provide section spacing
- provide section wrappers/dividers
- keep the panel visually unified

### `flutter_app/lib/screens/viewer/viewer_metadata_sidebar.dart`

This should become a thin-shell host for the shared pane.

Changes:

- replace hardcoded light panel background with theme-aware surface
- keep fixed width behavior
- use a left divider to separate sidebar from image viewer
- remove white inner card appearance from metadata section

Design role:

- host a native-feeling right info pane
- visually match the details page pane

### `flutter_app/lib/screens/image_detail_screen.dart`

This screen should stop presenting metadata as a floating rounded card.

Changes:

- keep desktop split layout
- make the right column read as an information pane
- reduce oversized card styling on the metadata side
- keep the image side more content-oriented, metadata side more utility-oriented

Design role:

- dedicated details page with the same pane language as the viewer sidebar

## Theme Strategy

All colors must be derived from `Theme.of(context).colorScheme` or directly-related theme tokens.

Required behavior:

- dark mode: no hardcoded light surfaces
- light mode: keep good contrast and avoid over-darkening
- future theme changes should automatically propagate through shared theme logic

Avoid:

- `Colors.white` for containers
- hardcoded light gray surfaces for the pane
- ad hoc foreground estimation tied to a forced light background

## Acceptance Criteria

The redesign is successful when:

1. Dark mode no longer shows white metadata cards or white pane backgrounds.
2. The viewer sidebar and image details page feel visually unified.
3. The metadata area reads like a Windows-style information pane rather than stacked cards.
4. Field labels, values, helper text, and controls all remain readable in dark mode.
5. The visual change improves aesthetics without altering core metadata workflows.

## Implementation Notes for Planning

When turning this into an implementation plan, favor:

- a shared pane style/token approach
- small, focused visual changes
- reuse between details page and viewer sidebar
- minimal behavioral refactors

When choosing between behavior changes and visual fixes, prefer visual fixes unless required for component reuse.

## Risks

- Over-preserving current rounded-card layout may weaken the Windows-native feel.
- Over-correcting into a fully custom layout may increase implementation scope unnecessarily.
- If styles are changed independently in two screens, the UI will drift again.

## Recommended Direction

Implement one shared metadata pane aesthetic and apply it consistently across:

- image details page
- viewer metadata sidebar
- metadata, tags, and AI-tag subsections

This gives the highest visual payoff while staying tightly scoped to the user’s complaint: the current details experience looks wrong in dark mode and should feel more like Windows Photos.
