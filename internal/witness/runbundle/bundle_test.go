// Package runbundle provides local run bundle creation and validation.
package runbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateBundleCreatesExpectedLayout(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-smoke01")
	now := time.Date(2026, 7, 9, 7, 17, 4, 0, time.UTC)

	bundle, err := Create(CreateOptions{
		Root:  root,
		RunID: runID,
		Now:   func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	expectedPath := filepath.Join(root, string(runID))
	if bundle.Path != expectedPath {
		t.Errorf("bundle.Path = %q, want %q", bundle.Path, expectedPath)
	}
	if bundle.ID != runID {
		t.Errorf("bundle.ID = %q, want %q", bundle.ID, runID)
	}
	if bundle.Root != root {
		t.Errorf("bundle.Root = %q, want %q", bundle.Root, root)
	}

	// Check all expected subdirectories exist
	expectedDirs := []string{"claims", "evidence", "digests", "traces", "verifier-results"}
	for _, subdir := range expectedDirs {
		dirPath := filepath.Join(expectedPath, subdir)
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("subdirectory %s does not exist: %v", subdir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s exists but is not a directory", subdir)
		}
	}
}

func TestCreateBundleWritesMetadata(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-smoke01")
	now := time.Date(2026, 7, 9, 7, 17, 4, 0, time.UTC)

	_, err := Create(CreateOptions{
		Root:     root,
		RunID:    runID,
		Now:      func() time.Time { return now },
		ToolName: "leamas",
		Version:  "v0.1.0",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Read and parse metadata
	metadataPath := filepath.Join(root, string(runID), "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read metadata.json: %v", err)
	}

	meta, err := StrictDecode(data)
	if err != nil {
		t.Fatalf("failed to decode metadata.json: %v", err)
	}

	if meta.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", meta.SchemaVersion, SchemaVersion)
	}
	if meta.RunID != runID {
		t.Errorf("run_id = %q, want %q", meta.RunID, runID)
	}

	expectedTime := now.Format(time.RFC3339Nano)
	actualTime := meta.CreatedAt.Format(time.RFC3339Nano)
	if actualTime != expectedTime {
		t.Errorf("created_at = %q, want %q", actualTime, expectedTime)
	}

	if meta.Tool.Name != "leamas" {
		t.Errorf("tool.name = %q, want %q", meta.Tool.Name, "leamas")
	}
	if meta.Tool.Version != "v0.1.0" {
		t.Errorf("tool.version = %q, want %q", meta.Tool.Version, "v0.1.0")
	}

	if !meta.Doctrine.LocalOnly {
		t.Error("doctrine.local_only should be true")
	}
	if !meta.Doctrine.ReadOnly {
		t.Error("doctrine.read_only should be true")
	}
	if !meta.Doctrine.NoDatabase {
		t.Error("doctrine.no_database should be true")
	}
}

func TestCreateBundleUsesDeterministicClock(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-test01")
	now := time.Date(2026, 7, 9, 7, 17, 4, 0, time.UTC)

	bundle1, err := Create(CreateOptions{
		Root:  root,
		RunID: runID,
		Now:   func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	opened, meta, err := Open(root, runID)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	if opened.ID != bundle1.ID {
		t.Errorf("opened.ID = %q, want %q", opened.ID, bundle1.ID)
	}
	if meta.CreatedAt.Unix() != now.Unix() {
		t.Errorf("created_at mismatch: got %v, want %v", meta.CreatedAt, now)
	}
}

func TestCreateBundleRejectsUnsafeRunID(t *testing.T) {
	root := t.TempDir()
	unsafeIDs := []RunID{"", "../escape", "/absolute", ".", ".."}

	for _, id := range unsafeIDs {
		t.Run(string(id), func(t *testing.T) {
			_, err := Create(CreateOptions{Root: root, RunID: id})
			if err == nil {
				t.Errorf("Create() should reject unsafe run ID %q", id)
			}
		})
	}
}

func TestCreateBundleRejectsEmptyRoot(t *testing.T) {
	_, err := Create(CreateOptions{Root: "", RunID: "run-20260709T071704Z-smoke01"})
	if err != ErrEmptyRoot {
		t.Errorf("Create() should reject empty root, got: %v", err)
	}
}

func TestCreateBundleDoesNotCreateOutsideRoot(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-safe01")

	_, err := Create(CreateOptions{Root: root, RunID: runID})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	expectedPath := filepath.Join(root, string(runID))
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Errorf("bundle should exist at %s: %v", expectedPath, err)
	}
	if !info.IsDir() {
		t.Errorf("bundle at %s should be a directory", expectedPath)
	}
}

func TestOpenBundleReadsMetadata(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-open01")
	now := time.Date(2026, 7, 9, 7, 17, 4, 0, time.UTC)

	_, err := Create(CreateOptions{
		Root:     root,
		RunID:    runID,
		Now:      func() time.Time { return now },
		ToolName: "leamas",
		Version:  "v0.1.0",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	bundle, meta, err := Open(root, runID)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	if bundle.ID != runID {
		t.Errorf("bundle.ID = %q, want %q", bundle.ID, runID)
	}
	if bundle.Root != root {
		t.Errorf("bundle.Root = %q, want %q", bundle.Root, root)
	}
	if meta.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", meta.SchemaVersion, SchemaVersion)
	}
	if meta.RunID != runID {
		t.Errorf("run_id = %q, want %q", meta.RunID, runID)
	}
}

func TestOpenBundleRejectsUnknownMetadataFields(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-unknown01")

	bundlePath := filepath.Join(root, string(runID))
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	badMetadata := `{
  "schema_version": "leamas.runbundle.v1",
  "run_id": "run-20260709T071704Z-unknown01",
  "created_at": "2026-07-09T07:17:04Z",
  "tool": {"name": "leamas", "version": ""},
  "doctrine": {"local_only": true, "read_only": true, "no_database": true},
  "unknown_field": "should cause error"
}`
	metadataPath := filepath.Join(bundlePath, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(badMetadata), 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	_, _, err := Open(root, runID)
	if err == nil {
		t.Error("Open() should reject metadata with unknown fields")
	}
}

func TestOpenBundleRejectsSchemaVersionMismatch(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-schema01")

	bundlePath := filepath.Join(root, string(runID))
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	badMetadata := Metadata{
		SchemaVersion: "leamas.runbundle.v99",
		RunID:         runID,
		CreatedAt:     time.Now(),
		Tool:          ToolInfo{Name: "leamas"},
		Doctrine:      Doctrine{LocalOnly: true, ReadOnly: true, NoDatabase: true},
	}
	data, _ := json.Marshal(badMetadata)
	metadataPath := filepath.Join(bundlePath, "metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	_, _, err := Open(root, runID)
	if err == nil {
		t.Error("Open() should reject schema version mismatch")
	}
}

func TestOpenBundleRejectsRunIDMismatch(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-mismatch01")
	wrongID := RunID("run-20260709T071704Z-wrong01")

	bundlePath := filepath.Join(root, string(runID))
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	badMetadata := Metadata{
		SchemaVersion: SchemaVersion,
		RunID:         wrongID,
		CreatedAt:     time.Now(),
		Tool:          ToolInfo{Name: "leamas"},
		Doctrine:      Doctrine{LocalOnly: true, ReadOnly: true, NoDatabase: true},
	}
	data, _ := json.Marshal(badMetadata)
	metadataPath := filepath.Join(bundlePath, "metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	_, _, err := Open(root, runID)
	if err == nil {
		t.Error("Open() should reject run ID mismatch")
	}
}

func TestOpenBundleRejectsMissingMetadata(t *testing.T) {
	root := t.TempDir()
	runID := RunID("run-20260709T071704Z-nometadata01")

	bundlePath := filepath.Join(root, string(runID))
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	_, _, err := Open(root, runID)
	if err == nil {
		t.Error("Open() should reject missing metadata")
	}
	if err != ErrMissingMetadata {
		t.Errorf("Open() should return ErrMissingMetadata, got: %v", err)
	}
}

func TestOpenBundleRejectsEmptyRoot(t *testing.T) {
	_, _, err := Open("", "run-20260709T071704Z-smoke01")
	if err != ErrEmptyRoot {
		t.Errorf("Open() should reject empty root, got: %v", err)
	}
}

func TestOpenBundleRejectsInvalidRunID(t *testing.T) {
	root := t.TempDir()
	_, _, err := Open(root, "")
	if err != ErrEmptyRunID {
		t.Errorf("Open() should reject empty run ID, got: %v", err)
	}

	_, _, err = Open(root, "invalid")
	if err != ErrRunIDNoPrefix {
		t.Errorf("Open() should reject run ID without 'run-' prefix, got: %v", err)
	}
}

func TestMetadataJSONFormat(t *testing.T) {
	now := time.Date(2026, 7, 9, 7, 17, 4, 123456789, time.UTC)
	meta := NewMetadata("run-20260709T071704Z-test", now, "leamas", "v0.1.0")

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() returned error: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{
		`"schema_version": "leamas.runbundle.v1"`,
		`"run_id": "run-20260709T071704Z-test"`,
		`"tool"`,
		`"doctrine"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output should contain %q", field)
		}
	}
}
