# Image Detail Left Panel and Tag Card Layout Design

## Goal

Improve the image detail page information panel so it feels more structured and polished, while making the tag card easier to scan and manage during day-to-day tagging work.

## Problem Summary

The current image detail left panel works functionally, but the visual hierarchy is weak:

- `flutter_app/lib/screens/image_detail_screen.dart` uses a desktop split layout with a narrow left metadata column beside a large image preview area, and also reuses the same metadata panel in compact layout.
- `flutter_app/lib/widgets/image_metadata_panel.dart` stacks three sections in one scrollable column: basic metadata, AI tag actions, and manual tags.
- The AI card and tag card use the same section shell, but their internal layouts do not communicate different priorities clearly.
- The path field is visually heavy and can dominate the metadata card.
- The manual tag area already separates `已确认 / 待确认 / 已拒绝`, but all three states still live inside one card with limited subsection emphasis, which makes it harder to answer three immediate questions quickly: what needs review, what is already retained, and what was rejected.

This leaves the left panel feeling like a generic form stack instead of a clear information-and-action sidebar.

## Approved Direction

The approved direction is a **layered card sidebar**:

- keep the existing three-part sidebar model rather than fully redesigning the page shell
- strengthen the vertical sequence as **Overview → AI Analysis → Tags**
- make each card responsible for one clear job
- keep the visual language restrained and product-like rather than turning the panel into a heavy admin console
- redesign the tag card around state-based grouping: **Pending / Accepted / Rejected**

This is the “balanced” option: clearer than the current layout, but not overly minimal or overly tool-dense.

## Scope

In scope:

- internal metadata panel layout used by both desktop and compact image detail layouts, with desktop visual rhythm treated as the primary target
- internal layout and hierarchy of the basic info card
- internal layout and hierarchy of the AI tag card
- internal layout and grouping rules of the manual tag card
- spacing, typography, and section emphasis within the left sidebar

Out of scope:

- changing backend metadata fields or tag APIs
- changing the right-side image preview region
- changing core tag behaviors such as confirm, reject, merge, edit, or remove
- introducing a mobile-specific redesign
- redesigning the entire app theme system

## Codebase Evidence

- `flutter_app/lib/widgets/image_metadata_panel.dart` renders the left panel as a `SingleChildScrollView` with three vertically stacked sections: metadata, AI, and tags.
- `_buildAITagSection` places title, generate button, status pill, custom prompt toggle, prompt editor, and helper text into a single card.
- `_buildTagsSection` renders manual tag management inside one card using wrapped tag chips.
- `flutter_app/lib/widgets/image_metadata_pane_theme.dart` already provides a shared section decoration, so the redesign can stay structurally consistent with the existing component model.

## UX Principles

1. The metadata pane should read as a **control and information sidebar**, not as equal-weight cards fighting for attention.
2. Users should understand the scan order immediately: file overview first, AI generation second, tags last.
3. The most actionable tag state should appear first.
4. Long metadata such as file paths should be visually contained so they stop breaking rhythm.
5. Rejected tags should remain available, but they should not compete visually with accepted or pending tags.
6. The redesign should favor clarity and polish over novelty.

## Layout Model

### 1. Left Sidebar Structure

Keep the current narrow desktop sidebar, but make the internal rhythm more deliberate.

Recommended vertical order:

1. **Basic Information Card**
2. **AI Analysis Card**
3. **Tags Card**

Card rules:

- keep one consistent outer shell style across all three cards
- increase the sense of separation through spacing and header hierarchy, not radically different card styles
- each card should have a compact header area and a clearly bounded content area
- maintain the current scrollable column behavior

### 2. Basic Information Card

This card should become a compact overview block instead of a loose stack of long text rows.

Current field set from `_buildMetadataSection` should be preserved, but reordered into a clearer scan model.

Required field set:

1. file name
2. dimensions
3. format
4. file size
5. import time
6. path

Recommended presentation order:

1. file name
2. dimensions + format
3. file size + import time
4. path

Layout rules:

- file name remains visible and should not be removed from the pane
- dimensions and format may be presented more compactly than today
- file size and import time should share the same compact metadata rhythm
- use a tighter label/value presentation for short fields like size and time
- keep the path on its own row rather than forcing it into the same rhythm as short metadata
- default path display is **single-line truncated text** with ellipsis
- provide a lightweight copy affordance for the full path at the row end
- full path access should come from copy and tooltip/hover title behavior, not from expanding the card height by default
- avoid making path text the strongest visual element in the card

### 3. AI Analysis Card

This card should communicate one primary action: generate or regenerate AI tags.

Header structure:

- left: icon + title
- right: primary action button
- optional inline status appears near the action, but should remain secondary to the button

Body structure:

- the custom prompt setting becomes a secondary option rather than the visual center of the card
- helper text is shortened to one concise line
- prompt editing is shown only when explicitly enabled

State rules:

- default state: title + primary button + optional helper text only
- when `_useCustomPrompt == true`: show the text field inline below the toggle row
- when `_isAITriggered == true`: keep the status visible in the header area, but do not let it replace the card title or dominate the button area
- the generate/regenerate action remains the strongest control in all non-error states

Design intent:

- the card should feel actionable, not verbose
- the primary button should remain the strongest element in the card
- the custom prompt should feel available, but not mandatory

### 4. Tags Card

This card becomes the main working surface in the sidebar.

Header structure:

- left: title
- optional near-title badge or inline count for total tags
- right: add-tag action

Body structure must group tags by meaning instead of mixing all chips together.

Recommended order:

1. **Pending**
2. **Accepted** (`confirmed` in code)
3. **Rejected**

Why this order:

- pending is the highest-attention state because it usually needs a decision
- accepted represents the final useful output
- rejected is historical context and should be visually quieter

## Tag Card Grouping Model

### 1. Section Behavior

Each tag state is rendered as its own subsection inside the card.

Each subsection contains:

- compact subsection label
- item count
- chip flow area

Subsection rules:

- show the subsection only when it has content, except where empty-state messaging is helpful
- rejected tags remain visible by default in this approved direction; they should be visually deemphasized rather than collapsed
- pending should never be collapsed by default
- if all three groups are empty, keep the existing simple empty state equivalent to `暂无标签`

### 2. Visual Emphasis

Emphasis order:

1. pending
2. accepted (`confirmed` in provider state)
3. rejected

How to express that hierarchy:

- pending uses the strongest local contrast and clearest subtitle treatment
- accepted uses normal chip styling and feels stable
- rejected uses muted text, weaker borders, or softer fill so it recedes

### 3. Chip Layout Rules

Tag chips should remain in wrap layout, but the card should feel denser and more organized.

Rules:

- tighten vertical spacing between chip rows slightly
- keep chip spacing consistent across all sections
- avoid over-wide empty gaps that make the card look sparse
- use subsection boundaries to create readability instead of relying on one uninterrupted chip cloud

### 4. Reading Flow

After the redesign, a user should be able to answer these questions in one scan:

1. Are there tags waiting for review?
2. Which tags are actually retained on the image?
3. Which suggestions were rejected?

If the layout cannot answer those three questions quickly, it has failed.

## Typography and Spacing

### Header Hierarchy

- card title: medium emphasis, stable across all cards
- subsection label: smaller and quieter than card title
- metadata label: muted
- metadata value: stronger than label, but not oversized
- helper copy: smallest and most muted text in the card

### Spacing Strategy

- use slightly larger spacing between cards than between elements inside a card
- keep card headers compact
- use subsection spacing inside the tag card to establish hierarchy cleanly
- prefer shorter helper copy over large empty blocks

## Interaction Notes

### Basic Information

- path should support copy without making the card visually busier than necessary
- long path display should favor truncation plus access to the full value over permanent multiline dominance

### AI Analysis

- “自定义提示词” should be treated as an advanced option
- turning it on may reveal the text field inline, but the default state should remain compact
- AI status should not displace the primary action visually

### Tags

- existing chip behaviors for confirm, reject, edit, merge, and remove should remain intact
- the redesign is about organization and emphasis, not about introducing a new tag workflow

## Implementation Seams to Preserve

The later implementation plan should treat these functions as the main change points:

- `_buildMetadataSection` in `flutter_app/lib/screens/image_detail_screen.dart`
- `_buildAITagSection` in `flutter_app/lib/widgets/image_metadata_panel.dart`
- `_buildTagsSection` in `flutter_app/lib/widgets/image_metadata_panel.dart`

These behaviors must remain unchanged unless explicitly re-approved:

- provider loading and polling logic
- AI trigger behavior
- tag confirm / reject / merge / edit / remove actions
- overall reuse of `ImageMetadataPanel` in both desktop and compact layouts

## Acceptance Criteria

The redesign is successful when:

1. The metadata pane reads as a clear three-step sidebar: overview, AI, tags.
2. Long path content no longer dominates the first card.
3. The AI card feels more compact and action-led.
4. The tag card clearly separates pending, accepted (`confirmed`), and rejected states.
5. Rejected tags remain accessible but visually deemphasized.
6. The redesign can be implemented inside the existing `ImageMetadataPanel` and `_buildMetadataSection` structure without unnecessary behavioral refactors.

## Risks

- Over-minimizing the sidebar could hide useful metadata or make actions feel too subtle.
- Over-tooling the tag card could make the page feel like an admin panel instead of an image details surface.
- If path handling is not carefully designed, the top card may still feel visually noisy.
- If state grouping is applied inconsistently, the chip layout may become more fragmented rather than clearer.

## Implementation Notes for Planning

When this moves into implementation planning, prefer:

- small structural changes inside `ImageMetadataPanel`
- minimal disruption to existing provider logic and tag actions
- reuse of the current section container model
- theme-aware spacing and emphasis changes rather than one-off styling hacks

The implementation should focus first on hierarchy and grouping, then on polish details.

## Recommended Direction Summary

Implement a balanced, layered sidebar where:

- the basic information card becomes more compact and path-aware
- the AI analysis card centers the generate action and demotes advanced prompt editing
- the tags card becomes a grouped review surface ordered by pending, accepted, then rejected

This keeps the current architecture intact while making the image detail left panel noticeably clearer and more intentional.
