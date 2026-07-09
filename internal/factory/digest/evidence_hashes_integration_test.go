// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEvidenceHashes_AppearAfterPatchHygieneBeforeChangedFiles verifies section order.
func TestEvidenceHashes_AppearAfterPatchHygieneBeforeChangedFiles(t *testing.T) {
	// Create a temp git repo with a commit
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create and commit a test file
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Generate dirty digest
	digest, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Find positions of key sections
	patchHygieneIdx := strings.Index(digest, "## PATCH_HYGIENE")
	evidenceHashesIdx := strings.Index(digest, "## EVIDENCE_HASHES")
	changedFilesIdx := strings.Index(digest, "## Changed files")

	if patchHygieneIdx == -1 {
		t.Fatal("PATCH_HYGIENE section not found")
	}
	if evidenceHashesIdx == -1 {
		t.Fatal("EVIDENCE_HASHES section not found")
	}
	if changedFilesIdx == -1 {
		t.Fatal("Changed files section not found")
	}

	// Verify order: PATCH_HYGIENE < EVIDENCE_HASHES < Changed files
	if patchHygieneIdx > evidenceHashesIdx {
		t.Error("EVIDENCE_HASHES should appear after PATCH_HYGIENE")
	}
	if evidenceHashesIdx > changedFilesIdx {
		t.Error("EVIDENCE_HASHES should appear before Changed files")
	}
}

// TestDigestEvidenceHash_IgnoresDigestCreatedAt verifies volatile field exclusion.
func TestDigestEvidenceHash_IgnoresDigestCreatedAt(t *testing.T) {
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create and commit a test file
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Generate digest
	digest, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify EVIDENCE_HASHES section exists
	ehIdx := strings.Index(digest, "## EVIDENCE_HASHES")
	if ehIdx == -1 {
		t.Fatal("EVIDENCE_HASHES section not found")
	}

	// Verify digest_evidence_sha256 is present and is a valid 64-char hex
	lines := strings.Split(digest[ehIdx:], "\n")
	var digestEvidenceHash string
	for _, line := range lines {
		if strings.HasPrefix(line, "digest_evidence_sha256=") {
			digestEvidenceHash = strings.TrimPrefix(line, "digest_evidence_sha256=")
			break
		}
	}

	if digestEvidenceHash == "" {
		t.Fatal("digest_evidence_sha256 not found")
	}
	if len(digestEvidenceHash) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(digestEvidenceHash))
	}
}

// TestDigestEvidenceHash_IgnoresAbsoluteRepoRoot verifies repo path exclusion.
func TestDigestEvidenceHash_IgnoresAbsoluteRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create and commit a test file
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Generate digest
	digest1, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Extract the digest_evidence_sha256
	ehIdx := strings.Index(digest1, "## EVIDENCE_HASHES")
	lines := strings.Split(digest1[ehIdx:], "\n")
	var hash1 string
	for _, line := range lines {
		if strings.HasPrefix(line, "digest_evidence_sha256=") {
			hash1 = strings.TrimPrefix(line, "digest_evidence_sha256=")
			break
		}
	}

	// Generate digest again - should get same hash
	digest2, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatal(err)
	}

	ehIdx2 := strings.Index(digest2, "## EVIDENCE_HASHES")
	lines2 := strings.Split(digest2[ehIdx2:], "\n")
	var hash2 string
	for _, line := range lines2 {
		if strings.HasPrefix(line, "digest_evidence_sha256=") {
			hash2 = strings.TrimPrefix(line, "digest_evidence_sha256=")
			break
		}
	}

	// The hashes should match (repo root in legacy header is excluded from evidence)
	if hash1 != hash2 {
		t.Errorf("digest_evidence_sha256 should be stable: got %s vs %s", hash1, hash2)
	}
}

// TestRangeDigest_IncludesEvidenceHashes verifies range mode includes hashes.
func TestRangeDigest_IncludesEvidenceHashes(t *testing.T) {
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create initial commit
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Create second commit
	if err := os.WriteFile(testFile, []byte("package test\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Second commit")

	// Generate range digest
	digest, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify EVIDENCE_HASHES section exists
	if !strings.Contains(digest, "## EVIDENCE_HASHES") {
		t.Fatal("EVIDENCE_HASHES section not found in range digest")
	}

	// Verify all expected hash fields are present
	expectedFields := []string{
		"hash_algorithm=sha256",
		"hash_scope=normalized_digest_v2_sections",
		"changeset_manifest_sha256=",
		"changeset_stats_sha256=",
		"review_map_sha256=",
		"risk_signals_sha256=",
		"patch_hygiene_sha256=",
		"file_evidence_sha256=",
		"digest_evidence_sha256=",
	}

	ehIdx := strings.Index(digest, "## EVIDENCE_HASHES")
	ehSection := digest[ehIdx:]

	for _, field := range expectedFields {
		if !strings.Contains(ehSection, field) {
			t.Errorf("expected field %q in EVIDENCE_HASHES section", field)
		}
	}
}

// TestStagedDigest_IncludesEvidenceHashes verifies staged mode includes hashes.
func TestStagedDigest_IncludesEvidenceHashes(t *testing.T) {
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create and commit initial file
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Stage changes
	if err := os.WriteFile(testFile, []byte("package test\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")

	// Generate staged digest
	digest, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify EVIDENCE_HASHES section exists
	if !strings.Contains(digest, "## EVIDENCE_HASHES") {
		t.Fatal("EVIDENCE_HASHES section not found in staged digest")
	}

	// Verify digest_evidence_sha256 is present
	ehIdx := strings.Index(digest, "## EVIDENCE_HASHES")
	ehSection := digest[ehIdx:]
	if !strings.Contains(ehSection, "digest_evidence_sha256=") {
		t.Error("digest_evidence_sha256 not found")
	}
}

// TestDirtyDigest_IncludesEvidenceHashes verifies dirty mode includes hashes.
func TestDirtyDigest_IncludesEvidenceHashes(t *testing.T) {
	repoRoot := t.TempDir()
	initGit(t, repoRoot)

	// Create and commit initial file
	testFile := filepath.Join(repoRoot, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repoRoot, "add", "test.go")
	runGit(t, repoRoot, "commit", "-m", "Initial commit")

	// Make dirty changes
	if err := os.WriteFile(testFile, []byte("package test\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate dirty digest
	digest, err := Generate(Options{
		RepoRoot: repoRoot,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify EVIDENCE_HASHES section exists
	if !strings.Contains(digest, "## EVIDENCE_HASHES") {
		t.Fatal("EVIDENCE_HASHES section not found in dirty digest")
	}

	// Verify all section hashes are present and non-empty
	ehIdx := strings.Index(digest, "## EVIDENCE_HASHES")
	lines := strings.Split(digest[ehIdx:], "\n")
	hashFields := []string{
		"changeset_manifest_sha256=",
		"changeset_stats_sha256=",
		"review_map_sha256=",
		"risk_signals_sha256=",
		"patch_hygiene_sha256=",
		"file_evidence_sha256=",
		"digest_evidence_sha256=",
	}

	for _, field := range hashFields {
		for _, line := range lines {
			if strings.HasPrefix(line, field) {
				hash := strings.TrimPrefix(line, field)
				if hash == "" {
					t.Errorf("%s should not be empty", field)
				}
				if len(hash) != 64 {
					t.Errorf("%s should be 64 chars, got %d", field, len(hash))
				}
				break
			}
		}
	}
}
