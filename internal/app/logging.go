package app

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var goLogStdout io.Writer = os.Stdout

type LogSourcePaths struct {
	GoLogPath string
}

func ResolveLogSourcePaths() LogSourcePaths {
	logsDir := resolveLogsDir()
	return LogSourcePaths{
		GoLogPath: filepath.Join(logsDir, "go.log"),
	}
}

func SetupGoLogging(paths LogSourcePaths) (func(), error) {
	goLogPath := strings.TrimSpace(paths.GoLogPath)
	if goLogPath == "" {
		return func() {}, nil
	}

	if err := os.MkdirAll(filepath.Dir(goLogPath), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(goLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	previousWriter := log.Writer()
	log.SetOutput(io.MultiWriter(goLogStdout, file))

	var once sync.Once
	return func() {
		once.Do(func() {
			log.SetOutput(previousWriter)
			_ = file.Close()
		})
	}, nil
}

func resolveLogsDir() string {
	logsDir := strings.TrimSpace(os.Getenv(logsDirEnv))
	if logsDir != "" {
		return filepath.Clean(logsDir)
	}

	runtimeRoot := strings.TrimSpace(os.Getenv(portableRuntimeRootEnv))
	if runtimeRoot != "" {
		return resolvePortableRuntimeLayoutRoot(runtimeRoot).LogsDir
	}

	return filepath.Join("runtime", "logs")
}
