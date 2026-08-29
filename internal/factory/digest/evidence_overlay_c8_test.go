// SPDX-License-Identifier: Apache-2.0

// C8 typed-sentinel classification tests for the local
// evidence overlay. The classifier returns
// errEvidenceSymlink / errEvidenceNonRegular sentinel errors
// and the collector decides which stable EvidenceOmission
// reason code to record via errors.Is. C10: fixtures make
// the test artefacts Git-ignored so the production collector
// actually exercises them.
package digest

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEvidence_C8_SymlinkSentinel proves the collector uses
// errEvidenceSymlink and surfaces OmitSymlink without
// inspecting error strings. C10: the fixture makes the
// symlink path Git-ignored so the production collector
// actually exercises it.
func TestEvidence_C8_SymlinkSentinel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "ignored/")
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("/nonexistent-target",
		filepath.Join(dir, "ignored", "dead")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	// C10: confirm Git actually lists this path as ignored.
	lsOut, lsErr := RunGit(dir, []string{
		"ls-files", "-z", "--others", "--ignored",
		"--exclude-standard", "--", "ignored",
	})
	if lsErr != nil {
		t.Fatalf("ls-files: %v", lsErr)
	}
	if !strings.Contains(lsOut, "ignored/dead") {
		t.Fatalf("C10 symlink path is not Git-ignored: %q", lsOut)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	// Cross-check: InspectSymlink must return true for the
	// raw error returned by inspectEvidenceFile.
	entry, rawErr := inspectEvidenceFile(dir, "ignored/dead")
	if entry.Size != 0 || rawErr == nil {
		t.Fatalf("inspectEvidenceFile did not return an error: %+v", entry)
	}
	if !InspectSymlink(rawErr) {
		t.Errorf("C8: InspectSymlink did not classify symlink error: %v", rawErr)
	}
	if !errors.Is(rawErr, errEvidenceSymlink) {
		t.Errorf("C8: errors.Is did not classify symlink: %v", rawErr)
	}
	// And the omission reason is the stable external code.
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored/dead" && om.Reason == OmitSymlink {
			found = true
		}
	}
	if !found {
		t.Errorf("C8: symlink omission missing: %+v", overlay.Omitted)
	}
}

// TestEvidence_C8_NonRegularSentinel proves non-regular file
// modes classify through the typed sentinel. C10: the FIFO
// lives under a Git-ignored directory, but Git itself does
// not surface non-regular files via `ls-files --ignored`.
// The classification is therefore exercised through the
// inspect-time primitive directly; the production collector
// additionally relies on the typed sentinel for any
// non-regular entry it ever does see.
func TestEvidence_C8_NonRegularSentinel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "ignored/")
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := unixMkfifo(filepath.Join(dir, "ignored", "fifo"), 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	defer os.Remove(filepath.Join(dir, "ignored", "fifo"))

	// Production path: the collector must NOT silently drop a
	// listed FIFO without an omission row. We exercise the
	// classifier directly because Git does not surface FIFOs
	// through `ls-files --ignored`. The cross-check proves the
	// typed sentinel is the authority, not string matching.
	_, rawErr := inspectEvidenceFile(dir, "ignored/fifo")
	if rawErr == nil {
		t.Fatalf("inspectEvidenceFile did not error for FIFO")
	}
	if !InspectNonRegular(rawErr) {
		t.Errorf("C8: InspectNonRegular did not classify FIFO: %v", rawErr)
	}
	if !errors.Is(rawErr, errEvidenceNonRegular) {
		t.Errorf("C8: errors.Is did not classify FIFO: %v", rawErr)
	}
}

// TestEvidence_C8_InspectFailedNonSentinel proves raw fs errors
// (e.g. permission, missing) are NOT classified as the
// symlink/non-regular sentinels; they flow through as plain
// wrapped errors.
func TestEvidence_C8_InspectFailedNonSentinel(t *testing.T) {
	dir, _ := makeReporootDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	entry, rawErr := inspectEvidenceFile(dir, "ignored/does-not-exist")
	if entry.Size != 0 || rawErr == nil {
		t.Fatalf("expected error for missing path, got %+v", entry)
	}
	if InspectSymlink(rawErr) || InspectNonRegular(rawErr) {
		t.Errorf("C8: InspectSymlink/NonRegular wrongly matched fs error: %v", rawErr)
	}
	if errors.Is(rawErr, errEvidenceSymlink) ||
		errors.Is(rawErr, errEvidenceNonRegular) {
		t.Errorf("C8: errors.Is wrongly matched fs error: %v", rawErr)
	}
}
