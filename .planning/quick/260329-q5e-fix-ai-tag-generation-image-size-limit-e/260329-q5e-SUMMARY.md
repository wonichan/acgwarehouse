---
phase: 260329-q5e
plan: 01
subsystem: ai
tags: [pixel-limit, image-compression, tdd]
requires: []
provides: [PIXEL-LIMIT-01]
affects: [qwen_provider.go, doubao_provider.go]
tech-stack:
  added: [maxAIPixelCount, targetPixelCount, calculateResizeDimensions]
  patterns: [early-return, proportional-resize]
key-files:
  created: []
  modified:
    - internal/ai/image_compression.go
    - internal/ai/image_compression_test.go
decisions:
  - Pixel limit (36M) checked BEFORE file size (10MB) to prevent API 400 errors
  - Target pixel count (30M) provides safety margin under 36M API limit
  - Proportional resize preserves aspect ratio using Lanczos filter
  - Original format preserved for small files under both limits
metrics:
  duration: 5 min
  completed_at: 2026-03-29
  tasks: 3
  commits: 3
  files_modified: 2
  test_coverage: 3 new tests
---

# Fix AI Tag Generation Image Pixel Count Limit Summary

## One-Liner

Added pixel count validation (36M limit) to `CompressImageIfNeeded` with proportional resize before file size check, preventing AI API 400 errors for large images.

## Objective Achieved

✅ Pixel limit constant defined (maxAIPixelCount = 36000000)
✅ Images exceeding pixel limit resized proportionally
✅ Aspect ratio preserved after resize
✅ Images under pixel limit not unnecessarily resized
✅ All tests pass: `go test ./internal/ai/... -v`

## Key Changes

### Constants Added
```go
maxAIPixelCount     = 36000000  // 36M pixels - API pixel limit
targetPixelCount    = 30000000  // 30M pixels - target to stay safely under limit
```

### Helper Function
```go
func calculateResizeDimensions(width, height int, maxPixels int) (int, int)
```
- Calculates proportional dimensions using `scale = sqrt(maxPixels / pixels)`
- Preserves aspect ratio exactly

### Modified Flow in `CompressImageIfNeeded`
1. **Load image FIRST** to check pixel dimensions (before file size)
2. **Early return**: if pixel count AND file size under limits → return original unchanged
3. **Resize if needed**: if pixels > 36M → resize to target 30M using Lanczos
4. **Encode and check file size**: if > 10MB → compress further

## Test Coverage

| Test | Description | Result |
|------|-------------|--------|
| `TestCompressImageIfNeeded_ExceedsPixelLimit` | 10000x10000 (100M) → resized to ~30M | ✅ |
| `TestCompressImageIfNeeded_NonSquarePixelLimit` | 15000x5000 (75M) → ratio 3:1 preserved | ✅ |
| `TestCompressImageIfNeeded_UnderPixelLimit` | 5000x5000 (25M) → no resize needed | ✅ |

## Deviations from Plan

None - plan executed exactly as written following TDD cycle.

## Commits

| Commit | Type | Message |
|--------|------|---------|
| df6e9bd | test | test(ai): add failing test for pixel limit validation |
| 8e5a592 | feat | feat(ai): add pixel count resize before API call |
| d31026f | test | test(ai): add edge case tests for pixel resize |

## Verification

```bash
go test ./internal/ai/... -v
# PASS: 28 tests, 0 failures
```

## Self-Check: PASSED

- [x] Created files exist: SUMMARY.md
- [x] Commits exist: df6e9bd, 8e5a592, d31026f
- [x] All tests pass
- [x] No regressions in existing tests