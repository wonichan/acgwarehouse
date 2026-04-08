package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogStreamNewServiceCreatesExpectedState(t *testing.T) {
	t.Parallel()

	svc := NewLogStreamService("go.log")
	if svc == nil {
		t.Fatal("NewLogStreamService() returned nil")
	}
	if svc.goLogPath != "go.log" {
		t.Fatalf("goLogPath = %q, want %q", svc.goLogPath, "go.log")
	}
	if svc.subscribers == nil {
		t.Fatal("subscribers should be initialized")
	}
	if svc.buffers == nil {
		t.Fatal("buffers should be initialized")
	}
	if got := svc.buffers[LogSourceGo]; got == nil {
		t.Fatal("go buffer should be initialized")
	}
}

func TestLogStreamSubscribeReturnsSnapshotThenLines(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"boot", "ready"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	ch, unsubscribe := svc.Subscribe(LogSourceGo, 2)
	defer unsubscribe()

	snapshot := waitForLogEvent(t, ch, "snapshot")
	assertSnapshotPayload(t, snapshot, []string{"boot", "ready"})

	appendLogLines(t, goLog, []string{"worker started"})

	line := waitForLogEvent(t, ch, "line")
	if line.Source != string(LogSourceGo) {
		t.Fatalf("line.Source = %q, want %q", line.Source, LogSourceGo)
	}
	if line.Payload != "worker started" {
		t.Fatalf("line.Payload = %q, want %q", line.Payload, "worker started")
	}
	if line.Severity != "normal" {
		t.Fatalf("line.Severity = %q, want %q", line.Severity, "normal")
	}
	if line.Timestamp.IsZero() {
		t.Fatal("line.Timestamp should be set")
	}
}

func TestLogStreamLineEventIncludesDetectedSeverity(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"boot"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	ch, unsubscribe := svc.Subscribe(LogSourceGo, 1)
	defer unsubscribe()
	_ = waitForLogEvent(t, ch, "snapshot")

	appendLogLines(t, goLog, []string{"warning: retrying request"})

	line := waitForLogEvent(t, ch, "line")
	if line.Severity != "warning" {
		t.Fatalf("line.Severity = %q, want %q", line.Severity, "warning")
	}
}

func TestLogStreamSubscribeToMissingFileReturnsStatusEvent(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.log")
	svc := NewLogStreamService(missing)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	ch, unsubscribe := svc.Subscribe(LogSourceGo, 10)
	defer unsubscribe()

	event := waitForLogEvent(t, ch, "status")
	if event.Payload != "log file not found" {
		t.Fatalf("status.Payload = %q, want %q", event.Payload, "log file not found")
	}
}

func TestLogStreamUnsubscribeStopsDelivery(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"boot"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	ch, unsubscribe := svc.Subscribe(LogSourceGo, 1)
	_ = waitForLogEvent(t, ch, "snapshot")

	unsubscribe()

	appendLogLines(t, goLog, []string{"after unsubscribe"})

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after unsubscribe")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for unsubscribed channel to close")
	}
}

func TestLogStreamFileTruncationHandledGracefully(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"before truncate", "old tail"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	ch, unsubscribe := svc.Subscribe(LogSourceGo, 10)
	defer unsubscribe()
	assertSnapshotPayload(t, waitForLogEvent(t, ch, "snapshot"), []string{"before truncate", "old tail"})

	writeLogLines(t, goLog, []string{"fresh start"}, true)

	event := waitForLogEvent(t, ch, "snapshot")
	assertSnapshotPayload(t, event, []string{"fresh start"})

	appendLogLines(t, goLog, []string{"after truncate"})
	line := waitForLogEvent(t, ch, "line")
	if line.Payload != "after truncate" {
		t.Fatalf("line.Payload = %q, want %q", line.Payload, "after truncate")
	}
}

func TestLogStreamBufferRetainsRecentLines(t *testing.T) {
	t.Parallel()

	buffer := newRingBuffer(3)
	for _, line := range []string{"one", "two", "three", "four"} {
		buffer.add(line)
	}

	got := buffer.last(10)
	want := []string{"two", "three", "four"}
	if len(got) != len(want) {
		t.Fatalf("len(last) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("last[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLogStreamDetectSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "error", line: "fatal: worker failed", want: "error"},
		{name: "warning", line: "warning: retrying request", want: "warning"},
		{name: "normal", line: "server started successfully", want: "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := detectSeverity(tt.line); got != tt.want {
				t.Fatalf("detectSeverity(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestReadLastLinesFromPathReturnsRequestedTail(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tail.log")
	writeLogLines(t, path, []string{"one", "two", "three", "four"}, false)

	got, err := readLastLinesFromPath(path, 2)
	if err != nil {
		t.Fatalf("readLastLinesFromPath() error = %v", err)
	}
	want := []string{"three", "four"}
	if len(got) != len(want) {
		t.Fatalf("len(readLastLinesFromPath) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("readLastLinesFromPath[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadLastLinesUsesTailReader(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "tail.log")
	writeLogLines(t, path, []string{"one", "two", "three"}, false)

	got := readLastLines(path, 1)
	if len(got) != 1 || got[0] != "three" {
		t.Fatalf("readLastLines() = %v, want [three]", got)
	}
}

func TestLogStreamStopCleansUpGoroutines(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"boot"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	svc.Start(ctx)
	svc.Stop()
	cancel()

	if svc.cancel != nil {
		t.Fatal("cancel should be cleared after Stop")
	}
	if svc.running[LogSourceGo] {
		t.Fatal("go source should not be running after Stop")
	}
}

func TestLogStreamMultipleSubscribers(t *testing.T) {
	t.Parallel()

	goLog := filepath.Join(t.TempDir(), "go.log")
	writeLogLines(t, goLog, []string{"boot"}, false)

	svc := NewLogStreamService(goLog)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	svc.Start(ctx)
	defer svc.Stop()

	first, firstUnsubscribe := svc.Subscribe(LogSourceGo, 1)
	defer firstUnsubscribe()
	second, secondUnsubscribe := svc.Subscribe(LogSourceGo, 1)
	defer secondUnsubscribe()

	assertSnapshotPayload(t, waitForLogEvent(t, first, "snapshot"), []string{"boot"})
	assertSnapshotPayload(t, waitForLogEvent(t, second, "snapshot"), []string{"boot"})

	appendLogLines(t, goLog, []string{"shared line"})

	if event := waitForLogEvent(t, first, "line"); event.Payload != "shared line" {
		t.Fatalf("first.Payload = %q, want %q", event.Payload, "shared line")
	}
	if event := waitForLogEvent(t, second, "line"); event.Payload != "shared line" {
		t.Fatalf("second.Payload = %q, want %q", event.Payload, "shared line")
	}
}

func waitForLogEvent(t *testing.T, ch <-chan LogEvent, wantType string) LogEvent {
	t.Helper()

	deadline := time.After(3 * time.Second)
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before expected event")
			}
			if event.Type == wantType {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q event", wantType)
		}
	}
}

func assertSnapshotPayload(t *testing.T, event LogEvent, want []string) {
	t.Helper()

	if event.Type != "snapshot" {
		t.Fatalf("event.Type = %q, want %q", event.Type, "snapshot")
	}
	var got []string
	if err := json.Unmarshal([]byte(event.Payload), &got); err != nil {
		t.Fatalf("json.Unmarshal(snapshot payload) error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(snapshot) = %d, want %d (payload=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("snapshot[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func writeLogLines(t *testing.T, path string, lines []string, truncate bool) {
	t.Helper()

	flag := os.O_CREATE | os.O_WRONLY
	if truncate {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_APPEND
	}

	f, err := os.OpenFile(path, flag, 0o600)
	if err != nil {
		t.Fatalf("os.OpenFile(%q) error = %v", path, err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(strings.TrimRight(line, "\n") + "\n"); err != nil {
			t.Fatalf("WriteString(%q) error = %v", path, err)
		}
	}
}

func appendLogLines(t *testing.T, path string, lines []string) {
	t.Helper()
	writeLogLines(t, path, lines, false)
}
