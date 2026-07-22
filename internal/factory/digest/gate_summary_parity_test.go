// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGateSummaryRenderDigestWithResolvedParity proves that RenderDigestWithResolved
// produces the same GATE_SUMMARY section and hash as RenderDigest.
func TestGateSummaryRenderDigestWithResolvedParity(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Install v2-minimal artifact so we test actual adapter
	installGateSummaryArtifact(t, tmpDir, "v2-minimal.json")

	file := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(file, []byte("content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	files, err := GetDirtyFiles(tmpDir)
	if err != nil {
		t.Fatalf("failed to get dirty files: %v", err)
	}

	dirtyDigest, err := RenderDigest(ModeDirty, tmpDir, files)
	if err != nil {
		t.Fatalf("RenderDigest failed: %v", err)
	}

	resolved := &ResolvedMode{Mode: ModeDirty, Reason: "test"}
	resolvedDigest, err := RenderDigestWithResolved(
		ModeDirty, tmpDir, files, resolved, true)
	if err != nil {
		t.Fatalf("RenderDigestWithResolved failed: %v", err)
	}

	dirtySection := extractSection(dirtyDigest, "GATE_SUMMARY")
	resolvedSection := extractSection(resolvedDigest, "GATE_SUMMARY")

	// Assert present schema-v2 sections
	if !strings.Contains(dirtySection, "schema_version=2\n") {
		t.Fatalf("dirty section did not exercise schema v2:\n%s", dirtySection)
	}
	if !strings.Contains(resolvedSection, "schema_version=2\n") {
		t.Fatalf("resolved section did not exercise schema v2:\n%s", resolvedSection)
	}

	if dirtySection != resolvedSection {
		t.Errorf("RenderDigest and RenderDigestWithResolved " +
			"should produce identical GATE_SUMMARY")
	}

	// Assert non-empty exact hashes
	dirtyHash := evidenceHash(dirtyDigest, "gate_summary_sha256")
	resolvedHash := evidenceHash(resolvedDigest, "gate_summary_sha256")
	if dirtyHash == "" {
		t.Fatal("gate_summary_sha256 missing in dirty digest")
	}
	if resolvedHash == "" {
		t.Fatal("gate_summary_sha256 missing in resolved digest")
	}
	if dirtyHash != resolvedHash {
		t.Errorf("GATE_SUMMARY hashes should match: dirty=%s resolved=%s",
			dirtyHash, resolvedHash)
	}
}

// TestGateSummaryRenderRangeDigestWithResolvedParity proves that
// RenderRangeDigestWithResolved produces the same GATE_SUMMARY section and
// hash as RenderRangeDigest.
func TestGateSummaryRenderRangeDigestWithResolvedParity(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Install v2-minimal artifact so we test actual adapter
	installGateSummaryArtifact(t, tmpDir, "v2-minimal.json")

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("first\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "first")

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("second\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "-m", "second")

	rangeStr := "HEAD~1..HEAD"
	files, err := GetRangeFiles(tmpDir, rangeStr)
	if err != nil {
		t.Fatalf("failed to get range files: %v", err)
	}

	rangeDigest, err := RenderRangeDigest(tmpDir, files, rangeStr)
	if err != nil {
		t.Fatalf("RenderRangeDigest failed: %v", err)
	}

	resolved := &ResolvedMode{
		Mode:   ModeRange,
		Range:  rangeStr,
		Reason: "explicit range mode",
	}
	resolvedDigest, err := RenderRangeDigestWithResolved(tmpDir, files, resolved)
	if err != nil {
		t.Fatalf("RenderRangeDigestWithResolved failed: %v", err)
	}

	rangeSection := extractSection(rangeDigest, "GATE_SUMMARY")
	resolvedSection := extractSection(resolvedDigest, "GATE_SUMMARY")

	// Assert present schema-v2 sections
	if !strings.Contains(rangeSection, "schema_version=2\n") {
		t.Fatalf("range section did not exercise schema v2:\n%s", rangeSection)
	}
	if !strings.Contains(resolvedSection, "schema_version=2\n") {
		t.Fatalf("resolved section did not exercise schema v2:\n%s", resolvedSection)
	}

	if rangeSection != resolvedSection {
		t.Errorf("RenderRangeDigest and RenderRangeDigestWithResolved " +
			"should produce identical GATE_SUMMARY")
	}

	// Assert non-empty exact hashes
	rangeHash := evidenceHash(rangeDigest, "gate_summary_sha256")
	resolvedHash := evidenceHash(resolvedDigest, "gate_summary_sha256")
	if rangeHash == "" {
		t.Fatal("gate_summary_sha256 missing in range digest")
	}
	if resolvedHash == "" {
		t.Fatal("gate_summary_sha256 missing in resolved digest")
	}
	if rangeHash != resolvedHash {
		t.Errorf("GATE_SUMMARY hashes should match: range=%s resolved=%s",
			rangeHash, resolvedHash)
	}

	// Bind hash to rendered section
	wantHash := ComputeSectionHash(rangeSection)
	if rangeHash != wantHash {
		t.Fatalf("range gate_summary_sha256 = %q, want %q",
			rangeHash, wantHash)
	}
	if resolvedHash != ComputeSectionHash(resolvedSection) {
		t.Fatalf("resolved range hash does not match rendered section")
	}
}
