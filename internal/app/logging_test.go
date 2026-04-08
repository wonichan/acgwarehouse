package app

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLogSourcePathsPrefersExplicitLogsDir(t *testing.T) {
	t.Setenv(logsDirEnv, filepath.Join("explicit", "logs"))
	t.Setenv(portableRuntimeRootEnv, filepath.Join("portable", "root"))

	paths := ResolveLogSourcePaths()
	wantDir := filepath.Join("explicit", "logs")

	if paths.GoLogPath != filepath.Join(wantDir, "go.log") {
		t.Fatalf("GoLogPath = %q, want %q", paths.GoLogPath, filepath.Join(wantDir, "go.log"))
	}
}

func TestResolveLogSourcePathsFallsBackToPortableRuntimeRoot(t *testing.T) {
	t.Setenv(logsDirEnv, "")
	t.Setenv(portableRuntimeRootEnv, filepath.Join("portable", "root"))

	paths := ResolveLogSourcePaths()
	wantDir := filepath.Join("portable", "root", "runtime", "logs")

	if paths.GoLogPath != filepath.Join(wantDir, "go.log") {
		t.Fatalf("GoLogPath = %q, want %q", paths.GoLogPath, filepath.Join(wantDir, "go.log"))
	}
}

func TestResolveLogSourcePathsFallsBackToDevRuntimeLogs(t *testing.T) {
	t.Setenv(logsDirEnv, "")
	t.Setenv(portableRuntimeRootEnv, "")

	paths := ResolveLogSourcePaths()
	wantDir := filepath.Join("runtime", "logs")

	if paths.GoLogPath != filepath.Join(wantDir, "go.log") {
		t.Fatalf("GoLogPath = %q, want %q", paths.GoLogPath, filepath.Join(wantDir, "go.log"))
	}
}

func TestSetupGoLoggingCreatesFileAndTeesLogOutput(t *testing.T) {
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalStdout := goLogStdout
	log.SetFlags(0)
	defer func() {
		goLogStdout = originalStdout
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}()

	stdoutBuffer := &bytes.Buffer{}
	paths := LogSourcePaths{GoLogPath: filepath.Join(t.TempDir(), "logs", "go.log")}
	goLogStdout = stdoutBuffer

	cleanup, err := SetupGoLogging(paths)
	if err != nil {
		t.Fatalf("SetupGoLogging() error = %v", err)
	}
	defer cleanup()

	log.Print("tee check")
	cleanup()

	raw, err := os.ReadFile(paths.GoLogPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "tee check") {
		t.Fatalf("go log file content = %q, want message", content)
	}
	if !strings.Contains(stdoutBuffer.String(), "tee check") {
		t.Fatalf("stdout content = %q, want message", stdoutBuffer.String())
	}
}
