package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildRuntimeManifestPayloadIncludesRequiredFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 9, 30, 15, 0, time.UTC)
	payload, err := BuildRuntimeManifestPayload("http://127.0.0.1:51423", now)
	if err != nil {
		t.Fatalf("BuildRuntimeManifestPayload() error = %v", err)
	}

	if payload.Version != RuntimeManifestSchemaVersion {
		t.Fatalf("Version = %d, want %d", payload.Version, RuntimeManifestSchemaVersion)
	}
	if payload.GeneratedAt != now.Format(time.RFC3339) {
		t.Fatalf("GeneratedAt = %q, want %q", payload.GeneratedAt, now.Format(time.RFC3339))
	}
	if payload.Go.BaseURL != "http://127.0.0.1:51423" {
		t.Fatalf("Go.BaseURL = %q, want %q", payload.Go.BaseURL, "http://127.0.0.1:51423")
	}
	if !payload.Go.Ready {
		t.Fatal("Go.Ready = false, want true")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(raw), "python") {
		t.Fatalf("manifest payload leaked python endpoint field: %s", string(raw))
	}
}

func TestWriteRuntimeManifestAtomicUsesTempThenRename(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "runtime-manifest.json")
	payload, err := BuildRuntimeManifestPayload("http://127.0.0.1:51423", time.Now().UTC())
	if err != nil {
		t.Fatalf("BuildRuntimeManifestPayload() error = %v", err)
	}

	var renameCalled bool
	originalRename := runtimeManifestRenameFile
	runtimeManifestRenameFile = func(oldpath, newpath string) error {
		renameCalled = true
		if !strings.Contains(filepath.Base(oldpath), "runtime-manifest") {
			t.Fatalf("rename oldpath = %q, want temp runtime-manifest file", oldpath)
		}
		if newpath != manifestPath {
			t.Fatalf("rename newpath = %q, want %q", newpath, manifestPath)
		}
		return originalRename(oldpath, newpath)
	}
	t.Cleanup(func() {
		runtimeManifestRenameFile = originalRename
	})

	if err := WriteRuntimeManifestAtomic(manifestPath, payload); err != nil {
		t.Fatalf("WriteRuntimeManifestAtomic() error = %v", err)
	}
	if !renameCalled {
		t.Fatal("WriteRuntimeManifestAtomic() did not call rename")
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var got RuntimeManifestPayload
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; raw=%s", err, string(raw))
	}
	if got.Go.BaseURL == "" {
		t.Fatal("manifest Go.BaseURL is empty")
	}
}

func TestWriteRuntimeManifestAtomicDoesNotLeavePartialJSONOnRenameFailure(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "runtime-manifest.json")
	original := []byte(`{"version":1,"go":{"base_url":"http://127.0.0.1:50000","ready":true},"generated_at":"2026-04-04T09:30:15Z"}`)
	if err := os.WriteFile(manifestPath, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalRename := runtimeManifestRenameFile
	runtimeManifestRenameFile = func(string, string) error {
		return errors.New("rename failed")
	}
	t.Cleanup(func() {
		runtimeManifestRenameFile = originalRename
	})

	payload, err := BuildRuntimeManifestPayload("http://127.0.0.1:51423", time.Now().UTC())
	if err != nil {
		t.Fatalf("BuildRuntimeManifestPayload() error = %v", err)
	}

	err = WriteRuntimeManifestAtomic(manifestPath, payload)
	if err == nil {
		t.Fatal("WriteRuntimeManifestAtomic() error = nil, want rename failure")
	}

	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("manifest content changed after rename failure, got=%s want=%s", string(got), string(original))
	}
}
