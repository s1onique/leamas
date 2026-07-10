//go:build unix || darwin || linux

package execution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// newProcessVerifier creates a new process verifier for a test.
func newProcessVerifier(t *testing.T) (*processVerifier, func()) {
	t.Helper()

	helperPath, err := getHelperPath()
	if err != nil {
		t.Fatalf("test helper not found: %v", err)
	}
	_ = helperPath

	f, err := os.CreateTemp("", "leamas-pid-manifest-*.jsonl")
	if err != nil {
		t.Fatalf("failed to create manifest file: %v", err)
	}
	f.Close()

	return &processVerifier{
			manifestFile: f.Name(),
			t:            t,
		}, func() {
			_ = os.Remove(f.Name())
		}
}

// ManifestFile returns the path to the PID manifest file.
func (v *processVerifier) ManifestFile() string {
	return v.manifestFile
}

// parseManifest reads and parses the PID manifest.
func (v *processVerifier) parseManifest() ([]PIDRecord, error) {
	v.t.Helper()

	f, err := os.Open(v.manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer f.Close()

	records := make([]PIDRecord, 0)
	scanner := bufio.NewScanner(f)

	const maxLineLength = 1024
	buf := make([]byte, maxLineLength)
	scanner.Buffer(buf, maxLineLength)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec PIDRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("line %d: malformed JSON: %w", lineNum, err)
		}
		if rec.Role == "" || rec.PID == 0 {
			return nil, fmt.Errorf("line %d: missing required fields", lineNum)
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	v.records = records
	return records, nil
}

// requireNonEmptyManifest asserts that the manifest contains at least one record.
func (v *processVerifier) requireNonEmptyManifest() {
	v.t.Helper()
	if len(v.records) == 0 {
		v.t.Fatal("manifest is empty - no PID records recorded")
	}
}

// requireExpectedRoles asserts that the manifest contains the expected roles.
func (v *processVerifier) requireExpectedRoles(mode string) {
	v.t.Helper()

	expected, ok := expectedRolesForMode[mode]
	if !ok {
		return
	}

	recordedRoles := make(map[string]int)
	for _, rec := range v.records {
		recordedRoles[rec.Role]++
	}

	for _, role := range expected {
		if count := recordedRoles[role]; count == 0 {
			v.t.Errorf("expected role %q not found in manifest", role)
		}
	}
}

// requireExactlyOnePGID asserts that all records share the same PGID.
func (v *processVerifier) requireExactlyOnePGID() int {
	v.t.Helper()
	if len(v.records) == 0 {
		v.t.Fatal("no records to check PGID")
	}

	pgid := v.records[0].PGID
	for _, rec := range v.records[1:] {
		if rec.PGID != pgid {
			v.t.Errorf("PGID mismatch: first=%d, got=%d", pgid, rec.PGID)
		}
	}
	return pgid
}

// getProcessGroups returns unique process group IDs from the manifest.
func (v *processVerifier) getProcessGroups() map[int]struct{} {
	groups := make(map[int]struct{})
	for _, rec := range v.records {
		groups[rec.PGID] = struct{}{}
	}
	return groups
}
