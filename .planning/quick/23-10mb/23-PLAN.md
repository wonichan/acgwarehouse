---
phase: quick-23
plan: 01
type: tdd
wave: 1
depends_on: []
files_modified:
  - internal/ai/image_compression.go
  - internal/ai/image_compression_test.go
  - internal/ai/qwen_provider.go
  - internal/ai/doubao_provider.go
autonomous: true
requirements: [QUICK-23]
user_setup: []

must_haves:
  truths:
    - "Images under 10MB are sent to AI unchanged"
    - "Images over 10MB are compressed before sending to AI"
    - "Compressed images are under 10MB in size"
    - "AI tag generation works for large images"
  artifacts:
    - path: "internal/ai/image_compression.go"
      provides: "Image compression utility"
      exports: ["CompressImageIfNeeded"]
    - path: "internal/ai/image_compression_test.go"
      provides: "Test coverage for compression"
    - path: "internal/ai/qwen_provider.go"
      provides: "Qwen provider with compression"
    - path: "internal/ai/doubao_provider.go"
      provides: "Doubao provider with compression"
  key_links:
    - from: "internal/ai/qwen_provider.go"
      to: "image_compression.go"
      via: "CompressImageIfNeeded call in processImageURL"
    - from: "internal/ai/doubao_provider.go"
      to: "image_compression.go"
      via: "CompressImageIfNeeded call in processImageURL"
---

<objective>
Add automatic image compression for files exceeding 10MB before sending to AI APIs for tag generation.

Purpose: Prevent API errors when processing large images for AI tagging.
Output: Working compression utility integrated into both AI providers.
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# Key Context from Codebase

## Current Implementation
- `internal/ai/qwen_provider.go`: `processImageURL()` reads file and encodes to base64 without size check
- `internal/ai/doubao_provider.go`: Same pattern as qwen_provider
- `internal/service/thumbnail_service.go`: Has image compression logic using `imaging` library

## Existing Dependencies (can reuse)
- `github.com/disintegration/imaging` - already imported in thumbnail_service
- `golang.org/x/image/webp` - already imported for webp support

## Key Interfaces
From `internal/ai/provider.go`:
```go
type AIProvider interface {
    Name() string
    GenerateTags(ctx interface{}, imageURL, prompt string) (*TagResult, error)
}
```

From `internal/ai/qwen_provider.go`:
```go
func (p *QwenProvider) processImageURL(imageURL string) (string, error) {
    // Currently: os.ReadFile(imageURL) -> base64 encode -> return data URI
}
```
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write failing tests for image compression</name>
  <files>internal/ai/image_compression_test.go</files>
  <behavior>
    - Test 1: File under 10MB returns original bytes unchanged
    - Test 2: File over 10MB returns compressed bytes under 10MB
    - Test 3: Compressed image is valid (can be decoded)
    - Test 4: Compression preserves image content (same visual after decode)
  </behavior>
  <action>
    Create `internal/ai/image_compression_test.go` with the following tests:

    1. `TestCompressImageIfNeeded_SmallFileUnchanged`: Create a small test image (~1MB), call CompressImageIfNeeded, verify returned bytes equal original file bytes.

    2. `TestCompressImageIfNeeded_LargeFileCompressed`: Create a large test image (~15MB using a generated image), call CompressImageIfNeeded, verify returned bytes are less than 10MB.

    3. `TestCompressImageIfNeeded_CompressedImageValid`: After compression, decode the returned bytes to verify it's a valid image.

    Use `testing` package and create temp files with `os.CreateTemp`.

    The tests should FAIL because CompressImageIfNeeded doesn't exist yet.
  </action>
  <verify>
    <automated>go test ./internal/ai/... -run TestCompressImageIfNeeded -v 2>&1 | grep -E "(FAIL|undefined)"</automated>
  </verify>
  <done>Test file exists with 3+ test functions, tests fail (function not implemented)</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement image compression utility</name>
  <files>internal/ai/image_compression.go, internal/ai/image_compression_test.go</files>
  <behavior>
    - Same tests as Task 1 should now PASS
    - Compression algorithm: reduce quality progressively until under 10MB
    - If quality reduction insufficient, reduce dimensions
  </behavior>
  <action>
    Create `internal/ai/image_compression.go`:

    ```go
    package ai

    import (
        "bytes"
        "fmt"
        "os"
        "strings"

        "github.com/disintegration/imaging"
        _ "golang.org/x/image/webp"
    )

    const maxAIImageSize = 10 * 1024 * 1024 // 10MB

    // CompressImageIfNeeded reads an image file and returns base64-ready bytes.
    // If the file exceeds 10MB, it compresses the image until under the limit.
    // Returns: (imageData, contentType, error)
    func CompressImageIfNeeded(filePath string) ([]byte, string, error) {
        // 1. Read original file
        // 2. If size <= 10MB, return as-is
        // 3. Otherwise, load image with imaging.Open
        // 4. Encode as JPEG with quality 90, check size
        // 5. If still > 10MB, reduce quality progressively (85, 80, 75, ...)
        // 6. If quality at 10 and still > 10MB, reduce dimensions by 10% and retry
        // 7. Return compressed bytes and content type
    }
    ```

    Implementation details:
    - Detect content type from file extension (jpeg, png, gif, webp)
    - Output always as JPEG for compression efficiency
    - Start with quality 90, reduce by 5 each iteration
    - Minimum quality: 50
    - If quality at 50 still > 10MB, reduce dimensions by 10% and retry quality reduction
    - Maximum 20 iterations to prevent infinite loops

    Make all tests pass.
  </action>
  <verify>
    <automated>go test ./internal/ai/... -run TestCompressImageIfNeeded -v</automated>
  </verify>
  <done>All CompressImageIfNeeded tests pass, function correctly compresses large images under 10MB</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Update providers to use compression</name>
  <files>internal/ai/qwen_provider.go, internal/ai/doubao_provider.go</files>
  <behavior>
    - Existing provider tests still pass
    - Large images are automatically compressed before API call
  </behavior>
  <action>
    Update `processImageURL` in both `qwen_provider.go` and `doubao_provider.go`:

    Replace the current implementation:
    ```go
    // Current code:
    data, err := os.ReadFile(imageURL)
    // ... encode to base64
    ```

    With:
    ```go
    // Use compression utility
    data, contentType, err := CompressImageIfNeeded(imageURL)
    if err != nil {
        return "", fmt.Errorf("process image: %w", err)
    }
    return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
    ```

    Remove the manual content type detection since CompressImageIfNeeded returns it.

    Ensure all existing tests still pass.
  </action>
  <verify>
    <automated>go test ./internal/ai/... -v</automated>
  </verify>
  <done>Both providers use CompressImageIfNeeded, all tests pass</done>
</task>

</tasks>

<verification>
1. Run all AI package tests: `go test ./internal/ai/... -v`
2. Run all worker tests: `go test ./internal/worker/... -v`
3. Verify no compilation errors: `go build ./...`
</verification>

<success_criteria>
- All tests pass: `go test ./internal/ai/...` and `go test ./internal/worker/...`
- Images over 10MB are compressed before sending to AI APIs
- Images under 10MB are sent unchanged
- Code compiles without errors
</success_criteria>

<output>
After completion, create `.planning/quick/23-10mb/23-SUMMARY.md`
</output>