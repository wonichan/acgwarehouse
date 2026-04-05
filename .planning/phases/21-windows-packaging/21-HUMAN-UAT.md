# Phase 21 Human UAT — Windows Packaging

**Phase:** 21 — Windows Packaging  
**Requirement:** `OPS-03`  
**Status:** Awaiting human execution

## Goal

Confirm that the packaged Windows ZIP works on a machine without Python installed and that startup failures are diagnosable.

## Blocking Preconditions

1. `powershell -ExecutionPolicy Bypass -File deploy/windows/package-smoke.ps1` has passed locally.
2. This checkpoint is still required because Phase 21 demands a real unzip-and-run check on a Windows x64 machine without Python installed.
3. Do **not** treat the phase as approved until the human UAT steps below pass.

## Steps

1. Copy `dist/windows-zip/ACGWarehouse-windows-x64-portable.zip` to a Windows x64 machine without Python.
2. Extract it to a writable directory.
3. Run `ACGWarehouse.exe`.
4. Confirm the app opens without separate Go/Python startup steps.
5. Trigger duplicate detection and confirm the packaged sidecar path works.
6. Close the app and confirm runtime files remain within the extracted directory.
7. Optional negative test: rename the packaged sidecar executable, relaunch, and confirm the startup dialog shows a classified failure and log locations.

## Expected Result

- Portable ZIP launches successfully.
- Duplicate detection works with the bundled Python runtime.
- Startup failure UX is explicit and actionable.

## Resume Signal

- Type `approved` only after both conditions are true:
  - automated smoke is green, and
  - the clean-machine unzip-and-run check passed.
- If it still fails, reply with the exact observed failure, including whether the break was `go`, `python`, `startup_chain`, or `human-uat`.

## Approval

- [ ] Approved
- [ ] Rejected with notes
