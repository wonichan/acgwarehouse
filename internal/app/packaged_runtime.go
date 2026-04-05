package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	portableRuntimeRootEnv    = "ACG_RUNTIME_ROOT"
	startupDiagnosticFileName = "startup-error.json"
)

var startupDiagnosticAllowedComponents = map[string]struct{}{
	"go":            {},
	"python":        {},
	"startup_chain": {},
}

type PortableRuntimeLayout struct {
	RootDir               string
	AppDir                string
	RuntimeDir            string
	LogsDir               string
	DiagnosticsDir        string
	ConfigDir             string
	DataDir               string
	StorageDir            string
	LibraryDir            string
	ManifestPath          string
	StartupDiagnosticPath string
	ServerExecutablePath  string
	SidecarExecutablePath string
	FlutterExecutablePath string
}

type startupDiagnosticPayload struct {
	Component string   `json:"component"`
	Message   string   `json:"message"`
	LogPaths  []string `json:"log_paths"`
	Timestamp string   `json:"timestamp"`
}

func ResolvePortableRuntimeLayout(executablePath string) (PortableRuntimeLayout, error) {
	trimmed := strings.TrimSpace(executablePath)
	if trimmed == "" {
		return PortableRuntimeLayout{}, fmt.Errorf("portable runtime executable path is required")
	}

	return resolvePortableRuntimeLayoutRoot(filepath.Dir(filepath.Clean(trimmed))), nil
}

func ResolveRuntimeManifestPathForLayout(layout PortableRuntimeLayout) string {
	return filepath.Join(layout.RootDir, "runtime", runtimeManifestFileName)
}

func WriteStartupDiagnostic(path, component, message string, logPaths []string) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Errorf("startup diagnostic path is required")
	}

	trimmedComponent := strings.TrimSpace(component)
	if _, ok := startupDiagnosticAllowedComponents[trimmedComponent]; !ok {
		return fmt.Errorf("startup diagnostic component %q is not supported", component)
	}

	payload := startupDiagnosticPayload{
		Component: trimmedComponent,
		Message:   strings.TrimSpace(message),
		LogPaths:  append([]string(nil), logPaths...),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal startup diagnostic: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(trimmedPath), 0o755); err != nil {
		return fmt.Errorf("create startup diagnostic directory: %w", err)
	}
	if err := os.WriteFile(trimmedPath, raw, 0o600); err != nil {
		return fmt.Errorf("write startup diagnostic: %w", err)
	}

	return nil
}

func resolvePortableRuntimeLayoutRoot(rootDir string) PortableRuntimeLayout {
	cleanRoot := filepath.Clean(rootDir)
	runtimeDir := filepath.Join(cleanRoot, "runtime")
	diagnosticsDir := filepath.Join(runtimeDir, "diagnostics")
	return PortableRuntimeLayout{
		RootDir:               cleanRoot,
		AppDir:                filepath.Join(cleanRoot, "app"),
		RuntimeDir:            runtimeDir,
		LogsDir:               filepath.Join(runtimeDir, "logs"),
		DiagnosticsDir:        diagnosticsDir,
		ConfigDir:             filepath.Join(cleanRoot, "config"),
		DataDir:               filepath.Join(cleanRoot, "data"),
		StorageDir:            filepath.Join(cleanRoot, "storage"),
		LibraryDir:            filepath.Join(cleanRoot, "library"),
		ManifestPath:          filepath.Join(runtimeDir, runtimeManifestFileName),
		StartupDiagnosticPath: filepath.Join(diagnosticsDir, startupDiagnosticFileName),
		ServerExecutablePath:  filepath.Join(runtimeDir, "bin", "acgwarehouse-server.exe"),
		SidecarExecutablePath: filepath.Join(runtimeDir, "python-sidecar", "acgwarehouse-sidecar.exe"),
		FlutterExecutablePath: filepath.Join(cleanRoot, "ACGWarehouse.exe"),
	}
}
