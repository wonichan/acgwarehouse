# Gallery Image Context Menu Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Windows desktop gallery right-click actions for opening source files, adding to collections (including create-and-add), and permanently deleting source image plus thumbnails.

**Architecture:** Add two backend image action endpoints (`open-source`, `permanent-delete`) in the existing image handler and wire them through a narrow file-operation service that handles OS open and disk deletion. Add Fluent gallery tile right-click menu + collection dialog in Flutter, reusing existing collection APIs. Keep Material path behavior-compatible and refresh gallery state after successful actions.

**Tech Stack:** Go (Gin, SQLite), Flutter (Fluent UI, Provider), existing collection/image APIs, filesystem + process invocation on Windows.

---

## Chunk 1: Backend image actions and consistency

### Task 1: Add file-operation service and action contracts

**Files:**
- Create: `internal/service/image_file_action_service.go`
- Test: `internal/service/image_file_action_service_test.go`

- [ ] **Step 1: Write failing tests for file action service behavior**

```go
func TestImageFileActionServiceOpenSourceFile(t *testing.T) { /* ... */ }
func TestImageFileActionServiceDeleteHandlesMissingFilesAsCleanup(t *testing.T) { /* ... */ }
func TestImageFileActionServiceDeleteFailsOnPermissionErrors(t *testing.T) { /* ... */ }
func TestImageFileActionServiceResolveThumbnailPaths(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run tests to verify failures**

Run: `go test ./internal/service -run ImageFileActionService -count=1`
Expected: FAIL (service not implemented).

- [ ] **Step 3: Implement minimal service**

```go
type ImageFileActionService struct {
  OpenFile func(path string) error
  RemoveFile func(path string) error
}

func (s *ImageFileActionService) OpenSource(path string) error
func (s *ImageFileActionService) PermanentDelete(path string, thumbSmallURL string, thumbLargeURL string) (DeleteResult, error)
```

Required seam details:
- Implement a thumbnail-deletion abstraction that accepts persisted thumbnail URLs and performs one of:
  - storage-key/object deletion for COS-style URLs, or
  - local file deletion when URL/path maps to local disk.
- Do **not** treat request-facing thumbnail URLs as raw filesystem paths.
- Add unit tests that prove URL parsing + deletion dispatch for both local and remote-style thumbnail values.

- [ ] **Step 4: Run service tests to green**

Run: `go test ./internal/service -run ImageFileActionService -count=1`
Expected: PASS.

### Task 2: Add image action endpoints in handler/routes

**Files:**
- Modify: `internal/handler/image_handler.go`
- Modify: `internal/handler/routes.go`
- Modify: `internal/handler/image_handler_test.go`

- [ ] **Step 1: Write failing handler tests first**

```go
func TestImageHandlerOpenSourceFileReturnsOK(t *testing.T) { /* POST /api/v1/images/:id/open-source */ }
func TestImageHandlerPermanentDeleteRemovesImageRecord(t *testing.T) { /* DELETE /api/v1/images/:id/permanent */ }
func TestImageHandlerPermanentDeleteReturnsConflictOnConsistencyFailure(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run failing handler tests**

Run: `go test ./internal/handler -run "ImageHandler(OpenSource|PermanentDelete)" -count=1`
Expected: FAIL (routes/handlers absent).

- [ ] **Step 3: Implement endpoints and dependency wiring**

```go
// routes
images.POST(":id/open-source", imageOpenSource)
images.DELETE(":id/permanent", imagePermanentDelete)

// handler
func (h *ImageHandler) OpenSourceFile(c *gin.Context)
func (h *ImageHandler) PermanentDeleteImage(c *gin.Context)
```

- [ ] **Step 4: Run handler tests**

Run: `go test ./internal/handler -run "ImageHandler(OpenSource|PermanentDelete)" -count=1`
Expected: PASS.

### Task 3: Keep collection count/cover consistent after image delete

**Files:**
- Modify: `internal/repository/collection_repository.go`
- Modify: `internal/service/collection_service.go`
- Modify: `internal/service/collection_service_test.go` (or create if absent)

- [ ] **Step 1: Add failing tests for collection metadata reconciliation**

```go
func TestCollectionServiceRepairAfterImageDeleteUpdatesImageCount(t *testing.T) { /* ... */ }
func TestCollectionServiceRepairAfterImageDeleteRepairsCover(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run tests to verify failures**

Run: `go test ./internal/service -run CollectionService.*Repair -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement repair helper used by permanent delete flow**

```go
func (s *CollectionService) RepairAfterImageDelete(ctx context.Context, imageID int64) error
```

Required sequencing:
- Add a pre-delete query to collect affected collection IDs (from `collection_images`) before `imageRepo.Delete` executes.
- Repair must operate on captured collection IDs after delete to refresh:
  - `collections.image_count`
  - `collections.cover_image_id` according to chosen rule.
- Add repository support if needed, e.g. `FindCollectionIDsByImage(ctx, imageID)` and `RecountCollectionImageCount(ctx, collectionID)`.

- [ ] **Step 4: Run service tests**

Run: `go test ./internal/service -run CollectionService.*Repair -count=1`
Expected: PASS.

## Chunk 2: Flutter Fluent gallery context menu and collection dialog

### Task 4: Add API methods + provider refresh helper

**Files:**
- Modify: `flutter_app/lib/services/api_service.dart`
- Modify: `flutter_app/lib/providers/image_provider.dart`
- Modify: `flutter_app/test/services/api_service_test.dart`
- Modify: `flutter_app/test/providers/image_provider_test.dart`

- [ ] **Step 1: Add failing tests for new API methods**

```dart
test('openSourceFile posts to image open endpoint', () async { /* ... */ });
test('permanentDeleteImage calls delete endpoint', () async { /* ... */ });
```

- [ ] **Step 2: Run failing Flutter tests**

Run: `flutter test test/services/api_service_test.dart --plain-name "openSourceFile"`
Expected: FAIL.

- [ ] **Step 3: Implement API + provider removal helper**

```dart
Future<void> openImageSourceFile(int imageId)
Future<void> permanentDeleteImage(int imageId)
void removeImageById(int imageId)
```

- [ ] **Step 4: Run targeted tests**

Run: `flutter test test/services/api_service_test.dart`
Expected: PASS.

### Task 5: Add collection picker dialog (choose existing or create-and-add)

**Files:**
- Create: `flutter_app/lib/widgets/image_collection_picker_dialog.dart`
- Modify: `flutter_app/test/widgets/fluent_gallery_content_test.dart`
- Create: `flutter_app/test/widgets/image_collection_picker_dialog_test.dart`

- [ ] **Step 1: Write failing widget tests for dialog behavior**

```dart
testWidgets('shows collection list and adds image to selected collection', (tester) async { /* ... */ });
testWidgets('create collection then add image in one flow', (tester) async { /* ... */ });
```

- [ ] **Step 2: Run failing widget tests**

Run: `flutter test test/widgets/image_collection_picker_dialog_test.dart`
Expected: FAIL.

- [ ] **Step 3: Implement dialog widget with collection service reuse**

UI feedback requirements in this task:
- Keep dialog open on collection load/add failure and show inline actionable error text.
- For create-success/add-fail, show explicit partial-result message.
- On success, show lightweight user feedback and close dialog.

- [ ] **Step 4: Re-run dialog tests**

Run: `flutter test test/widgets/image_collection_picker_dialog_test.dart`
Expected: PASS.

### Task 6: Add right-click menu to Fluent gallery image tiles

**Files:**
- Modify: `flutter_app/lib/widgets/fluent_image_card.dart`
- Modify: `flutter_app/lib/widgets/fluent_gallery_content.dart`
- Modify: `flutter_app/lib/app/fluent_screens.dart` (only if needed for refresh hooks)
- Modify: `flutter_app/test/widgets/fluent_image_card_test.dart`
- Modify: `flutter_app/test/widgets/fluent_gallery_content_test.dart`

- [ ] **Step 1: Write failing tests for right-click menu actions**

```dart
testWidgets('secondary click shows context actions', (tester) async { /* ... */ });
testWidgets('delete action removes image from provider list on success', (tester) async { /* ... */ });
```

- [ ] **Step 2: Run failing tests**

Run: `flutter test test/widgets/fluent_gallery_content_test.dart --plain-name "secondary click"`
Expected: FAIL.

- [ ] **Step 3: Implement context menu + action handlers**

Actions:
- 打开源文件 -> `ApiService.openImageSourceFile`
- 收藏 -> show `ImageCollectionPickerDialog`
- 删除源文件及缩略图 -> confirm dialog + `ApiService.permanentDeleteImage`

UI feedback requirements in this task:
- Open-source failure must show user-visible error.
- Permanent delete failure must show user-visible error and preserve current tile state.
- Permanent delete success must remove tile and show lightweight success feedback.

- [ ] **Step 4: Run fluent widget tests**

Run: `flutter test test/widgets/fluent_image_card_test.dart test/widgets/fluent_gallery_content_test.dart`
Expected: PASS.

## Chunk 3: End-to-end verification and regression

### Task 7: Backend + Flutter verification sweep

**Files:**
- Verify only (no new files)

- [ ] **Step 1: Run Go tests for touched packages**

Run: `go test ./internal/handler ./internal/service ./internal/repository -count=1`
Expected: PASS.

- [ ] **Step 2: Run Flutter targeted tests**

Run: `flutter test test/widgets/fluent_image_card_test.dart test/widgets/fluent_gallery_content_test.dart test/widgets/image_collection_picker_dialog_test.dart test/services/api_service_test.dart`
Expected: PASS.

- [ ] **Step 3: Run Flutter analyzer for touched files**

Run: `flutter analyze`
Expected: No new errors from changed files.

- [ ] **Step 4: Manual desktop acceptance on Windows build/runtime**

Checklist:
- Right-click menu appears on gallery image tiles.
- Open source file launches default Windows app.
- 收藏 dialog supports select existing + create-and-add.
- Permanent delete removes file, thumbnail artifacts, DB record, and UI tile.
- Existing gallery double-click/open behaviors remain intact.
