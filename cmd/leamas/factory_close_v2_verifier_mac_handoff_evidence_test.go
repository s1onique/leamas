// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_mac_handoff_evidence_test.go
// owns the dogfood result type and the deterministic
// evidence writer for
// TestClosureCLIV2VerifierMacHandoff.
//
// Splitting the evidence writer out of the main test file
// keeps the test file under the LLM-friendly 400-line
// threshold while preserving a single closure over the
// dogfood result shape.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// macHandoffDogfoodResult captures every literal value the
// test observes. The struct is the JSON shape of the
// committed evidence file.
type macHandoffDogfoodResult struct {
	FinalCommit            string
	FinalTree              string
	BuildSourceCommit      string
	BuildSourceTree        string
	BuildSourceStatusEmpty bool
	BuildSourceDetached    bool
	BinaryPath             string
	BinarySHA256           string
	BinaryVCSRevision      string
	BinaryVCSModified      bool
	Subject                string
	SubjectTree            string
	Freeze                 string
	FreezeTree             string
	Closure                string
	ClosureTree            string
	PlanPath               string
	PlanBlob               string
	PlanSHA256             string
	ManifestPath           string
	ManifestBlob           string
	ManifestSHA256         string
	CallerHeadBefore       string
	CallerHeadAfter        string
	CallerTreeBefore       string
	CallerTreeAfter        string
	CallerStatusBeforeSHA  string
	CallerStatusAfterSHA   string
	WorktreesBeforeSHA     string
	WorktreesAfterSHA      string
	RefsBeforeSHA          string
	RefsAfterSHA           string
	StdoutBytes            int
	StderrBytes            int
	StdoutSHA256           string
	StderrSHA256           string
	ExitCode               int
	TimedOut               bool
	StdoutTruncated        bool
	StderrTruncated        bool
	RunErr                 error
	VerifierOutputPath     string
	VerifierOutputSHA256   string
	EvidencePath           string
	EvidenceSidecarPath    string
	EvidenceSHA256         string
}

var lastMacHandoffDogfood macHandoffDogfoodResult

// writeMacHandoffEvidence serialises the dogfood result
// to deterministic JSON outside the Leamas checkout, then
// computes the file's SHA-256 and writes it to a sibling
// sidecar. The same pattern as R2C-R3: the on-disk
// evidence file does NOT embed its own SHA-256; the
// sidecar holds the digest.
func writeMacHandoffEvidence(t *testing.T, r *macHandoffDogfoodResult) {
	t.Helper()
	dir := os.Getenv("LEAMAS_MAC_HANDOFF_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	jsonPath := filepath.Join(dir, "mac-handoff-evidence.json")
	sidecarPath := filepath.Join(dir, "mac-handoff-evidence.json.sha256")

	r.EvidencePath = jsonPath
	r.EvidenceSidecarPath = sidecarPath
	r.EvidenceSHA256 = "" // populated after write

	if err := writeMacHandoffEvidenceAtomic(jsonPath, r); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read evidence: %v", err)
	}
	sum := sha256HexBytes(raw)
	if err := os.WriteFile(sidecarPath, []byte(sum+"\n"), 0o644); err != nil {
		t.Fatalf("write evidence sidecar: %v", err)
	}
	r.EvidenceSHA256 = sum

	finalBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read final evidence: %v", err)
	}
	if got := sha256HexBytes(finalBytes); got != sum {
		t.Fatalf("evidence file SHA mismatch: got %s want %s", got, sum)
	}
}

// writeMacHandoffEvidenceAtomic writes r to path via a
// temp-file rename so a partial write can never leave a
// half-formed evidence file behind.
func writeMacHandoffEvidenceAtomic(path string, r *macHandoffDogfoodResult) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mac-handoff-evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp evidence: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		tmp.Close()
		return fmt.Errorf("encode evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp evidence: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename evidence: %w", err)
	}
	return nil
}
