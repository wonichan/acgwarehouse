# Phase 20 Verification

- Phase: `20-operations-monitoring`
- Verified on: `2026-04-05`
- Status: `passed`
- Verifier scope: Phase 20 must-haves for `OPS-01`, `OPS-02`, plus Oracle-remediated integration seams
- Context note: this re-verification is refreshed after Oracle-remediation commits `985a72d`, `356485a`, `7e33615`, `027ed5d`, `e07d4bf`

## Inputs Reviewed

- `.planning/REQUIREMENTS.md`
- `.planning/phases/20-operations-monitoring/20-CONTEXT.md`
- `.planning/phases/20-operations-monitoring/20-01-PLAN.md`
- `.planning/phases/20-operations-monitoring/20-02-PLAN.md`
- `.planning/phases/20-operations-monitoring/20-03-PLAN.md`
- `internal/handler/admin_handler.go`
- `internal/handler/routes.go`
- `internal/handler/ws_handler.go`
- `internal/service/monitoring_event_bus.go`
- `internal/service/task_read_service.go`
- `internal/app/app.go`
- `internal/app/runtime_manifest.go`
- `flutter_app/lib/bootstrap/runtime_manifest_loader.dart`
- `flutter_app/lib/config/api_config.dart`
- `flutter_app/lib/main.dart`
- `flutter_app/lib/models/monitoring_models.dart`
- `flutter_app/lib/services/monitoring_service.dart`
- `flutter_app/lib/services/monitoring_channel_factory.dart`
- `flutter_app/lib/services/monitoring_channel_factory_io.dart`
- `flutter_app/lib/providers/monitoring_provider.dart`
- `flutter_app/lib/providers/navigation_provider.dart`
- `flutter_app/lib/app/fluent_app_shell.dart`
- `flutter_app/lib/widgets/monitoring/monitoring_workspace.dart`
- `flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart`
- `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart`
- `flutter_app/test/services/monitoring_service_test.dart`
- `flutter_app/test/providers/monitoring_provider_test.dart`

## Verification Result

Phase 20 goal achievement is re-verified as `passed` in the current repository state.

Rationale:

1. `OPS-01` remains satisfied: the desktop shell still exposes `运营监控` as the 6th navigation item, and the monitoring workspace still loads overview, batch, and task data through the provider-driven page.
2. `OPS-02` remains satisfied: the sidecar diagnostics area still exposes runtime status, recent error summary, queue/worker/pending metrics, and a guarded restart flow.
3. Oracle-blocking contract drift is resolved: backend batch payloads now carry the expected timestamp fields and the Flutter client accepts both `batches` and legacy `task_batches` envelopes plus nested restart response data.
4. Oracle-blocking auth drift is resolved: the desktop runtime manifest now emits admin Basic auth, the Flutter bootstrap loads it into `ApiConfig`, REST calls send it, and desktop WebSocket creation passes it as headers.
5. Oracle-blocking realtime drift is resolved: live monitoring events still come from the Go event bus/WebSocket path, and the Flutter provider now refreshes batch/task drilldown rows after incoming events so visible data stays aligned with server state.
6. Fresh verification commands succeeded for the Go monitoring/backend scope, full Go build, and focused Flutter Phase 20 monitoring scope including the runtime-manifest bootstrap seam.

## Requirement Traceability

| Requirement | Expected outcome | Repository evidence | Result |
|-------------|------------------|---------------------|--------|
| `OPS-01` | Admin can enter a desktop monitoring area and inspect import-task batches and task status | `flutter_app/lib/providers/navigation_provider.dart` keeps `operationsMonitoringIndex = 5` and `itemCount = 6`; `flutter_app/lib/app/fluent_app_shell.dart` mounts the `运营监控` pane item; `flutter_app/lib/widgets/monitoring/monitoring_workspace.dart` connects on page enter, disconnects on leave, and exposes reconnect/service-unavailable UX; `flutter_app/lib/providers/monitoring_provider.dart` loads overview/batches/tasks and refreshes detail rows after live events; `flutter_app/lib/services/monitoring_service.dart` accepts current batch contracts; `internal/handler/routes.go` still wires `/admin/api/task-batches`, `/admin/api/tasks`, and `/admin/api/monitoring/ws`; `internal/service/task_read_service.go` includes `created_at` / `finished_at` in batch models | `passed` |
| `OPS-02` | Admin can inspect Python sidecar runtime status, recent error summary, and use a manual restart entry | `internal/handler/admin_handler.go` still reports `interrupted_task_count` through `/admin/api/actions/sidecar/restart`; `internal/app/runtime_manifest.go` and `internal/app/app.go` publish desktop admin auth in the runtime manifest; `flutter_app/lib/bootstrap/runtime_manifest_loader.dart`, `flutter_app/lib/config/api_config.dart`, and `flutter_app/lib/main.dart` bootstrap that auth into the desktop app; `flutter_app/lib/services/monitoring_service.dart` sends Basic auth for REST and exposes WebSocket headers; `flutter_app/lib/services/monitoring_channel_factory_io.dart` passes headers during socket connect; `flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart` shows state, metrics, error summary, and restart confirmation | `passed` |

## Oracle Remediation Re-check

### Contract fixes

- Commit `985a72d` aligns backend monitoring batch payload keys and preserves timestamp fields needed by the desktop UI.
- Commit `7e33615` aligns Flutter parsing with the live backend contract by accepting both `batches` and legacy `task_batches` response envelopes, nested restart response data, and batch timestamps.
- Evidence: `internal/service/task_read_service.go` now serializes `created_at` / `finished_at`; `flutter_app/lib/models/monitoring_models.dart` parses those fields; `flutter_app/test/services/monitoring_service_test.dart` verifies batch timestamp parsing and nested restart payload parsing.

### Auth fixes

- Commit `356485a` adds `admin_basic_auth` to the runtime manifest generated by the desktop backend.
- Commit `027ed5d` carries that manifest value through Flutter bootstrap into `ApiConfig` and the monitoring service.
- Evidence: `internal/app/runtime_manifest.go` writes `go.admin_basic_auth`; `flutter_app/lib/bootstrap/runtime_manifest_loader.dart` extracts it; `flutter_app/lib/main.dart` injects `ApiConfig.adminBasicAuthHeader` into `MonitoringService`; `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart` and `flutter_app/test/services/monitoring_service_test.dart` verify the manifest/apply/send path.

### Realtime fixes

- Commit `e07d4bf` ensures desktop monitoring sockets connect with auth headers and that overview/batch/sidecar WebSocket events trigger fresh batch/task row reloads.
- Evidence: `internal/service/monitoring_event_bus.go` still publishes periodic `overview` snapshots; `internal/handler/ws_handler.go` still authenticates, subscribes, and cleans up on disconnect; `flutter_app/lib/providers/monitoring_provider.dart` now schedules detail refreshes after live events and reconnects with backoff; `flutter_app/test/providers/monitoring_provider_test.dart` verifies reconnect, auth-header propagation, and live event refresh behavior.

## Must-have Evaluation

### Plan 20-01 must-haves

- Multiple WebSocket clients can receive monitoring snapshots: still supported by `internal/service/monitoring_event_bus.go` subscriber fan-out and `internal/handler/ws_handler.go` subscription-based WebSocket pumping.
- Restart endpoint reports exact running-task impact before execution: still supported by `internal/handler/admin_handler.go` via `countRunningTasks()` before `Stop()` / `Start()`.
- WebSocket resources clean up on disconnect: still supported by `internal/handler/ws_handler.go` with `unsubscribe()` and connection close on exit.

### Plan 20-02 must-haves

- Desktop state loads overview, batches, tasks, sidecar status, and Go runtime metrics through typed models: still supported by `flutter_app/lib/models/monitoring_models.dart`, `flutter_app/lib/services/monitoring_service.dart`, and `flutter_app/lib/providers/monitoring_provider.dart`.
- Provider manages connect/disconnect and reconnect behavior: still supported by `flutter_app/lib/providers/monitoring_provider.dart` with `connect()`, `disconnect()`, and exponential backoff reconnect loop.
- Provider exposes sidecar restart action for confirmation UX: still supported by `flutter_app/lib/providers/monitoring_provider.dart` and `flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart`.

### Plan 20-03 must-haves

- Admin can navigate to `运营监控` and see batch monitoring plus sidecar diagnostics in one desktop workspace: still supported by `flutter_app/lib/app/fluent_app_shell.dart` and `flutter_app/lib/widgets/monitoring/monitoring_workspace.dart`.
- Batch rows show semantic status, progress, timestamps, and task drill-down: still supported by `flutter_app/lib/widgets/monitoring/batch_list_section.dart` plus the refreshed batch contract in `flutter_app/lib/models/monitoring_models.dart`.
- Sidecar diagnostic card shows status, metrics, error summary, and restart confirmation: still supported by `flutter_app/lib/widgets/monitoring/sidecar_diagnostic_section.dart`.
- WebSocket connection state is visible and disconnects are recoverable: still supported by `flutter_app/lib/widgets/monitoring/monitoring_workspace.dart` and `flutter_app/lib/providers/monitoring_provider.dart`.

## Fresh Verification Evidence

### Commands run

- `go test ./internal/service/ ./internal/handler/ -count=1`
- `go build ./...`
- `cd flutter_app && flutter test test/bootstrap/runtime_manifest_loader_test.dart test/providers/navigation_provider_test.dart test/app/fluent_app_shell_test.dart test/services/monitoring_service_test.dart test/providers/monitoring_provider_test.dart test/widgets/monitoring/monitoring_workspace_test.dart test/widgets/monitoring/sidecar_diagnostic_section_test.dart`

### Results

- Go monitoring/backend scope: passed
  - `ok   github.com/wonichan/acgwarehouse-backend/internal/service 7.493s`
  - `ok   github.com/wonichan/acgwarehouse-backend/internal/handler 2.282s`
- Go build: passed (`go build ./...` exited `0`)
- Flutter Phase 20 monitoring scope plus auth bootstrap seam: passed (`38` tests, `All tests passed!`)

## Conclusion

`OPS-01` and `OPS-02` are achieved in the current codebase, Oracle-remediated contract/auth/realtime blockers are re-verified as fixed, and Phase 20 is verified as `passed`.
