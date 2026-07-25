// SPDX-License-Identifier: Apache-2.0

// Package authority: required_capabilities_loader_test.go pins the
// loader contract for `.factory/required-capabilities.json`.
// Loader-level tests use temporary files and are fully isolated
// from the canonical-repository contract assertion, which lives
// in correction01_capability_test.go.
package authority

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifest writes a JSON document with the supplied map and
// returns its absolute path.
func writeManifest(t *testing.T, dir string, raw map[string]int) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".factory"), 0o755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(dir, ".factory", "required-capabilities.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadRequiredValid(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, map[string]int{
		CapDigestAutoRange: 1,
	})
	rc, err := LoadRequired(path)
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if got := rc.Raw[CapDigestAutoRange]; got != 1 {
		t.Fatalf("level=%d want 1", got)
	}
}

func TestLoadRequiredMissingReturnsEmptyMap(t *testing.T) {
	rc, err := LoadRequired(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if len(rc.Raw) != 0 {
		t.Fatalf("missing file must yield empty map, got %v", rc.Raw)
	}
}

func TestLoadRequiredEmptyManifest(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, map[string]int{})
	rc, err := LoadRequired(path)
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if rc.Raw == nil || len(rc.Raw) != 0 {
		t.Fatalf("empty file must yield empty map, got %v", rc.Raw)
	}
}

func TestLoadRequiredMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".factory", "required-capabilities.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadRequired(path)
	if err == nil {
		t.Fatalf("malformed manifest must not load silently")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestLoadRequiredRejectsNonIntegerLevel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".factory", "required-capabilities.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"factory_digest_auto_range":"high"}`),
		0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadRequired(path)
	if err == nil {
		t.Fatalf("string level must not load silently")
	}
}

func TestLoadRequiredRetainsDuplicateKeysLastWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".factory", "required-capabilities.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw := `{"factory_digest_auto_range":1, "factory_digest_auto_range":7}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	rc, err := LoadRequired(path)
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if got := rc.Raw[CapDigestAutoRange]; got != 7 {
		t.Fatalf("duplicate last-wins contract broken; got %d want 7", got)
	}
}

func TestLoadRequiredRequiredCapabilityAbsent(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, map[string]int{
		"other_capability": 1,
	})
	rc, err := LoadRequired(path)
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if _, ok := rc.Raw[CapDigestAutoRange]; ok {
		t.Fatalf("absent capability must not appear: %v", rc.Raw)
	}
}

func TestLoadRequiredCanonicalFailsClosedOnMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	_, err := LoadRequiredCanonical(path)
	if err == nil {
		t.Fatalf("missing canonical file must fail closed")
	}
	if !strings.Contains(err.Error(), "read required capabilities") {
		t.Fatalf("expected read error wrapper, got %v", err)
	}
}

func TestLoadRequiredCanonicalParses(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, map[string]int{
		CapDigestAutoRange: 2,
		CapClosureProtocol: 1,
	})
	rc, err := LoadRequiredCanonical(path)
	if err != nil {
		t.Fatalf("LoadRequiredCanonical: %v", err)
	}
	if rc.Raw[CapDigestAutoRange] != 2 || rc.Raw[CapClosureProtocol] != 1 {
		t.Fatalf("canonical loader produced wrong values: %v", rc.Raw)
	}
}
