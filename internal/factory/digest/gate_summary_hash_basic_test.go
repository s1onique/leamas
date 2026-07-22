// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
	"testing"
)

// requirePresent asserts that a section rendered successfully with present status.
func requirePresent(t *testing.T, section string) {
	t.Helper()
	if !strings.Contains(section, "source_status=present\n") {
		t.Fatalf("fixture did not render as present:\n%s", section)
	}
	if strings.Contains(section, "failure_stage=") {
		t.Fatalf("fixture unexpectedly rendered a failure:\n%s", section)
	}
}

// TestGateSummaryHashUsesExactRenderedSection proves that the SHA-256 hash is computed
// from exactly the rendered GATE_SUMMARY section via ComputeEvidenceHashes.
func TestGateSummaryHashUsesExactRenderedSection(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)

	section := buildGateSummarySection(tmpDir)
	requirePresent(t, section)

	wantHash := ComputeSectionHash(section)

	hashes := ComputeEvidenceHashes(
		"", "", "", "", "", "", "", section, "",
	)

	if hashes.GateSummarySHA256 != wantHash {
		t.Fatalf("GateSummarySHA256 = %q, want %q",
			hashes.GateSummarySHA256, wantHash)
	}

	if len(hashes.GateSummarySHA256) != 64 {
		t.Errorf("expected 64-char hex hash, got %d",
			len(hashes.GateSummarySHA256))
	}

	t.Logf("Section hash: %s", hashes.GateSummarySHA256)
}

// TestGateSummaryLifecycleChangesHash proves that changes to status fields alter the hash.
func TestGateSummaryLifecycleChangesHash(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-minimal.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	scopeFixture := strings.Replace(string(fixture),
		`"scope_status": "CLOSED"`, `"scope_status": "OPEN"`, 1)
	parentFixture := strings.Replace(string(fixture),
		`"parent_status": "CLOSED"`, `"parent_status": "OPEN"`, 1)

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, fixture)
	baseSection := buildGateSummarySection(tmpDir)
	requirePresent(t, baseSection)
	baseHash := ComputeSectionHash(baseSection)

	writeGateSummaryFile(t, tmpDir, []byte(scopeFixture))
	scopeSection := buildGateSummarySection(tmpDir)
	requirePresent(t, scopeSection)
	scopeHash := ComputeSectionHash(scopeSection)

	writeGateSummaryFile(t, tmpDir, []byte(parentFixture))
	parentSection := buildGateSummarySection(tmpDir)
	requirePresent(t, parentSection)
	parentHash := ComputeSectionHash(parentSection)

	if baseHash == scopeHash {
		t.Errorf("different scope_status should have different hash")
	}
	if baseHash == parentHash {
		t.Errorf("different parent_status should have different hash")
	}
}

// TestGateSummaryV1AndV2ProduceDifferentHashes proves v1 and v2 produce different hashes.
func TestGateSummaryV1AndV2ProduceDifferentHashes(t *testing.T) {
	t.Parallel()
	v1Fixture := `{
		"schema_version": 1,
		"generated_at": "2024-01-01T00:00:00Z",
		"overall_status": "pass",
		"checks": [
			{"name": "check", "status": "pass", "evidence": "check.sh"}
		]
	}`

	v2Fixture, err := readTestFixture("v2-minimal.json")
	if err != nil {
		t.Fatalf("failed to read v2 fixture: %v", err)
	}

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, []byte(v1Fixture))
	v1Section := buildGateSummarySection(tmpDir)
	requirePresent(t, v1Section)
	v1Hash := ComputeSectionHash(v1Section)

	writeGateSummaryFile(t, tmpDir, v2Fixture)
	v2Section := buildGateSummarySection(tmpDir)
	requirePresent(t, v2Section)
	v2Hash := ComputeSectionHash(v2Section)

	if v1Hash == v2Hash {
		t.Errorf("v1 and v2 should produce different hashes")
	}
}

// TestGateSummaryLifecycleOverallStatusChangesHash proves that overall_status changes the hash.
func TestGateSummaryLifecycleOverallStatusChangesHash(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-minimal.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	failFixture := strings.Replace(string(fixture),
		`"overall_status": "pass"`, `"overall_status": "fail"`, 1)
	failFixture = strings.Replace(failFixture,
		`"status": "pass"`, `"status": "fail"`, 1)
	failFixture = strings.Replace(failFixture,
		`"exit_code": 0`, `"exit_code": 1`, 1)

	tmpDir := t.TempDir()

	writeGateSummaryFile(t, tmpDir, fixture)
	passSection := buildGateSummarySection(tmpDir)
	requirePresent(t, passSection)
	passHash := ComputeSectionHash(passSection)

	writeGateSummaryFile(t, tmpDir, []byte(failFixture))
	failSection := buildGateSummarySection(tmpDir)
	requirePresent(t, failSection)
	failHash := ComputeSectionHash(failSection)

	if passHash == failHash {
		t.Errorf("different overall_status should produce different hash")
	}
}

// TestGateSummarySectionAppearsExactlyOnce proves the GATE_SUMMARY section
// appears exactly once in the output.
func TestGateSummarySectionAppearsExactlyOnce(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)

	count := strings.Count(section, "## GATE_SUMMARY")
	if count != 1 {
		t.Errorf("GATE_SUMMARY header should appear exactly once, got %d", count)
	}
}
