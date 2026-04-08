package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPortableRuntimeLayoutResolvesBundleLocalPaths(t *testing.T) {
	t.Parallel()

	layout, err := ResolvePortableRuntimeLayout(`C:/bundle/ACGWarehouse.exe`)
	if err != nil {
		t.Fatalf("ResolvePortableRuntimeLayout() error = %v", err)
	}

	if layout.RootDir != filepath.Clean(`C:/bundle`) {
		t.Fatalf("RootDir = %q, want %q", layout.RootDir, filepath.Clean(`C:/bundle`))
	}
	if layout.AppDir != filepath.Join(layout.RootDir, "app") {
		t.Fatalf("AppDir = %q, want %q", layout.AppDir, filepath.Join(layout.RootDir, "app"))
	}
	if layout.RuntimeDir != filepath.Join(layout.RootDir, "runtime") {
		t.Fatalf("RuntimeDir = %q, want %q", layout.RuntimeDir, filepath.Join(layout.RootDir, "runtime"))
	}
	if layout.LogsDir != filepath.Join(layout.RuntimeDir, "logs") {
		t.Fatalf("LogsDir = %q, want %q", layout.LogsDir, filepath.Join(layout.RuntimeDir, "logs"))
	}
	if layout.DiagnosticsDir != filepath.Join(layout.RuntimeDir, "diagnostics") {
		t.Fatalf("DiagnosticsDir = %q, want %q", layout.DiagnosticsDir, filepath.Join(layout.RuntimeDir, "diagnostics"))
	}
	if layout.ConfigDir != filepath.Join(layout.RootDir, "config") {
		t.Fatalf("ConfigDir = %q, want %q", layout.ConfigDir, filepath.Join(layout.RootDir, "config"))
	}
	if layout.DataDir != filepath.Join(layout.RootDir, "data") {
		t.Fatalf("DataDir = %q, want %q", layout.DataDir, filepath.Join(layout.RootDir, "data"))
	}
	if layout.StorageDir != filepath.Join(layout.RootDir, "storage") {
		t.Fatalf("StorageDir = %q, want %q", layout.StorageDir, filepath.Join(layout.RootDir, "storage"))
	}
	if layout.LibraryDir != filepath.Join(layout.RootDir, "library") {
		t.Fatalf("LibraryDir = %q, want %q", layout.LibraryDir, filepath.Join(layout.RootDir, "library"))
	}
	if layout.ManifestPath != filepath.Join(layout.RuntimeDir, runtimeManifestFileName) {
		t.Fatalf("ManifestPath = %q, want %q", layout.ManifestPath, filepath.Join(layout.RuntimeDir, runtimeManifestFileName))
	}
	if layout.StartupDiagnosticPath != filepath.Join(layout.DiagnosticsDir, "startup-error.json") {
		t.Fatalf("StartupDiagnosticPath = %q, want %q", layout.StartupDiagnosticPath, filepath.Join(layout.DiagnosticsDir, "startup-error.json"))
	}
}

func TestResolveRuntimeManifestPathPrefersPortableRootWhenConfigured(t *testing.T) {
	t.Setenv(runtimeManifestEnvPath, "")
	t.Setenv(portableRuntimeRootEnv, `C:/bundle`)

	got := ResolveRuntimeManifestPath()
	want := filepath.Join(filepath.Clean(`C:/bundle`), "runtime", runtimeManifestFileName)
	if got != want {
		t.Fatalf("ResolveRuntimeManifestPath() = %q, want %q", got, want)
	}
}

func TestResolveRuntimeManifestPathFallsBackToTempWithoutPortableRoot(t *testing.T) {
	t.Setenv(runtimeManifestEnvPath, "")
	t.Setenv(portableRuntimeRootEnv, "")

	got := ResolveRuntimeManifestPath()
	want := filepath.Join(os.TempDir(), "acgwarehouse", runtimeManifestFileName)
	if got != want {
		t.Fatalf("ResolveRuntimeManifestPath() = %q, want %q", got, want)
	}
}

func TestWriteStartupDiagnosticWritesStructuredJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "runtime", "diagnostics", "startup-error.json")
	logPaths := []string{"runtime/logs/go.log"}

	if err := WriteStartupDiagnostic(path, "go", "startup failed", logPaths); err != nil {
		t.Fatalf("WriteStartupDiagnostic() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload struct {
		Component string   `json:"component"`
		Message   string   `json:"message"`
		LogPaths  []string `json:"log_paths"`
		Timestamp string   `json:"timestamp"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Component != "go" {
		t.Fatalf("Component = %q, want %q", payload.Component, "go")
	}
	if payload.Message != "startup failed" {
		t.Fatalf("Message = %q, want %q", payload.Message, "startup failed")
	}
	if len(payload.LogPaths) != len(logPaths) {
		t.Fatalf("len(LogPaths) = %d, want %d", len(payload.LogPaths), len(logPaths))
	}
	if payload.Timestamp == "" {
		t.Fatal("Timestamp is empty")
	}
}

func TestWriteStartupDiagnosticRejectsUnknownComponent(t *testing.T) {
	t.Parallel()

	err := WriteStartupDiagnostic(filepath.Join(t.TempDir(), "startup-error.json"), "flutter", "boom", nil)
	if err == nil {
		t.Fatal("WriteStartupDiagnostic() error = nil, want invalid component error")
	}
}
