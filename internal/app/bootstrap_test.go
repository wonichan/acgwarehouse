package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

func TestPackagedSidecarBootstrapUsesExplicitExecutableAndPort(t *testing.T) {
	rootDir := t.TempDir()
	sidecarPath := filepath.Join(rootDir, "runtime", "python-sidecar", "acgwarehouse-sidecar.exe")
	if err := os.MkdirAll(filepath.Dir(sidecarPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sidecarPath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv(portableRuntimeRootEnv, rootDir)
	t.Setenv("ACG_SIDECAR_EXECUTABLE", filepath.Join("runtime", "python-sidecar", "acgwarehouse-sidecar.exe"))
	t.Setenv("ACG_SIDECAR_PORT", "9311")
	t.Setenv("ACG_DIAGNOSTICS_DIR", filepath.Join(rootDir, "runtime", "diagnostics"))
	t.Setenv("ACG_LOGS_DIR", filepath.Join(rootDir, "runtime", "logs"))

	capture := &bootstrapRuntimeCapture{}
	originalRuntimeFactory := newSidecarRuntime
	newSidecarRuntime = func(cfg sidecar.RuntimeConfig) sidecarRuntimeLifecycle {
		capture.cfg = cfg
		return capture
	}
	defer func() {
		newSidecarRuntime = originalRuntimeFactory
	}()

	originalStarter := startSidecarProcess
	var gotExecutable string
	var gotArgs []string
	startSidecarProcess = func(context.Context, string, []string, string) (sidecar.Process, error) {
		gotExecutable = sidecarPath
		gotArgs = []string{"--host", "127.0.0.1", "--port", "9311"}
		return noopSidecarProcess{}, nil
	}
	defer func() {
		startSidecarProcess = originalStarter
	}()

	originalHTTPDo := sidecarHTTPDo
	var probeURL string
	sidecarHTTPDo = func(req *http.Request) (*http.Response, error) {
		probeURL = req.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	defer func() {
		sidecarHTTPDo = originalHTTPDo
	}()

	app := &App{refillStopCh: make(chan struct{})}
	app.initSidecarRuntime()

	if app.sidecarBaseURL != "http://127.0.0.1:9311" {
		t.Fatalf("sidecarBaseURL = %q, want %q", app.sidecarBaseURL, "http://127.0.0.1:9311")
	}
	if _, err := capture.cfg.CommandFactory(context.Background()); err != nil {
		t.Fatalf("CommandFactory() error = %v", err)
	}
	if gotExecutable != sidecarPath {
		t.Fatalf("sidecar executable = %q, want %q", gotExecutable, sidecarPath)
	}
	if !reflect.DeepEqual(gotArgs, []string{"--host", "127.0.0.1", "--port", "9311"}) {
		t.Fatalf("sidecar args = %#v, want %#v", gotArgs, []string{"--host", "127.0.0.1", "--port", "9311"})
	}
	if err := capture.cfg.Probe(context.Background()); err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if probeURL != "http://127.0.0.1:9311/health" {
		t.Fatalf("probe URL = %q, want %q", probeURL, "http://127.0.0.1:9311/health")
	}
}

func TestPackagedSidecarBootstrapWritesDiagnosticOnStartFailure(t *testing.T) {
	rootDir := t.TempDir()
	sidecarPath := filepath.Join(rootDir, "runtime", "python-sidecar", "acgwarehouse-sidecar.exe")
	if err := os.MkdirAll(filepath.Dir(sidecarPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sidecarPath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	diagnosticsDir := filepath.Join(rootDir, "runtime", "diagnostics")
	logsDir := filepath.Join(rootDir, "runtime", "logs")

	t.Setenv(portableRuntimeRootEnv, rootDir)
	t.Setenv("ACG_SIDECAR_EXECUTABLE", sidecarPath)
	t.Setenv("ACG_SIDECAR_PORT", "9444")
	t.Setenv("ACG_DIAGNOSTICS_DIR", diagnosticsDir)
	t.Setenv("ACG_LOGS_DIR", logsDir)

	capture := &bootstrapRuntimeCapture{}
	originalRuntimeFactory := newSidecarRuntime
	newSidecarRuntime = func(cfg sidecar.RuntimeConfig) sidecarRuntimeLifecycle {
		capture.cfg = cfg
		return capture
	}
	defer func() {
		newSidecarRuntime = originalRuntimeFactory
	}()

	originalStarter := startSidecarProcess
	startSidecarProcess = func(context.Context, string, []string, string) (sidecar.Process, error) {
		return nil, errors.New("spawn failed")
	}
	defer func() {
		startSidecarProcess = originalStarter
	}()

	app := &App{refillStopCh: make(chan struct{})}
	app.initSidecarRuntime()

	err := capture.Start(context.Background())
	if err == nil {
		t.Fatal("Start() error = nil, want sidecar startup failure")
	}

	diagnosticPath := filepath.Join(diagnosticsDir, startupDiagnosticFileName)
	raw, readErr := os.ReadFile(diagnosticPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if !strings.Contains(string(raw), `"component":"python"`) {
		t.Fatalf("diagnostic payload = %s, want python component", string(raw))
	}
}

func TestStartSidecarProcessClosesLogFileOnStartFailure(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "sidecar.log")

	proc, err := startSidecarProcess(context.Background(), filepath.Join(t.TempDir(), "missing.exe"), nil, logPath)
	if err == nil {
		t.Fatal("startSidecarProcess() error = nil, want start failure")
	}
	if proc != nil {
		t.Fatal("startSidecarProcess() process != nil on start failure")
	}
	if removeErr := os.Remove(logPath); removeErr != nil {
		t.Fatalf("Remove(%q) error = %v, want closed log file handle", logPath, removeErr)
	}
}

type bootstrapRuntimeCapture struct {
	cfg sidecar.RuntimeConfig
}

func (c *bootstrapRuntimeCapture) Start(ctx context.Context) error {
	if c.cfg.CommandFactory == nil {
		return nil
	}
	_, err := c.cfg.CommandFactory(ctx)
	return err
}

func (c *bootstrapRuntimeCapture) Stop(context.Context) error { return nil }

func (c *bootstrapRuntimeCapture) State() sidecar.State { return sidecar.StateNotStarted }

func (c *bootstrapRuntimeCapture) Status() sidecar.Status { return sidecar.Status{} }

type noopSidecarProcess struct{}

func (noopSidecarProcess) Kill() error { return nil }

func (noopSidecarProcess) Wait() error { return nil }
