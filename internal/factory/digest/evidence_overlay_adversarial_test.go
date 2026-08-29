// SPDX-License-Identifier: Apache-2.0

// Adversarial-test helpers for the C7/C8/C11 evidence overlay
// suites. Each suite (C7, C8, C11) lives in its own test file
// to keep every file LLM-friendly (<= 400 lines).
package digest

import (
	"os"
	"path/filepath"
	"testing"
)

// makeReporootDir returns a hermetic repo root for adversarial
// tests. The root is git-init'd with a single seeded commit so
// subsequent pathspec queries behave like production.
func makeReporootDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	mustGitNoErr(t, dir, "init", "-q", "--initial-branch=main")
	mustGitNoErr(t, dir, "config", "user.email", "test@example.com")
	mustGitNoErr(t, dir, "config", "user.name", "Test")
	mustGitNoErr(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	mustGitNoErr(t, dir, "add", "README")
	mustGitNoErr(t, dir, "commit", "-q", "-m", "seed")
	return dir, func() {}
}

// ignorePath appends a single ignore entry to repoRoot/.gitignore
// so git ls-files --others --ignored will surface subsequent
// writes under that path. The helper is idempotent on repeat
// entries.
func ignorePath(t *testing.T, repoRoot, entry string) {
	t.Helper()
	gi := filepath.Join(repoRoot, ".gitignore")
	body := []byte(entry + "\n")
	if existing, rErr := os.ReadFile(gi); rErr == nil {
		body = append(existing, body...)
	}
	if err := os.WriteFile(gi, body, 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
}

// collectFor runs the production collect path against root.
func collectFor(t *testing.T, root, pathspec string) EvidenceOverlay {
	t.Helper()
	roots, err := ValidateEvidenceRoots(root, []string{pathspec})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(root, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return overlay
}

// unsafeControlOpen is the test-only "what would an unsafe
// implementation see?" helper. It deliberately uses ordinary
// pathname-based open rather than os.Root, so a C7/C9/C11
// test can prove the laboratory can detect the leak the
// secure implementation prevents.
func unsafeControlOpen(t *testing.T, repoRoot, relPath string) (string, error) {
	t.Helper()
	full := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
