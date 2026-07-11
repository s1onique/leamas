package doctrine

import (
	"path/filepath"
	"testing"
)

// --- R5 chained symlink regression tests ---

// AGENTS.md -> safe/next, safe/next -> /outside/missing.md.
// Multi-hop chain ending outside the root must emit ECF010.
func TestCheckECF_PathEscape_Chained_FinalSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Remove AGENTS.md; the canonical fixture rebuilds it.
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))

	// safe/next -> /outside/missing.md
	safeDir := filepath.Join(tmpDir, "safe")
	mustMkdirAll(t, safeDir)
	mustSymlink(t, "/outside/missing.md", filepath.Join(safeDir, "next"))

	// AGENTS.md -> safe/next
	mustSymlink(t, "safe/next", filepath.Join(tmpDir, "AGENTS.md"))

	findings := CheckExecutableContractFirst(tmpDir)
	if got := countByKindPath(findings, ecf010, ecfAgentsMDFile); got != 1 {
		t.Errorf("expected exactly one ECF010 on AGENTS.md for chained escape, got %d (findings:", got)
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// .github -> in-root symlink -> outside target.
// Parent directory is itself an in-root symlink whose chain reaches
// outside the root.
func TestCheckECF_PathEscape_Chained_ParentSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Replace .github with an in-root symlink whose target is in-root.
	githubDir := filepath.Join(tmpDir, ".github")
	mustRemove(t, githubDir)

	// inside_dir -> /outside/missing-dir. Remove the directory the
	// canonical fixture created so the symlink can take its place.
	insideDir := filepath.Join(tmpDir, "inside_dir")
	mustRemove(t, insideDir)
	mustSymlink(t, "/outside/missing-dir", insideDir)

	// .github -> inside_dir (in-root relative link).
	mustSymlink(t, "inside_dir", githubDir)

	findings := CheckExecutableContractFirst(tmpDir)
	if got := countByKindPath(findings, ecf010, ecfCopilotFile); got != 1 {
		t.Errorf("expected exactly one ECF010 on copilot-instructions.md for chained parent escape, got %d (findings:", got)
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}

// Two-hop chain entirely inside the root: accepted; no ECF010/ECF011.
func TestCheckECF_PathEscape_Chained_InRoot(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// Override AGENTS.md to point at a safe/next chain inside root.
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))

	// real_agents.md with valid content.
	mustWriteFile(t, filepath.Join(tmpDir, "real_agents.md"), canonicalAgentInstruction)

	// safe/next -> real_agents.md (relative, in-root). safe/next does
	// not exist yet (mustMkdirAll only created safe/ itself).
	safeDir := filepath.Join(tmpDir, "safe")
	mustMkdirAll(t, safeDir)
	mustSymlink(t, "../real_agents.md", filepath.Join(safeDir, "next"))

	// AGENTS.md -> safe/next
	mustSymlink(t, "safe/next", filepath.Join(tmpDir, "AGENTS.md"))

	findings := CheckExecutableContractFirst(tmpDir)
	if findByKindPath(findings, ecf010, ecfAgentsMDFile) {
		t.Errorf("in-root two-hop chain must not produce ECF010")
	}
	if findByKindPath(findings, ecf011, ecfAgentsMDFile) {
		t.Errorf("in-root two-hop chain must not produce ECF011")
	}
}

// Symlink loop: AGENTS.md -> self. Must produce a deterministic
// finding (not hang).
func TestCheckECF_PathEscape_Chained_SymlinkLoop(t *testing.T) {
	tmpDir := t.TempDir()
	createCanonicalECFStructure(t, tmpDir)

	// AGENTS.md -> AGENTS.md (self-loop)
	mustRemove(t, filepath.Join(tmpDir, "AGENTS.md"))
	mustSymlink(t, "AGENTS.md", filepath.Join(tmpDir, "AGENTS.md"))

	// Must terminate without hanging. The expected classification is
	// ECF010 (the bounded resolver exceeds the hop limit) but ECF011
	// (the open fails to resolve) is also acceptable; what is not
	// acceptable is missing the failure or hanging.
	findings := CheckExecutableContractFirst(tmpDir)
	if !(findByKindPath(findings, ecf010, ecfAgentsMDFile) ||
		findByKindPath(findings, ecf011, ecfAgentsMDFile)) {
		t.Errorf("symlink loop must produce ECF010 or ECF011, got:")
		for _, f := range findings {
			t.Logf("  %s: %s: %s", f.Path, f.Kind, f.Message)
		}
	}
}
