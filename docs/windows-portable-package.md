# Windows Portable Package Guide

This guide explains how to build, launch, upgrade, and troubleshoot the Windows x64 portable ZIP package for ACGWarehouse.

## Bundle Layout

The portable ZIP extracts into one self-contained root directory. The required portable paths are:

```text
ACGWarehouse.exe
data/
runtime/
runtime/bin/acgwarehouse-server.exe
runtime/python-sidecar/acgwarehouse-sidecar.exe
runtime/logs/
runtime/diagnostics/
config/
storage/
library/
```

The extracted root also includes the Flutter Windows support DLLs that must stay next to `ACGWarehouse.exe`. Keep the entire extracted directory together.

## Build Command

Run the authoritative packaging command from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1
```

Or use the matching Make target:

```bash
make package-windows-portable
```

The packaging pipeline builds the Go server, packages the Python sidecar through PyInstaller onedir mode via `services/python-sidecar/sidecar.spec`, runs `flutter build windows --release`, assembles the portable tree in `dist/windows-portable/`, and creates `dist/windows-zip/ACGWarehouse-windows-x64-portable.zip`.

## First Launch

1. Extract `dist/windows-zip/ACGWarehouse-windows-x64-portable.zip` to a writable directory.
2. Confirm the extracted root still contains `ACGWarehouse.exe`, `data/`, `runtime/`, `config/`, `storage/`, and `library/`.
3. If you are migrating an existing installation, copy your populated `data/acgwarehouse.db` into the extracted `data/` directory before first use.
4. Copy `deploy/config/config.example.yaml` values into `config/` as needed before first use.
5. Start the app by running `ACGWarehouse.exe`.
6. Keep `ACGWarehouse.exe`, `data/`, and `runtime/` in the same extracted root so the packaged startup chain can find the Go runtime, sidecar, logs, diagnostics, and SQLite database.

## In-Place Overwrite Upgrade

This package supports in-place overwrite upgrades.

- Always close the running app before overwrite.
- Preserve `config/`, `data/`, `storage/`, and `library/` during upgrades because those directories hold operator configuration and user data.
- Replace the Flutter executable + data/ assets + runtime/ binaries as a unit so the packaged launcher, Go runtime, and Python sidecar stay on the same runtime compatibility level.
- D-10 validation must explicitly consider stale runtime files, runtime compatibility, file locks, and user-data preservation.
- If overwrite fails due to file locks, close the running app before overwrite and then delete only old runtime binaries after the app is closed.
- Do not partially mix old and new runtime files. Partial replacement can leave stale runtime files behind and break startup.

Recommended overwrite flow:

1. Back up `config/`, `data/`, `storage/`, and `library/` if you need a rollback point.
2. Close ACGWarehouse and confirm no packaged processes are still running.
3. Extract the new ZIP over the existing directory, replacing `ACGWarehouse.exe`, `data/`, and `runtime/` together.
4. If Windows blocks replacement because of file locks, close the app completely and retry. Only remove stale runtime files from the old `runtime/` tree after the app is closed.
5. Start `ACGWarehouse.exe` again and confirm the packaged stack starts normally.

## Troubleshooting

If startup fails, classify the failure as `go`, `python`, or `startup_chain` and inspect the packaged diagnostics before changing files.

- `go`: the packaged Go runtime failed before the backend became ready.
- `python`: the packaged Python sidecar failed to start or respond.
- `startup_chain`: the Flutter → Go → Python startup chain did not reach a ready state.

Use `runtime/diagnostics/startup-error.json` as the first source of truth. The failure details should point you to the packaged log files and help identify whether the fault is in `go`, `python`, or the wider `startup_chain`.

Common checks:

- Confirm `runtime/bin/acgwarehouse-server.exe` and `runtime/python-sidecar/acgwarehouse-sidecar.exe` both exist.
- Confirm `data/` stayed next to `ACGWarehouse.exe` after extraction or overwrite.
- Remove only stale runtime files after the app is fully closed if an interrupted overwrite left mixed binaries behind.
- Rebuild the package if runtime compatibility between the launcher, Go binary, and sidecar is uncertain.

## Log and Diagnostic Locations

- Go log: `runtime/logs/go.log`
- Python sidecar log: `runtime/logs/python-sidecar.log`
- Flutter bootstrap log: `runtime/logs/flutter-bootstrap.log`
- Startup diagnostic: `runtime/diagnostics/startup-error.json`

Keep these files with the extracted bundle when reporting issues so operators can verify startup behavior, stale runtime files, runtime compatibility, file locks, and user-data preservation outcomes from the same portable root.
