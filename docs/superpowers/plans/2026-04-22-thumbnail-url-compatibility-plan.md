# Thumbnail URL Compatibility Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make thumbnail URL resolution work consistently for MinIO and Tencent COS across runtime manifest generation, frontend display, and AI tagging payload creation.

**Architecture:** Keep provider-specific branching entirely in the Go backend by teaching `ResolveThumbnailBaseURL` to normalize provider names and return the correct base URL for the configured storage backend. Preserve the existing frontend contract of “absolute URL passthrough, relative path + `thumbnail_base_url` join”, and reuse the same backend thumbnail-resolution contract for AI image payloads so UI and AI do not diverge.

**Tech Stack:** Go 1.23, Gin, SQLite, Flutter, Flutter test, Go test

---

## File Structure

- Modify: `internal/service/thumbnail_path.go`
  - Central helper for provider-aware thumbnail base URL resolution and raw thumbnail URL joining.
- Create: `internal/service/thumbnail_path_test.go`
  - Dedicated unit tests for provider normalization, COS/MinIO base URL rules, and fallback behavior.
- Modify: `internal/app/runtime_manifest_test.go`
  - Lock runtime manifest behavior when `thumbnail_base_url` is present, COS-shaped, or intentionally empty.
- Modify: `internal/service/ai_image_source_test.go`
  - Lock AI thumbnail-path resolution for absolute URLs, COS-relative URLs, and empty/invalid base URL fallback.
- Modify: `internal/handler/ai_tag_handler_test.go`
  - Verify the manual single-image AI trigger writes the correct payload path under both normal and degraded thumbnail-base conditions.
- Modify: `internal/service/ai_backfill_service_test.go`
  - Verify AI backfill payload generation uses the shared thumbnail contract, including degraded base URL handling.
- Modify: `internal/service/ai_tag_auto_scheduler_test.go`
  - Verify scheduler-generated AI payloads use the same thumbnail contract.
- Modify: `flutter_app/test/config/api_config_test.dart`
  - Lock frontend URL joining behavior for MinIO, COS, absolute URLs, and empty base URLs.
- Modify: `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart`
  - Lock frontend runtime manifest parsing for COS-style `thumbnail_base_url` values.

## Chunk 1: Backend provider resolution and shared contract

### Task 1: Add failing tests for provider-aware thumbnail base resolution

**Files:**
- Create: `internal/service/thumbnail_path_test.go`
- Modify: `internal/service/thumbnail_path.go`

- [ ] **Step 1: Write the failing test for COS base URL resolution**

```go
func TestResolveThumbnailBaseURLUsesCOSBucketURL(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		ThumbnailStorageProvider: "cos",
		COS: config.COSConfig{BucketURL: "https://acg-1250000000.cos.ap-shanghai.myqcloud.com/"},
	}

	got := ResolveThumbnailBaseURL(cfg)
	if got != "https://acg-1250000000.cos.ap-shanghai.myqcloud.com" {
		t.Fatalf("ResolveThumbnailBaseURL() = %q", got)
	}
}
```

- [ ] **Step 2: Run the targeted Go test and verify it fails**

Run: `go test ./internal/service/... -run TestResolveThumbnailBaseURLUsesCOSBucketURL -v`

Expected: FAIL because `ResolveThumbnailBaseURL` still ignores `ThumbnailStorageProvider` and reads MinIO only.

- [ ] **Step 3: Add the remaining failing helper tests before implementation**

Add tests in the same file for:

```go
func TestResolveThumbnailBaseURLNormalizesProviderCase(t *testing.T) {}
func TestResolveThumbnailBaseURLUsesMinIOEndpoint(t *testing.T) {}
func TestResolveThumbnailBaseURLReturnsEmptyForEmptyProvider(t *testing.T) {}
func TestResolveThumbnailBaseURLReturnsEmptyForNilConfig(t *testing.T) {}
func TestResolveThumbnailBaseURLReturnsEmptyForInvalidCOSBucketURL(t *testing.T) {}
func TestResolveThumbnailBaseURLReturnsEmptyForInvalidMinIOEndpoint(t *testing.T) {}
func TestResolveThumbnailBaseURLReturnsEmptyForUnknownProvider(t *testing.T) {}
func TestResolveThumbnailURLReturnsRelativePathWhenBaseURLIsEmpty(t *testing.T) {}
func TestResolveThumbnailURLReturnsRelativePathWhenBaseURLIsInvalid(t *testing.T) {}
func TestResolveThumbnailURLReturnsEmptyWhenRawURLIsEmpty(t *testing.T) {}
func TestResolveThumbnailURLKeepsAbsoluteURLWithoutRewriting(t *testing.T) {}
```

- [ ] **Step 4: Run the BaseURL helper tests and verify the new assertions fail for the expected reason**

Run: `go test ./internal/service/... -run TestResolveThumbnailBaseURL -v`

Expected: FAIL with mismatches around provider handling, not syntax errors.

- [ ] **Step 5: Run the URL-joining helper tests and verify the new assertions fail for the expected reason**

Run: `go test ./internal/service/... -run TestResolveThumbnailURL -v`

Expected: FAIL with mismatches around provider handling and fallback behavior, not syntax errors.

- [ ] **Step 6: Implement the minimal provider-aware resolution logic**

Update `internal/service/thumbnail_path.go` to:

```go
func ResolveThumbnailBaseURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.ThumbnailStorageProvider))
	switch provider {
	case "minio":
		base := BuildThumbnailBaseURL(cfg.Minio.Endpoint, cfg.Minio.UseSSL)
		if parsed, err := url.Parse(base); err != nil || parsed == nil || !parsed.IsAbs() {
			return ""
		}
		return base
	case "cos":
		raw := strings.TrimRight(strings.TrimSpace(cfg.COS.BucketURL), "/")
		parsed, err := url.Parse(raw)
		if err != nil || parsed == nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ""
		}
		return raw
	default:
		return ""
	}
}
```

- [ ] **Step 7: Run the BaseURL helper tests and verify they pass**

Run: `go test ./internal/service/... -run TestResolveThumbnailBaseURL -v`

Expected: PASS for the BaseURL helper tests.

- [ ] **Step 8: Run the URL-joining helper tests and verify they pass**

Run: `go test ./internal/service/... -run TestResolveThumbnailURL -v`

Expected: PASS for the URL helper tests.

- [ ] **Step 9: Commit the helper-only slice**

```bash
git add internal/service/thumbnail_path.go internal/service/thumbnail_path_test.go
git commit -m "fix: resolve thumbnail base url by storage provider"
```

### Task 2: Add runtime manifest regression tests around empty and COS thumbnail base URLs

**Files:**
- Modify: `internal/app/runtime_manifest_test.go`
- Modify: `internal/app/runtime_manifest.go`

- [ ] **Step 1: Add the runtime manifest regression test for COS-shaped thumbnail base URLs**

```go
func TestBuildRuntimeManifestPayloadPreservesCOSBaseURL(t *testing.T) {
	t.Parallel()
	payload, err := BuildRuntimeManifestPayload(
		"http://127.0.0.1:51423",
		"https://acg-1250000000.cos.ap-shanghai.myqcloud.com/",
		"admin",
		"secret",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("BuildRuntimeManifestPayload() error = %v", err)
	}
	if payload.Go.ThumbnailBaseURL != "https://acg-1250000000.cos.ap-shanghai.myqcloud.com" {
		t.Fatalf("ThumbnailBaseURL = %q", payload.Go.ThumbnailBaseURL)
	}
}
```

- [ ] **Step 2: Add the runtime manifest regression test for empty thumbnail base URLs**

```go
func TestBuildRuntimeManifestPayloadAllowsEmptyThumbnailBaseURL(t *testing.T) {
	t.Parallel()
	payload, err := BuildRuntimeManifestPayload("http://127.0.0.1:51423", "", "", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("BuildRuntimeManifestPayload() error = %v", err)
	}
	if payload.Go.ThumbnailBaseURL != "" {
		t.Fatalf("ThumbnailBaseURL = %q, want empty", payload.Go.ThumbnailBaseURL)
	}
}
```

- [ ] **Step 3: Run the COS runtime manifest regression test immediately**

Run: `go test ./internal/app/... -run TestBuildRuntimeManifestPayloadPreservesCOSBaseURL -v`

Expected: PASS if current behavior already matches; FAIL only if the test exposed a real mismatch.

- [ ] **Step 4: Run the empty-thumbnail-base regression test immediately**

Run: `go test ./internal/app/... -run TestBuildRuntimeManifestPayloadAllowsEmptyThumbnailBaseURL -v`

Expected: PASS if current behavior already matches; FAIL only if the test exposed a real mismatch.

- [ ] **Step 5: Make the smallest code adjustment only if a test exposed a mismatch**

Keep `internal/app/runtime_manifest.go` unchanged unless the tests prove a behavioral gap. If needed, only normalize `thumbnailBaseURL` via existing trim logic—do not redesign the manifest format.

- [ ] **Step 6: Re-run the full runtime manifest payload test group**

Run: `go test ./internal/app/... -run TestBuildRuntimeManifestPayload -v`

Expected: PASS.

- [ ] **Step 7: Commit the manifest slice**

```bash
git add internal/app/runtime_manifest_test.go internal/app/runtime_manifest.go
git commit -m "test: lock runtime manifest thumbnail base url behavior"
```

## Chunk 2: AI payload generation uses the same thumbnail contract

### Task 3: Add shared AI image-source regression tests before touching AI entry points

**Files:**
- Modify: `internal/service/ai_image_source_test.go`
- Modify: `internal/service/ai_image_source.go`

- [ ] **Step 1: Add the AI image-source regression test for COS-relative thumbnail paths**

```go
func TestResolveAITagImagePathBuildsCOSURLFromRelativeThumbnailPath(t *testing.T) {
	t.Parallel()
	image := &domain.Image{
		Path:              "/images/original.png",
		ThumbnailLargeUrl: "thumbnails/example-large.jpg",
	}
	got := ResolveAITagImagePath(image, "https://acg-1250000000.cos.ap-shanghai.myqcloud.com")
	want := "https://acg-1250000000.cos.ap-shanghai.myqcloud.com/thumbnails/example-large.jpg"
	if got != want {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the COS-relative AI image-source test immediately**

Run: `go test ./internal/service/... -run TestResolveAITagImagePathBuildsCOSURLFromRelativeThumbnailPath -v`

Expected: PASS if `ResolveThumbnailURL` already enforces the contract; otherwise FAIL with a real mismatch.

- [ ] **Step 3: Add the AI image-source regression test for degraded thumbnail base URLs**

```go
func TestResolveAITagImagePathKeepsRelativeThumbnailWhenBaseURLIsEmpty(t *testing.T) {
	t.Parallel()
	image := &domain.Image{ThumbnailLargeUrl: "acg/thumbnails/20260419/example-large.jpg"}
	got := ResolveAITagImagePath(image, "")
	if got != image.ThumbnailLargeUrl {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, image.ThumbnailLargeUrl)
	}
}
```

- [ ] **Step 4: Run the degraded-base AI image-source test immediately**

Run: `go test ./internal/service/... -run TestResolveAITagImagePathKeepsRelativeThumbnailWhenBaseURLIsEmpty -v`

Expected: PASS if `ResolveThumbnailURL` already enforces the contract; otherwise FAIL with a real mismatch.

- [ ] **Step 5: Apply the minimal change only if tests proved a gap**

If the tests already pass through `ResolveThumbnailURL`, do not touch `internal/service/ai_image_source.go`. If they fail, keep the implementation minimal and continue routing through `ResolveThumbnailURL`.

- [ ] **Step 6: Re-run the AI image-source test group**

Run: `go test ./internal/service/... -run TestResolveAITagImagePath -v`

Expected: PASS.

- [ ] **Step 7: Commit the shared AI image-source slice**

```bash
git add internal/service/ai_image_source_test.go internal/service/ai_image_source.go
git commit -m "test: lock ai thumbnail path resolution contract"
```

### Task 4: Extend AI entry-point tests so payload generation cannot drift

**Files:**
- Modify: `internal/handler/ai_tag_handler_test.go`
- Modify: `internal/handler/ai_tag_handler.go`
- Modify: `internal/service/ai_backfill_service_test.go`
- Modify: `internal/service/ai_backfill_service.go`
- Modify: `internal/service/ai_tag_auto_scheduler_test.go`
- Modify: `internal/service/ai_tag_auto_scheduler.go`

- [ ] **Step 1: Write the failing handler test for degraded base URL fallback**

```go
func TestAITagTriggerKeepsRelativeThumbnailPathWhenBaseURLIsUnavailable(t *testing.T) {
	t.Parallel()
	router, repos := newAITagHandlerTestRouterWithConfig(t, &config.Config{
		ThumbnailStorageProvider: "cos",
		COS: config.COSConfig{BucketURL: "not-a-url"},
	})
	if _, err := repos.db.Exec(`UPDATE images SET thumbnail_large_url = ? WHERE id = 1`, "acg/thumbnails/20260419/1-large.jpg"); err != nil {
		t.Fatalf("seed large thumbnail url: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/1/ai-tags", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	var resp struct{ JobIDs []int64 `json:"job_ids"` }
	decodeAIJSONBody(t, w.Body.Bytes(), &resp)
	job, err := repos.jobRepo.FindByID(resp.JobIDs[0])
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	var payload worker.AITagPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.Path != "acg/thumbnails/20260419/1-large.jpg" {
		t.Fatalf("payload.Path = %q, want original relative path", payload.Path)
	}
}
```

- [ ] **Step 2: Write the failing backfill test for degraded base URL fallback**

```go
func TestExecuteBackfillKeepsRelativeThumbnailPathWhenBaseURLIsUnavailable(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "ai-backfill-invalid-base.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}
	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	taskRepo := repository.NewPlatformTaskRepository(db)
	batchRepo := repository.NewTaskBatchRepository(db)
	taskPlatformSvc := NewTaskPlatformService(batchRepo, taskRepo, jobRepo)
	svc := NewAIBackfillService(imageRepo, taskPlatformSvc, nil, func() *config.Config {
		return &config.Config{ThumbnailStorageProvider: "cos", COS: config.COSConfig{BucketURL: "not-a-url"}}
	})
	image := &domain.Image{Path: "/images/original.png", Filename: "original.png", SourceRoot: "/images", FileSize: 1024, Width: 100, Height: 100, Format: "png", PHash: 1, ThumbnailLargeUrl: "acg/thumbnails/20260419/original-large.jpg", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := imageRepo.SaveImage(image); err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	result, err := svc.ExecuteBackfill(context.Background(), repository.BackfillCandidateFilter{HasTags: boolPtr(false)}, "")
	if err != nil {
		t.Fatalf("ExecuteBackfill() error = %v", err)
	}
	jobs, err := jobRepo.FindByPlatformTaskID(result.CreatedTaskList[0].ID)
	if err != nil {
		t.Fatalf("FindByPlatformTaskID() error = %v", err)
	}
	var payload struct{ Path string `json:"path"` }
	if err := json.Unmarshal([]byte(jobs[0].Payload), &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.Path != "acg/thumbnails/20260419/original-large.jpg" {
		t.Fatalf("payload.Path = %q, want original relative path", payload.Path)
	}
}
```

- [ ] **Step 3: Write the failing scheduler test for degraded base URL fallback**

```go
func TestAITagAutoSchedulerKeepsRelativeThumbnailPathWhenBaseURLIsUnavailable(t *testing.T) {
	t.Parallel()
	cfg := schedulerTestConfig()
	cfg.ThumbnailStorageProvider = "cos"
	cfg.COS = config.COSConfig{BucketURL: "not-a-url"}
	images := []domain.Image{{
		ID:                1,
		Path:              "/images/1.png",
		ThumbnailLargeUrl: "acg/thumbnails/20260419/1-large.jpg",
		SourceRoot:        "/images",
		FileSize:          10,
	}}
	finder := &fakeAITagImageFinder{images: images}
	platform := &fakeAITagTaskPlatform{planResult: &TaskPlatformPlanResult{CreatedTasks: []domain.PlatformTask{{ID: 11, ImageID: 1}}}}
	scheduler := NewAITagAutoScheduler(finder, platform, cfg)
	queued, err := scheduler.ScanAndEnqueue(context.Background())
	if err != nil {
		t.Fatalf("ScanAndEnqueue() error = %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1", queued)
	}
	var payload struct{ Path string `json:"path"` }
	if err := json.Unmarshal([]byte(platform.queuedPayloads[0]), &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.Path != "acg/thumbnails/20260419/1-large.jpg" {
		t.Fatalf("payload.Path = %q, want original relative path", payload.Path)
	}
}
```

- [ ] **Step 4: Run the handler entry-point tests to verify each failure is real**

Run: `go test ./internal/handler/... -run TestAITagTrigger -v`

- [ ] **Step 5: Run the backfill entry-point tests to verify each failure is real**

Run: `go test ./internal/service/... -run TestExecuteBackfill -v`

- [ ] **Step 6: Run the scheduler entry-point tests to verify each failure is real**

Run: `go test ./internal/service/... -run TestAITagAutoScheduler -v`

Expected: FAIL only where payload generation does not yet match the shared contract.

- [ ] **Step 7: Implement the smallest fix in production code only if an entry point drift is exposed**

Production code should keep this shape and only change if a test proves drift:

```go
thumbnailBaseURL := ResolveThumbnailBaseURL(h.currentConfig())
payload, err := json.Marshal(worker.AITagPayload{
	ImageID: task.ImageID,
	Path:    service.ResolveAITagImagePath(image, thumbnailBaseURL),
	Prompt:  prompt,
})
```

The same rule applies to `AIBackfillService` and `AITagAutoScheduler`: do not invent custom joining logic.

- [ ] **Step 8: Re-run the handler entry-point tests**

Run: `go test ./internal/handler/... -run TestAITagTrigger -v`

- [ ] **Step 9: Re-run the backfill entry-point tests**

Run: `go test ./internal/service/... -run TestExecuteBackfill -v`

- [ ] **Step 10: Re-run the scheduler entry-point tests**

Run: `go test ./internal/service/... -run TestAITagAutoScheduler -v`

Expected: PASS.

- [ ] **Step 11: Commit the AI entry-point slice**

```bash
git add internal/handler/ai_tag_handler_test.go internal/service/ai_backfill_service_test.go internal/service/ai_tag_auto_scheduler_test.go internal/handler/ai_tag_handler.go internal/service/ai_backfill_service.go internal/service/ai_tag_auto_scheduler.go
git commit -m "test: align ai task payloads with thumbnail url contract"
```

## Chunk 3: Frontend parsing tests stay aligned with backend contract

### Task 5: Add frontend regression tests for URL parsing and manifest loading

**Files:**
- Modify: `flutter_app/test/config/api_config_test.dart`
- Modify: `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart`
- Modify: `flutter_app/lib/config/api_config.dart`
- Modify: `flutter_app/lib/bootstrap/runtime_manifest_loader.dart`

- [ ] **Step 1: Add the frontend regression test for COS base URL joins**

```dart
test('resolveThumbnailUrl builds COS URL from relative path', () {
  expect(
    ApiConfig.resolveThumbnailUrl(
      'thumbnails/example-large.jpg',
      thumbnailBaseUrl: 'https://acg-1250000000.cos.ap-shanghai.myqcloud.com',
    ),
    'https://acg-1250000000.cos.ap-shanghai.myqcloud.com/thumbnails/example-large.jpg',
  );
});
```

- [ ] **Step 2: Add the frontend regression test for empty-base fallback if not already present**

```dart
test('resolveThumbnailUrl keeps relative path when thumbnail base URL is empty', () {
  expect(
    ApiConfig.resolveThumbnailUrl('acg/thumbnails/20260419/example-large.jpg'),
    'acg/thumbnails/20260419/example-large.jpg',
  );
});
```

- [ ] **Step 3: Add the manifest-loading regression test for COS thumbnail base URLs**

```dart
test('loads COS thumbnail_base_url from runtime manifest', () async {
  final loader = RuntimeManifestLoader(
    readText: (_) async => '{"version":1,"generated_at":"2026-04-22T10:00:00Z","go":{"base_url":"http://127.0.0.1:51423","thumbnail_base_url":"https://acg-1250000000.cos.ap-shanghai.myqcloud.com","ready":true}}',
  );
  final result = await loader.load(isDevelopmentMode: false, isDesktopTarget: true);
  expect(result.appliedThumbnailBaseUrl, 'https://acg-1250000000.cos.ap-shanghai.myqcloud.com');
});
```

- [ ] **Step 4: Run the focused Flutter config tests and verify failures are real**

Run (workdir=`flutter_app`): `flutter test test/config/api_config_test.dart`

- [ ] **Step 5: Run the focused Flutter manifest-loader tests and verify failures are real**

Run (workdir=`flutter_app`): `flutter test test/bootstrap/runtime_manifest_loader_test.dart`

Expected: PASS if the behavior already exists; otherwise FAIL with clear assertion mismatches.

- [ ] **Step 6: Make the smallest frontend code change only if a test exposed a mismatch**

Prefer keeping production code unchanged if the tests already pass. If a fix is required, keep it inside:

```dart
// flutter_app/lib/config/api_config.dart
static String? resolveThumbnailUrl(String? rawUrl, {String? thumbnailBaseUrl})

// flutter_app/lib/bootstrap/runtime_manifest_loader.dart
String? _extractThumbnailBaseUrl(String? raw)
```

Do not add provider-specific branches to Flutter widgets or providers.

- [ ] **Step 7: Re-run the Flutter config tests**

Run (workdir=`flutter_app`): `flutter test test/config/api_config_test.dart`

- [ ] **Step 8: Re-run the Flutter manifest-loader tests**

Run (workdir=`flutter_app`): `flutter test test/bootstrap/runtime_manifest_loader_test.dart`

Expected: PASS.

- [ ] **Step 9: Commit the frontend slice**

```bash
git add flutter_app/test/config/api_config_test.dart flutter_app/test/bootstrap/runtime_manifest_loader_test.dart flutter_app/lib/config/api_config.dart flutter_app/lib/bootstrap/runtime_manifest_loader.dart
git commit -m "test: lock frontend thumbnail url compatibility"
```

### Task 6: Run the verification set before shipping implementation work

**Files:**
- Modify: none (verification only)

- [ ] **Step 1: Run the backend-focused regression set**

Run: `go test ./internal/service/... ./internal/handler/... ./internal/app/...`

Expected: PASS.

- [ ] **Step 2: Run the focused Flutter regression set**

Run (workdir=`flutter_app`): `flutter test test/config/api_config_test.dart test/bootstrap/runtime_manifest_loader_test.dart`

Expected: PASS.

- [ ] **Step 3: Run the Go formatter for touched Go files**

Run: `gofmt -w internal/service/thumbnail_path.go internal/service/thumbnail_path_test.go internal/service/ai_image_source_test.go internal/handler/ai_tag_handler_test.go internal/service/ai_backfill_service_test.go internal/service/ai_tag_auto_scheduler_test.go internal/app/runtime_manifest_test.go`

Expected: No output.

- [ ] **Step 4: Run the Flutter formatter only if Flutter production files changed**

Run (workdir=`flutter_app`): `dart format lib/config/api_config.dart lib/bootstrap/runtime_manifest_loader.dart`

Expected: Only formatting diffs if those files changed.

- [ ] **Step 5: Re-run the narrowest affected Go tests after formatting**

Run: `go test ./internal/service/... -run 'TestResolveThumbnail|TestResolveAITagImagePath|TestExecuteBackfill|TestAITagAutoScheduler' -v`

Expected: PASS.

- [ ] **Step 6: Re-run the focused Flutter tests after formatting if any Flutter files changed**

Run (workdir=`flutter_app`): `flutter test test/config/api_config_test.dart test/bootstrap/runtime_manifest_loader_test.dart`

Expected: PASS.

- [ ] **Step 7: Commit the verification/cleanup slice**

```bash
git add internal/service/thumbnail_path.go internal/service/thumbnail_path_test.go internal/service/ai_image_source_test.go internal/handler/ai_tag_handler_test.go internal/service/ai_backfill_service_test.go internal/service/ai_tag_auto_scheduler_test.go internal/app/runtime_manifest_test.go flutter_app/test/config/api_config_test.dart flutter_app/test/bootstrap/runtime_manifest_loader_test.dart
git commit -m "fix: unify thumbnail url resolution across runtime and ai paths"
```
