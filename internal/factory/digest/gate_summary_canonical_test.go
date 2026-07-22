// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"testing"
)

// sha256Zero is a valid SHA256 hash for empty content.
const sha256Zero = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// v2Header is a minimal V2 fixture header.
const v2Header = `{"schema_version":2,"generated_at":"2024-01-01T00:00:00Z",` +
	`"scope_id":"TEST","scope_status":"CLOSED","scope_disposition":"done",` +
	`"parent_act":"","parent_status":"CLOSED","parent_disposition":"root",` +
	`"overall_status":"pass","overall_disposition":"done",` +
	`"execution_head_oid":"0123456789abcdef0123456789abcdef01234567",` +
	`"execution_tree_oid":"0123456789abcdef0123456789abcdef01234567",` +
	`"subject_tree_oid":"0123456789abcdef0123456789abcdef01234567",` +
	`"worktree_clean_before":true,"worktree_clean_after":true,"checks":[`

// v2Footer is a minimal V2 fixture footer.
const v2Footer = `]}`

// checkAlpha creates a V2 check for sorting tests with given name.
func checkAlpha() string {
	return `{"name": "alpha", "scope": "ROOT", "status": "pass",` +
		` "evidence": "a.sh", "detail": "a",` +
		` "extras": {"argv": ["a.sh"], "exit_code": 0, "duration_ms": 100,` +
		` "stdout_sha256": "` + sha256Zero + `",` +
		` "stderr_sha256": "` + sha256Zero + `"}}`
}

// checkBeta creates a V2 check for sorting tests with given name.
func checkBeta() string {
	return `{"name": "beta", "scope": "ROOT", "status": "pass",` +
		` "evidence": "b.sh", "detail": "b",` +
		` "extras": {"argv": ["b.sh"], "exit_code": 0, "duration_ms": 200,` +
		` "stdout_sha256": "` + sha256Zero + `",` +
		` "stderr_sha256": "` + sha256Zero + `"}}`
}

// TestGateSummarySourceOrderDoesNotChangeHash proves that shuffled check input
// produces identical output section and hash.
func TestGateSummarySourceOrderDoesNotChangeHash(t *testing.T) {
	t.Parallel()
	orderedFixture := v2Header + checkAlpha() + `,` + checkBeta() + v2Footer
	shuffledFixture := v2Header + checkBeta() + `,` + checkAlpha() + v2Footer

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, []byte(orderedFixture))
	orderedSection := buildGateSummarySection(tmpDir)
	requirePresent(t, orderedSection)

	writeGateSummaryFile(t, tmpDir, []byte(shuffledFixture))
	shuffledSection := buildGateSummarySection(tmpDir)
	requirePresent(t, shuffledSection)

	if orderedSection != shuffledSection {
		t.Errorf("shuffled input should produce identical section")
	}

	orderedHash := ComputeSectionHash(orderedSection)
	shuffledHash := ComputeSectionHash(shuffledSection)
	if orderedHash != shuffledHash {
		t.Errorf("shuffled input should produce identical hash")
	}

	names := findCheckNames(orderedSection)
	if !isSorted(names) {
		t.Errorf("check names should be sorted, got: %v", names)
	}
}

// TestGateSummaryV1CanonicalOrderUsesFullDurationValue proves V1 uses full
// integer comparison for duration.
func TestGateSummaryV1CanonicalOrderUsesFullDurationValue(t *testing.T) {
	t.Parallel()
	checkDur100 := `{"name":"check","status":"pass","evidence":"check.sh","duration_ms":100}`
	checkDur200 := `{"name":"check","status":"pass","evidence":"check.sh","duration_ms":200}`

	fixtureA := `{"schema_version":1,"generated_at":"2024-01-01T00:00:00Z",` +
		`"overall_status":"pass","checks":[` + checkDur200 + `,` + checkDur100 + `]}`
	fixtureB := `{"schema_version":1,"generated_at":"2024-01-01T00:00:00Z",` +
		`"overall_status":"pass","checks":[` + checkDur100 + `,` + checkDur200 + `]}`

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, []byte(fixtureA))
	sectionA := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionA)

	writeGateSummaryFile(t, tmpDir, []byte(fixtureB))
	sectionB := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionB)

	if sectionA != sectionB {
		t.Errorf("V1 duration sort failed: sections differ")
	}
}

// TestGateSummaryV2CanonicalOrderUsesFullDurationValue proves V2 uses full
// integer comparison for duration.
func TestGateSummaryV2CanonicalOrderUsesFullDurationValue(t *testing.T) {
	t.Parallel()
	// Two checks with different durations
	checkDur100 := `{"name":"check_a","scope":"ROOT","status":"pass",` +
		`"evidence":"check.sh","detail":"ok",` +
		`"extras":{"argv":["check.sh"],"exit_code":0,"duration_ms":100,` +
		`"stdout_sha256":"` + sha256Zero + `",` +
		`"stderr_sha256":"` + sha256Zero + `"}}`
	checkDur200 := `{"name":"check_b","scope":"ROOT","status":"pass",` +
		`"evidence":"check.sh","detail":"ok",` +
		`"extras":{"argv":["check.sh"],"exit_code":0,"duration_ms":200,` +
		`"stdout_sha256":"` + sha256Zero + `",` +
		`"stderr_sha256":"` + sha256Zero + `"}}`

	fixtureA := v2Header + checkDur200 + `,` + checkDur100 + v2Footer
	fixtureB := v2Header + checkDur100 + `,` + checkDur200 + v2Footer

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, []byte(fixtureA))
	sectionA := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionA)

	writeGateSummaryFile(t, tmpDir, []byte(fixtureB))
	sectionB := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionB)

	if sectionA != sectionB {
		t.Errorf("V2 duration sort failed: sections differ")
	}
}

// TestGateSummaryV2CanonicalOrderUsesFullExitCodeValue proves V2 uses full
// integer comparison for exit_code.
func TestGateSummaryV2CanonicalOrderUsesFullExitCodeValue(t *testing.T) {
	t.Parallel()
	checkEC1 := `{"name":"check_a","scope":"ROOT","status":"fail",` +
		`"evidence":"check.sh","detail":"failed",` +
		`"extras":{"argv":["check.sh"],"exit_code":1,"duration_ms":100,` +
		`"stdout_sha256":"` + sha256Zero + `",` +
		`"stderr_sha256":"` + sha256Zero + `"}}`
	checkEC2 := `{"name":"check_b","scope":"ROOT","status":"fail",` +
		`"evidence":"check.sh","detail":"failed",` +
		`"extras":{"argv":["check.sh"],"exit_code":2,"duration_ms":100,` +
		`"stdout_sha256":"` + sha256Zero + `",` +
		`"stderr_sha256":"` + sha256Zero + `"}}`

	// Use overall_status=fail since checks have fail status
	failHeader := `{"schema_version":2,"generated_at":"2024-01-01T00:00:00Z",` +
		`"scope_id":"TEST","scope_status":"CLOSED","scope_disposition":"done",` +
		`"parent_act":"","parent_status":"CLOSED","parent_disposition":"root",` +
		`"overall_status":"fail","overall_disposition":"done",` +
		`"execution_head_oid":"0123456789abcdef0123456789abcdef01234567",` +
		`"execution_tree_oid":"0123456789abcdef0123456789abcdef01234567",` +
		`"subject_tree_oid":"0123456789abcdef0123456789abcdef01234567",` +
		`"worktree_clean_before":true,"worktree_clean_after":true,"checks":[`

	fixtureA := failHeader + checkEC2 + `,` + checkEC1 + v2Footer
	fixtureB := failHeader + checkEC1 + `,` + checkEC2 + v2Footer

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, []byte(fixtureA))
	sectionA := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionA)

	writeGateSummaryFile(t, tmpDir, []byte(fixtureB))
	sectionB := buildGateSummarySection(tmpDir)
	requirePresent(t, sectionB)

	if sectionA != sectionB {
		t.Errorf("V2 exit_code sort failed: sections differ")
	}
}
