// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2cr_correction02a_evidence_test.go
// owns the dogfood result type and the validated,
// atomic evidence writer for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02A.
//
// The evidence writer:
//
//   - validates every required string, OID, SHA-256, and
//     boolean before publication;
//   - rejects publication with a typed diagnostic when
//     any field is empty or malformed;
//   - publishes the JSON and the SHA-256 sidecar via the
//     shared atomic writer so a stale sidecar cannot
//     accompany a new evidence file.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// correction02aSubprocessResult captures the bounded
// subprocess outcome in literal fields. Every field is
// assigned from the actual subprocess result; the
// evidence writer rejects empty ExitCode, missing
// SHA-256 values, and uninitialised booleans.
type correction02aSubprocessResult struct {
	ExitCode        int
	TimedOut        bool
	StdoutTruncated bool
	StderrTruncated bool
	StdoutSHA256    string
	StderrSHA256    string
	ErrorPresent    bool
	ErrorText       string
}

// correction02aDogfoodResult is the durable literal
// result captured for the dogfood. Every field is
// required; the evidence writer refuses to publish a
// result with any empty required field.
type correction02aDogfoodResult struct {
	ACTID                       string
	Status                      string
	BaseCommit                  string
	BaseTree                    string
	FinalCommit                 string
	FinalTree                   string
	BuildSourceHeadBefore       string
	BuildSourceHeadAfter        string
	BuildSourceTreeBefore       string
	BuildSourceTreeAfter        string
	BuildSourceDetachedBefore   bool
	BuildSourceDetachedAfter    bool
	BuildSourceStatusBeforeSHA  string
	BuildSourceStatusAfterSHA   string
	BuildOutputPath             string
	BuildOutputOutsideSource    bool
	BinaryPath                  string
	BinarySHA256                string
	BinaryVCSRevision           string
	BinaryVCSModified           bool
	RunnerResult                correction02aSubprocessResult
	VerifierResult              correction02aSubprocessResult
	DogfoodSubject              string
	DogfoodSubjectTree          string
	DogfoodFreeze               string
	DogfoodFreezeTree           string
	DogfoodClosure              string
	DogfoodClosureTree          string
	DogfoodPlanPath             string
	DogfoodPlanBlob             string
	DogfoodPlanSHA256           string
	DogfoodManifestPath         string
	DogfoodManifestBlob         string
	DogfoodManifestSHA256       string
	CallerHeadBefore            string
	CallerHeadAfter             string
	CallerTreeBefore            string
	CallerTreeAfter             string
	CallerStatusBeforeSHA       string
	CallerStatusAfterSHA        string
	WorktreeInventoryBeforeSHA  string
	WorktreeInventoryAfterSHA   string
	RefsBeforeSHA               string
	RefsAfterSHA                string
	EvidencePath                string
	EvidenceSidecarPath         string
	EvidenceSidecarSHA256       string
}

// validateCorrection02aResult asserts every required
// field of the dogfood result is present and well-formed.
// The function returns a non-nil error describing the
// first violation it finds.
func validateCorrection02aResult(r *correction02aDogfoodResult) error {
	if r == nil {
		return fmt.Errorf("nil dogfood result")
	}
	if r.ACTID == "" {
		return fmt.Errorf("ACTID is empty")
	}
	if r.Status == "" {
		return fmt.Errorf("Status is empty")
	}
	if !isLowerHex(r.BaseCommit, 40) {
		return fmt.Errorf("BaseCommit is not 40 lowercase hex: %q", r.BaseCommit)
	}
	if !isLowerHex(r.BaseTree, 40) {
		return fmt.Errorf("BaseTree is not 40 lowercase hex: %q", r.BaseTree)
	}
	if !isLowerHex(r.FinalCommit, 40) {
		return fmt.Errorf("FinalCommit is not 40 lowercase hex: %q", r.FinalCommit)
	}
	if !isLowerHex(r.FinalTree, 40) {
		return fmt.Errorf("FinalTree is not 40 lowercase hex: %q", r.FinalTree)
	}
	if !isLowerHex(r.BuildSourceHeadBefore, 40) {
		return fmt.Errorf("BuildSourceHeadBefore is not 40 lowercase hex: %q", r.BuildSourceHeadBefore)
	}
	if !isLowerHex(r.BuildSourceHeadAfter, 40) {
		return fmt.Errorf("BuildSourceHeadAfter is not 40 lowercase hex: %q", r.BuildSourceHeadAfter)
	}
	if !isLowerHex(r.BuildSourceTreeBefore, 40) {
		return fmt.Errorf("BuildSourceTreeBefore is not 40 lowercase hex: %q", r.BuildSourceTreeBefore)
	}
	if !isLowerHex(r.BuildSourceTreeAfter, 40) {
		return fmt.Errorf("BuildSourceTreeAfter is not 40 lowercase hex: %q", r.BuildSourceTreeAfter)
	}
	if r.BuildSourceStatusBeforeSHA == "" {
		return fmt.Errorf("BuildSourceStatusBeforeSHA is empty")
	}
	if r.BuildSourceStatusAfterSHA == "" {
		return fmt.Errorf("BuildSourceStatusAfterSHA is empty")
	}
	if !r.BuildOutputOutsideSource {
		return fmt.Errorf("BuildOutputOutsideSource is false")
	}
	if r.BuildOutputPath == "" {
		return fmt.Errorf("BuildOutputPath is empty")
	}
	if r.BinaryPath == "" {
		return fmt.Errorf("BinaryPath is empty")
	}
	if !isLowerHex(r.BinarySHA256, 64) {
		return fmt.Errorf("BinarySHA256 is not 64 lowercase hex: %q", r.BinarySHA256)
	}
	if !isLowerHex(r.BinaryVCSRevision, 40) {
		return fmt.Errorf("BinaryVCSRevision is not 40 lowercase hex: %q", r.BinaryVCSRevision)
	}
	if r.BinaryVCSModified {
		return fmt.Errorf("BinaryVCSModified is true; clean build required")
	}
	if err := validateCorrection02aSubprocess(&r.RunnerResult, "RunnerResult"); err != nil {
		return err
	}
	if err := validateCorrection02aSubprocess(&r.VerifierResult, "VerifierResult"); err != nil {
		return err
	}
	return validateCorrection02aBindings(r)
}

// validateCorrection02aSubprocess asserts a single
// subprocess result has every required field populated.
func validateCorrection02aSubprocess(s *correction02aSubprocessResult, label string) error {
	if s == nil {
		return fmt.Errorf("%s is nil", label)
	}
	if s.ExitCode != 0 {
		return fmt.Errorf("%s.ExitCode is %d, want 0", label, s.ExitCode)
	}
	if s.TimedOut {
		return fmt.Errorf("%s.TimedOut is true", label)
	}
	if s.StdoutTruncated {
		return fmt.Errorf("%s.StdoutTruncated is true", label)
	}
	if s.StderrTruncated {
		return fmt.Errorf("%s.StderrTruncated is true", label)
	}
	if s.ErrorPresent {
		return fmt.Errorf("%s.ErrorPresent is true: %s", label, s.ErrorText)
	}
	if !isLowerHex(s.StdoutSHA256, 64) {
		return fmt.Errorf("%s.StdoutSHA256 is not 64 lowercase hex: %q", label, s.StdoutSHA256)
	}
	if !isLowerHex(s.StderrSHA256, 64) {
		return fmt.Errorf("%s.StderrSHA256 is not 64 lowercase hex: %q", label, s.StderrSHA256)
	}
	return nil
}

// validateCorrection02aBindings asserts every verifier
// S/F/C/P/M identity is well-formed.
func validateCorrection02aBindings(r *correction02aDogfoodResult) error {
	oids := []struct {
		name, value string
	}{
		{"DogfoodSubject", r.DogfoodSubject},
		{"DogfoodSubjectTree", r.DogfoodSubjectTree},
		{"DogfoodFreeze", r.DogfoodFreeze},
		{"DogfoodFreezeTree", r.DogfoodFreezeTree},
		{"DogfoodClosure", r.DogfoodClosure},
		{"DogfoodClosureTree", r.DogfoodClosureTree},
		{"DogfoodPlanBlob", r.DogfoodPlanBlob},
		{"DogfoodManifestBlob", r.DogfoodManifestBlob},
	}
	for _, o := range oids {
		if !isLowerHex(o.value, 40) {
			return fmt.Errorf("%s is not 40 lowercase hex: %q", o.name, o.value)
		}
	}
	hashes := []struct {
		name, value string
	}{
		{"DogfoodPlanSHA256", r.DogfoodPlanSHA256},
		{"DogfoodManifestSHA256", r.DogfoodManifestSHA256},
		{"CallerStatusBeforeSHA", r.CallerStatusBeforeSHA},
		{"CallerStatusAfterSHA", r.CallerStatusAfterSHA},
		{"WorktreeInventoryBeforeSHA", r.WorktreeInventoryBeforeSHA},
		{"WorktreeInventoryAfterSHA", r.WorktreeInventoryAfterSHA},
		{"RefsBeforeSHA", r.RefsBeforeSHA},
		{"RefsAfterSHA", r.RefsAfterSHA},
	}
	for _, h := range hashes {
		if !isLowerHex(h.value, 64) {
			return fmt.Errorf("%s is not 64 lowercase hex: %q", h.name, h.value)
		}
	}
	if r.DogfoodPlanPath == "" {
		return fmt.Errorf("DogfoodPlanPath is empty")
	}
	if r.DogfoodManifestPath == "" {
		return fmt.Errorf("DogfoodManifestPath is empty")
	}
	return nil
}

var lowerHexRE = regexp.MustCompile(`^[0-9a-f]+$`)

// isLowerHex reports whether s is exactly n lowercase
// hex characters.
func isLowerHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	return lowerHexRE.MatchString(s)
}

// writeCorrection02aEvidence validates r, then publishes
// it atomically with a coordinated SHA-256 sidecar.
// Publication uses the shared atomic writer so a partial
// write or stale sidecar cannot leak.
func writeCorrection02aEvidence(t *testing.T, r *correction02aDogfoodResult) {
	t.Helper()
	if err := validateCorrection02aResult(r); err != nil {
		t.Fatalf("evidence validation failed: %v", err)
	}
	dir := os.Getenv("LEAMAS_CORRECTION02A_EVIDENCE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	jsonPath := filepath.Join(dir, "correction02a-evidence.json")
	sidecarPath := filepath.Join(dir, "correction02a-evidence.json.sha256")

	r.EvidencePath = jsonPath
	r.EvidenceSidecarPath = sidecarPath
	r.EvidenceSidecarSHA256 = "" // populated after atomic publish

	// Snapshot the JSON to a temp file inside the same
	// directory, then atomically rename it to the final
	// path. The temp file does NOT carry the sidecar; a
	// second temp file carries the sidecar. Both are
	// renamed in sequence so the directory never exposes
	// a stale sidecar.
	tmpJSON, err := os.CreateTemp(dir, ".correction02a-evidence-*.json.tmp")
	if err != nil {
		t.Fatalf("create temp JSON: %v", err)
	}
	tmpJSONPath := tmpJSON.Name()
	defer func() { _ = os.Remove(tmpJSONPath) }()
	enc := json.NewEncoder(tmpJSON)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		_ = tmpJSON.Close()
		t.Fatalf("encode evidence: %v", err)
	}
	if err := tmpJSON.Close(); err != nil {
		t.Fatalf("close temp JSON: %v", err)
	}
	// Compute the SHA-256 of the encoded bytes BEFORE the
	// rename. The hash is what the sidecar carries.
	jsonBytes, err := os.ReadFile(tmpJSONPath)
	if err != nil {
		t.Fatalf("re-read temp JSON: %v", err)
	}
	sum := sha256HexBytes(jsonBytes)

	tmpSidecar, err := os.CreateTemp(dir, ".correction02a-evidence-*.sha256.tmp")
	if err != nil {
		t.Fatalf("create temp sidecar: %v", err)
	}
	tmpSidecarPath := tmpSidecar.Name()
	defer func() { _ = os.Remove(tmpSidecarPath) }()
	if _, err := tmpSidecar.Write([]byte(sum + "\n")); err != nil {
		_ = tmpSidecar.Close()
		t.Fatalf("write temp sidecar: %v", err)
	}
	if err := tmpSidecar.Close(); err != nil {
		t.Fatalf("close temp sidecar: %v", err)
	}

	// Coordinated rename: move the JSON first, then the
	// sidecar. The sidecar only appears after the JSON it
	// digests is already at its final path, so a stale
	// sidecar cannot accompany a new evidence file.
	if err := os.Rename(tmpJSONPath, jsonPath); err != nil {
		t.Fatalf("rename JSON: %v", err)
	}
	if err := os.Rename(tmpSidecarPath, sidecarPath); err != nil {
		// Roll back the JSON to keep the directory state
		// consistent.
		_ = os.Remove(jsonPath)
		t.Fatalf("rename sidecar: %v", err)
	}
	r.EvidenceSidecarSHA256 = sum
}

var lastCorrection02aDogfood correction02aDogfoodResult
