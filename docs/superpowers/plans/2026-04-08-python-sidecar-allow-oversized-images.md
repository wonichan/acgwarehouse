# Python Sidecar Oversized Image Allowance Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow trusted oversized images to be hashed in the Python sidecar duplicate-detection flow without Pillow blocking on the default decompression-bomb pixel limit.

**Architecture:** Apply a module-level Pillow configuration in the existing hashing module so the current `imagededup.PHash.encode_image(...)` path can open oversized images. Keep the change narrow to the hashing entrypoint and verify it with a regression test plus a real oversized-image run.

**Tech Stack:** Python, FastAPI sidecar, imagededup, Pillow, pytest.

---

## Chunk 1: Oversized-image allowance in hashing flow

### Task 1: Add regression test for Pillow pixel-limit configuration

**Files:**
- Modify: `services/python-sidecar/tests/test_hashing.py`
- Modify: `services/python-sidecar/compute/hashing.py`

- [ ] **Step 1: Write the failing test**

```python
def test_hashing_module_disables_pillow_pixel_limit_for_trusted_images():
    assert Image.MAX_IMAGE_PIXELS is None
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest services/python-sidecar/tests/test_hashing.py -k pixel_limit -q`
Expected: FAIL because `Image.MAX_IMAGE_PIXELS` still has Pillow's default threshold.

- [ ] **Step 3: Implement minimal code in hashing module**

```python
from PIL import Image

Image.MAX_IMAGE_PIXELS = None
```

Constraints:
- Keep the override local to the Python sidecar hashing module.
- Do not change request/response contracts or duplicate grouping logic.
- Do not add warning filters unless the test proves they are needed.

- [ ] **Step 4: Re-run the targeted test**

Run: `pytest services/python-sidecar/tests/test_hashing.py -k pixel_limit -q`
Expected: PASS.

### Task 2: Verify existing hashing behavior and real oversized-image processing

**Files:**
- Modify: `services/python-sidecar/tests/test_hashing.py` (only if additional assertions are needed)

- [ ] **Step 1: Run hashing test file**

Run: `pytest services/python-sidecar/tests/test_hashing.py -q`
Expected: PASS.

- [ ] **Step 2: Run diagnostics on changed Python files**

Run: language-server diagnostics for `services/python-sidecar/compute/hashing.py` and `services/python-sidecar/tests/test_hashing.py`
Expected: zero new errors.

- [ ] **Step 3: Manual QA with a real oversized image**

Run a short Python command that:
1. Creates a trusted image above Pillow's default pixel threshold.
2. Calls `compute_image_hashes()` from `services/python-sidecar/compute/hashing.py`.
3. Prints the resulting `error`, `sha256` presence, and `phash` presence.

Expected: `error` is `None`, `sha256` is populated, and `phash` is populated.
