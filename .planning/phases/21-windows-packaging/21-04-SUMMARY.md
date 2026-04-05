# Phase 21-04 Summary

## Outcome

- Updated `deploy/windows/package-smoke.ps1` into the authoritative packaged smoke runner.
- Added dynamic launcher port handling, seeded duplicate-detection fixture data, and validated the packaged Flutter → Go → Python chain against the ZIP artifact.
- Preserved the blocking human verification gate in `.planning/phases/21-windows-packaging/21-HUMAN-UAT.md`.

## Verification Evidence

- `powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1 -SkipTests` rebuilt the portable artifact successfully.
- `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1` passed.
- `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1 -VerifyBundleLayout` passed.
- `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1 -VerifyDocsOnly` passed.

## Automated Smoke Coverage

- Extracts `dist/windows-zip/ACGWarehouse-windows-x64-portable.zip` to a temporary directory.
- Verifies bundle layout paths including `ACGWarehouse.exe`, `data/`, `runtime/bin/`, `runtime/python-sidecar/`, `config/`, `storage/`, and `library/`.
- Launches the packaged desktop app with Python removed from `PATH`.
- Waits for `runtime/runtime-manifest.json`, probes `/health`, and confirms `runtime/logs/` plus `runtime/diagnostics/` exist.
- Seeds two packaged-library duplicate fixtures into the bundled SQLite database, calls `POST /api/v1/duplicates/detect`, and confirms `runtime/logs/python-sidecar.log` changes.

## Remaining Checkpoint

- Phase 21 still requires the human verification checkpoint from `21-HUMAN-UAT.md`.
- Automated smoke is now green, so the only remaining blocker is clean-machine unzip-and-run approval.

## Resume Signal

- Resume with `approved` only after a Windows x64 machine without Python installed passes `21-HUMAN-UAT.md`.
- Otherwise resume with the exact observed failure and classify it as `go`, `python`, `startup_chain`, or `human-uat`.
