# Phase 15 Research: Compute Sidecar Infrastructure

## Objective

Answering: **What do we need to know to plan Phase 15 well?**

This document is planning-focused (not implementation), aligned to:

- `.planning/phases/15-compute-sidecar-infrastructure/15-CONTEXT.md`
- `.planning/REQUIREMENTS.md` (COMP-01, COMP-02, COMP-06)
- `.planning/ROADMAP.md` (Phase 15 success criteria)
- `.planning/STATE.md` (current milestone position)
- `.planning/PROJECT.md` (Go orchestration and Python compute boundary)

---

## Non-Negotiable Decisions to Preserve (D-01..D-15)

Planning must preserve these decisions exactly:

1. **Go is the single orchestrator** for Python sidecar lifecycle (start, wait, probe, stop, error propagation).
2. **Flutter depends only on Go service contract**, never on Python lifecycle details.
3. **Runtime manifest is required** for desktop-time Go address discovery; hardcoded `localhost:8080` is not the product contract.
4. **Python address is internal to Go orchestration**, not a Flutter-facing contract.
5. **Health semantics are layered**:
   - `/health` = Go process liveness only
   - `/ready` = Go service readiness to accept traffic
   - Python sidecar diagnostics belong in admin overview, not base health
6. **Phase 15 is degraded-available by design** when Python is unavailable; gallery core flow remains usable.
7. **Python remains pure-compute boundary** (no DB persistence, no business-ID semantics, no task orchestration takeover).
8. **Local desktop delivery is current target**, but contracts must remain separable for future non-local deployment.

---

## 1) Recommended Architecture and Lifecycle (for D-01..D-15)

## Topology

- `Flutter -> Go API` is the only UI-facing runtime dependency.
- `Go -> Python sidecar (HTTP localhost)` is internal orchestration.
- Runtime manifest bridges Flutter to current Go endpoint (non-hardcoded).

## Go-side lifecycle model

Plan a dedicated sidecar runtime component with explicit states:

- `NotStarted`
- `Starting`
- `Ready`
- `Degraded` (Go up, Python unavailable)
- `Stopping`
- `Stopped`

Minimum lifecycle responsibilities:

1. **Spawn Python process** (`os/exec`) with bounded startup timeout.
2. **Discover Python listen endpoint** through controlled startup handshake (stdout contract or equivalent explicit channel).
3. **Readiness probe loop** for Python endpoint before marking sidecar as `Ready`.
4. **Background health probe loop** with last-success and last-error capture.
5. **Graceful stop** on Go shutdown:
   - attempt Python HTTP shutdown endpoint first
   - fallback to process kill on timeout
   - always wait/reap process to avoid zombie/leaked children
6. **Degraded mode transition** when sidecar unavailable; no full Go unready unless Go itself is not ready.

## Phase boundary discipline

Phase 15 should stop at infrastructure:

- lifecycle + probing + diagnostics plumbing
- manifest-based Flutter discovery for Go
- contract shell for future compute APIs (optional)

Phase 15 should **not** migrate duplicate detection computation itself (Phase 16).

---

## 2) Runtime Manifest Design (Flutter discovers Go URL)

## Why manifest

Current Flutter code still defaults to `http://localhost:8080` in:

- `flutter_app/lib/config/api_config.dart`
- `flutter_app/lib/providers/config_provider.dart`

Phase 15 needs runtime discovery that avoids hardcoded fixed ports in product behavior.

## Recommended manifest contract

Use a small JSON file written by Go during startup, for example:

```json
{
  "version": 1,
  "generated_at": "2026-04-04T00:00:00Z",
  "go": {
    "base_url": "http://127.0.0.1:51423",
    "ready": true
  },
  "diagnostics": {
    "instance_id": "<uuid>",
    "pid": 12345
  }
}
```

Guidance:

- Include only what Flutter needs for Phase 15 (`go.base_url` + minimal metadata).
- Do **not** expose Python endpoint as Flutter contract (D-06).
- Add schema version for forward compatibility.
- Write atomically (temp file + rename) to avoid partial-read races.
- Keep a development-only fallback path to `localhost:8080` for local dev convenience (D-05).

## Suggested ownership

- Go owns manifest write/refresh/cleanup.
- Flutter reads manifest early in app boot and updates API base URL before first real API traffic.

---

## 3) Readiness, Health, Degraded Availability, Diagnostics Boundaries

## API semantics to lock in planning

- `/health` remains simple Go liveness (`internal/handler/health_handler.go`).
- `/ready` remains Go readiness-to-serve; it should not become “Python-ready”.
- Sidecar details move to admin monitoring contract (existing overview pattern) via `AdminService`.

## Admin diagnostics payload scope (Phase 15)

Add sidecar diagnostics under overview-style structure (not base health):

- sidecar state (`starting|ready|degraded|stopped`)
- last probe timestamp
- last probe result
- startup attempts / restart count (if tracked)
- last error summary (short, diagnosable)

Keep this bounded:

- no deep crash-forensics subsystem in Phase 15
- no full auto-recovery policy commitments beyond basic degraded state signaling

---

## 4) Likely Integration Points in Current Repository

These are high-probability planning anchors:

- `internal/app/app.go`
  - central lifecycle and shutdown orchestration
  - natural host for sidecar runtime start/stop wiring
- `internal/app/bootstrap.go`
  - service/runtime initialization wiring
  - place to assemble sidecar runtime dependencies
- `cmd/server/main.go`
  - process-level startup sequence and top-level shutdown signals
- `internal/handler/health_handler.go`
  - keep `/health` and `/ready` semantics constrained
- `internal/handler/routes.go`
  - registration point for any Phase 15 admin/readiness-facing endpoints
- `internal/service/admin_service.go`
  - existing aggregated overview contract; best location for sidecar diagnostic exposure
- `flutter_app/lib/config/api_config.dart`
  - current static host default; needs runtime update path integration
- `flutter_app/lib/providers/config_provider.dart`
  - runtime mutable host config surface for app bootstrapping

Also relevant for later-but-adjacent compute migration alignment:

- `internal/service/task_platform_service.go`
- `internal/worker/job_manager.go`

---

## 5) Concrete TDD Strategy (RED/GREEN) and Candidate Tests

Phase 15 is infrastructure-heavy; TDD should focus on contract and lifecycle invariants.

## RED/GREEN sequence pattern

For each behavior:

1. **RED**: Add failing test asserting target invariant.
2. **GREEN**: Minimal implementation to satisfy test.
3. **REFACTOR**: Keep boundaries clean (especially sidecar state model).
4. **RE-RUN** focused tests, then broader suite.

## Candidate Go tests

Existing patterns show strong Go unit/integration tests in:

- `internal/app/app_test.go`
- `internal/service/admin_service_test.go`
- `internal/handler/admin_handler_test.go`
- `internal/handler/routes_test.go`

Add Phase 15-focused tests such as:

1. **Lifecycle start success**
   - Given sidecar startup handshake succeeds
   - Expect state transition to `Ready`
2. **Startup timeout -> degraded**
   - Given sidecar never becomes reachable
   - Expect bounded timeout and `Degraded` state
3. **Shutdown reaps child process**
   - Given started sidecar
   - Expect shutdown path attempts graceful stop then force-kill fallback and wait/reap
4. **Health endpoint boundary**
   - `/health` unaffected by sidecar down
5. **Ready endpoint boundary**
   - `/ready` reflects Go readiness, not sidecar compute readiness
6. **Admin overview includes sidecar diagnostics**
   - Verify state + last error + probe fields appear in overview response
7. **Manifest writer correctness**
   - atomic write, valid schema, and expected `go.base_url` field

## Candidate Flutter tests

Likely add tests under `flutter_app/test` for startup discovery behavior:

1. **Manifest parse and base URL update**
   - Given valid manifest, ensure `ApiConfig.updateBaseUrl(...)` is applied
2. **Fallback behavior for missing/invalid manifest**
   - In dev mode, fallback allowed; in production path, surface diagnosable startup issue per design
3. **No hardcoded assumption test**
   - Assert startup logic does not assume `localhost:8080` when manifest exists

## Validation command layers (execution-time)

- Focused Go tests (new/changed packages)
- Full relevant Go package tests (`go test ./internal/...`)
- Flutter targeted tests (`flutter test` with focused groups first)

---

## 6) Suggested Planner Decomposition (Ultrawork-ready)

Recommended planning decomposition for implementation phase:

1. **Sidecar runtime core (Go)**
   - process start/stop, state machine, probe loop, shutdown guarantees
2. **Manifest infrastructure (Go + Flutter bootstrap)**
   - schema, atomic write, Flutter startup read and base URL apply
3. **Health/readiness boundary hardening**
   - enforce D-08/D-09 semantics in handlers and tests
4. **Admin diagnostics extension**
   - sidecar status + error summaries in overview contract
5. **Degraded availability behavior**
   - ensure Python failure does not block core gallery serving path
6. **Verification pass + hardening**
   - focused tests -> broader tests, race/timeouts tuning, docs update

Each decomposition unit should be planned as independent, test-backed vertical slices.

---

## 7) Validation Architecture (for Nyquist Validation Authoring)

Nyquist-style validation should assert system invariants rather than implementation details.

## Validation domains

1. **Lifecycle invariants**
   - no orphan sidecar process after Go shutdown
   - startup timeout is bounded and diagnosable
2. **Contract invariants**
   - Flutter discovers Go endpoint via manifest
   - hardcoded port is not required for success path
3. **Boundary invariants**
   - `/health` and `/ready` semantics remain Go-scoped
   - sidecar details appear in admin diagnostics, not base health
4. **Availability invariants**
   - sidecar failure leads to degraded mode, not total service blackout
5. **Observability invariants**
   - latest sidecar probe/error state available through admin overview

## Suggested Nyquist artifact shape

- **Scenario matrix**:
  - normal startup
  - sidecar startup timeout
  - sidecar crash after startup
  - manifest missing/corrupted
  - shutdown during startup
- **Assertions per scenario**:
  - endpoint status codes
  - state transitions
  - diagnostics payload fields
  - process cleanup outcomes

---

## Dependencies and Sequencing Considerations

## Intra-repo sequencing

- Must align with existing app lifecycle wiring in `internal/app`.
- Must preserve current admin overview contract style and auth flow.
- Must not force immediate duplicate-computation migration (Phase 16 responsibility).

## External/runtime dependencies

- Python sidecar runtime process contract (startup message/readiness endpoint/shutdown endpoint).
- Windows process behavior (graceful then force terminate fallback).
- File-system location strategy for manifest that works in desktop packaging and development.

## Risk-driven sequencing

Implement and test in this order:

1. lifecycle + shutdown correctness
2. manifest discovery
3. diagnostics and degraded semantics

This order minimizes hidden startup/shutdown failures early.

---

## Common Pitfalls to Guard in the Plan

1. Treating Python availability as `/health` truth source (violates D-08/D-09).
2. Leaking Python endpoint to Flutter contract (violates D-06).
3. Hardcoding `localhost:8080` as production behavior (violates COMP-06 / D-05).
4. Missing process reaping/wait path (zombie risk).
5. Non-atomic manifest writes causing intermittent Flutter boot failures.
6. Expanding scope into duplicate algorithm migration in Phase 15.

---

## Atomic Commit Recommendations (for later execution plan)

Use small, behavior-scoped commits with tests:

1. **Commit A: sidecar runtime skeleton + failing tests**
2. **Commit B: lifecycle start/stop/probe implementation + passing tests**
3. **Commit C: runtime manifest writer + parser/bootstrap integration + tests**
4. **Commit D: admin overview sidecar diagnostics contract + tests**
5. **Commit E: health/readiness boundary assertions + tests**
6. **Commit F: degraded-availability behavior hardening + regression tests**
7. **Commit G: documentation updates for manifest and diagnostics contract**

This commit shape keeps rollback risk low and review quality high.

---

## Planning Readiness Checklist

Phase 15 is ready to plan when the plan explicitly includes:

- D-01..D-15 boundary lock and non-goals
- sidecar lifecycle states and transitions
- manifest schema/location/update/cleanup strategy
- health/readiness/admin diagnostics contract boundaries
- RED/GREEN test list mapped to each behavior slice
- degraded-mode acceptance criteria
- validation architecture scenario matrix for Nyquist
- atomic commit sequence for execution
