# Gallery Image Context Menu Design

## Goal

Add a desktop-oriented right-click menu to gallery images so users can open the original file with the Windows default application, add the image to an existing collection or create a new collection and add it immediately, and permanently delete the source image together with its thumbnails.

## Problem Summary

The current gallery supports left-click navigation, long-press selection, and batch operations, but it does not support per-image desktop right-click actions.

Codebase evidence:

- `flutter_app/lib/screens/gallery_screen.dart` renders the gallery and delegates item interaction to `ResponsiveImageGrid`.
- `flutter_app/lib/widgets/responsive_image_grid.dart` supports tap and long-press selection, but no right-click interaction.
- `flutter_app/lib/app/adaptive_app.dart` selects Fluent UI for packaged Windows desktop.
- `flutter_app/lib/app/fluent_screens.dart` wires the active Windows gallery screen to `FluentGalleryContent`.
- `flutter_app/lib/widgets/fluent_gallery_content.dart` renders `FluentImageCard` for grid and masonry items on Windows desktop.
- `flutter_app/lib/services/collection_service.dart` and `flutter_app/lib/providers/collection_provider.dart` already support collection CRUD and add-image flows.
- `internal/repository/image_repository.go` supports deleting the image record, but there is no confirmed backend API for deleting the source file plus thumbnail files from disk.
- No confirmed existing implementation was found for opening a local file with the operating system default app.

This leaves a gap between the current app behavior and expected desktop gallery behavior.

## Approved Direction

The approved direction is:

- add a right-click menu on each gallery image item
- keep the right-click menu to one level only
- make `收藏` open a dialog instead of a nested submenu
- reuse the existing `collections` domain rather than introducing a separate favorites model
- make delete mean permanent deletion of the original file, thumbnail files, database record, and collection relationships
- make `打开源文件` open the original file with the Windows default application rather than the in-app viewer

## Scope

In scope:

- gallery image right-click handling in the Flutter desktop UI
- context menu UI for image-specific actions
- collection-picker dialog for choosing an existing collection
- inline create-collection flow inside the collection-picker dialog
- backend or desktop integration for opening a local file in the operating system
- backend API for permanent delete of original file, thumbnail files, and database record
- gallery refresh and local UI state cleanup after successful actions
- user-visible success and error feedback for all new actions

Out of scope:

- changing the existing left-click image viewer flow
- introducing a dedicated favorites table or new storage model separate from collections
- adding “reveal in Explorer” or “open containing folder”
- redesigning collections outside the new add-to-collection dialog flow
- changing batch operations unless strictly needed for reuse

## UX Model

### 1. Right-Click Menu

Every image tile in the gallery gains a desktop right-click affordance.

The menu contains exactly these actions:

1. `打开源文件`
2. `收藏`
3. `删除源文件及缩略图`

Design rules:

- keep the menu flat rather than introducing a second-level submenu
- the menu should feel lightweight and contextual to the image tile
- existing left-click and long-press behavior must remain unchanged
- selection mode must continue to work as it does today

### 2. Open Source File

`打开源文件` must open the original file on disk using the Windows default application.

Behavior:

- do not route through the current in-app image detail screen or viewer window
- use the image record’s `path` as the source of truth
- if the file does not exist or cannot be opened, surface a clear user-visible error
- successful open should not show an extra success toast

### 3. Add to Collection Flow

`收藏` must open a dialog rather than a nested context menu.

Dialog structure:

- top section: existing collections list
- middle interaction: choose one collection and confirm add
- lower section: create a new collection inline
- create success path: immediately add the current image to the new collection

Design rules:

- reuse the existing collection concept and APIs
- prefer “收藏到合集” semantics in copy, while keeping the underlying model as collections
- keep the flow to one dialog and avoid redirecting users to the collections screen
- if add fails, keep the dialog open and show the failure state
- if create succeeds but add fails, tell the user explicitly that the collection was created but the image was not added

### 4. Permanent Delete Flow

`删除源文件及缩略图` is destructive and irreversible.

Behavior:

- show a strong confirmation dialog before any deletion work begins
- copy must make clear this is permanent deletion, not a soft remove from the gallery
- after confirmation, delete the original image file from disk
- delete any associated thumbnail files from disk
- remove the image record from the database
- rely on existing database and repository semantics to clear collection and tag associations
- remove the image from the current gallery UI immediately after success

## Data and System Design

### Existing Reuse

These parts should be reused rather than replaced:

- `flutter_app/lib/services/collection_service.dart` for collection API calls
- `flutter_app/lib/providers/collection_provider.dart` as an optional reuse point for in-memory collection state and optimistic count updates
- `internal/handler/collection_handler.go`, `internal/service/collection_service.go`, and `internal/repository/collection_repository.go` for collection membership logic
- `internal/repository/image_repository.go` for database record deletion semantics

Provider registration note:

- `flutter_app/lib/main.dart` does not currently register `CollectionProvider` in the root provider tree
- the implementation plan must explicitly choose whether to register `CollectionProvider` globally or keep the new collection dialog self-contained with direct `CollectionService` usage
- this spec requires reuse of the existing collection model and APIs, but it does not require `CollectionProvider` specifically unless registration is added on purpose

### New Capability: Open Local File

The codebase does not currently show a confirmed implementation for opening a local file with the operating system, and `internal/handler/routes.go` does not currently expose an image-open endpoint.

The design therefore requires a new application capability with this contract:

- input: image ID or resolved image path
- resolution: validate the image exists and resolve the original file path from the database record
- action: invoke the Windows shell/default app for the file path
- failure: return a user-displayable error when the file is missing or the launch fails

The implementation plan must choose one explicit seam:

- a Windows Flutter-desktop integration path that opens the file locally, or
- a new backend endpoint dedicated to opening the original file

Implementation should prefer the narrowest possible contract: “open this image’s original file”, not a broad arbitrary-shell execution primitive.

### New Capability: Permanent Image Delete

The current repository supports deleting the database row, but the design needs a higher-level delete workflow.

Required delete contract:

1. resolve the image record from the database
2. attempt to delete the original file from disk
3. resolve thumbnail storage paths from persisted metadata
4. attempt to delete any stored thumbnail files from disk
5. reconcile collection metadata affected by image removal
6. remove the image record from the database
7. return enough information for the UI to remove the image from the current view

Behavior rules:

- if the original file is already missing, the operation may continue as a cleanup path rather than failing outright
- if thumbnail files are already missing, treat that as cleanup rather than a hard failure
- if a true filesystem permission or IO error occurs, stop and return a clear error
- do not return success unless the final persisted state is consistent

Thumbnail-path rule:

- the current model stores `thumbnail_small_url` and `thumbnail_large_url`
- `internal/handler/thumbnail_url_rewrite.go` shows these are request-facing URLs, not proven filesystem paths
- the implementation plan must define a safe mapping from persisted thumbnail metadata to actual thumbnail file paths before file deletion is attempted
- the delete implementation must not assume that a request URL can be deleted directly as a local filesystem path

### Consistency Model

The delete flow should behave as transactionally as practical, even though disk IO and database operations are not one database transaction.

The required consistency priorities are:

1. avoid false-success states
2. prefer explicit failure over partial silent cleanup
3. allow removal of stale database records when files are already gone
4. keep the final database state aligned with disk reality whenever the operation reports success
5. keep denormalized collection metadata aligned after image deletion

Collection consistency rules:

- foreign-key cascade will remove rows from `collection_images` and `image_tags`
- foreign-key cascade alone will not recalculate `collections.image_count`
- the permanent delete workflow must explicitly recalculate or repair `image_count` for affected collections
- `migrations/001_initial_schema.up.sql` defines `collections.cover_image_id` with `ON DELETE SET NULL`
- the implementation plan must choose whether delete leaves null cover images in place or performs an immediate replacement-cover repair for affected collections
- default recommendation: allow `cover_image_id` to become null during deletion, then run an explicit cover-repair step for affected collections where a replacement image exists

## UI Integration Points

### Gallery Tile Integration

Primary attachment point for packaged Windows desktop:

- `flutter_app/lib/widgets/fluent_gallery_content.dart`
- `flutter_app/lib/widgets/fluent_image_card.dart`

Secondary attachment point for the Material gallery path:

- `flutter_app/lib/widgets/responsive_image_grid.dart`

Reasoning:

- packaged Windows desktop uses the Fluent gallery path selected by `AdaptiveApp`, so the feature must be implemented there first
- the Material path should be kept behaviorally aligned if it remains in use outside packaged Windows
- adding right-click behavior at the tile layer keeps the change local to image tiles rather than spreading menu logic through the whole screen

Supporting integration points likely include:

- `flutter_app/lib/app/fluent_screens.dart` for desktop gallery-level refresh hooks or dialog launch behavior
- `flutter_app/lib/screens/gallery_screen.dart` for parity if the Material gallery path remains relevant

### Collection Dialog Integration

The new dialog should be a dedicated reusable widget rather than ad hoc inline code inside the gallery screen.

Its responsibilities:

- load available collections
- render selection UI
- create a collection inline
- call add-to-collection after selection or creation
- report success or actionable error states

This keeps collection selection logic isolated from the gallery grid rendering logic.

The dialog must not assume a globally-available collection provider unless the implementation explicitly adds that provider to app bootstrap.

## Error Handling Model

### Open Source File Errors

- if the image record no longer resolves, show a clear failure message
- if the file path does not exist, show “source file missing or inaccessible” style feedback
- if the OS open call fails, show a specific open failure message

### Collection Flow Errors

- if collection loading fails, keep the dialog open and show retryable feedback
- if add-to-collection fails, do not dismiss the dialog automatically
- if create succeeds but add fails, explicitly communicate the partial outcome

### Delete Errors

- require explicit confirmation before delete starts
- if a filesystem delete fails due to permissions or IO, surface that error and do not report success
- if the file is already missing, continue cleanup rather than blocking the user from removing the stale record
- if the database delete fails after file deletion work, surface the inconsistency and keep the operation visibly failed so it can be investigated or retried
- if collection count repair or cover repair fails, return a failure rather than reporting a clean delete

## Acceptance Criteria

The design is successful when:

1. Right-clicking any gallery image reliably shows the three approved actions.
2. Left-click image viewing and long-press selection continue to work as before.
3. `打开源文件` opens the original image in the Windows default application.
4. If the original file is missing, the user sees a clear error instead of silent failure.
5. `收藏` opens a dialog that allows choosing an existing collection.
6. The same dialog allows creating a new collection and immediately adding the image.
7. Successful add-to-collection gives clear lightweight feedback.
8. `删除源文件及缩略图` requires confirmation and permanently removes the original file, thumbnail files, database row, and collection relationships.
9. After successful delete, the image disappears from the gallery without requiring a full app restart.
10. If the source file is already missing, delete still supports cleaning up the stale gallery record.
11. Affected collections have correct `image_count` values after delete completes.
12. Affected collection covers follow the chosen repair rule consistently after delete completes.
13. New right-click functionality does not break batch operations, sorting, filtering, or existing collection CRUD.

## Testing and Verification Guidance

When this is turned into an implementation plan, verification must cover:

- widget-level interaction for right-click menu invocation and collection dialog behavior
- frontend service/provider tests for collection add and create-then-add flows where practical
- backend tests for permanent delete edge cases, especially missing-file cleanup, thumbnail-path resolution, true filesystem failures, collection count repair, and cover-image repair behavior
- desktop/manual verification on Windows for opening the original file with the default app
- regression coverage for existing gallery tap, long-press, and refresh behavior

## Risks

- The current app appears to be Flutter-first with backend HTTP APIs, so desktop-specific file open behavior may require a new integration seam.
- Permanent delete spans filesystem and database state, so partial-failure handling must be designed carefully.
- If right-click handling is only added to one gallery rendering path, grid and masonry or Fluent and Material variants may drift.
- Reusing collections avoids schema growth, but poor dialog copy could still make the feature feel more like generic collection management than user-facing favorites.

## Recommended Direction

Implement the simplest desktop-friendly model:

- flat right-click menu on image tiles
- collection-picker dialog for `收藏`
- reuse existing collections APIs and state
- add one narrow desktop/open-file capability
- add one high-level permanent-delete capability that coordinates filesystem cleanup and database deletion

This approach matches the approved user experience while minimizing unnecessary new concepts in the codebase.
