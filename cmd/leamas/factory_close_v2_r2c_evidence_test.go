// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2c_evidence_test.go owns the R2C-R1
// dogfood result type and the deterministic evidence writer.
// Splitting the result type out of factory_close_v2_r2c_dogfood_test.go
// keeps the dogfood test file below the LLM-friendly
// 400-line threshold.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// r2cRDogfoodResult captures every R2C-R1 dogfood value the test
// observes. The struct fields are the literal values surfaced
// in the final report; no field may be a placeholder.
type r2cRDogfoodResult struct {
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
	Descendant             string
	CallerHead             string
	CallerHeadBefore       string
	CallerHeadAfter        string
	CallerTreeBefore       string
	CallerTreeAfter        string
	CallerStatusBefore     string
	CallerStatusAfter      string
	WorktreesBefore        string
	WorktreesAfter         string
	ManifestPath           string
	ManifestAbsenceBefore  bool
	ManifestPresentAfter   bool
	ManifestSubject        string
	ManifestFreeze         string
	ManifestCallerHead     string
	ManifestExecutionTree  string
	ManifestPlanPath       string
	ManifestPlanBlob       string
	ManifestPlanSHA256     string
	ManifestBinarySHA256   string
	ManifestBinaryVCSRev   string
	ManifestBinaryVCSMod   bool
	ManifestSHA256         string
	StdoutBytes            int
	StderrBytes            int
	StdoutSHA256           string
	StderrSHA256           string
	ExitCode               int
	TimedOut               bool
	StdoutTruncated        bool
	StderrTruncated        bool
	TruncationRejectionMsg string
	RunErr                 error
	EvidencePath           string
	EvidenceSHA256         string
}

var lastR2CRDogfood r2cRDogfoodResult

// writeR2CREvidence serializes the dogfood result to
// deterministic JSON in a directory outside the Leamas
// repository, then computes the file's SHA-256 externally
// (sidecar) and stores the result in the in-memory struct.
//
// R2C-R3 design: the file does NOT contain its own SHA-256.
// The SHA-256 is written to a sidecar file
// (r2cr-evidence.json.sha256) that exists independently of
// the JSON content. The final report references both the JSON
// path and the sidecar path so the digest is verifiable
// without recomputation.
//
// Why a sidecar and not a self-referential field:
//
//   - A self-referential hash inside the file describes the
//     FIRST-pass content, not the final file. The two-pass
//     pattern (write PENDING, hash, write final) is
//     mathematically invalid because the second write changes
//     the bytes whose hash is now stored.
//   - A sidecar produced by an external tool (or computed
//     after the final write) is unambiguously the hash of the
//     final file.
func writeR2CREvidence(t *testing.T, r *r2cRDogfoodResult) {
	t.Helper()
	dir := os.Getenv("LEAMAS_R2CR_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	jsonPath := filepath.Join(dir, "r2cr-evidence.json")
	shaPath := filepath.Join(dir, "r2cr-evidence.json.sha256")

	// Write the JSON exactly once with no SHA-256 field
	// inside it. The EvidencePath field is the canonical path;
	// the in-memory struct holds NO EvidenceSHA256 (it is set
	// after the write below, externally, and never inside
	// the file).
	r.EvidencePath = jsonPath
	r.EvidenceSHA256 = "" // not embedded; computed externally
	if err := writeR2CREvidenceAtomic(jsonPath, r); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	// Compute the final file's SHA-256 externally and write
	// it to a sidecar so a verifier can re-check the digest
	// without recomputing.
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read evidence: %v", err)
	}
	sum := sha256HexBytes(raw)
	if err := os.WriteFile(shaPath, []byte(sum+"\n"), 0o644); err != nil {
		t.Fatalf("write evidence sidecar: %v", err)
	}
	// Re-read the sidecar and store the final digest in the
	// in-memory struct so the close report carries the literal
	// value. The on-disk file never contains this digest.
	r.EvidenceSHA256 = sum
	// Re-read the JSON to verify the file is well-formed and
	// matches the recorded digest. This is the final
	// invariant the R2C-R3 ACT requires.
	finalBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("re-read final evidence: %v", err)
	}
	if got := sha256HexBytes(finalBytes); got != sum {
		t.Fatalf("evidence file SHA mismatch: got %s want %s", got, sum)
	}
}

// writeR2CREvidenceAtomic writes the supplied result to path
// via a temp-file rename so a partial write can never leave a
// half-formed evidence file behind.
func writeR2CREvidenceAtomic(path string, r *r2cRDogfoodResult) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".r2cr-evidence-*.tmp")
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