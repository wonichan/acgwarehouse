package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewInitializesAdminActionsWithWorkerManager(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeTestConfig(t, dbPath)

	app, err := New(cfgPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	jobID, err := app.adminSvc.TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("TriggerScan() error = %v", err)
	}
	if jobID <= 0 {
		t.Fatalf("TriggerScan() jobID = %d, want > 0", jobID)
	}
}

func writeTestConfig(t *testing.T, dbPath string) string {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := []byte("server:\n  host: 127.0.0.1\n  port: 0\n  env: test\ndatabase:\n  type: sqlite\n  path: " + dbPath + "\nstorage:\n  scan_roots: []\nai: {}\ncos: {}\nadmin:\n  username: \"\"\n  password: \"\"\nworker_pool:\n  worker_count: 1\n  queue_size: 8\n  refill_interval_seconds: 1\n  refill_threshold: 0.5\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return cfgPath
}
