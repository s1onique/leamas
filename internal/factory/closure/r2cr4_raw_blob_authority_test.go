// SPDX-License-Identifier: Apache-2.0

package closure

// r2cr4_raw_blob_authority_test.go covers the raw-blob
// authority path and the trailing-newline regression
// required by ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R2C-R4.
//
// The byte-authority contract requires:
//   - the manifest plan_sha256 equals SHA-256 of the raw
//     blob bytes returned by `git cat-file blob <oid>`;
//   - the trimmed-byte SHA-256 differs so a regression
//     that silently strips the trailing newline is caught;
//   - leading whitespace and trailing spaces also
//     round-trip through the byte-authority path.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestR2CRDogfoodPlanSHAIncludesTrailingNewline proves the
// manifest plan SHA-256 equals the SHA-256 of the raw blob
// bytes returned by `git cat-file blob <oid>` and that the
// trailing newline is preserved end-to-end. The trimmed
// byte SHA-256 must differ so a regression that silently
// strips the trailing newline is caught.
//
// The fixture explicitly appends a single '\n' to the
// canonical Plan Contract v1 document so the F:P blob ends
// with exactly one trailing newline.
func TestR2CRDogfoodPlanSHAIncludesTrailingNewline(t *testing.T) {
	dir := initRepo(t)
	planPath := "docs/closure-plans/NEWLINE.json"
	// Subject is the parent of freeze in v2 topology.
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	if subjectTree == "" {
		t.Fatalf("subject tree must be non-empty")
	}
	frozenBytes := buildR2CRNewlineFixture(t, subject, subjectTree)
	if frozenBytes[len(frozenBytes)-1] != '\n' {
		t.Fatalf("fixture must end with '\\n'")
	}
	if bytes.Count(frozenBytes, []byte{'\n'}) != 1 {
		t.Fatalf("fixture must contain exactly one trailing newline, got %d newlines",
			bytes.Count(frozenBytes, []byte{'\n'}))
	}
	freeze := makeCommit(t, dir, "freeze: trailing newline fixture", map[string]string{
		planPath: string(frozenBytes),
	})

	blobOID := mustRunGit(t, dir, "rev-parse", freeze+":"+planPath)
	if len(blobOID) != 40 {
		t.Fatalf("frozen blob OID must be 40 chars, got %d", len(blobOID))
	}

	rawBytes, err := runR2CRGitRaw(context.Background(), dir, blobOID)
	if err != nil {
		t.Fatalf("runR2CRGitRaw(%s): %v", blobOID, err)
	}

	if len(rawBytes) == 0 || rawBytes[len(rawBytes)-1] != '\n' {
		t.Fatalf("raw blob must end with '\\n', got last byte 0x%02x", rawBytes[len(rawBytes)-1])
	}

	if !bytes.Equal(rawBytes, frozenBytes) {
		t.Fatalf("raw blob bytes disagree with in-memory fixture: raw=%d want=%d",
			len(rawBytes), len(frozenBytes))
	}

	trimmedBytes := bytes.TrimRight(rawBytes, " \t\r\n")
	if bytes.Equal(trimmedBytes, rawBytes) {
		t.Fatalf("trimming produced identical bytes; trailing whitespace test is vacuous")
	}

	rawSHA := sha256Hex(rawBytes)
	trimmedSHA := sha256Hex(trimmedBytes)
	if rawSHA == trimmedSHA {
		t.Fatalf("raw and trimmed SHA-256 must differ (raw=%s trimmed=%s)", rawSHA, trimmedSHA)
	}

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         manifestPath,
	}
	// Pre-publication assertion: the manifest path MUST be
	// absent before invocation so the run actually publishes.
	if _, err := os.Stat(manifestPath); err == nil {
		t.Fatalf("manifest path must be absent before invocation: %s", manifestPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected manifest stat error: %v", err)
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.PlanBlob != blobOID {
		t.Fatalf("manifest.plan_blob: got=%s want=%s", manifest.PlanBlob, blobOID)
	}
	if manifest.PlanSHA256 != rawSHA {
		t.Fatalf("manifest.plan_sha256: got=%s want=%s", manifest.PlanSHA256, rawSHA)
	}
	if manifest.PlanSHA256 == trimmedSHA {
		t.Fatalf("manifest.plan_sha256 must not equal trimmed SHA-256 (both %s)", manifest.PlanSHA256)
	}
	wantSum := sha256.Sum256(rawBytes)
	if manifest.PlanSHA256 != hex.EncodeToString(wantSum[:]) {
		t.Fatalf("manifest.plan_sha256 disagrees with raw-byte SHA-256")
	}
}

// buildR2CRNewlineFixture returns a contract-valid Plan
// Contract v1 document that ends with exactly one trailing
// newline so the F:P blob satisfies the trailing-newline
// regression requirement. The supplied subject / tree OIDs
// are bound into the baseline so the runner accepts the
// fixture.
func buildR2CRNewlineFixture(t *testing.T, subject, subjectTree string) []byte {
	t.Helper()
	doc := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2CR-NEWLINE",
		"baseline":         map[string]string{"commit_oid": subject, "tree_oid": subjectTree},
		"execution":        map[string]string{"mode": "serial_fail_fast"},
		"checks": []map[string]any{{
			"id":                "noop",
			"mode":              "run",
			"argv":              []string{"true"},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(raw, '\n')
}

// TestR2CRDogfoodPlanSHACoverLeadingAndTrailingWhitespace
// proves the byte-authority contract also covers leading
// whitespace and trailing spaces. The test constructs the
// S < F topology explicitly: subject first, freeze as the
// child. The leading/trailing whitespace must round-trip
// through the byte-authority path unchanged.
func TestR2CRDogfoodPlanSHACoverLeadingAndTrailingWhitespace(t *testing.T) {
	dir := initRepo(t)
	planPath := "docs/closure-plans/WS.json"
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes := buildR2CRWhitespaceFixture(t, subject, subjectTree)
	freeze := makeCommit(t, dir, "freeze: whitespace fixture", map[string]string{
		planPath: string(frozenBytes),
	})
	blobOID := mustRunGit(t, dir, "rev-parse", freeze+":"+planPath)
	rawBytes, err := runR2CRGitRaw(context.Background(), dir, blobOID)
	if err != nil {
		t.Fatalf("runR2CRGitRaw: %v", err)
	}
	if !bytes.Equal(rawBytes, frozenBytes) {
		t.Fatalf("raw bytes disagree with fixture: %d vs %d", len(rawBytes), len(frozenBytes))
	}
	wantSHA := sha256Hex(rawBytes)
	manifestPath := filepath.Join(t.TempDir(), "manifest-ws.json")
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               planPath,
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         manifestPath,
	}
	// Pre-publication assertion: the manifest path MUST be
	// absent before invocation so the run actually publishes.
	if _, err := os.Stat(manifestPath); err == nil {
		t.Fatalf("manifest path must be absent before invocation: %s", manifestPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected manifest stat error: %v", err)
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), req)
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.PlanSHA256 != wantSHA {
		t.Fatalf("manifest.plan_sha256: got=%s want=%s", manifest.PlanSHA256, wantSHA)
	}
	if rawBytes[0] != ' ' {
		t.Fatalf("fixture must start with space, got 0x%02x", rawBytes[0])
	}
	if rawBytes[len(rawBytes)-1] != ' ' {
		t.Fatalf("fixture must end with space, got 0x%02x", rawBytes[len(rawBytes)-1])
	}
}

// buildR2CRWhitespaceFixture returns JSON bytes that begin
// and end with a single space so the SHA-256 differs from
// the unmangled document. The supplied subject / tree OIDs
// are bound into the baseline so the runner accepts the
// fixture.
func buildR2CRWhitespaceFixture(t *testing.T, subject, subjectTree string) []byte {
	t.Helper()
	doc := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2CR-WS",
		"baseline":         map[string]string{"commit_oid": subject, "tree_oid": subjectTree},
		"execution":        map[string]string{"mode": "serial_fail_fast"},
		"checks": []map[string]any{{
			"id":                "noop",
			"mode":              "run",
			"argv":              []string{"true"},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(append([]byte(" "), raw...), ' ')
}
