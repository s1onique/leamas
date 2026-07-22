// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGateSummaryDigestModeParity proves that gate_summary section is identical
// across dirty, staged/resolved, and range rendering modes.
func TestGateSummaryDigestModeParity(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Initialize git repo
	initGit(t, tmpDir)

	// Install v2-minimal artifact
	installGateSummaryArtifact(t, tmpDir, "v2-minimal.json")

	// Render dirty digest
	dirtyDigest, err := RenderDigest(ModeDirty, tmpDir, nil)
	if err != nil {
		t.Fatalf("dirty render failed: %v", err)
	}
	dirtySection := extractSection(dirtyDigest, "GATE_SUMMARY")

	// Render staged digest
	stagedDigest, err := RenderDigest(ModeStaged, tmpDir, nil)
	if err != nil {
		t.Fatalf("staged render failed: %v", err)
	}
	stagedSection := extractSection(stagedDigest, "GATE_SUMMARY")

	// Render range digest
	rangeDigest, err := RenderDigest(ModeRange, tmpDir, nil)
	if err != nil {
		t.Fatalf("range render failed: %v", err)
	}
	rangeSection := extractSection(rangeDigest, "GATE_SUMMARY")

	// All modes must produce identical GATE_SUMMARY section
	if dirtySection != stagedSection {
		t.Errorf("dirty and staged sections differ")
	}
	if dirtySection != rangeSection {
		t.Errorf("dirty and range sections differ")
	}

	t.Logf("GATE_SUMMARY section length: %d bytes", len(dirtySection))
}

// TestGateSummarySectionOrdering proves that sections appear in correct order
// with GATE_SUMMARY after EVIDENCE_HASHES and before PUBLIC_SURFACE_DELTA.
func TestGateSummarySectionOrdering(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Initialize git repo
	initGit(t, tmpDir)

	// Install v2-minimal artifact
	installGateSummaryArtifact(t, tmpDir, "v2-minimal.json")

	// Render digest
	digestText, err := RenderDigest(ModeDirty, tmpDir, nil)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Find positions of key sections
	ehIdx := strings.Index(digestText, "## EVIDENCE_HASHES")
	gsIdx := strings.Index(digestText, "## GATE_SUMMARY")
	psIdx := strings.Index(digestText, "## PUBLIC_SURFACE_DELTA")
	ddIdx := strings.Index(digestText, "## DEPENDENCY_DELTA")

	if ehIdx == -1 {
		t.Fatal("EVIDENCE_HASHES section missing")
	}
	if gsIdx == -1 {
		t.Fatal("GATE_SUMMARY section missing")
	}
	if psIdx == -1 {
		t.Fatal("PUBLIC_SURFACE_DELTA section missing")
	}
	if ddIdx == -1 {
		t.Fatal("DEPENDENCY_DELTA section missing")
	}

	// Verify order: EVIDENCE_HASHES < GATE_SUMMARY < PUBLIC_SURFACE_DELTA < DEPENDENCY_DELTA
	if !(ehIdx < gsIdx && gsIdx < psIdx && psIdx < ddIdx) {
		t.Errorf("section order incorrect: EVIDENCE_HASHES=%d, GATE_SUMMARY=%d, PUBLIC_SURFACE_DELTA=%d, DEPENDENCY_DELTA=%d",
			ehIdx, gsIdx, psIdx, ddIdx)
	}

	t.Logf("Section order verified: EVIDENCE_HASHES(%d) < GATE_SUMMARY(%d) < PUBLIC_SURFACE_DELTA(%d) < DEPENDENCY_DELTA(%d)",
		ehIdx, gsIdx, psIdx, ddIdx)
}

// TestGateSummarySectionAppearsExactlyOnceInDigest proves the GATE_SUMMARY heading
// appears exactly once in the full digest output.
func TestGateSummarySectionAppearsExactlyOnceInDigest(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Initialize git repo
	initGit(t, tmpDir)

	// Install v2-full artifact
	installGateSummaryArtifact(t, tmpDir, "v2-full.json")

	// Render digest
	digestText, err := RenderDigest(ModeDirty, tmpDir, nil)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Count occurrences of GATE_SUMMARY heading
	count := strings.Count(digestText, "## GATE_SUMMARY")
	if count != 1 {
		t.Errorf("GATE_SUMMARY heading should appear exactly once, got %d", count)
	}
}

// TestGateSummaryRenderedHashMatchesDigest proves that the gate_summary_sha256
// in EVIDENCE_HASHES matches ComputeSectionHash of the actual GATE_SUMMARY section.
func TestGateSummaryRenderedHashMatchesDigest(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Initialize git repo
	initGit(t, tmpDir)

	// Install v2-minimal artifact
	installGateSummaryArtifact(t, tmpDir, "v2-minimal.json")

	// Render digest
	digestText, err := RenderDigest(ModeDirty, tmpDir, nil)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Extract GATE_SUMMARY section
	gsSection := extractSection(digestText, "GATE_SUMMARY")

	// Extract hash from EVIDENCE_HASHES
	reportedHash := evidenceHash(digestText, "gate_summary_sha256")
	if reportedHash == "" {
		t.Fatal("gate_summary_sha256 not found in EVIDENCE_HASHES")
	}

	// Compute expected hash
	expectedHash := ComputeSectionHash(gsSection)

	// Verify match
	if reportedHash != expectedHash {
		t.Errorf("gate_summary_sha256 mismatch: in digest=%q, computed=%q", reportedHash, expectedHash)
	}

	t.Logf("Hash verified: %s", reportedHash)
}

// installGateSummaryArtifact creates the .factory directory and installs
// the specified gate-summary.json fixture.
func installGateSummaryArtifact(t *testing.T, repoRoot, fixtureName string) {
	t.Helper()

	fixture, err := readTestFixture(fixtureName)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixtureName, err)
	}

	factoryDir := filepath.Join(repoRoot, ".factory")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatalf("failed to create .factory directory: %v", err)
	}

	gsPath := filepath.Join(factoryDir, "gate-summary.json")
	if err := os.WriteFile(gsPath, fixture, 0644); err != nil {
		t.Fatalf("failed to write gate-summary.json: %v", err)
	}
}
