package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_adds_tag_when_name_is_valid(t *testing.T) {
	// Given
	databasePath := t.TempDir() + "/tagctl.db"
	t.Setenv("SQLITE_PATH", databasePath)
	t.Setenv("JWT_SECRET", "")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	streams := commandStreams{stdout: &stdout, stderr: &stderr}

	// When
	err := run(context.Background(), []string{"-a", "  新标签  "}, streams)

	// Then
	if err != nil {
		t.Fatalf("run add tag error = %v, want nil", err)
	}
	if !strings.Contains(stdout.String(), "created tag") || !strings.Contains(stdout.String(), "新标签") {
		t.Fatalf("stdout = %q, want created tag message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRun_updates_tag_when_old_name_exists(t *testing.T) {
	// Given
	databasePath := t.TempDir() + "/tagctl.db"
	t.Setenv("SQLITE_PATH", databasePath)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	streams := commandStreams{stdout: &stdout, stderr: &stderr}
	if err := run(context.Background(), []string{"-a", "旧标签"}, streams); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	stdout.Reset()
	stderr.Reset()

	// When
	err := run(context.Background(), []string{"-u", "旧标签", "-name", "新标签"}, streams)

	// Then
	if err != nil {
		t.Fatalf("run update tag error = %v, want nil", err)
	}
	if !strings.Contains(stdout.String(), "updated tag") || !strings.Contains(stdout.String(), "新标签") {
		t.Fatalf("stdout = %q, want updated tag message", stdout.String())
	}
}

func TestRun_deletes_tag_when_name_exists(t *testing.T) {
	// Given
	databasePath := t.TempDir() + "/tagctl.db"
	t.Setenv("SQLITE_PATH", databasePath)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	streams := commandStreams{stdout: &stdout, stderr: &stderr}
	if err := run(context.Background(), []string{"-a", "待删除"}, streams); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	stdout.Reset()
	stderr.Reset()

	// When
	err := run(context.Background(), []string{"-d", "待删除"}, streams)

	// Then
	if err != nil {
		t.Fatalf("run delete tag error = %v, want nil", err)
	}
	if !strings.Contains(stdout.String(), "deleted tag") || !strings.Contains(stdout.String(), "待删除") {
		t.Fatalf("stdout = %q, want deleted tag message", stdout.String())
	}
}

func TestParseOperation_rejects_multiple_flags_when_one_value_is_blank(t *testing.T) {
	// Given
	stderr := bytes.Buffer{}
	args := []string{"-a", "   ", "-d", "seed"}

	// When
	_, err := parseOperation(args, &stderr)

	// Then
	if err == nil {
		t.Fatal("parseOperation error = nil, want ambiguous operation error")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("error = %v, want exactly one operation error", err)
	}
}

func TestRun_rejects_invalid_operation_arguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no operation", args: []string{}},
		{name: "multiple operations", args: []string{"-a", "tag", "-d", "tag"}},
		{name: "update missing new name", args: []string{"-u", "old"}},
		{name: "blank add name", args: []string{"-a", "   "}},
		{name: "missing delete target", args: []string{"-d", "missing"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			databasePath := t.TempDir() + "/tagctl.db"
			t.Setenv("SQLITE_PATH", databasePath)
			stdout := bytes.Buffer{}
			stderr := bytes.Buffer{}

			streams := commandStreams{stdout: &stdout, stderr: &stderr}

			// When
			err := run(context.Background(), tt.args, streams)

			// Then
			if err == nil {
				t.Fatalf("run(%v) error = nil, want error", tt.args)
			}
		})
	}
}
