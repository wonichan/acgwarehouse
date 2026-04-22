# Thumbnail Storage: Absolute URLs to Relative Paths

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change thumbnail storage from absolute URLs (e.g., `https://cdn.example.com/bucket/thumbnails/img.jpg`) to relative paths (e.g., `thumbnails/20260102/img-small.jpg`) and add batch migration for existing DB rows.

**Architecture:** 
- Services return relative paths from Upload methods
- Handler prepends configured base URL when serving responses
- Migration script converts existing absolute URLs to relative paths by extracting the object key

**Tech Stack:** Go 1.23, SQLite, MinIO, COS, Gin

---

## Task 1: Modify MinIO Service to Return Relative Path

**Files:**
- Modify: `internal/service/minio_service.go:48-73`
- Test: `internal/worker/thumbnail_handler_test.go` (existing, verify behavior)

- [ ] **Step 1: Write failing test for relative path return**

Add test case in `internal/worker/thumbnail_handler_test.go` that expects relative path (e.g., `thumbnails/20260102/test-image-small.jpg`) instead of absolute URL.

```go
func TestThumbnailHandler_UploadReturnsRelativePath(t *testing.T) {
    t.Parallel()
    thumbSvc := &stubThumbnailGenerator{
        small: &domain.Thumbnail{Data: []byte("small-bytes"), Size: "small"},
        large: &domain.Thumbnail{Data: []byte("large-bytes"), Size: "large"},
    }
    cosSvc := &stubThumbnailUploader{}
    repo := &stubThumbnailImageRepo{}

    h := NewThumbnailHandler(thumbSvc, cosSvc, repo)
    err := h.Handle(context.Background(), 1, `{"image_id":1,"path":"C:/tmp/a.png","filename":"test-image"}`)
    if err != nil {
        t.Fatalf("Handle() error = %v", err)
    }

    // Should store relative path, not absolute URL
    if !strings.HasPrefix(repo.smallURL, "thumbnails/") {
        t.Fatalf("small url = %q, want relative path starting with 'thumbnails/'", repo.smallURL)
    }
}
```

Run: `go test ./internal/worker/... -run TestThumbnailHandler_UploadReturnsRelativePath -v`
Expected: FAIL - repository receives absolute URL instead of relative path

- [ ] **Step 2: Modify MinIO Upload to return relative path**

In `internal/service/minio_service.go`, change the return value from `finalURL` to just `key`:

```go
func (s *MinioService) Upload(ctx context.Context, filename, size string, data []byte) (string, error) {
    // ... existing upload logic ...
    
    // Return relative path (key) instead of absolute URL
    logger.Infof("[service] MinIO Upload completed: key=%s", key)
    return key, nil
}
```

Run: `go test ./internal/worker/... -run TestThumbnailHandler_UploadReturnsRelativePath -v`
Expected: PASS

- [ ] **Step 3: Run all thumbnail tests**

Run: `go test ./internal/worker/... -v`
Expected: All pass (may need updates to existing test assertions)

- [ ] **Step 4: Commit**

```bash
git add internal/service/minio_service.go internal/worker/thumbnail_handler_test.go
git commit -m "refactor: store thumbnail relative path instead of absolute URL in MinIO"
```

---

## Task 2: Modify COS Service to Return Relative Path

**Files:**
- Modify: `internal/service/cos_service.go:61-87`
- Test: Verify via existing tests

- [ ] **Step 1: Write failing test for COS relative path**

Add similar test to verify COS service returns relative path.

Run: `go test ./internal/service/... -run TestCOSUpload -v`
Expected: FAIL - currently returns absolute URL

- [ ] **Step 2: Modify COS Upload to return relative path**

In `internal/service/cos_service.go`, change return value:

```go
func (s *COSService) Upload(ctx context.Context, filename, size string, data []byte) (string, error) {
    // ... existing upload logic ...
    
    // Return relative path (key) instead of absolute URL
    logger.Infof("[service] COS Upload completed: key=%s", key)
    return key, nil
}
```

Run: `go test ./internal/service/... -run TestCOSUpload -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/service/cos_service.go
git commit -m "refactor: store thumbnail relative path instead of absolute URL in COS"
```

---

## Task 3: Modify Handler to Prepend Base URL for Responses

**Files:**
- Modify: `internal/handler/thumbnail_url_rewrite.go`
- Test: `internal/handler/thumbnail_url_rewrite_test.go`

**Context:** The handler currently rewrites localhost URLs to match request host. With relative paths stored, we need to prepend a configured base URL instead.

- [ ] **Step 1: Write failing test for relative path handling**

In `internal/handler/thumbnail_url_rewrite_test.go`, add test that DB contains relative paths:

```go
func TestThumbnailURLRewriteWithRelativePaths(t *testing.T) {
    t.Parallel()
    router, repos := newImageHandlerTestRouter(t)
    // Store relative paths in DB
    setImageThumbnailURLs(t, repos.db, 1,
        "thumbnails/20260102/image-1-small.jpg",
        "thumbnails/20260102/image-1-large.jpg",
    )

    resp := performRequest(t, router, http.MethodGet, "http://127.0.0.1:4321/api/v1/images/1", nil)
    // Response should have full URL with base URL prepended
    assertThumbnailURLs(t, decodeToMap(t, resp),
        "http://127.0.0.1:4321/thumbnails/20260102/image-1-small.jpg",
        "http://127.0.0.1:4321/thumbnails/20260102/image-1-large.jpg",
    )
}
```

Run: `go test ./internal/handler/... -run TestThumbnailURLRewriteWithRelativePaths -v`
Expected: FAIL - currently returns relative path as-is

- [ ] **Step 2: Add base URL config and modify handler**

The handler needs access to a configured base URL. This likely requires:
1. Adding thumbnail base URL to config
2. Creating a thumbnail URL builder helper
3. Modifying `rewriteThumbnailURLForRequest` to prepend base URL

This is a larger change - verify exact config structure first:

```bash
grep -r "ThumbnailBaseURL\|thumbnail_base_url\|ThumbnailURL" internal/config/
```

- [ ] **Step 3: Run handler tests**

Run: `go test ./internal/handler/... -v`
Expected: All pass

- [ ] **Step 4: Commit**

---

## Task 4: Add Batch Migration for Existing DB Rows

**Files:**
- Modify: `cmd/migrate-thumbnails/main.go`
- Test: Run migration on test database

**Context:** Existing rows have absolute URLs. Need to extract relative path from URL.

- [ ] **Step 1: Write migration logic**

In `cmd/migrate-thumbnails/main.go`, add new migration mode:

```go
// Add flag for URL-to-path conversion
convertToRelative := flag.Bool("convert-urls", false, "Convert absolute thumbnail URLs to relative paths")

// In main(), after stats:
// ─── Step 4: Convert absolute URLs to relative paths ───
if *convertToRelative && withThumbnail > 0 {
    fmt.Printf("[Step 4] 转换绝对URL为相对路径...")
    if *dryRun {
        fmt.Printf(" (dry-run, 跳过)\n")
        // Preview: show what would be converted
    } else {
        result, err := db.Exec(`
            UPDATE images
            SET thumbnail_small_url = 
                CASE 
                    WHEN thumbnail_small_url LIKE 'http%://%/%' 
                    THEN substr(thumbnail_small_url, instr(thumbnail_small_url, '/', 10) + 1)
                    ELSE thumbnail_small_url
                END,
                thumbnail_large_url = 
                CASE 
                    WHEN thumbnail_large_url LIKE 'http%://%/%'
                    THEN substr(thumbnail_large_url, instr(thumbnail_large_url, '/', 10) + 1)
                    ELSE thumbnail_large_url
                END,
                updated_at = CURRENT_TIMESTAMP
            WHERE thumbnail_small_url LIKE 'http%://%/%' 
               OR thumbnail_large_url LIKE 'http%://%/%'
        `)
        // ... handle result
    }
}
```

- [ ] **Step 2: Test migration on sample data**

```bash
# Create test DB with absolute URLs
sqlite3 test.db "UPDATE images SET thumbnail_small_url='https://cdn.example.com/bucket/thumbnails/img1.jpg' WHERE id=1"

# Run migration
go run cmd/migrate-thumbnails/main.go -config config.yaml -convert-urls -dry-run

# Verify
sqlite3 test.db "SELECT thumbnail_small_url FROM images WHERE id=1"
# Should show: bucket/thumbnails/img1.jpg (or thumbnails/img1.jpg depending on extraction)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/migrate-thumbnails/main.go
git commit -m "feat: add batch migration to convert absolute thumbnail URLs to relative paths"
```

---

## Verification Commands

After all tasks complete:

```bash
# Run all tests
go test ./internal/service/... ./internal/handler/... ./internal/worker/... -v

# Run migration help
go run cmd/migrate-thumbnails/main.go -h

# Dry-run migration
go run cmd/migrate-thumbnails/main.go -convert-urls -dry-run

# Apply migration (production)
go run cmd/migrate-thumbnails/main.go -convert-urls
```

---

## Risk Notes

- **Backward compatibility**: Existing thumbnail URLs in `<img>` tags or cached responses will break after migration. Consider adding URL rewrite at CDN/proxy level or gradual rollout.
- **URL extraction logic**: The regex/INSTR approach may need adjustment based on actual URL formats in your database (with/without bucket prefix, query params, etc.). Test with real data first.
- **Handler change**: Task 3 requires config changes that may affect other handlers.
