# Phase 21 Research — Windows Packaging

**Phase:** 21 — Windows Packaging  
**Requirement:** `OPS-03`  
**Language:** English  
**Status:** Complete

---

## Executive Summary

Phase 21 should ship as a **single ZIP portable bundle for Windows x64** that contains:

- the Flutter desktop executable as the single user-facing entry point,
- the Go backend binary,
- the Python sidecar binary plus its bundled runtime,
- portable `config/`, `storage/`, `library/`, and `runtime/` directories.

The recommended implementation is:

1. Keep **Go as the only orchestrator** for the Python sidecar.
2. Keep the **Flutter Windows executable** as the single user-facing entry point.
3. Package the Python sidecar with **PyInstaller `--onedir`**, not `--onefile`.
4. Write runtime manifest, logs, and startup diagnostics into a **fixed bundle-local runtime directory**.
5. Build the final distributable as **one ZIP file**, while keeping the internal bundle multi-file and debuggable.

This approach satisfies the locked decisions in `21-CONTEXT.md` without introducing an installer, without requiring a system Python, and without pushing lifecycle ownership out of Go.

---

## Decision Coverage From Research

| Decision | Research outcome |
|----------|------------------|
| `D-01` green portable package | Use bundle-local directories; no installer; zip release only |
| `D-02` single ZIP distributable | Produce one archive containing the complete portable tree |
| `D-03` Windows x64 only | Build only Windows x64 artifacts in packaging script |
| `D-04` embedded Python runtime | Package Python sidecar with PyInstaller so user needs no external Python |
| `D-05` one unified entry point | Keep the Flutter desktop executable as the only user-visible start target |
| `D-06` Go remains orchestrator | Flutter starts Go; Go starts Python; Flutter never manages Python directly |
| `D-07` fully portable layout | Keep config/storage/library/logs/runtime under extracted root |
| `D-08` fixed runtime diagnostics directory | Use bundle-local `runtime/`, `runtime/logs/`, `runtime/diagnostics/` |
| `D-09` in-place overwrite upgrades | Build a stable directory layout and document overwrite-safe behavior |
| `D-10` upgrade safety as explicit verification | Add smoke checks for stale-file tolerance and file-lock handling guidance |
| `D-11` clear startup error page/dialog | Launcher must show a startup error dialog on bootstrap failures |
| `D-12` classify Go/Python/chain failures | Diagnostic file must encode `go`, `python`, or `startup_chain` cause |
| `D-13` no silent sidecar failure | Startup failure must stop launch with explicit diagnostics |

---

## Packaging Strategy Options

### Option A — PyInstaller `--onefile` for Python sidecar

**Pros**
- Single Python executable file
- Simple artifact naming

**Cons**
- Worse debugging experience
- Self-extraction overhead on every start
- Harder to inspect missing DLL/import issues
- PyInstaller guidance recommends solving packaging issues in `--onedir` first

### Option B — PyInstaller `--onedir` for Python sidecar **(Recommended)**

**Pros**
- Best fit for a portable ZIP bundle
- Easier debugging and missing-dependency inspection
- Stable file layout for diagnostics and support
- No requirement that the internal bundle be single-file as long as the distributed artifact is one ZIP

**Cons**
- More files in the extracted directory
- Packaging script must copy a directory tree instead of one file

### Option C — CPython embeddable package + raw Python scripts

**Pros**
- Official Python runtime distribution
- Fine-grained control over runtime layout

**Cons**
- More manual dependency/bootstrap work
- More complex FastAPI/uvicorn/Pillow packaging story
- Higher maintenance burden than PyInstaller for this phase

**Recommendation:** Use **Option B**. The user asked for a single ZIP bundle, not a single executable. `--onedir` gives a more supportable portable product while still fully satisfying `OPS-03`.

---

## Recommended Bundle Layout

```text
ACGWarehouse/
  ACGWarehouse.exe                    # Flutter desktop executable; single user-facing entry
  data/                               # Flutter release assets
  runtime/
    bin/
      acgwarehouse-server.exe         # Go backend
    python-sidecar/
      acgwarehouse-sidecar.exe        # PyInstaller output
      ... bundled DLLs/files ...
    runtime-manifest.json             # Go writes at startup
    logs/
      go.log
      python-sidecar.log
      flutter-bootstrap.log
    diagnostics/
      startup-error.json
  config/
    config.yaml
    config.example.yaml
  storage/
    acgwarehouse.db
  library/
```

This layout keeps the launcher obvious, the runtime artifacts grouped, and the mutable user data portable.

---

## Startup Architecture Recommendation

### Recommended startup chain

1. User runs `ACGWarehouse.exe`.
2. Flutter packaged bootstrap resolves the extracted bundle root and validates required child binaries.
3. Flutter packaged bootstrap allocates runtime environment values:
   - Go host: `127.0.0.1`
   - Go port: `0` (ephemeral)
   - sidecar port: free loopback port chosen by Flutter bootstrap
   - runtime manifest path: bundle-local `runtime/runtime-manifest.json`
   - diagnostics path: bundle-local `runtime/diagnostics/startup-error.json`
4. Flutter packaged bootstrap starts `runtime/bin/acgwarehouse-server.exe` with these environment variables.
5. Go starts the Python sidecar from `runtime/python-sidecar/acgwarehouse-sidecar.exe`.
6. Go writes `runtime/runtime-manifest.json` when ready.
7. Flutter packaged bootstrap waits for the manifest or a startup error file before the shell becomes usable.
8. If startup fails, Flutter packaged bootstrap reads `startup-error.json` and shows a startup error dialog or failure screen that distinguishes:
   - `go`
   - `python`
   - `startup_chain`

This preserves the locked architecture and user decision fidelity: the user launches Flutter first, Flutter starts Go, Go starts Python, and the app only becomes usable after Go/Python readiness is established.

---

## Required Runtime Contracts

### Go runtime contract

Go should become packaging-aware via environment variables and relative bundle layout resolution:

- `ACG_RUNTIME_ROOT`
- `ACG_RUNTIME_MANIFEST_PATH`
- `ACG_DIAGNOSTICS_DIR`
- `ACG_LOGS_DIR`
- `ACG_SIDECAR_EXECUTABLE`
- `ACG_SIDECAR_PORT`

Go should continue to own:

- sidecar lifecycle,
- readiness probing,
- manifest generation,
- explicit degraded/stopped state semantics,
- startup error classification.

### Python sidecar contract

The packaged sidecar executable should support:

- `--host 127.0.0.1`
- `--port <chosen-port>`
- stdout/stderr safeguards for windowed/no-console mode,
- health endpoint support,
- log file redirection passed by Go if needed.

### Packaged Flutter bootstrap contract

The packaged Flutter bootstrap must:

- use only bundle-relative child executable paths,
- never rely on `python` from PATH,
- surface startup failure dialogs/screens with log locations,
- shut down Go when Flutter exits.

---

## External Research Notes

### PyInstaller guidance

Authoritative documentation supports these planning conclusions:

- Do **not** use `--onefile` as the first debugging path.
- `--noconsole` / windowed mode can leave `sys.stdout` and `sys.stderr` as `None`.
- resource path handling and hidden import collection need explicit validation.

Implication for this phase:

- package the sidecar in `--onedir`,
- make `main.py` safe in no-console mode,
- add explicit packaging smoke tests that run only against the packaged sidecar artifact.

### CPython Windows embeddable distribution

The official embeddable package is viable for embedded applications, but it is a lower-level option than needed here. For a FastAPI + uvicorn + Pillow + imagehash sidecar, PyInstaller provides a faster path to a supportable portable bundle.

---

## Validation Architecture

The validation strategy for Phase 21 should combine **unit tests, packaging smoke tests, and a final operator verification step**.

### Fast automated checks

- Go unit tests for runtime layout resolution, startup diagnostics, and packaged bootstrap inputs
- Python tests for packaged sidecar startup argument parsing and no-console safety
- Flutter tests for runtime manifest path handling where applicable
- PowerShell packaging smoke tests for bundle assembly and launch behavior

### Critical smoke assertions

The packaged bundle must prove all of the following:

1. Launch uses only bundle-local child binaries.
2. No external `python.exe` is required on PATH.
3. `runtime/runtime-manifest.json` is created inside the extracted bundle.
4. `runtime/logs/` and `runtime/diagnostics/` are created inside the extracted bundle.
5. Startup failures create a structured diagnostic file with classified component cause.
6. Duplicate detection still works when the app runs from the packaged bundle.

### Manual-only confirmation

One final verification remains human-facing:

- unzip on a Windows machine that does not have Python installed,
- run `ACGWarehouse.exe`,
- confirm the app opens and duplicate detection still works,
- confirm startup failure dialog quality by forcing one failure mode if feasible.

---

## Recommended Atomic Commit Strategy

For ultrawork execution, use **small TDD-shaped commits per task**:

1. `test(21-xx): add failing coverage for ...`
2. `feat(21-xx): implement ...`
3. `refactor(21-xx): tighten packaging/runtime wiring` *(only if needed)*
4. `docs(21-xx): document portable packaging flow` *(docs-only tasks)*

Avoid giant cross-stack commits. Each plan should leave the tree in a runnable, partially verifiable state.

---

## Planning Implications

Phase 21 should be split into four execution plans:

1. **Portable runtime contract** — Go/Python packaging-aware paths, logs, diagnostics, sidecar executable contract.
2. **Packaged Flutter bootstrap** — Flutter-first startup, manifest wait, startup error dialog/screen, child-process shutdown.
3. **Bundle assembly** — PowerShell packaging pipeline, artifact layout, ZIP creation, operator docs.
4. **Verification closure** — automated packaged smoke checks plus a blocking human verification step.

That split matches ultrawork execution: isolated file ownership, minimal plan overlap, and clean atomic commit boundaries.
