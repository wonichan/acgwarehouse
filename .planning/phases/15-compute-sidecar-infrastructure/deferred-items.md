# Deferred Items

- 2026-04-04: `go test ./internal/app/... -run "Manifest|Runtime" -count=1` currently fails due to unrelated `internal/app/app_test.go` references (`sidecarRuntime`, `prepareSidecarStartup`, `sidecarModeDegraded`) introduced outside plan `15-02`. Not fixed here per plan scope boundary.
